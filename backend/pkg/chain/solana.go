package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	solNetwork   = "solana"
	solDecimals  = 9
	solPageSize  = 20
	solFetchPar  = 4
	solCoverage  = "Derived from parsed System Program and SPL Token instructions of the most recent transactions that reference this address. Transfers made through programs that do not emit parsed instructions, and balance changes that are not transfers, can be missing; token symbols are not resolved."
	solTxNote    = "Transfers are derived from parsed System Program and SPL Token instructions. Value is the sum of System Program transfers only. Transfers inside other programs may not appear; compare with the account balance changes if completeness matters. SPL tokens are shown by mint address; names are not resolved."
)

// NormalizeSolanaAddress validates a base58 public key (32 bytes).
func NormalizeSolanaAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	b, ok := base58Decode(s)
	if !ok || len(b) != 32 {
		return "", fmt.Errorf("%w: not a valid Solana address", ErrInvalid)
	}
	return s, nil
}

// NormalizeSolanaSignature validates a base58 transaction signature (64 bytes).
func NormalizeSolanaSignature(s string) (string, error) {
	s = strings.TrimSpace(s)
	b, ok := base58Decode(s)
	if !ok || len(b) != 64 {
		return "", fmt.Errorf("%w: not a valid Solana transaction signature", ErrInvalid)
	}
	return s, nil
}

// Solana reads accounts and transactions through a Solana JSON-RPC node.
type Solana struct{ rpc *RPC }

func NewSolana(rpc *RPC) *Solana { return &Solana{rpc: rpc.WithSource("SOLANA_JSON_RPC")} }

func (s *Solana) Address(ctx context.Context, addr string) (*AddressSummary, error) {
	a, err := NormalizeSolanaAddress(addr)
	if err != nil {
		return nil, err
	}
	out := &AddressSummary{Address: a, Network: solNetwork, NativeSymbol: "SOL", NativeDecimals: solDecimals,
		ThirdPartyLabels: []ThirdPartyLabel{}, Notes: []string{}, Provenance: []Provenance{}}

	raw, p, err := s.rpc.Call(ctx, "getBalance", a, map[string]any{"commitment": "finalized"})
	if err != nil {
		return nil, err
	}
	var bal struct {
		Value uint64 `json:"value"`
	}
	if json.Unmarshal(raw, &bal) != nil {
		return nil, fmt.Errorf("%w: malformed balance response", ErrUpstream)
	}
	out.BalanceWei = strconv.FormatUint(bal.Value, 10)
	out.Provenance = append(out.Provenance, p)

	raw, p, err = s.rpc.Call(ctx, "getAccountInfo", a, map[string]any{"encoding": "base64", "commitment": "finalized"})
	if err != nil {
		return nil, err
	}
	out.Provenance = append(out.Provenance, p)
	var acc struct {
		Value *struct {
			Owner      string `json:"owner"`
			Executable bool   `json:"executable"`
		} `json:"value"`
	}
	if json.Unmarshal(raw, &acc) != nil {
		return nil, fmt.Errorf("%w: malformed account response", ErrUpstream)
	}
	if acc.Value == nil {
		out.Notes = append(out.Notes, "No account exists at this address on-chain (never funded, or closed).")
	} else {
		out.IsContract = acc.Value.Executable
		out.Notes = append(out.Notes, "Owner program: "+acc.Value.Owner)
	}
	out.Notes = append(out.Notes, "Solana accounts have no nonce; the nonce field is not applicable. Third-party address labels are not available for this network.")
	return out, nil
}

// ---- transaction parsing

type solKey struct {
	Pubkey string `json:"pubkey"`
	Signer bool   `json:"signer"`
}

type solIx struct {
	Program string          `json:"program"`
	Parsed  json.RawMessage `json:"parsed"`
}

type solTokBal struct {
	AccountIndex  int    `json:"accountIndex"`
	Mint          string `json:"mint"`
	Owner         string `json:"owner"`
	UITokenAmount struct {
		Decimals int `json:"decimals"`
	} `json:"uiTokenAmount"`
}

type solTx struct {
	Slot      uint64  `json:"slot"`
	BlockTime *int64  `json:"blockTime"`
	Meta      *solMeta `json:"meta"`
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys  []solKey `json:"accountKeys"`
			Instructions []solIx  `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`
}

type solMeta struct {
	Err               json.RawMessage `json:"err"`
	Fee               uint64          `json:"fee"`
	ComputeUnits      uint64          `json:"computeUnitsConsumed"`
	PreTokenBalances  []solTokBal     `json:"preTokenBalances"`
	PostTokenBalances []solTokBal     `json:"postTokenBalances"`
	InnerInstructions []struct {
		Index        int     `json:"index"`
		Instructions []solIx `json:"instructions"`
	} `json:"innerInstructions"`
}

