package chain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid input")
	ErrNotFound = errors.New("not found")
	ErrUpstream = errors.New("upstream data source error")
)

var (
	addrRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	hashRe = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
)

const maxBody = 8 << 20

func NormalizeAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !addrRe.MatchString(s) {
		return "", fmt.Errorf("%w: not a valid Ethereum address", ErrInvalid)
	}
	return strings.ToLower(s), nil
}

func NormalizeHash(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !hashRe.MatchString(s) {
		return "", fmt.Errorf("%w: not a valid transaction hash", ErrInvalid)
	}
	return strings.ToLower(s), nil
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// redact keeps only scheme://host so API keys in paths/queries never reach provenance.
func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(redacted)"
	}
	return u.Scheme + "://" + u.Host
}

func hexBig(s string) *big.Int {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return new(big.Int)
	}
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return new(big.Int)
	}
	return v
}

func hexUint(s string) uint64 { return hexBig(s).Uint64() }

func unquote(raw json.RawMessage) string {
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

// RPC is a minimal Ethereum JSON-RPC client.
type RPC struct {
	endpoint string
	hc       *http.Client
}

func NewRPC(endpoint string) *RPC {
	return &RPC{endpoint: endpoint, hc: &http.Client{Timeout: 15 * time.Second}}
}

func (r *RPC) Call(ctx context.Context, method string, params ...any) (json.RawMessage, Provenance, error) {
	if params == nil {
		params = []any{}
	}
	pj, _ := json.Marshal(params)
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, Provenance{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.hc.Do(req)
	if err != nil {
		return nil, Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, Provenance{}, fmt.Errorf("%w: RPC HTTP %d", ErrUpstream, resp.StatusCode)
	}
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, Provenance{}, fmt.Errorf("%w: malformed RPC response", ErrUpstream)
	}
	if out.Error != nil {
		return nil, Provenance{}, fmt.Errorf("%w: RPC error %d: %s", ErrUpstream, out.Error.Code, out.Error.Message)
	}
	return out.Result, Provenance{
		Source: "ETHEREUM_JSON_RPC", Endpoint: redact(r.endpoint), Query: method + string(pj),
		RetrievedAt: time.Now().UTC(), PayloadSHA256: sum(raw),
		Coverage: "Node state at query time; depends on the node's pruning policy.",
	}, nil
}
