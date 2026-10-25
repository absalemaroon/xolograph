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

const (
	maxFlowPages = 4
	priceGap     = 36 * time.Hour // a transfer is never valued with a price further away than this
)

// Flows summarises native-asset movements over the loaded window of activity. It is a movement summary,
// NOT profit or loss: it has no cost basis, ignores fees, internal/contract transfers, token swaps and anything before the window.
type Flows struct {
	Network        string             `json:"network"`
	Address        string             `json:"address"`
	Symbol         string             `json:"symbol"`
	Decimals       int                `json:"decimals"`
	PagesLoaded    int                `json:"pagesLoaded"`
	HistoryEnd     bool               `json:"historyExhausted"` // true when the source reported no further pages
	TransferCount  int                `json:"transferCount"`
	Failed         int                `json:"failedExcluded"`
	From           *time.Time         `json:"from,omitempty"`
	To             *time.Time         `json:"to,omitempty"`
	Inflow         string             `json:"inflow"`
	Outflow        string             `json:"outflow"`
	Net            string             `json:"net"` // may be negative
	InflowUSD      float64            `json:"inflowUsd"`
	OutflowUSD     float64            `json:"outflowUsd"`
	NetUSD         float64            `json:"netUsd"`
	ValuedCount    int                `json:"valuedCount"`
	UnvaluedCount  int                `json:"unvaluedCount"`
	Qualification  []string           `json:"qualification"`
	Provenance     []chain.Provenance `json:"provenance"`
}

// BuildFlows loads up to maxFlowPages pages of native activity and values each movement at the price nearest its time.
func BuildFlows(ctx context.Context, reg *chain.Registry, px prices.Provider, network, address string, pages int) (*Flows, error) {
	p, info, ok := reg.Get(network)
	if !ok || p == nil {
		return nil, fmt.Errorf("%w: unsupported network %q", chain.ErrInvalid, network)
	}
	if pages < 1 {
		pages = 1
	}
	if pages > maxFlowPages {
		pages = maxFlowPages
	}
	f := &Flows{Network: network, Symbol: info.NativeSymbol, Decimals: info.NativeDecimals, Qualification: []string{}, Provenance: []chain.Provenance{}}
	var items []chain.Transfer
	cursor := ""
	for i := 0; i < pages; i++ {
		pg, err := p.Activity(ctx, address, "native", cursor)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			f.Qualification = append(f.Qualification, "A later page could not be loaded: "+err.Error())
			break
		}
		if f.Address == "" {
			f.Address = pg.Address
		}
		f.PagesLoaded++
		f.Provenance = append(f.Provenance, pg.Provenance)
		items = append(items, pg.Items...)
		cursor = pg.NextCursor
		if cursor == "" {
			f.HistoryEnd = true
			break
		}
	}
	in, out := new(big.Int), new(big.Int)
	type mv struct {
		t     time.Time
		v     *big.Int
		inbnd bool
	}
	var moves []mv
	for _, t := range items {
		if t.Status == "FAILED" {
			f.Failed++
			continue
		}
		v, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			continue
		}
		inbound := strings.EqualFold(t.To, f.Address) && !strings.EqualFold(t.From, f.Address)
		outbound := strings.EqualFold(t.From, f.Address) && !strings.EqualFold(t.To, f.Address)
		switch {
		case inbound:
			in.Add(in, v)
		case outbound:
			out.Add(out, v)
		default: // self-transfer or unattributable: counted, not a flow
			f.TransferCount++
			continue
		}
		f.TransferCount++
		moves = append(moves, mv{t: t.Timestamp, v: v, inbnd: inbound})
		if !t.Timestamp.IsZero() {
			ts := t.Timestamp
			if f.From == nil || ts.Before(*f.From) {
				f.From = &ts
			}
			if f.To == nil || ts.After(*f.To) {
				f.To = &ts
			}
		}
	}
	f.Inflow, f.Outflow, f.Net = in.String(), out.String(), new(big.Int).Sub(in, out).String()

	// value each movement at the nearest historical price
	var series []prices.Point
	if px != nil && f.From != nil && f.To != nil {
		if coin, ok := prices.NativeCoin[network]; ok {
			from, to := f.From.Add(-priceGap), f.To.Add(priceGap)
			if to.After(time.Now()) {
				to = time.Now()
			}
			pts, prov, err := px.History(ctx, coin, from, to)
			if err != nil {
				f.Qualification = append(f.Qualification, "Historical prices unavailable, so nothing is valued in USD: "+err.Error())
			} else {
				series = pts
				f.Provenance = append(f.Provenance, prov)
			}
		} else {
			f.Qualification = append(f.Qualification, "No price mapping exists for this network's native asset.")
		}
	} else if px == nil {
		f.Qualification = append(f.Qualification, "The price source is disabled, so nothing is valued in USD.")
	}
	for _, m := range moves {
		pt, ok := prices.Nearest(series, m.t, priceGap)
		if !ok || m.t.IsZero() {
			f.UnvaluedCount++
			continue
		}
		amt, ok := units(m.v.String(), f.Decimals)
		if !ok {
			f.UnvaluedCount++
			continue
		}
		val := usd(amt, pt.USD)
		f.ValuedCount++
		if m.inbnd {
			f.InflowUSD += val
		} else {
			f.OutflowUSD += val
		}
	}
	f.NetUSD = f.InflowUSD - f.OutflowUSD
	if f.UnvaluedCount > 0 && len(series) > 0 {
		f.Qualification = append(f.Qualification, fmt.Sprintf("%d movement(s) had no price within %s of their time and are excluded from the USD totals (the native totals include them).", f.UnvaluedCount, priceGap))
	}
	if !f.HistoryEnd {
		f.Qualification = append(f.Qualification, fmt.Sprintf("Only the %d most recent page(s) of activity were loaded; older movements are not included.", f.PagesLoaded))
	}
	f.Qualification = append(f.Qualification,
		"This is a summary of native-asset movements, not profit or loss: there is no cost basis, and fees, internal and contract-initiated transfers, token swaps and holdings from before the window are not included.",
		"USD amounts use a third-party market reference price nearest each movement's time and are approximate.")
	return f, nil
}
