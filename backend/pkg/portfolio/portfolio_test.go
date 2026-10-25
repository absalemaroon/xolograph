package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xolograph/backend/pkg/chain"
	"github.com/xolograph/backend/pkg/prices"
)

const (
	me    = "0x1111111111111111111111111111111111111111"
	other = "0x2222222222222222222222222222222222222222"
	tokA  = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tokB  = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type fakeChain struct{ pages map[string]*chain.ActivityPage }

func (f fakeChain) Address(ctx context.Context, a string) (*chain.AddressSummary, error) {
	return nil, errors.New("unused")
}
func (f fakeChain) Transaction(ctx context.Context, h string) (*chain.Transaction, error) {
	return nil, errors.New("unused")
}
func (f fakeChain) Activity(ctx context.Context, a, kind, cursor string) (*chain.ActivityPage, error) {
	pg, ok := f.pages[cursor]
	if !ok {
		return nil, errors.New("no such page")
	}
	return pg, nil
}
func (f fakeChain) Holdings(ctx context.Context, a string) (*chain.Holdings, error) {
	return &chain.Holdings{Address: a, Network: "ethereum-mainnet", Notes: []string{"n"}, Items: []chain.Holding{
		{Kind: "NATIVE", Symbol: "ETH", Balance: "2500000000000000000", Decimals: 18},  // 2.5 ETH
		{Kind: "ERC20", Symbol: "USDC", TokenAddress: tokA, Balance: "1500000", Decimals: 6}, // 1.5 USDC
		{Kind: "ERC20", Symbol: "SPAM", TokenAddress: tokB, Balance: "999999999999", Decimals: 6},
	}}, nil
}

type fakePx struct{ fail bool }

func (f fakePx) NativeSpot(ctx context.Context, id string) (prices.Quote, chain.Provenance, error) {
	if f.fail {
		return prices.Quote{}, chain.Provenance{}, prices.ErrUpstream
	}
	return prices.Quote{USD: 2000, At: time.Unix(1700000000, 0), Source: "T"}, chain.Provenance{Source: "T"}, nil
}
func (f fakePx) TokenSpot(ctx context.Context, platform string, c []string) (map[string]prices.Quote, chain.Provenance, error) {
	return map[string]prices.Quote{tokA: {USD: 1, At: time.Unix(1700000000, 0), Source: "T"}}, chain.Provenance{Source: "T"}, nil
}
func (f fakePx) History(ctx context.Context, id string, from, to time.Time) ([]prices.Point, chain.Provenance, error) {
	return []prices.Point{{T: time.Unix(1700000000, 0), USD: 1000}, {T: time.Unix(1700172800, 0), USD: 2000}}, chain.Provenance{Source: "T"}, nil
}

func reg(c fakeChain) *chain.Registry {
	r := chain.NewRegistry()
	r.Add(chain.NetworkInfo{ID: "ethereum-mainnet", Name: "Ethereum", Family: "EVM", NativeSymbol: "ETH", NativeDecimals: 18}, c)
	return r
}

func TestWalletValuesOnlyPricedAssets(t *testing.T) {
	w, err := BuildWallet(context.Background(), reg(fakeChain{}), fakePx{}, "ethereum-mainnet", me)
	if err != nil {
		t.Fatal(err)
	}
	// 2.5 ETH * 2000 + 1.5 USDC * 1 = 5001.5 ; the unpriced SPAM token must not count (and is not worth "0")
	if w.PricedValue != 5001.5 || w.PricedCount != 2 || w.UnpricedCount != 1 {
		t.Fatalf("totals: %+v", w)
	}
	for _, a := range w.Assets {
		if a.Symbol == "SPAM" && (a.Priced || a.ValueUSD != nil || a.PriceUSD != nil) {
			t.Fatalf("unpriced asset must carry no value: %+v", a)
		}
		if a.Symbol == "ETH" && (a.ValueUSD == nil || *a.ValueUSD != 5000 || a.PriceSource != "T") {
			t.Fatalf("native valuation: %+v", a)
		}
	}
}

func TestWalletWithoutPricesOrWithPriceFailure(t *testing.T) {
	w, err := BuildWallet(context.Background(), reg(fakeChain{}), nil, "ethereum-mainnet", me)
	if err != nil || w.PricedValue != 0 || w.UnpricedCount != 3 {
		t.Fatalf("prices off: %+v %v", w, err)
	}
	w, err = BuildWallet(context.Background(), reg(fakeChain{}), fakePx{fail: true}, "ethereum-mainnet", me)
	if err != nil || w.PricedCount != 1 || w.UnpricedCount != 2 { // only the token price lookup succeeded
		t.Fatalf("native price failure must degrade, not fail: %+v %v", w, err)
	}
	if _, err := BuildWallet(context.Background(), reg(fakeChain{}), nil, "nowhere", me); !errors.Is(err, chain.ErrInvalid) {
		t.Fatalf("unknown network: %v", err)
	}
}

func TestFlowsAreSummariesNotProfit(t *testing.T) {
	t0, t1 := time.Unix(1700000000, 0).UTC(), time.Unix(1700172800, 0).UTC()
	c := fakeChain{pages: map[string]*chain.ActivityPage{
		"": {Address: me, NextCursor: "p2", Items: []chain.Transfer{
			{TxHash: "a", From: other, To: me, Value: "2000000000000000000", Timestamp: t1, Status: "SUCCESS"},  // +2 ETH @2000
			{TxHash: "b", From: me, To: other, Value: "500000000000000000", Timestamp: t1, Status: "SUCCESS"},   // -0.5 ETH @2000
			{TxHash: "c", From: me, To: other, Value: "9000000000000000000", Timestamp: t1, Status: "FAILED"},   // failed: excluded
			{TxHash: "d", From: me, To: me, Value: "1000000000000000000", Timestamp: t1, Status: "SUCCESS"},     // self: not a flow
		}},
		"p2": {Address: me, NextCursor: "", Items: []chain.Transfer{
			{TxHash: "e", From: other, To: me, Value: "1000000000000000000", Timestamp: t0, Status: "SUCCESS"}, // +1 ETH @1000
		}},
	}}
	f, err := BuildFlows(context.Background(), reg(c), fakePx{}, "ethereum-mainnet", me, 4)
	if err != nil {
		t.Fatal(err)
	}
	if f.Inflow != "3000000000000000000" || f.Outflow != "500000000000000000" || f.Net != "2500000000000000000" || f.Failed != 1 || !f.HistoryEnd || f.PagesLoaded != 2 {
		t.Fatalf("native totals: %+v", f)
	}
	if f.InflowUSD != 5000 || f.OutflowUSD != 1000 || f.NetUSD != 4000 || f.ValuedCount != 3 || f.UnvaluedCount != 0 {
		t.Fatalf("usd totals (each movement at its own-time price): %+v", f)
	}
	joined := ""
	for _, q := range f.Qualification {
		joined += q + "|"
	}
	if !contains(joined, "not profit or loss") {
		t.Fatalf("the not-P&L qualification must always be present: %v", f.Qualification)
	}
}

func TestFlowsStopAtPageLimitAndSayIncomplete(t *testing.T) {
	pg := func(next string) *chain.ActivityPage {
		return &chain.ActivityPage{Address: me, NextCursor: next, Items: []chain.Transfer{{TxHash: next, From: other, To: me, Value: "1", Timestamp: time.Unix(1700000000, 0), Status: "SUCCESS"}}}
	}
	c := fakeChain{pages: map[string]*chain.ActivityPage{"": pg("1"), "1": pg("2"), "2": pg("3")}}
	f, err := BuildFlows(context.Background(), reg(c), nil, "ethereum-mainnet", me, 2)
	if err != nil || f.PagesLoaded != 2 || f.HistoryEnd || f.ValuedCount != 0 {
		t.Fatalf("page limit: %+v %v", f, err)
	}
	if !contains(joinAll(f.Qualification), "most recent") || !contains(joinAll(f.Qualification), "price source is disabled") {
		t.Fatalf("qualifications: %v", f.Qualification)
	}
}

func joinAll(s []string) string {
	out := ""
	for _, x := range s {
		out += x + "|"
	}
	return out
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
