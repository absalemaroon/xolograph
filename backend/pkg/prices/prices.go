// Package prices fetches fiat reference prices. Every value carries its source, timestamp and provenance;
// a missing price is reported as missing, never as zero.
package prices

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xolograph/backend/pkg/chain"
)

var ErrUpstream = errors.New("price source error")

// Quote is one USD price. At is the provider's own last-updated time when it reports one, else the retrieval time.
type Quote struct {
	USD    float64   `json:"usd"`
	At     time.Time `json:"at"`
	Source string    `json:"source"`
}

type Point struct {
	T   time.Time
	USD float64
}

// Provider supplies reference prices. Implementations must be safe for concurrent use.
type Provider interface {
	NativeSpot(ctx context.Context, coinID string) (Quote, chain.Provenance, error)
	// TokenSpot returns quotes keyed by the lower-cased contract/mint address; unknown tokens are simply absent.
	TokenSpot(ctx context.Context, platform string, contracts []string) (map[string]Quote, chain.Provenance, error)
	History(ctx context.Context, coinID string, from, to time.Time) ([]Point, chain.Provenance, error)
}

// Coin ids and token platforms used by CoinGecko, per Xolograph network id.
var (
	NativeCoin = map[string]string{
		"ethereum-mainnet": "ethereum", "base": "ethereum", "optimism": "ethereum", "arbitrum": "ethereum", "scroll": "ethereum",
		"polygon": "polygon-ecosystem-token", "bsc": "binancecoin", "gnosis": "xdai", "avalanche": "avalanche-2", "celo": "celo",
		"solana": "solana", "bitcoin": "bitcoin",
	}
	TokenPlatform = map[string]string{
		"ethereum-mainnet": "ethereum", "base": "base", "optimism": "optimistic-ethereum", "arbitrum": "arbitrum-one", "polygon": "polygon-pos",
		"bsc": "binance-smart-chain", "gnosis": "xdai", "avalanche": "avalanche", "scroll": "scroll", "celo": "celo", "solana": "solana",
	}
)

const maxBody = 4 << 20

// CoinGecko implements Provider against the public (or demo-keyed) CoinGecko v3 API.
type CoinGecko struct {
	base string
	key  string
	hc   *http.Client
	ttl  time.Duration
	mu   sync.Mutex
	memo map[string]memo
}

type memo struct {
	raw  []byte
	prov chain.Provenance
	at   time.Time
}

func NewCoinGecko(base, apiKey string) *CoinGecko {
	if base == "" {
		base = "https://api.coingecko.com/api/v3"
	}
	return &CoinGecko{base: strings.TrimRight(base, "/"), key: apiKey, hc: &http.Client{Timeout: 20 * time.Second}, ttl: 60 * time.Second, memo: map[string]memo{}}
}

func redactHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(redacted)"
	}
	return u.Scheme + "://" + u.Host
}

// get returns the raw body and its provenance, serving repeated identical requests from a short cache
// (the cached provenance keeps its original retrieval time).
func (c *CoinGecko) get(ctx context.Context, path string, q url.Values, coverage string) ([]byte, chain.Provenance, error) {
	full := path + "?" + q.Encode()
	c.mu.Lock()
	if m, ok := c.memo[full]; ok && time.Since(m.at) < c.ttl {
		c.mu.Unlock()
		return m.raw, m.prov, nil
	}
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+full, nil)
	if err != nil {
		return nil, chain.Provenance{}, err
	}
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("x-cg-demo-api-key", c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, chain.Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, chain.Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, chain.Provenance{}, fmt.Errorf("%w: rate limited by the price source; try again shortly", ErrUpstream)
	case resp.StatusCode != http.StatusOK:
		return nil, chain.Provenance{}, fmt.Errorf("%w: HTTP %d", ErrUpstream, resp.StatusCode)
	}
	h := sha256.Sum256(raw)
	prov := chain.Provenance{Source: "COINGECKO_V3", Endpoint: redactHost(c.base), Query: full, RetrievedAt: time.Now().UTC(), PayloadSHA256: hex.EncodeToString(h[:]), Coverage: coverage}
	c.mu.Lock()
	c.memo[full] = memo{raw: raw, prov: prov, at: time.Now()}
	c.mu.Unlock()
	return raw, prov, nil
}

