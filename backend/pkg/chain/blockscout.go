package chain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Blockscout reads address history and public labels from a Blockscout v2 REST API
// (no API key required). Data is third-party and carries that provenance.
type Blockscout struct {
	base   string
	hc     *http.Client
	symbol string // native currency symbol; "ETH" when unset
}

func (b *Blockscout) nativeSymbol() string {
	if b.symbol == "" {
		return "ETH"
	}
	return b.symbol
}

func NewBlockscout(base string) *Blockscout {
	return &Blockscout{base: strings.TrimRight(base, "/"), hc: &http.Client{Timeout: 20 * time.Second}}
}

type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	switch {
	case s == "null":
		*f = ""
	case strings.HasPrefix(s, `"`):
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexString(v)
	default:
		*f = flexString(s)
	}
	return nil
}

type bsParty struct {
	Hash string `json:"hash"`
}

type bsTx struct {
	Hash        string   `json:"hash"`
	Timestamp   string   `json:"timestamp"`
	BlockNumber *uint64  `json:"block_number"`
	Block       *uint64  `json:"block"`
	From        bsParty  `json:"from"`
	To          *bsParty `json:"to"`
	Value       string   `json:"value"`
	Status      string   `json:"status"`
}

type bsTokenTransfer struct {
	TxHash      string     `json:"transaction_hash"`
	LogIndex    flexString `json:"log_index"`
	BlockNumber *uint64    `json:"block_number"`
	Timestamp   string     `json:"timestamp"`
	From        bsParty    `json:"from"`
	To          bsParty    `json:"to"`
	Token       struct {
		Address     string     `json:"address"`
		AddressHash string     `json:"address_hash"`
		Symbol      string     `json:"symbol"`
		Decimals    flexString `json:"decimals"`
		Type        string     `json:"type"`
	} `json:"token"`
	Total struct {
		Value    string     `json:"value"`
		Decimals flexString `json:"decimals"`
		TokenID  string     `json:"token_id"`
	} `json:"total"`
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func (t bsTx) transfer(symbol string) Transfer {
	var bn uint64
	if t.BlockNumber != nil {
		bn = *t.BlockNumber
	} else if t.Block != nil {
		bn = *t.Block
	}
	to := ""
	if t.To != nil {
		to = strings.ToLower(t.To.Hash)
	}
	status := "UNKNOWN"
	switch strings.ToLower(t.Status) {
	case "ok":
		status = "SUCCESS"
	case "error":
		status = "FAILED"
	}
	return Transfer{
		TxHash: strings.ToLower(t.Hash), LogIndex: -1, Kind: "NATIVE",
		From: strings.ToLower(t.From.Hash), To: to, Value: orZero(t.Value), Decimals: 18, Symbol: symbol,
		Timestamp: parseTime(t.Timestamp), BlockNumber: bn, Status: status,
	}
}

func (t bsTokenTransfer) transfer() Transfer {
	addr := t.Token.Address
	if addr == "" {
		addr = t.Token.AddressHash
	}
	dec := t.Total.Decimals
	if dec == "" {
		dec = t.Token.Decimals
	}
	d := -1
	if n, err := strconv.Atoi(string(dec)); err == nil && n >= 0 && n <= 77 {
		d = n
	}
	li := int64(0)
	if n, err := strconv.ParseInt(string(t.LogIndex), 10, 64); err == nil {
		li = n
	}
	var bn uint64
	if t.BlockNumber != nil {
		bn = *t.BlockNumber
	}
	return Transfer{
		TxHash: strings.ToLower(t.TxHash), LogIndex: li,
		Kind: strings.ReplaceAll(strings.ToUpper(t.Token.Type), "-", ""),
		From: strings.ToLower(t.From.Hash), To: strings.ToLower(t.To.Hash),
		Value: orZero(t.Total.Value), Decimals: d, Symbol: sanitize(t.Token.Symbol),
		TokenAddress: strings.ToLower(addr), TokenID: t.Total.TokenID,
		Timestamp: parseTime(t.Timestamp), BlockNumber: bn, Status: "SUCCESS",
	}
}

func (b *Blockscout) get(ctx context.Context, path string, q url.Values, out any) (Provenance, error) {
	u := b.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Provenance{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := b.hc.Do(req)
	if err != nil {
		return Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Provenance{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return Provenance{}, fmt.Errorf("%w: not indexed by the history source", ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return Provenance{}, fmt.Errorf("%w: history source HTTP %d", ErrUpstream, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return Provenance{}, fmt.Errorf("%w: malformed history response", ErrUpstream)
	}
	return Provenance{
		Source: "BLOCKSCOUT_V2", Endpoint: redact(b.base), Query: path + "?" + q.Encode(),
		RetrievedAt: time.Now().UTC(), PayloadSHA256: sum(raw),
		Coverage: "Third-party index; may lag the chain head and may omit internal transactions. Not a complete or authoritative history.",
	}, nil
}

func (b *Blockscout) AddressLabels(ctx context.Context, addr string) ([]ThirdPartyLabel, Provenance, error) {
	var meta struct {
		Name       *string `json:"name"`
		PublicTags []struct {
			Label       string `json:"label"`
			DisplayName string `json:"display_name"`
		} `json:"public_tags"`
	}
	p, err := b.get(ctx, "/api/v2/addresses/"+addr, nil, &meta)
	if err != nil {
		return nil, Provenance{}, err
	}
	labels := []ThirdPartyLabel{}
	if meta.Name != nil && *meta.Name != "" {
		labels = append(labels, ThirdPartyLabel{Name: sanitize(*meta.Name), Kind: "CONTRACT_NAME", Source: "BLOCKSCOUT_V2"})
	}
	for _, t := range meta.PublicTags {
		n := t.DisplayName
		if n == "" {
			n = t.Label
		}
		if n != "" {
			labels = append(labels, ThirdPartyLabel{Name: sanitize(n), Kind: "PUBLIC_TAG", Source: "BLOCKSCOUT_V2"})
		}
	}
	return labels, p, nil
}

var cursorKeyRe = regexp.MustCompile(`^[a-z_]{1,32}$`)

func encodeCursor(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if dec.Decode(&m) != nil {
		return ""
	}
	out := map[string]string{}
	for k, v := range m {
		switch t := v.(type) {
		case string:
			out[k] = t
		case json.Number:
			out[k] = t.String()
		case bool:
			out[k] = strconv.FormatBool(t)
		}
	}
	b, _ := json.Marshal(out)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(c string) (url.Values, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return nil, fmt.Errorf("%w: bad cursor", ErrInvalid)
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil || len(m) > 8 {
		return nil, fmt.Errorf("%w: bad cursor", ErrInvalid)
	}
	q := url.Values{}
	for k, v := range m {
		if !cursorKeyRe.MatchString(k) || len(v) > 100 {
			return nil, fmt.Errorf("%w: bad cursor", ErrInvalid)
		}
		q.Set(k, v)
	}
	return q, nil
}

func (b *Blockscout) Activity(ctx context.Context, addr, kind, cursor string) (*ActivityPage, error) {
	a, err := NormalizeAddress(addr)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if cursor != "" {
		if q, err = decodeCursor(cursor); err != nil {
			return nil, err
		}
	}
	page := &ActivityPage{Address: a, Kind: kind, Items: []Transfer{}}
	switch kind {
	case "native":
		var resp struct {
			Items []bsTx          `json:"items"`
			Next  json.RawMessage `json:"next_page_params"`
		}
		if page.Provenance, err = b.get(ctx, "/api/v2/addresses/"+a+"/transactions", q, &resp); err != nil {
			return nil, err
		}
		for _, it := range resp.Items {
			page.Items = append(page.Items, it.transfer(b.nativeSymbol()))
		}
		page.NextCursor = encodeCursor(resp.Next)
	case "token":
		var resp struct {
			Items []bsTokenTransfer `json:"items"`
			Next  json.RawMessage   `json:"next_page_params"`
		}
		if page.Provenance, err = b.get(ctx, "/api/v2/addresses/"+a+"/token-transfers", q, &resp); err != nil {
			return nil, err
		}
		for _, it := range resp.Items {
			page.Items = append(page.Items, it.transfer())
		}
		page.NextCursor = encodeCursor(resp.Next)
	default:
		return nil, fmt.Errorf("%w: kind must be native or token", ErrInvalid)
	}
	return page, nil
}