// solTransfer is one value movement found in a transaction's parsed instructions.
type solTransfer struct {
	index    int
	native   bool
	from, to string // owners for tokens, accounts for native
	amount   string
	mint     string
	decimals int
}

func (t *solTx) failed() bool {
	return t.Meta != nil && len(t.Meta.Err) > 0 && string(t.Meta.Err) != "null"
}

// transfers extracts System Program and SPL Token transfers. Token accounts are resolved to their owners via the
// transaction's token balances; a token transfer whose accounts cannot be resolved is reported with the token account itself.
func (t *solTx) transfers() []solTransfer {
	if t.Meta == nil {
		return nil
	}
	keys := t.Transaction.Message.AccountKeys
	type tokAcct struct {
		owner, mint string
		decimals    int
	}
	accts := map[string]tokAcct{}
	for _, list := range [][]solTokBal{t.Meta.PreTokenBalances, t.Meta.PostTokenBalances} {
		for _, b := range list {
			if b.AccountIndex >= 0 && b.AccountIndex < len(keys) {
				accts[keys[b.AccountIndex].Pubkey] = tokAcct{owner: b.Owner, mint: b.Mint, decimals: b.UITokenAmount.Decimals}
			}
		}
	}
	all := append([]solIx{}, t.Transaction.Message.Instructions...)
	for _, in := range t.Meta.InnerInstructions {
		all = append(all, in.Instructions...)
	}
	var out []solTransfer
	for _, ix := range all {
		var p struct {
			Type string          `json:"type"`
			Info json.RawMessage `json:"info"`
		}
		if len(ix.Parsed) == 0 || json.Unmarshal(ix.Parsed, &p) != nil { // memo etc. carry a plain string
			continue
		}
		switch {
		case ix.Program == "system" && p.Type == "transfer":
			var i struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
				Lamports    uint64 `json:"lamports"`
			}
			if json.Unmarshal(p.Info, &i) == nil && i.Source != "" && i.Destination != "" {
				out = append(out, solTransfer{index: len(out), native: true, from: i.Source, to: i.Destination, amount: strconv.FormatUint(i.Lamports, 10), decimals: solDecimals})
			}
		case (ix.Program == "spl-token" || ix.Program == "spl-token-2022") && (p.Type == "transfer" || p.Type == "transferChecked"):
			var i struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
				Mint        string `json:"mint"`
				Amount      string `json:"amount"`
				TokenAmount struct {
					Amount   string `json:"amount"`
					Decimals int    `json:"decimals"`
				} `json:"tokenAmount"`
			}
			if json.Unmarshal(p.Info, &i) != nil || i.Source == "" || i.Destination == "" {
				continue
			}
			amount, dec := i.Amount, -1
			if p.Type == "transferChecked" {
				amount, dec = i.TokenAmount.Amount, i.TokenAmount.Decimals
			}
			src, dst := accts[i.Source], accts[i.Destination]
			mint := i.Mint
			if mint == "" {
				mint = src.mint
			}
			if mint == "" {
				mint = dst.mint
			}
			if dec < 0 {
				if src.mint != "" {
					dec = src.decimals
				} else if dst.mint != "" {
					dec = dst.decimals
				}
			}
			from, to := src.owner, dst.owner
			if from == "" {
				from = i.Source
			}
			if to == "" {
				to = i.Destination
			}
			out = append(out, solTransfer{index: len(out), from: from, to: to, amount: amount, mint: mint, decimals: dec})
		}
	}
	return out
}

func (s *Solana) fetchTx(ctx context.Context, sig string) (*solTx, Provenance, error) {
	raw, p, err := s.rpc.Call(ctx, "getTransaction", sig, map[string]any{"encoding": "jsonParsed", "maxSupportedTransactionVersion": 0, "commitment": "confirmed"})
	if err != nil {
		return nil, p, err
	}
	if string(raw) == "null" {
		return nil, p, fmt.Errorf("%w: transaction %s", ErrNotFound, sig)
	}
	var t solTx
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, p, fmt.Errorf("%w: malformed transaction payload", ErrUpstream)
	}
	return &t, p, nil
}

