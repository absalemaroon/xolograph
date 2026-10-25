package chain

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	solTokenProgram     = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	solToken2022Program = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	maxHoldings         = 200
)

var digitsRe = regexp.MustCompile(`^\d+$`)

// Holding is one asset balance. Balance is in the asset's smallest unit.
type Holding struct {
	Kind         string `json:"kind"` // NATIVE | ERC20 | SPL
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	TokenAddress string `json:"tokenAddress,omitempty"`
	Balance      string `json:"balance"`
	Decimals     int    `json:"decimals"` // -1 when unknown
}

type Holdings struct {
	Address    string       `json:"address"`
	Network    string       `json:"network"`
	Items      []Holding    `json:"items"`
	Provenance []Provenance `json:"provenance"`
	Notes      []string     `json:"notes"`
}

// HoldingsProvider is implemented by adapters that can list more than the native balance.
type HoldingsProvider interface {
	Holdings(ctx context.Context, addr string) (*Holdings, error)
}

type bsTokenBalance struct {
	Token struct {
		Address     string     `json:"address"`
		AddressHash string     `json:"address_hash"`
		Name        flexString `json:"name"`
		Symbol      flexString `json:"symbol"`
		Decimals    flexString `json:"decimals"`
		Type        string     `json:"type"`
	} `json:"token"`
	Value flexString `json:"value"`
}

// TokenBalances lists ERC-20 balances. Non-fungible tokens are counted, not listed.
func (b *Blockscout) TokenBalances(ctx context.Context, addr string) (items []Holding, skipped int, prov Provenance, err error) {
	a, err := NormalizeAddress(addr)
	if err != nil {
		return nil, 0, Provenance{}, err
	}
	var rows []bsTokenBalance
	if prov, err = b.get(ctx, "/api/v2/addresses/"+url.PathEscape(a)+"/token-balances", url.Values{}, &rows); err != nil {
		return nil, 0, Provenance{}, err
	}
	for _, r := range rows {
		if !strings.EqualFold(r.Token.Type, "ERC-20") {
			skipped++
			continue
		}
		bal := string(r.Value)
		if !digitsRe.MatchString(bal) || strings.Trim(bal, "0") == "" {
			continue
		}
		addrHash := r.Token.AddressHash
		if addrHash == "" {
			addrHash = r.Token.Address
		}
		if norm, e := NormalizeAddress(addrHash); e == nil {
			addrHash = norm
		} else {
			continue
		}
		dec := -1
		if d, e := strconv.Atoi(string(r.Token.Decimals)); e == nil && d >= 0 && d <= 77 {
			dec = d
		}
		items = append(items, Holding{Kind: "ERC20", Symbol: sanitize(string(r.Token.Symbol)), Name: sanitize(string(r.Token.Name)), TokenAddress: addrHash, Balance: bal, Decimals: dec})
		if len(items) == maxHoldings {
			break
		}
	}
	return items, skipped, prov, nil
}

// Holdings returns the native balance plus ERC-20 balances when a history source is configured.
func (e *Ethereum) Holdings(ctx context.Context, addr string) (*Holdings, error) {
	s, err := e.Address(ctx, addr)
	if err != nil {
		return nil, err
	}
	h := &Holdings{Address: s.Address, Network: e.net, Provenance: append([]Provenance{}, s.Provenance...), Notes: []string{},
		Items: []Holding{{Kind: "NATIVE", Symbol: e.symbol, Name: e.symbol, Balance: s.BalanceWei, Decimals: 18}}}
	if e.bs == nil {
		h.Notes = append(h.Notes, "No token-balance source is configured for this network; only the native balance is listed.")
		return h, nil
	}
	items, skipped, p, err := e.bs.TokenBalances(ctx, s.Address)
	if err != nil {
		h.Notes = append(h.Notes, "Token balances unavailable: "+err.Error())
		return h, nil
	}
	h.Items = append(h.Items, items...)
	h.Provenance = append(h.Provenance, p)
	if skipped > 0 {
		h.Notes = append(h.Notes, fmt.Sprintf("%d non-fungible or unsupported token holding(s) are not listed or valued.", skipped))
	}
	if len(items) >= maxHoldings {
		h.Notes = append(h.Notes, fmt.Sprintf("Only the first %d token balances are listed.", maxHoldings))
	}
	h.Notes = append(h.Notes, "Token symbols and names come from the token contracts via a third-party index and are not verified; contracts can use misleading names.")
	return h, nil
}

