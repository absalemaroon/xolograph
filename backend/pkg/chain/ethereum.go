package chain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	maxTokenMeta  = 8 // contract metadata lookups per transaction
)

type tokenMeta struct {
	symbol   string
	decimals int
}

// Ethereum combines a JSON-RPC node (authoritative state) with an optional
// Blockscout instance (address history and third-party labels).
type Ethereum struct {
	net    string // network id
	symbol string // native currency symbol
	rpc    *RPC
	bs     *Blockscout
	mu     sync.Mutex
	tokens map[string]tokenMeta
}

func NewEthereum(rpc *RPC, bs *Blockscout) *Ethereum {
	return NewEVM("ethereum-mainnet", "ETH", rpc, bs)
}

// NewEVM builds an adapter for any EVM network (JSON-RPC for state, optional Blockscout for history and labels).
func NewEVM(network, nativeSymbol string, rpc *RPC, bs *Blockscout) *Ethereum {
	if bs != nil {
		bs.symbol = nativeSymbol
	}
	return &Ethereum{net: network, symbol: nativeSymbol, rpc: rpc, bs: bs, tokens: map[string]tokenMeta{}}
}

func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
	rs := []rune(s)
	if len(rs) > 32 {
		rs = rs[:32]
	}
	return string(rs)
}

func (e *Ethereum) Address(ctx context.Context, addr string) (*AddressSummary, error) {
	a, err := NormalizeAddress(addr)
	if err != nil {
		return nil, err
	}
	s := &AddressSummary{Address: a, Network: e.net, NativeSymbol: e.symbol, NativeDecimals: 18, ThirdPartyLabels: []ThirdPartyLabel{}, Notes: []string{}, Provenance: []Provenance{}}

	raw, p, err := e.rpc.Call(ctx, "eth_getBalance", a, "latest")
	if err != nil {
		return nil, err
	}
	s.BalanceWei = hexBig(unquote(raw)).String()
	s.Provenance = append(s.Provenance, p)

	raw, p, err = e.rpc.Call(ctx, "eth_getTransactionCount", a, "latest")
	if err != nil {
		return nil, err
	}
	s.Nonce = hexUint(unquote(raw))
	s.Provenance = append(s.Provenance, p)

	raw, p, err = e.rpc.Call(ctx, "eth_getCode", a, "latest")
	if err != nil {
		return nil, err
	}
	code := unquote(raw)
	s.IsContract = code != "" && code != "0x" && code != "0x0"
	s.Provenance = append(s.Provenance, p)

	if e.bs != nil {
		labels, lp, err := e.bs.AddressLabels(ctx, a)
		switch {
		case err == nil:
			s.ThirdPartyLabels = labels
			s.Provenance = append(s.Provenance, lp)
		case !errors.Is(err, ErrNotFound):
			s.Notes = append(s.Notes, "Third-party labels unavailable: "+err.Error())
		}
	}
	return s, nil
}

