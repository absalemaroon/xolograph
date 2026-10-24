package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	solA    = "So11111111111111111111111111111111111111112" // 32-byte keys used as stand-in accounts
	solB    = "Vote111111111111111111111111111111111111111"
	solSrc  = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	solDst  = "SysvarC1ock11111111111111111111111111111111"
	solMint = "ComputeBudget111111111111111111111111111111"
	btcBech = "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"
	btcOld  = "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"
)

func b58enc(b []byte) string {
	n := new(big.Int).SetBytes(b)
	base, mod := big.NewInt(58), new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, mod)
		out = append([]byte{b58Alphabet[mod.Int64()]}, out...)
	}
	for _, c := range b {
		if c != 0 {
			break
		}
		out = append([]byte{'1'}, out...)
	}
	return string(out)
}

func TestBase58AndNormalizers(t *testing.T) {
	if b, ok := base58Decode(solA); !ok || len(b) != 32 {
		t.Fatalf("wSOL mint should decode to 32 bytes: %d %v", len(b), ok)
	}
	if b, ok := base58Decode(strings.Repeat("1", 32)); !ok || len(b) != 32 {
		t.Fatal("leading ones are zero bytes")
	}
	if _, ok := base58Decode("0OIl"); ok {
		t.Fatal("characters outside the alphabet accepted")
	}
	if _, err := NormalizeSolanaAddress("0x" + strings.Repeat("a", 40)); !errors.Is(err, ErrInvalid) {
		t.Fatal("EVM address accepted as Solana")
	}
	if _, err := NormalizeSolanaSignature(solA); !errors.Is(err, ErrInvalid) {
		t.Fatal("32-byte key accepted as a signature")
	}
	sig := b58enc(make([]byte, 64))
	if _, err := NormalizeSolanaSignature(sig); err != nil {
		t.Fatalf("64-byte signature rejected: %v", err)
	}
	if got, err := NormalizeBitcoinAddress(strings.ToUpper(btcBech)); err != nil || got != btcBech {
		t.Fatalf("bech32 normalisation: %q %v", got, err)
	}
	if _, err := NormalizeBitcoinAddress("0x" + strings.Repeat("a", 40)); !errors.Is(err, ErrInvalid) {
		t.Fatal("EVM address accepted as Bitcoin")
	}
}

func solanaServer(t *testing.T, sig string) *httptest.Server {
	t.Helper()
	tx := fmt.Sprintf(`{"slot":777,"blockTime":1700000000,
	  "meta":{"err":null,"fee":5000,"computeUnitsConsumed":1234,
	    "preTokenBalances":[{"accountIndex":2,"mint":%q,"owner":%q,"uiTokenAmount":{"decimals":6}},{"accountIndex":3,"mint":%q,"owner":%q,"uiTokenAmount":{"decimals":6}}],
	    "postTokenBalances":[],"innerInstructions":[]},
	  "transaction":{"signatures":[%q],"message":{
	    "accountKeys":[{"pubkey":%q,"signer":true},{"pubkey":%q,"signer":false},{"pubkey":%q,"signer":false},{"pubkey":%q,"signer":false}],
	    "instructions":[
	      {"program":"system","parsed":{"type":"transfer","info":{"source":%q,"destination":%q,"lamports":1500000000}}},
	      {"program":"spl-token","parsed":{"type":"transferChecked","info":{"source":%q,"destination":%q,"mint":%q,"authority":%q,"tokenAmount":{"amount":"2500000","decimals":6}}}},
	      {"program":"spl-memo","parsed":"hello"}]}}}`,
		solMint, solA, solMint, solB, sig, solA, solB, solSrc, solDst, solA, solB, solSrc, solDst, solMint, solA)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var res string
		switch req.Method {
		case "getBalance":
			res = `{"context":{"slot":1},"value":2500000000}`
		case "getAccountInfo":
			res = `{"context":{"slot":1},"value":{"owner":"11111111111111111111111111111111","executable":false}}`
		case "getSignaturesForAddress":
			res = fmt.Sprintf(`[{"signature":%q}]`, sig)
		case "getTransaction":
			res = tx
		default:
			res = `null`
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + res + `}`))
	}))
}

