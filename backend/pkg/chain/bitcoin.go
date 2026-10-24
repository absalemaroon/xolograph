package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	btcNetwork  = "bitcoin"
	btcDecimals = 8
	btcPage     = 25 // Esplora returns 25 confirmed transactions per page
	btcCoverage = "Bitcoin is UTXO-based. Each row shows the net effect of one transaction on this address; the counterparty is the first other input or output address and is indicative only. Unconfirmed transactions are included when present."
)

var (
	btcLegacyRe = regexp.MustCompile(`^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$`)
	btcBech32Re = regexp.MustCompile(`^(?i:bc1)[a-zA-HJ-NP-Z0-9]{11,87}$`)
	btcTxRe     = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// NormalizeBitcoinAddress accepts mainnet legacy (1…, 3…) and bech32/bech32m (bc1…) formats. Bech32 is lower-cased.
func NormalizeBitcoinAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case btcLegacyRe.MatchString(s):
		return s, nil
	case btcBech32Re.MatchString(s):
		return strings.ToLower(s), nil
	}
	return "", fmt.Errorf("%w: not a valid Bitcoin mainnet address", ErrInvalid)
}

func NormalizeBitcoinTxid(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !btcTxRe.MatchString(s) {
		return "", fmt.Errorf("%w: not a valid Bitcoin transaction id", ErrInvalid)
	}
	return strings.ToLower(s), nil
}

// Bitcoin reads from an Esplora-compatible REST API (mempool.space, blockstream.info).
type Bitcoin struct {
	base string
	hc   *http.Client
}

func NewBitcoin(base string) *Bitcoin {
	return &Bitcoin{base: strings.TrimRight(base, "/"), hc: &http.Client{Timeout: 20 * time.Second}}
}

func (b *Bitcoin) get(ctx context.Context, path string, out any) (Provenance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.base+path, nil)
	if err != nil {
		return Provenance{}, err
	}
	resp, err := b.hc.Do(req)
	if err != nil {
		return Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return Provenance{}, fmt.Errorf("%w: %s", ErrNotFound, path)
	case resp.StatusCode == http.StatusBadRequest:
		return Provenance{}, fmt.Errorf("%w: %s", ErrInvalid, strings.TrimSpace(string(raw)))
	case resp.StatusCode != http.StatusOK:
		return Provenance{}, fmt.Errorf("%w: HTTP %d", ErrUpstream, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return Provenance{}, fmt.Errorf("%w: malformed response", ErrUpstream)
	}
	return Provenance{Source: "ESPLORA_REST", Endpoint: redact(b.base), Query: "GET " + path, RetrievedAt: time.Now().UTC(), PayloadSHA256: sum(raw),
		Coverage: "Index state at query time."}, nil
}

type btcStats struct {
	Funded  uint64 `json:"funded_txo_sum"`
	Spent   uint64 `json:"spent_txo_sum"`
	TxCount uint64 `json:"tx_count"`
}

func (b *Bitcoin) Address(ctx context.Context, addr string) (*AddressSummary, error) {
	a, err := NormalizeBitcoinAddress(addr)
	if err != nil {
		return nil, err
	}
	var r struct {
		Chain   btcStats `json:"chain_stats"`
		Mempool btcStats `json:"mempool_stats"`
	}
	p, err := b.get(ctx, "/address/"+a, &r)
	if err != nil {
		return nil, err
	}
	bal := new(big.Int).Sub(new(big.Int).SetUint64(r.Chain.Funded), new(big.Int).SetUint64(r.Chain.Spent))
	if bal.Sign() < 0 {
		bal.SetInt64(0)
	}
	s := &AddressSummary{Address: a, Network: btcNetwork, NativeSymbol: "BTC", NativeDecimals: btcDecimals, BalanceWei: bal.String(),
		ThirdPartyLabels: []ThirdPartyLabel{}, Provenance: []Provenance{p},
		Notes: []string{fmt.Sprintf("Confirmed transactions: %d. Balance is confirmed funds only; the nonce field is not applicable. Third-party address labels are not available for this network.", r.Chain.TxCount)}}
	if d := new(big.Int).Sub(new(big.Int).SetUint64(r.Mempool.Funded), new(big.Int).SetUint64(r.Mempool.Spent)); d.Sign() != 0 || r.Mempool.TxCount > 0 {
		s.Notes = append(s.Notes, fmt.Sprintf("Unconfirmed: %d transaction(s), net %s satoshi, not included in the balance.", r.Mempool.TxCount, d.String()))
	}
	return s, nil
}

type btcTx struct {
	Txid string `json:"txid"`
	Fee  uint64 `json:"fee"`
	Vin  []struct {
		IsCoinbase bool `json:"is_coinbase"`
		Prevout    *struct {
			Address string `json:"scriptpubkey_address"`
			Type    string `json:"scriptpubkey_type"`
			Value   uint64 `json:"value"`
		} `json:"prevout"`
	} `json:"vin"`
	Vout []struct {
		Address string `json:"scriptpubkey_address"`
		Type    string `json:"scriptpubkey_type"`
		Value   uint64 `json:"value"`
	} `json:"vout"`
	Status struct {
		Confirmed   bool    `json:"confirmed"`
		BlockHeight uint64  `json:"block_height"`
		BlockHash   string  `json:"block_hash"`
		BlockTime   *int64  `json:"block_time"`
	} `json:"status"`
}

