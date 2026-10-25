package prices

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoinGeckoSpotHistoryAndCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Header.Get("x-cg-demo-api-key") != "k" {
			t.Errorf("api key header missing")
		}
		switch r.URL.Path {
		case "/simple/price":
			_, _ = w.Write([]byte(`{"ethereum":{"usd":2500.5,"last_updated_at":1700000000}}`))
		case "/simple/token_price/ethereum":
			_, _ = w.Write([]byte(`{"0xAAAAaaaaAAAAaaaaAAAAaaaaAAAAaaaaAAAAaaaa":{"usd":1.01},"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb":{}}`))
		case "/coins/ethereum/market_chart/range":
			_, _ = w.Write([]byte(`{"prices":[[1700000000000,2000],[1700086400000,2100]]}`))
		case "/simple/price-bad":
		default:
			http.Error(w, "nope", http.StatusTooManyRequests)
		}
	}))
	defer srv.Close()
	c := NewCoinGecko(srv.URL, "k")
	ctx := context.Background()

	q, p, err := c.NativeSpot(ctx, "ethereum")
	if err != nil || q.USD != 2500.5 || q.At.Unix() != 1700000000 || p.Source != "COINGECKO_V3" || p.PayloadSHA256 == "" {
		t.Fatalf("native: %+v %+v %v", q, p, err)
	}
	if _, _, err := c.NativeSpot(ctx, "ethereum"); err != nil || atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("second identical call must come from the cache: hits=%d err=%v", hits, err)
	}
	if _, _, err := c.NativeSpot(ctx, "not-a-coin"); !errors.Is(err, ErrUpstream) {
		t.Fatalf("missing coin must be an error, not a zero price: %v", err)
	}

	toks, _, err := c.TokenSpot(ctx, "ethereum", []string{"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	if err != nil || len(toks) != 1 || toks["0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].USD != 1.01 {
		t.Fatalf("tokens (unpriced ones must be absent): %+v %v", toks, err)
	}

	pts, _, err := c.History(ctx, "ethereum", time.Unix(1700000000, 0), time.Unix(1700090000, 0))
	if err != nil || len(pts) != 2 || pts[1].USD != 2100 {
		t.Fatalf("history: %+v %v", pts, err)
	}
	if _, _, err := c.History(ctx, "ethereum", time.Unix(5, 0), time.Unix(5, 0)); !errors.Is(err, ErrUpstream) {
		t.Fatal("empty range accepted")
	}
	if _, _, err := c.History(ctx, "unknown-coin", time.Unix(1, 0), time.Unix(100, 0)); err == nil || !errors.Is(err, ErrUpstream) {
		t.Fatalf("429 must surface as an upstream error: %v", err)
	}
}

func TestNearestRefusesDistantPrices(t *testing.T) {
	base := time.Unix(1700000000, 0)
	pts := []Point{{T: base, USD: 1}, {T: base.Add(48 * time.Hour), USD: 2}}
	if p, ok := Nearest(pts, base.Add(3*time.Hour), 24*time.Hour); !ok || p.USD != 1 {
		t.Fatalf("nearest: %+v %v", p, ok)
	}
	if _, ok := Nearest(pts, base.Add(24*time.Hour+time.Minute), 12*time.Hour); ok {
		t.Fatal("a price 24h away must not be used with a 12h limit")
	}
	if _, ok := Nearest(nil, base, time.Hour); ok {
		t.Fatal("empty series")
	}
}