func (e *Ethereum) Transaction(ctx context.Context, hash string) (*Transaction, error) {
	h, err := NormalizeHash(hash)
	if err != nil {
		return nil, err
	}
	raw, p, err := e.rpc.Call(ctx, "eth_getTransactionByHash", h)
	if err != nil {
		return nil, err
	}
	provs := []Provenance{p}
	if string(raw) == "null" {
		return nil, fmt.Errorf("%w: transaction %s", ErrNotFound, h)
	}
	var tx struct {
		Hash        string  `json:"hash"`
		BlockNumber *string `json:"blockNumber"`
		BlockHash   *string `json:"blockHash"`
		From        string  `json:"from"`
		To          *string `json:"to"`
		Value       string  `json:"value"`
		GasPrice    string  `json:"gasPrice"`
	}
	if err := json.Unmarshal(raw, &tx); err != nil {
		return nil, fmt.Errorf("%w: malformed transaction payload", ErrUpstream)
	}
	out := &Transaction{
		Hash: strings.ToLower(tx.Hash), Network: e.net, NativeSymbol: e.symbol, NativeDecimals: 18, Status: "PENDING",
		From: strings.ToLower(tx.From), ValueWei: hexBig(tx.Value).String(),
		TokenTransfers: []TokenTransfer{}, Notes: []string{},
	}
	if tx.To != nil {
		out.To = strings.ToLower(*tx.To)
	}
	if tx.BlockNumber == nil {
		out.GasPriceWei, out.FeeWei = hexBig(tx.GasPrice).String(), "0"
		out.Notes = append(out.Notes, "Transaction is pending; it has no receipt or block yet.")
		out.Provenance = provs
		return out, nil
	}
	out.BlockNumber = hexUint(*tx.BlockNumber)
	if tx.BlockHash != nil {
		out.BlockHash = strings.ToLower(*tx.BlockHash)
	}

	rraw, p, err := e.rpc.Call(ctx, "eth_getTransactionReceipt", h)
	if err != nil {
		return nil, err
	}
	provs = append(provs, p)
	var rc struct {
		Status            string  `json:"status"`
		GasUsed           string  `json:"gasUsed"`
		EffectiveGasPrice string  `json:"effectiveGasPrice"`
		ContractAddress   *string `json:"contractAddress"`
		Logs              []struct {
			Address  string   `json:"address"`
			Topics   []string `json:"topics"`
			Data     string   `json:"data"`
			LogIndex string   `json:"logIndex"`
		} `json:"logs"`
	}
	if string(rraw) == "null" || json.Unmarshal(rraw, &rc) != nil {
		return nil, fmt.Errorf("%w: receipt unavailable", ErrUpstream)
	}
	switch rc.Status {
	case "0x1":
		out.Status = "SUCCESS"
	case "0x0":
		out.Status = "FAILED"
	default:
		out.Status = "UNKNOWN"
	}
	out.GasUsed = hexUint(rc.GasUsed)
	price := hexBig(rc.EffectiveGasPrice)
	if rc.EffectiveGasPrice == "" {
		price = hexBig(tx.GasPrice)
	}
	out.GasPriceWei = price.String()
	out.FeeWei = new(big.Int).Mul(new(big.Int).SetUint64(out.GasUsed), price).String()
	if rc.ContractAddress != nil {
		out.ContractCreated = strings.ToLower(*rc.ContractAddress)
	}

	braw, p, err := e.rpc.Call(ctx, "eth_getBlockByNumber", *tx.BlockNumber, false)
	if err != nil {
		return nil, err
	}
	provs = append(provs, p)
	var blk struct {
		Timestamp string `json:"timestamp"`
	}
	if json.Unmarshal(braw, &blk) == nil && blk.Timestamp != "" {
		t := time.Unix(int64(hexUint(blk.Timestamp)), 0).UTC()
		out.Timestamp = &t
	}

	lookups := 0
	for _, lg := range rc.Logs {
		if len(lg.Topics) < 3 || strings.ToLower(lg.Topics[0]) != transferTopic {
			continue
		}
		tt := TokenTransfer{
			LogIndex: int64(hexUint(lg.LogIndex)), Token: strings.ToLower(lg.Address),
			From: topicAddr(lg.Topics[1]), To: topicAddr(lg.Topics[2]), Decimals: -1,
		}
		switch len(lg.Topics) {
		case 3:
			tt.Standard, tt.Value = "ERC20", hexBig(lg.Data).String()
			if lookups < maxTokenMeta {
				lookups++
				m, mp := e.tokenInfo(ctx, tt.Token)
				provs = append(provs, mp...)
				tt.Symbol, tt.Decimals = m.symbol, m.decimals
			}
		case 4:
			tt.Standard, tt.Value, tt.TokenID = "ERC721", "1", hexBig(lg.Topics[3]).String()
		default:
			continue
		}
		out.TokenTransfers = append(out.TokenTransfers, tt)
	}
	if lookups > 0 {
		out.Notes = append(out.Notes, "Token symbols/decimals are read from the token contract and are not verified; contracts can report misleading symbols.")
	}
	out.Provenance = provs
	return out, nil
}

func topicAddr(t string) string {
	t = strings.ToLower(t)
	if len(t) < 40 {
		return ""
	}
	return "0x" + t[len(t)-40:]
}

func (e *Ethereum) tokenInfo(ctx context.Context, token string) (tokenMeta, []Provenance) {
	e.mu.Lock()
	m, ok := e.tokens[token]
	e.mu.Unlock()
	if ok {
		return m, nil
	}
	m = tokenMeta{decimals: -1}
	var provs []Provenance
	call := func(data string) string {
		raw, p, err := e.rpc.Call(ctx, "eth_call", map[string]string{"to": token, "data": data}, "latest")
		if err != nil {
			return ""
		}
		provs = append(provs, p)
		return unquote(raw)
	}
	m.symbol = decodeABIString(call("0x95d89b41"))
	if dr := call("0x313ce567"); dr != "" {
		if d := hexBig(dr); d.IsInt64() && d.Int64() <= 77 {
			m.decimals = int(d.Int64())
		}
	}
	e.mu.Lock()
	e.tokens[token] = m
	e.mu.Unlock()
	return m, provs
}

// decodeABIString decodes an ABI-encoded string or a bytes32 value.
func decodeABIString(h string) string {
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	if err != nil || len(b) == 0 {
		return ""
	}
	if len(b) == 32 {
		return sanitize(strings.TrimRight(string(b), "\x00"))
	}
	if len(b) < 64 {
		return ""
	}
	off := new(big.Int).SetBytes(b[:32])
	if !off.IsUint64() || off.Uint64()+32 > uint64(len(b)) {
		return ""
	}
	o := int(off.Uint64())
	ln := new(big.Int).SetBytes(b[o : o+32])
	if !ln.IsUint64() || ln.Uint64() > uint64(len(b)-o-32) {
		return ""
	}
	n := int(ln.Uint64())
	return sanitize(string(b[o+32 : o+32+n]))
}

// Activity delegates to the history source (Blockscout).
func (e *Ethereum) Activity(ctx context.Context, addr, kind, cursor string) (*ActivityPage, error) {
	if e.bs == nil {
		return nil, fmt.Errorf("%w: no address-history source configured", ErrUpstream)
	}
	return e.bs.Activity(ctx, addr, kind, cursor)
}
