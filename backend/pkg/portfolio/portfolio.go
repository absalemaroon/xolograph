// Package portfolio values holdings and summarises net flows. It reports what its sources support and states what they do not:
// an asset without a price is "unpriced", never worth zero, and flows are not presented as profit or loss.
package portfolio

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/xolograph/backend/pkg/chain"
	"github.com/xolograph/backend/pkg/prices"
)

type Asset struct {
	Kind         string     `json:"kind"`
	Symbol       string     `json:"symbol"`
	Name         string     `json:"name"`
	TokenAddress string     `json:"tokenAddress,omitempty"`
	Balance      string     `json:"balance"`
	Decimals     int        `json:"decimals"`
	Priced       bool       `json:"priced"`
	PriceUSD     *float64   `json:"priceUsd,omitempty"`
	ValueUSD     *float64   `json:"valueUsd,omitempty"`
	PriceAt      *time.Time `json:"priceAt,omitempty"`
	PriceSource  string     `json:"priceSource,omitempty"`
}

type Wallet struct {
	Network       string             `json:"network"`
	Address       string             `json:"address"`
	Assets        []Asset            `json:"assets"`
	PricedValue   float64            `json:"pricedValueUsd"` // sum of priced assets only
	PricedCount   int                `json:"pricedCount"`
	UnpricedCount int                `json:"unpricedCount"`
	AsOf          time.Time          `json:"asOf"`
	Provenance    []chain.Provenance `json:"provenance"`
	Notes         []string           `json:"notes"`
}

// units converts a smallest-unit integer string to a decimal amount. ok is false when decimals are unknown or the balance is malformed.
func units(balance string, decimals int) (*big.Float, bool) {
	if decimals < 0 || decimals > 77 {
		return nil, false
	}
	v, ok := new(big.Int).SetString(balance, 10)
	if !ok || v.Sign() < 0 {
		return nil, false
	}
	den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return new(big.Float).Quo(new(big.Float).SetInt(v), new(big.Float).SetInt(den)), true
}

func usd(amount *big.Float, price float64) float64 {
	f, _ := new(big.Float).Mul(amount, big.NewFloat(price)).Float64()
	return f
}

// BuildWallet lists a wallet's holdings and values the ones that have a price. px may be nil (prices disabled).
func BuildWallet(ctx context.Context, reg *chain.Registry, px prices.Provider, network, address string) (*Wallet, error) {
	p, _, ok := reg.Get(network)
	if !ok || p == nil {
		return nil, fmt.Errorf("%w: unsupported network %q", chain.ErrInvalid, network)
	}
	var h *chain.Holdings
	var err error
	if hp, ok := p.(chain.HoldingsProvider); ok {
		h, err = hp.Holdings(ctx, address)
	} else {
		var s *chain.AddressSummary
		if s, err = p.Address(ctx, address); err == nil {
			h = &chain.Holdings{Address: s.Address, Network: network, Provenance: s.Provenance, Notes: []string{"Only the native balance is available for this network."},
				Items: []chain.Holding{{Kind: "NATIVE", Symbol: s.NativeSymbol, Name: s.NativeSymbol, Balance: s.BalanceWei, Decimals: s.NativeDecimals}}}
		}
	}
	if err != nil {
		return nil, err
	}
	w := &Wallet{Network: network, Address: h.Address, Assets: []Asset{}, AsOf: time.Now().UTC(), Provenance: append([]chain.Provenance{}, h.Provenance...), Notes: append([]string{}, h.Notes...)}
	for _, it := range h.Items {
		w.Assets = append(w.Assets, Asset{Kind: it.Kind, Symbol: it.Symbol, Name: it.Name, TokenAddress: it.TokenAddress, Balance: it.Balance, Decimals: it.Decimals})
	}
	if px == nil {
		w.Notes = append(w.Notes, "The price source is disabled, so nothing is valued.")
		w.UnpricedCount = len(w.Assets)
		return w, nil
	}
	apply := func(a *Asset, q prices.Quote) {
		amt, ok := units(a.Balance, a.Decimals)
		if !ok {
			return
		}
		price, val, at := q.USD, usd(amt, q.USD), q.At
		a.Priced, a.PriceUSD, a.ValueUSD, a.PriceAt, a.PriceSource = true, &price, &val, &at, q.Source
	}
	// native asset
	if coin, ok := prices.NativeCoin[network]; ok {
		q, prov, err := px.NativeSpot(ctx, coin)
		if err != nil {
			w.Notes = append(w.Notes, "Native price unavailable: "+err.Error())
		} else {
			w.Provenance = append(w.Provenance, prov)
			for i := range w.Assets {
				if w.Assets[i].Kind == "NATIVE" {
					apply(&w.Assets[i], q)
				}
			}
		}
	} else {
		w.Notes = append(w.Notes, "No price mapping exists for this network's native asset.")
	}
	// tokens, by contract address
	if platform, ok := prices.TokenPlatform[network]; ok {
		var contracts []string
		idx := map[string][]int{}
		for i, a := range w.Assets {
			if a.Kind == "NATIVE" || a.TokenAddress == "" {
				continue
			}
			key := strings.ToLower(a.TokenAddress) // the price source keys its responses in lower case
			if _, seen := idx[key]; !seen {
				contracts = append(contracts, a.TokenAddress)
			}
			idx[key] = append(idx[key], i)
		}
		if len(contracts) > 0 {
			qs, prov, err := px.TokenSpot(ctx, platform, contracts)
			if err != nil {
				w.Notes = append(w.Notes, "Token prices unavailable: "+err.Error())
			} else {
				w.Provenance = append(w.Provenance, prov)
				for key, is := range idx {
					if q, ok := qs[key]; ok {
						for _, i := range is {
							apply(&w.Assets[i], q)
						}
					}
				}
			}
			if len(contracts) > 50 {
				w.Notes = append(w.Notes, "Only the first 50 tokens were looked up for prices; the rest are unpriced.")
			}
		}
	}
	for _, a := range w.Assets {
		if a.Priced && a.ValueUSD != nil {
			w.PricedValue += *a.ValueUSD
			w.PricedCount++
		} else {
			w.UnpricedCount++
		}
	}
	if w.UnpricedCount > 0 {
		w.Notes = append(w.Notes, fmt.Sprintf("%d asset(s) have no reference price and are NOT included in the total. Unlisted or spam tokens are expected to be unpriced.", w.UnpricedCount))
	}
	return w, nil
}