// Holdings returns the SOL balance plus SPL token balances (Token and Token-2022 programs), summed per mint.
func (s *Solana) Holdings(ctx context.Context, addr string) (*Holdings, error) {
	sum0, err := s.Address(ctx, addr)
	if err != nil {
		return nil, err
	}
	h := &Holdings{Address: sum0.Address, Network: solNetwork, Provenance: append([]Provenance{}, sum0.Provenance...), Notes: []string{},
		Items: []Holding{{Kind: "NATIVE", Symbol: "SOL", Name: "Solana", Balance: sum0.BalanceWei, Decimals: solDecimals}}}
	totals := map[string]*big.Int{}
	decs := map[string]int{}
	var order []string
	for _, prog := range []string{solTokenProgram, solToken2022Program} {
		raw, p, err := s.rpc.Call(ctx, "getTokenAccountsByOwner", sum0.Address, map[string]any{"programId": prog}, map[string]any{"encoding": "jsonParsed", "commitment": "finalized"})
		if err != nil {
			h.Notes = append(h.Notes, "Token accounts ("+prog[:8]+"…) unavailable: "+err.Error())
			continue
		}
		h.Provenance = append(h.Provenance, p)
		var r struct {
			Value []struct {
				Account struct {
					Data struct {
						Parsed struct {
							Info struct {
								Mint        string `json:"mint"`
								TokenAmount struct {
									Amount   string `json:"amount"`
									Decimals int    `json:"decimals"`
								} `json:"tokenAmount"`
							} `json:"info"`
						} `json:"parsed"`
					} `json:"data"`
				} `json:"account"`
			} `json:"value"`
		}
		if jsonUnmarshal(raw, &r) != nil {
			h.Notes = append(h.Notes, "Token accounts response was malformed and was ignored.")
			continue
		}
		for _, v := range r.Value {
			in := v.Account.Data.Parsed.Info
			amt, ok := new(big.Int).SetString(in.TokenAmount.Amount, 10)
			if in.Mint == "" || !ok || amt.Sign() == 0 {
				continue
			}
			if _, seen := totals[in.Mint]; !seen {
				totals[in.Mint] = new(big.Int)
				decs[in.Mint] = in.TokenAmount.Decimals
				order = append(order, in.Mint)
			}
			totals[in.Mint].Add(totals[in.Mint], amt)
		}
	}
	for i, m := range order {
		if i == maxHoldings {
			h.Notes = append(h.Notes, fmt.Sprintf("Only the first %d token balances are listed.", maxHoldings))
			break
		}
		h.Items = append(h.Items, Holding{Kind: "SPL", Symbol: "", Name: "", TokenAddress: m, Balance: totals[m].String(), Decimals: decs[m]})
	}
	h.Notes = append(h.Notes, "SPL tokens are listed by mint address; names and symbols are not resolved.")
	return h, nil
}

// Holdings returns the confirmed BTC balance (no other assets are supported).
func (b *Bitcoin) Holdings(ctx context.Context, addr string) (*Holdings, error) {
	s, err := b.Address(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &Holdings{Address: s.Address, Network: btcNetwork, Provenance: s.Provenance, Notes: []string{"Only the confirmed BTC balance is listed; Ordinals, Runes and BRC-20 are not supported."},
		Items: []Holding{{Kind: "NATIVE", Symbol: "BTC", Name: "Bitcoin", Balance: s.BalanceWei, Decimals: btcDecimals}}}, nil
}
