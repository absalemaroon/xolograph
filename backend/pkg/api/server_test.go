package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xolograph/backend/pkg/store"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st, "secret", nil).Handler()
}

func do(h http.Handler, method, path, body, token, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthOriginAndBasicFlow(t *testing.T) {
	h := newTestServer(t)
	if c := do(h, "GET", "/api/health", "", "", "").Code; c != 200 {
		t.Fatalf("health must be public, got %d", c)
	}
	if c := do(h, "GET", "/api/cases", "", "", "").Code; c != 401 {
		t.Fatalf("missing token must be 401, got %d", c)
	}
	if c := do(h, "GET", "/api/cases", "", "wrong", "").Code; c != 401 {
		t.Fatalf("wrong token must be 401, got %d", c)
	}
	if c := do(h, "GET", "/api/cases", "", "secret", "https://evil.example").Code; c != 403 {
		t.Fatalf("foreign origin must be 403, got %d", c)
	}
	if c := do(h, "POST", "/api/cases", `{"title":"T"}`, "secret", "http://localhost:3000").Code; c != 201 {
		t.Fatalf("create case: %d", c)
	}
	if c := do(h, "POST", "/api/cases", `{"title":"T","bogus":1}`, "secret", "").Code; c != 400 {
		t.Fatalf("unknown fields must be 400, got %d", c)
	}
	if c := do(h, "GET", "/api/chain/address/0x"+strings.Repeat("a", 40), "", "secret", "").Code; c != 503 {
		t.Fatalf("unconfigured chain must be 503, got %d", c)
	}
}