const spotCoverage = "Aggregated market reference price (USD) from a third party; not an execution price, and it can lag or be wrong for illiquid or manipulated assets."

type spotEntry struct {
	USD *float64 `json:"usd"`
	At  *int64   `json:"last_updated_at"`
}

func quoteOf(e spotEntry, retrieved time.Time) (Quote, bool) {
	if e.USD == nil {
		return Quote{}, false
	}
	at := retrieved
	if e.At != nil && *e.At > 0 {
		at = time.Unix(*e.At, 0).UTC()
	}
	return Quote{USD: *e.USD, At: at, Source: "COINGECKO_V3"}, true
}

func (c *CoinGecko) NativeSpot(ctx context.Context, coinID string) (Quote, chain.Provenance, error) {
	q := url.Values{"ids": {coinID}, "vs_currencies": {"usd"}, "include_last_updated_at": {"true"}}
	raw, prov, err := c.get(ctx, "/simple/price", q, spotCoverage)
	if err != nil {
		return Quote{}, prov, err
	}
	var out map[string]spotEntry
	if json.Unmarshal(raw, &out) != nil {
		return Quote{}, prov, fmt.Errorf("%w: malformed price response", ErrUpstream)
	}
	qt, ok := quoteOf(out[coinID], prov.RetrievedAt)
	if !ok {
		return Quote{}, prov, fmt.Errorf("%w: no price for %s", ErrUpstream, coinID)
	}
	return qt, prov, nil
}

func (c *CoinGecko) TokenSpot(ctx context.Context, platform string, contracts []string) (map[string]Quote, chain.Provenance, error) {
	out := map[string]Quote{}
	if len(contracts) == 0 {
		return out, chain.Provenance{}, nil
	}
	if len(contracts) > 50 {
		contracts = contracts[:50] // keep the request short; the caller reports the rest as unpriced
	}
	q := url.Values{"contract_addresses": {strings.Join(contracts, ",")}, "vs_currencies": {"usd"}, "include_last_updated_at": {"true"}}
	raw, prov, err := c.get(ctx, "/simple/token_price/"+url.PathEscape(platform), q, spotCoverage)
	if err != nil {
		return nil, prov, err
	}
	var resp map[string]spotEntry
	if json.Unmarshal(raw, &resp) != nil {
		return nil, prov, fmt.Errorf("%w: malformed price response", ErrUpstream)
	}
	for k, v := range resp {
		if qt, ok := quoteOf(v, prov.RetrievedAt); ok {
			out[strings.ToLower(k)] = qt
		}
	}
	return out, prov, nil
}

func (c *CoinGecko) History(ctx context.Context, coinID string, from, to time.Time) ([]Point, chain.Provenance, error) {
	if !to.After(from) {
		return nil, chain.Provenance{}, fmt.Errorf("%w: empty time range", ErrUpstream)
	}
	q := url.Values{"vs_currency": {"usd"}, "from": {fmt.Sprint(from.Unix())}, "to": {fmt.Sprint(to.Unix())}}
	raw, prov, err := c.get(ctx, "/coins/"+url.PathEscape(coinID)+"/market_chart/range", q,
		"Historical market reference prices (USD) from a third party; granularity depends on the range (hourly up to 90 days, otherwise daily).")
	if err != nil {
		return nil, prov, err
	}
	var resp struct {
		Prices [][]float64 `json:"prices"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil, prov, fmt.Errorf("%w: malformed history response", ErrUpstream)
	}
	pts := make([]Point, 0, len(resp.Prices))
	for _, p := range resp.Prices {
		if len(p) == 2 {
			pts = append(pts, Point{T: time.UnixMilli(int64(p[0])).UTC(), USD: p[1]})
		}
	}
	return pts, prov, nil
}

// Nearest returns the price point closest in time to t, and false when the series is empty or the nearest point
// is further than maxGap away (so a transfer is never valued with a price from a different era).
func Nearest(pts []Point, t time.Time, maxGap time.Duration) (Point, bool) {
	var best Point
	bestD := time.Duration(1<<63 - 1)
	for _, p := range pts {
		d := p.T.Sub(t)
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = p, d
		}
	}
	if bestD > maxGap || len(pts) == 0 {
		return Point{}, false
	}
	return best, true
}