func (b *Bitcoin) Transaction(ctx context.Context, hash string) (*Transaction, error) {
	id, err := NormalizeBitcoinTxid(hash)
	if err != nil {
		return nil, err
	}
	var t btcTx
	p, err := b.get(ctx, "/tx/"+id, &t)
	if err != nil {
		return nil, err
	}
	out := &Transaction{Hash: t.Txid, Network: btcNetwork, NativeSymbol: "BTC", NativeDecimals: btcDecimals, Status: "PENDING",
		BlockNumber: t.Status.BlockHeight, BlockHash: t.Status.BlockHash, GasPriceWei: "0", FeeWei: strconv.FormatUint(t.Fee, 10),
		TokenTransfers: []TokenTransfer{}, Inputs: []TxIO{}, Outputs: []TxIO{}, Provenance: []Provenance{p},
		Notes: []string{"UTXO transaction: From shows only the first input address and there is no single recipient; see the inputs and outputs. Value is the sum of all outputs."}}
	if t.Status.Confirmed {
		out.Status = "SUCCESS"
	} else {
		out.Notes = append(out.Notes, "Transaction is unconfirmed (in the mempool).")
	}
	if t.Status.BlockTime != nil {
		bt := time.Unix(*t.Status.BlockTime, 0).UTC()
		out.Timestamp = &bt
	}
	total := new(big.Int)
	for _, v := range t.Vout {
		out.Outputs = append(out.Outputs, TxIO{Address: v.Address, Value: strconv.FormatUint(v.Value, 10), Type: v.Type})
		total.Add(total, new(big.Int).SetUint64(v.Value))
	}
	for _, v := range t.Vin {
		if v.IsCoinbase {
			out.Inputs = append(out.Inputs, TxIO{Type: "coinbase", Value: "0"})
			continue
		}
		if v.Prevout != nil {
			out.Inputs = append(out.Inputs, TxIO{Address: v.Prevout.Address, Value: strconv.FormatUint(v.Prevout.Value, 10), Type: v.Prevout.Type})
		}
	}
	for _, in := range out.Inputs {
		if in.Address != "" {
			out.From = in.Address
			break
		}
	}
	out.ValueWei = total.String()
	return out, nil
}

// netEffect returns the net satoshi change for addr in tx and an indicative counterparty.
func (t *btcTx) netEffect(addr string) (net *big.Int, firstInput, firstOther string) {
	net = new(big.Int)
	for _, v := range t.Vin {
		if v.Prevout == nil {
			continue
		}
		if v.Prevout.Address == addr {
			net.Sub(net, new(big.Int).SetUint64(v.Prevout.Value))
		} else if firstInput == "" {
			firstInput = v.Prevout.Address
		}
	}
	for _, v := range t.Vout {
		if v.Address == addr {
			net.Add(net, new(big.Int).SetUint64(v.Value))
		} else if firstOther == "" {
			firstOther = v.Address
		}
	}
	return net, firstInput, firstOther
}

func (b *Bitcoin) Activity(ctx context.Context, addr, kind, cursor string) (*ActivityPage, error) {
	a, err := NormalizeBitcoinAddress(addr)
	if err != nil {
		return nil, err
	}
	if kind != "native" && kind != "token" {
		return nil, fmt.Errorf("%w: kind must be native or token", ErrInvalid)
	}
	page := &ActivityPage{Address: a, Kind: kind, Items: []Transfer{}}
	if kind == "token" {
		page.Provenance = Provenance{Source: "ESPLORA_REST", Endpoint: redact(b.base), Query: "(none)", RetrievedAt: time.Now().UTC(), PayloadSHA256: sum([]byte("none")),
			Coverage: "Bitcoin has no token transfers in this adapter (Ordinals/Runes/BRC-20 are not supported)."}
		return page, nil
	}
	path := "/address/" + a + "/txs"
	if cursor != "" {
		id, err := NormalizeBitcoinTxid(cursor)
		if err != nil {
			return nil, fmt.Errorf("%w: bad cursor", ErrInvalid)
		}
		path = "/address/" + a + "/txs/chain/" + id
	}
	var txs []btcTx
	if page.Provenance, err = b.get(ctx, path, &txs); err != nil {
		return nil, err
	}
	page.Provenance.Coverage = btcCoverage
	for i := range txs {
		t := &txs[i]
		net, firstIn, firstOther := t.netEffect(a)
		tr := Transfer{TxHash: t.Txid, LogIndex: -1, Kind: "NATIVE", Decimals: btcDecimals, Symbol: "BTC", BlockNumber: t.Status.BlockHeight, Status: "UNKNOWN"}
		if t.Status.Confirmed {
			tr.Status = "SUCCESS"
		}
		if t.Status.BlockTime != nil {
			tr.Timestamp = time.Unix(*t.Status.BlockTime, 0).UTC()
		}
		tr.Value = new(big.Int).Abs(net).String()
		if net.Sign() >= 0 {
			tr.From, tr.To = firstIn, a
			if len(t.Vin) > 0 && t.Vin[0].IsCoinbase {
				tr.From = "coinbase"
			}
		} else {
			tr.From, tr.To = a, firstOther
		}
		page.Items = append(page.Items, tr)
	}
	if n := len(txs); n >= btcPage && txs[n-1].Status.Confirmed {
		page.NextCursor = txs[n-1].Txid
	}
	return page, nil
}