func TestSolanaAdapter(t *testing.T) {
	sig := b58enc(append([]byte{1}, make([]byte, 63)...))
	srv := solanaServer(t, sig)
	defer srv.Close()
	s := NewSolana(NewRPC(srv.URL))
	ctx := context.Background()

	a, err := s.Address(ctx, solA)
	if err != nil || a.BalanceWei != "2500000000" || a.NativeSymbol != "SOL" || a.NativeDecimals != 9 || a.IsContract || a.Network != "solana" {
		t.Fatalf("address: %+v %v", a, err)
	}
	if len(a.Provenance) != 2 || a.Provenance[0].Source != "SOLANA_JSON_RPC" || a.Provenance[0].PayloadSHA256 == "" {
		t.Fatalf("provenance: %+v", a.Provenance)
	}

	tx, err := s.Transaction(ctx, sig)
	if err != nil || tx.Status != "SUCCESS" || tx.FeeWei != "5000" || tx.GasUsed != 1234 || tx.BlockNumber != 777 || tx.From != solA || tx.ValueWei != "1500000000" {
		t.Fatalf("tx: %+v %v", tx, err)
	}
	if len(tx.TokenTransfers) != 2 || tx.TokenTransfers[0].Standard != "NATIVE" || tx.TokenTransfers[1].Standard != "SPL" ||
		tx.TokenTransfers[1].From != solA || tx.TokenTransfers[1].To != solB || tx.TokenTransfers[1].Decimals != 6 || tx.TokenTransfers[1].Token != solMint {
		t.Fatalf("transfers (token accounts must resolve to owners): %+v", tx.TokenTransfers)
	}

	nat, err := s.Activity(ctx, solA, "native", "")
	if err != nil || len(nat.Items) != 1 || nat.Items[0].Value != "1500000000" || nat.Items[0].From != solA || nat.Items[0].To != solB || nat.Items[0].Symbol != "SOL" || nat.NextCursor != "" {
		t.Fatalf("native activity: %+v %v", nat, err)
	}
	tok, err := s.Activity(ctx, solB, "token", "")
	if err != nil || len(tok.Items) != 1 || tok.Items[0].Kind != "SPL" || tok.Items[0].Value != "2500000" || tok.Items[0].TokenAddress != solMint || tok.Items[0].Decimals != 6 {
		t.Fatalf("token activity: %+v %v", tok, err)
	}
	if _, err := s.Activity(ctx, solA, "native", "not-a-signature"); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad cursor accepted")
	}
	if _, err := s.Transaction(ctx, solA); !errors.Is(err, ErrInvalid) {
		t.Fatal("address accepted as a signature")
	}
}

func TestBitcoinAdapter(t *testing.T) {
	txid := strings.Repeat("ab", 32)
	tx := fmt.Sprintf(`{"txid":%q,"fee":1000,
	  "vin":[{"is_coinbase":false,"prevout":{"scriptpubkey_address":%q,"scriptpubkey_type":"p2pkh","value":100000}}],
	  "vout":[{"scriptpubkey_address":%q,"scriptpubkey_type":"v0_p2wpkh","value":60000},{"scriptpubkey_address":%q,"scriptpubkey_type":"p2pkh","value":39000}],
	  "status":{"confirmed":true,"block_height":800000,"block_hash":"00ff","block_time":1700000000}}`, txid, btcOld, btcBech, btcOld)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/address/"+btcBech:
			_, _ = w.Write([]byte(`{"chain_stats":{"funded_txo_sum":160000,"spent_txo_sum":60000,"tx_count":2},"mempool_stats":{"funded_txo_sum":0,"spent_txo_sum":0,"tx_count":0}}`))
		case strings.HasSuffix(r.URL.Path, "/txs"):
			_, _ = w.Write([]byte("[" + tx + "]"))
		case r.URL.Path == "/tx/"+txid:
			_, _ = w.Write([]byte(tx))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	b := NewBitcoin(srv.URL)
	ctx := context.Background()

	a, err := b.Address(ctx, strings.ToUpper(btcBech))
	if err != nil || a.BalanceWei != "100000" || a.NativeSymbol != "BTC" || a.NativeDecimals != 8 {
		t.Fatalf("address: %+v %v", a, err)
	}
	if _, err := b.Address(ctx, btcOld); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown address should be not found: %v", err)
	}
	got, err := b.Transaction(ctx, strings.ToUpper(txid))
	if err != nil || got.Status != "SUCCESS" || got.FeeWei != "1000" || got.ValueWei != "99000" || got.From != btcOld || len(got.Inputs) != 1 || len(got.Outputs) != 2 || got.BlockNumber != 800000 {
		t.Fatalf("tx: %+v %v", got, err)
	}
	in, err := b.Activity(ctx, btcBech, "native", "")
	if err != nil || len(in.Items) != 1 || in.Items[0].Value != "60000" || in.Items[0].From != btcOld || in.Items[0].To != btcBech {
		t.Fatalf("incoming: %+v %v", in, err)
	}
	out, err := b.Activity(ctx, btcOld, "native", "")
	if err != nil || len(out.Items) != 1 || out.Items[0].Value != "61000" || out.Items[0].From != btcOld || out.Items[0].To != btcBech {
		t.Fatalf("outgoing (net of change): %+v %v", out, err)
	}
	if tok, err := b.Activity(ctx, btcBech, "token", ""); err != nil || len(tok.Items) != 0 {
		t.Fatalf("token kind: %+v %v", tok, err)
	}
}

func TestRegistryOrdersEthereumFirst(t *testing.T) {
	r := NewRegistry()
	r.Add(NetworkInfo{ID: "solana", Name: "Solana"}, nil)
	r.Add(NetworkInfo{ID: "base", Name: "Base"}, nil)
	r.Add(NetworkInfo{ID: "ethereum-mainnet", Name: "Ethereum"}, nil)
	ns := r.Networks()
	if len(ns) != 3 || ns[0].ID != "ethereum-mainnet" || ns[1].ID != "base" || ns[2].ID != "solana" {
		t.Fatalf("order: %+v", ns)
	}
	if _, _, ok := r.Get("nope"); ok {
		t.Fatal("unknown network found")
	}
}
