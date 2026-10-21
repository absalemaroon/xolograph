package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


func pad32(addr string) string { return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(addr, "0x") }

func fakeRPC(t *testing.T) *httptest.Server {
	t.Helper()
	from, to, token := "0x"+strings.Repeat("a1", 20), "0x"+strings.Repeat("b2", 20), "0x"+strings.Repeat("c3", 20)
	h := "0x" + strings.Repeat("ab", 32)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var res string
		switch req.Method {
		case "eth_getBalance":
			res = `"0xde0b6b3a7640000"`
		case "eth_getTransactionCount":
			res = `"0x5"`
		case "eth_getCode":
			res = `"0x"`
		case "eth_getTransactionByHash":
			res = fmt.Sprintf(`{"hash":%q,"blockNumber":"0x10","blockHash":"0x%s","from":%q,"to":%q,"value":"0xde0b6b3a7640000","gasPrice":"0x3b9aca00"}`, h, strings.Repeat("bb", 32), strings.ToUpper(from[:2])+from[2:], to)
		case "eth_getTransactionReceipt":
			res = fmt.Sprintf(`{"status":"0x1","gasUsed":"0x5208","effectiveGasPrice":"0x3b9aca00","contractAddress":null,"logs":[{"address":%q,"topics":["%s","%s","%s"],"data":"0x%064x","logIndex":"0x2"}]}`, token, transferTopic, pad32(from), pad32(to), 1500000)
		case "eth_getBlockByNumber":
			res = `{"timestamp":"0x65a1b2c3"}`
		case "eth_call":
			var p map[string]string
			_ = json.Unmarshal(req.Params[0], &p)
			if p["data"] == "0x95d89b41" {
				res = fmt.Sprintf(`"0x%064x%064x%s"`, 32, 4, "5553444300000000000000000000000000000000000000000000000000000000")
			} else {
				res = fmt.Sprintf(`"0x%064x"`, 6)
			}
		default:
			res = "null"
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%s}`, res)
	}))
}

func TestTransactionDecoding(t *testing.T) {
	srv := fakeRPC(t)
	defer srv.Close()
	e := NewEthereum(NewRPC(srv.URL), nil)
	tx, err := e.Transaction(context.Background(), "0x"+strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	if tx.Status != "SUCCESS" || tx.ValueWei != "1000000000000000000" || tx.FeeWei != "21000000000000" || tx.BlockNumber != 16 {
		t.Fatalf("bad tx: %+v", tx)
	}
	if tx.From != "0x"+strings.Repeat("a1", 20) || tx.Timestamp == nil {
		t.Fatalf("from/timestamp wrong: %+v", tx)
	}
	if len(tx.TokenTransfers) != 1 {
		t.Fatalf("want 1 token transfer, got %d", len(tx.TokenTransfers))
	}
	tt := tx.TokenTransfers[0]
	if tt.Standard != "ERC20" || tt.Value != "1500000" || tt.Symbol != "USDC" || tt.Decimals != 6 || tt.To != "0x"+strings.Repeat("b2", 20) {
		t.Fatalf("bad token transfer: %+v", tt)
	}
	if len(tx.Provenance) < 3 || tx.Provenance[0].PayloadSHA256 == "" || strings.Contains(tx.Provenance[0].Endpoint, "/") && !strings.HasPrefix(tx.Provenance[0].Endpoint, "http") {
		t.Fatalf("provenance missing: %+v", tx.Provenance)
	}
}

func TestTransactionNotFoundAndInvalid(t *testing.T) {
	srv := fakeRPC(t)
	defer srv.Close()
	e := NewEthereum(NewRPC(srv.URL), nil)
	if _, err := e.Transaction(context.Background(), "0x123"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if _, err := e.Address(context.Background(), "nope"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func fakeBlockscout(t *testing.T, seen *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.URL.RawQuery
		switch {
		case strings.HasSuffix(r.URL.Path, "/transactions"):
			next := `{"block_number":123,"index":5,"items_count":50}`
			if r.URL.Query().Get("block_number") != "" {
				next = "null"
			}
			_, _ = fmt.Fprintf(w, `{"items":[{"hash":"0xAB","timestamp":"2024-01-02T03:04:05.000000Z","block_number":99,"from":{"hash":"0xAAA0000000000000000000000000000000000001"},"to":null,"value":"5","status":"ok"}],"next_page_params":%s}`, next)
		case strings.HasSuffix(r.URL.Path, "/token-transfers"):
			_, _ = fmt.Fprint(w, `{"items":[{"transaction_hash":"0xCD","log_index":7,"block_number":100,"timestamp":"2024-01-02T03:04:05Z","from":{"hash":"0x01"},"to":{"hash":"0x02"},"token":{"address":"0xTOK","symbol":"USDT","decimals":"6","type":"ERC-20"},"total":{"value":"2500000","decimals":"6"}}],"next_page_params":null}`)
		default:
			_, _ = fmt.Fprint(w, `{"name":"Foo Router","is_contract":true,"public_tags":[{"label":"dex","display_name":"DEX Router"}]}`)
		}
	}))
}

func TestBlockscoutActivityAndCursor(t *testing.T) {
	var seen string
	srv := fakeBlockscout(t, &seen)
	defer srv.Close()
	bs := NewBlockscout(srv.URL)
	addr := "0x" + strings.Repeat("1f", 20)

	p1, err := bs.Activity(context.Background(), addr, "native", "")
	if err != nil || len(p1.Items) != 1 || p1.NextCursor == "" {
		t.Fatalf("page1: %+v %v", p1, err)
	}
	it := p1.Items[0]
	if it.Kind != "NATIVE" || it.To != "" || it.Status != "SUCCESS" || it.BlockNumber != 99 || it.LogIndex != -1 || it.Timestamp.IsZero() {
		t.Fatalf("bad native item: %+v", it)
	}
	if _, err := bs.Activity(context.Background(), addr, "native", p1.NextCursor); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"block_number=123", "index=5", "items_count=50"} {
		if !strings.Contains(seen, want) {
			t.Fatalf("cursor not forwarded, query=%q missing %q", seen, want)
		}
	}
	tk, err := bs.Activity(context.Background(), addr, "token", "")
	if err != nil || len(tk.Items) != 1 {
		t.Fatalf("token: %+v %v", tk, err)
	}
	if x := tk.Items[0]; x.Kind != "ERC20" || x.Decimals != 6 || x.Symbol != "USDT" || x.Value != "2500000" || x.LogIndex != 7 {
		t.Fatalf("bad token item: %+v", x)
	}
	if _, err := bs.Activity(context.Background(), addr, "bogus", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if _, err := bs.Activity(context.Background(), addr, "native", "!!!notbase64"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad cursor must be rejected, got %v", err)
	}
}

func TestAddressWithThirdPartyLabels(t *testing.T) {
	rpc := fakeRPC(t)
	defer rpc.Close()
	var seen string
	bsrv := fakeBlockscout(t, &seen)
	defer bsrv.Close()
	e := NewEthereum(NewRPC(rpc.URL), NewBlockscout(bsrv.URL))
	s, err := e.Address(context.Background(), "0x"+strings.Repeat("1f", 20))
	if err != nil {
		t.Fatal(err)
	}
	if s.BalanceWei != "1000000000000000000" || s.Nonce != 5 || s.IsContract {
		t.Fatalf("bad summary: %+v", s)
	}
	if len(s.ThirdPartyLabels) != 2 || s.ThirdPartyLabels[0].Source != "BLOCKSCOUT_V2" {
		t.Fatalf("labels: %+v", s.ThirdPartyLabels)
	}
}

func TestRedact(t *testing.T) {
	if got := redact("https://node.example/v2/SECRETKEY?x=1"); got != "https://node.example" {
		t.Fatalf("redact leaked: %s", got)
	}
}