func (s *Solana) Transaction(ctx context.Context, hash string) (*Transaction, error) {
	sig, err := NormalizeSolanaSignature(hash)
	if err != nil {
		return nil, err
	}
	t, p, err := s.fetchTx(ctx, sig)
	if err != nil {
		return nil, err
	}
	out := &Transaction{Hash: sig, Network: solNetwork, NativeSymbol: "SOL", NativeDecimals: solDecimals, Status: "SUCCESS",
		BlockNumber: t.Slot, GasPriceWei: "0", ValueWei: "0", FeeWei: "0", TokenTransfers: []TokenTransfer{}, Notes: []string{solTxNote}, Provenance: []Provenance{p}}
	for _, k := range t.Transaction.Message.AccountKeys {
		if k.Signer {
			out.From = k.Pubkey
			break
		}
	}
	if t.BlockTime != nil {
		bt := time.Unix(*t.BlockTime, 0).UTC()
		out.Timestamp = &bt
	}
	if t.Meta != nil {
		out.FeeWei, out.GasUsed = strconv.FormatUint(t.Meta.Fee, 10), t.Meta.ComputeUnits
	}
	if t.failed() {
		out.Status = "FAILED"
		out.Notes = append(out.Notes, "The transaction failed on-chain; the transfers below were attempted but did not take effect. Error: "+string(t.Meta.Err))
	}
	total := new(big.Int)
	for _, tr := range t.transfers() {
		tt := TokenTransfer{LogIndex: int64(tr.index), From: tr.from, To: tr.to, Value: tr.amount, Decimals: tr.decimals, Token: tr.mint}
		if tr.native {
			tt.Standard, tt.Symbol = "NATIVE", "SOL"
			if v, ok := new(big.Int).SetString(tr.amount, 10); ok {
				total.Add(total, v)
			}
		} else {
			tt.Standard = "SPL"
		}
		out.TokenTransfers = append(out.TokenTransfers, tt)
	}
	out.ValueWei = total.String()
	return out, nil
}

func (s *Solana) Activity(ctx context.Context, addr, kind, cursor string) (*ActivityPage, error) {
	a, err := NormalizeSolanaAddress(addr)
	if err != nil {
		return nil, err
	}
	if kind != "native" && kind != "token" {
		return nil, fmt.Errorf("%w: kind must be native or token", ErrInvalid)
	}
	opts := map[string]any{"limit": solPageSize, "commitment": "confirmed"}
	if cursor != "" {
		if _, err := NormalizeSolanaSignature(cursor); err != nil {
			return nil, fmt.Errorf("%w: bad cursor", ErrInvalid)
		}
		opts["before"] = cursor
	}
	raw, p0, err := s.rpc.Call(ctx, "getSignaturesForAddress", a, opts)
	if err != nil {
		return nil, err
	}
	var sigs []struct {
		Signature string `json:"signature"`
	}
	if json.Unmarshal(raw, &sigs) != nil {
		return nil, fmt.Errorf("%w: malformed signature list", ErrUpstream)
	}
	type result struct {
		tx *solTx
		p  Provenance
		ok bool
	}
	res := make([]result, len(sigs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, solFetchPar)
	for i := range sigs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			t, p, err := s.fetchTx(ctx, sigs[i].Signature)
			res[i] = result{tx: t, p: p, ok: err == nil}
		}(i)
	}
	wg.Wait()

	page := &ActivityPage{Address: a, Kind: kind, Items: []Transfer{}}
	hashes := p0.PayloadSHA256
	failed := 0
	for i, r := range res {
		if !r.ok {
			failed++
			continue
		}
		hashes += r.p.PayloadSHA256
		status := "SUCCESS"
		if r.tx.failed() {
			status = "FAILED"
		}
		var ts time.Time
		if r.tx.BlockTime != nil {
			ts = time.Unix(*r.tx.BlockTime, 0).UTC()
		}
		for _, tr := range r.tx.transfers() {
			if tr.native != (kind == "native") || (tr.from != a && tr.to != a) {
				continue
			}
			t := Transfer{TxHash: sigs[i].Signature, LogIndex: int64(tr.index), From: tr.from, To: tr.to, Value: tr.amount, Decimals: tr.decimals,
				Timestamp: ts, BlockNumber: r.tx.Slot, Status: status}
			if tr.native {
				t.Kind, t.Symbol = "NATIVE", "SOL"
			} else {
				t.Kind, t.TokenAddress = "SPL", tr.mint
			}
			page.Items = append(page.Items, t)
		}
	}
	if len(sigs) > 0 && failed == len(sigs) {
		return nil, fmt.Errorf("%w: none of the %d transactions could be retrieved", ErrUpstream, len(sigs))
	}
	cov := solCoverage
	if failed > 0 {
		cov += fmt.Sprintf(" %d of %d transactions could not be retrieved and are missing.", failed, len(sigs))
	}
	page.Provenance = Provenance{Source: "SOLANA_JSON_RPC", Endpoint: p0.Endpoint, Query: fmt.Sprintf("getSignaturesForAddress + getTransaction x%d", len(sigs)),
		RetrievedAt: time.Now().UTC(), PayloadSHA256: sum([]byte(hashes)), Coverage: cov}
	if len(sigs) == solPageSize {
		page.NextCursor = sigs[len(sigs)-1].Signature
	}
	return page, nil
}
