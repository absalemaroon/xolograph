// Package api exposes the store over a token-protected loopback HTTP API.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/xolograph/backend/pkg/chain"
	"github.com/xolograph/backend/pkg/inspect"
	"github.com/xolograph/backend/pkg/prices"
	"github.com/xolograph/backend/pkg/store"
)

// ChainProvider is the blockchain data source used by the explorer endpoints.
type ChainProvider interface {
	Address(ctx context.Context, addr string) (*chain.AddressSummary, error)
	Transaction(ctx context.Context, hash string) (*chain.Transaction, error)
	Activity(ctx context.Context, addr, kind, cursor string) (*chain.ActivityPage, error)
}

type Server struct {
	st    *store.Store
	token string
	chain ChainProvider // may be nil: chain endpoints then answer 503
	reg   *chain.Registry // multi-network registry; when set it takes precedence over chain
	px    prices.Provider // nil when prices are disabled
}

// WithPrices enables valuation. Without it portfolios list holdings but value nothing.
func (s *Server) WithPrices(px prices.Provider) *Server { s.px = px; return s }

func New(st *store.Store, token string, cp ChainProvider) *Server {
	return &Server{st: st, token: token, chain: cp}
}

// NewMulti serves every network in the registry; the `network` query parameter selects one (default ethereum-mainnet).
func NewMulti(st *store.Store, token string, reg *chain.Registry) *Server {
	return &Server{st: st, token: token, reg: reg}
}

// provider resolves the data source for the request's `network` parameter, writing the error response itself on failure.
func (s *Server) provider(w http.ResponseWriter, r *http.Request) (ChainProvider, bool) {
	network := r.URL.Query().Get("network")
	if network == "" {
		network = "ethereum-mainnet"
	}
	if s.reg != nil {
		p, _, ok := s.reg.Get(network)
		if !ok || p == nil {
			writeErr(w, 400, "unsupported network: "+network)
			return nil, false
		}
		return p, true
	}
	if s.chain == nil {
		writeErr(w, 503, "blockchain data source not configured")
		return nil, false
	}
	if network != "ethereum-mainnet" {
		writeErr(w, 400, "unsupported network: "+network)
		return nil, false
	}
	return s.chain, true
}

func (s *Server) chainNetworks(w http.ResponseWriter, r *http.Request) {
	if s.reg != nil {
		writeJSON(w, 200, s.reg.Networks())
		return
	}
	writeJSON(w, 200, []chain.NetworkInfo{{ID: "ethereum-mainnet", Name: "Ethereum", Family: "EVM", NativeSymbol: "ETH", NativeDecimals: 18, HasHistory: true}})
}

var allowedOrigins = map[string]bool{
	"http://localhost:3000":  true, // vite dev
	"http://127.0.0.1:3000":  true,
	"tauri://localhost":      true, // tauri (macOS/Linux)
	"http://tauri.localhost": true, // tauri (Windows)
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "xolographd"})
	})
	m.HandleFunc("GET /api/cases", s.listCases)
	m.HandleFunc("POST /api/cases", s.createCase)
	m.HandleFunc("GET /api/cases/{id}", s.getCase)
	m.HandleFunc("PATCH /api/cases/{id}", s.patchCase)
	m.HandleFunc("GET /api/cases/{id}/board", s.getBoard)
	m.HandleFunc("POST /api/cases/{id}/board/nodes", s.addBoardNode)
	m.HandleFunc("POST /api/cases/{id}/board/positions", s.moveBoardNodes)
	m.HandleFunc("POST /api/cases/{id}/board/links", s.addBoardLink)
	m.HandleFunc("PATCH /api/board-nodes/{id}", s.patchBoardNode)
	m.HandleFunc("DELETE /api/board-nodes/{id}", s.deleteBoardNode)
	m.HandleFunc("PATCH /api/board-links/{id}", s.patchBoardLink)
	m.HandleFunc("DELETE /api/board-links/{id}", s.deleteBoardLink)
	m.HandleFunc("POST /api/board-links/{id}/evidence", s.addBoardLinkEvidence)
	m.HandleFunc("DELETE /api/board-link-evidence/{id}", s.deleteBoardLinkEvidence)
	m.HandleFunc("GET /api/cases/{id}/notes", s.listNotes)
	m.HandleFunc("POST /api/cases/{id}/notes", s.addNote)
	m.HandleFunc("GET /api/cases/{id}/evidence", s.listEvidence)
	m.HandleFunc("POST /api/cases/{id}/evidence", s.importEvidence)
	m.HandleFunc("GET /api/cases/{id}/findings", s.listFindings)
	m.HandleFunc("POST /api/cases/{id}/findings", s.addFinding)
	m.HandleFunc("POST /api/evidence/{id}/verify", s.verifyEvidence)
	m.HandleFunc("GET /api/evidence/{id}/custody", s.custody)
	m.HandleFunc("POST /api/tools/inspect", s.inspectFile)
	m.HandleFunc("POST /api/evidence/{id}/inspect", s.inspectEvidence)
	m.HandleFunc("GET /api/portfolio/wallet", s.walletPortfolio)
	m.HandleFunc("GET /api/portfolio/flows", s.walletFlows)
	m.HandleFunc("GET /api/entities/{id}/portfolio", s.entityPortfolio)
	m.HandleFunc("GET /api/chain/networks", s.chainNetworks)
	m.HandleFunc("GET /api/search", s.search)
	m.HandleFunc("GET /api/chain/address/{addr}", s.chainAddress)
	m.HandleFunc("GET /api/chain/address/{addr}/activity", s.chainActivity)
	m.HandleFunc("GET /api/chain/tx/{hash}", s.chainTx)
	m.HandleFunc("POST /api/entities/lookup", s.lookupEntities)
	m.HandleFunc("GET /api/graph-snapshots", s.listGraphSnapshots)
	m.HandleFunc("POST /api/graph-snapshots", s.saveGraphSnapshot)
	m.HandleFunc("GET /api/graph-snapshots/{id}", s.getGraphSnapshot)
	m.HandleFunc("DELETE /api/graph-snapshots/{id}", s.deleteGraphSnapshot)
	m.HandleFunc("GET /api/entities", s.listEntities)
	m.HandleFunc("POST /api/entities", s.createEntity)
	m.HandleFunc("GET /api/entities/{id}", s.getEntity)
	m.HandleFunc("PATCH /api/entities/{id}", s.patchEntity)
	m.HandleFunc("DELETE /api/entities/{id}", s.deleteEntity)
	m.HandleFunc("POST /api/entities/{id}/addresses", s.addEntityAddress)
	m.HandleFunc("PATCH /api/entity-addresses/{id}", s.patchEntityAddress)
	m.HandleFunc("DELETE /api/entity-addresses/{id}", s.deleteEntityAddress)
	m.HandleFunc("POST /api/entities/{id}/identifiers", s.addEntityIdentifier)
	m.HandleFunc("DELETE /api/entity-identifiers/{id}", s.deleteEntityIdentifier)
	m.HandleFunc("POST /api/entities/{id}/evidence", s.linkEntityEvidence)
	m.HandleFunc("DELETE /api/entity-evidence/{id}", s.unlinkEntityEvidence)
	m.HandleFunc("GET /api/labels", s.listLabels)
	m.HandleFunc("POST /api/cases/{id}/labels", s.addLabel)
	m.HandleFunc("POST /api/cases/{id}/report", s.exportReport)
	m.HandleFunc("POST /api/backup", s.createBackup)
	m.HandleFunc("GET /api/restore", s.restoreStatus)
	m.HandleFunc("POST /api/restore/stage", s.stageRestore)
	m.HandleFunc("DELETE /api/restore/stage", s.cancelRestore)
	m.HandleFunc("GET /api/audit", s.audit)
	m.HandleFunc("GET /api/audit/verify", s.verifyAudit)
	return s.middleware(m)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			if !allowedOrigins[o] {
				writeErr(w, 403, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path != "/api/health" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
				writeErr(w, 401, "unauthorized")
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, chain.ErrNotFound):
		writeErr(w, 404, "not found")
	case errors.Is(err, store.ErrInvalid), errors.Is(err, chain.ErrInvalid), errors.Is(err, inspect.ErrInvalid):
		writeErr(w, 400, err.Error())
	case errors.Is(err, chain.ErrUpstream), errors.Is(err, prices.ErrUpstream):
		log.Printf("upstream error: %v", err)
		writeErr(w, 502, err.Error())
	default:
		log.Printf("internal error: %v", err)
		writeErr(w, 500, "internal error")
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, 400, "invalid JSON body")
		return false
	}
	return true
}

type actorBody struct {
	Actor string `json:"actor"`
}

func (s *Server) listCases(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.ListCases(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) createCase(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.CreateCase(r.Context(), b.Title, b.Description, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) getCase(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.GetCase(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) patchCase(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Status string `json:"status"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.SetCaseStatus(r.Context(), r.PathValue("id"), b.Status, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.ListNotes(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Body string `json:"body"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddNote(r.Context(), r.PathValue("id"), b.Body, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) listEvidence(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.ListEvidence(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) importEvidence(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Path string `json:"path"` // chosen via the native file dialog
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.ImportEvidence(r.Context(), r.PathValue("id"), b.Path, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) listFindings(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.ListFindings(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addFinding(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Kind       string `json:"kind"`
		RefType    string `json:"refType"`
		RefID      string `json:"refId"`
		Title      string `json:"title"`
		Note       string `json:"note"`
		SourceJSON string `json:"sourceJson"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddFinding(r.Context(), store.Finding{CaseID: r.PathValue("id"), Kind: b.Kind,
		RefType: b.RefType, RefID: b.RefID, Title: b.Title, Note: b.Note, SourceJSON: b.SourceJSON}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) verifyEvidence(w http.ResponseWriter, r *http.Request) {
	var b actorBody
	_ = json.NewDecoder(r.Body).Decode(&b) // body optional
	v, err := s.st.VerifyEvidence(r.Context(), r.PathValue("id"), b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) custody(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.st.GetEvidence(r.Context(), id); err != nil {
		fail(w, err)
		return
	}
	ev, err := s.st.ListCustody(r.Context(), id)
	if err != nil {
		fail(w, err)
		return
	}
	chain, err := s.st.VerifyCustodyChain(r.Context(), id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"events": ev, "chain": chain})
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := s.st.ListAudit(r.Context(), limit)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) verifyAudit(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.VerifyAuditChain(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) noChain(w http.ResponseWriter) bool {
	if s.chain == nil {
		writeErr(w, 503, "blockchain data source not configured")
		return true
	}
	return false
}

func (s *Server) chainAddress(w http.ResponseWriter, r *http.Request) {
	p, ok := s.provider(w, r)
	if !ok {
		return
	}
	v, err := p.Address(r.Context(), r.PathValue("addr"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) chainActivity(w http.ResponseWriter, r *http.Request) {
	p, ok := s.provider(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	v, err := p.Activity(r.Context(), r.PathValue("addr"), q.Get("kind"), q.Get("cursor"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) chainTx(w http.ResponseWriter, r *http.Request) {
	p, ok := s.provider(w, r)
	if !ok {
		return
	}
	v, err := p.Transaction(r.Context(), r.PathValue("hash"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) listLabels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	v, err := s.st.ListLabels(r.Context(), q.Get("caseId"), q.Get("address"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addLabel(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Address    string `json:"address"`
		Name       string `json:"name"`
		Category   string `json:"category"`
		Source     string `json:"source"`
		Confidence string `json:"confidence"`
		Basis      string `json:"basis"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddLabel(r.Context(), store.Label{CaseID: r.PathValue("id"), Address: b.Address, Name: b.Name,
		Category: b.Category, Source: b.Source, Confidence: b.Confidence, Basis: b.Basis}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) exportReport(w http.ResponseWriter, r *http.Request) {
	var b actorBody
	_ = json.NewDecoder(r.Body).Decode(&b) // body optional
	v, err := s.st.ExportReport(r.Context(), r.PathValue("id"), b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) createBackup(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Path            string `json:"path"`
		IncludeEvidence *bool  `json:"includeEvidence"`
	}
	if !decode(w, r, &b) {
		return
	}
	include := true
	if b.IncludeEvidence != nil {
		include = *b.IncludeEvidence
	}
	v, err := s.st.Backup(r.Context(), b.Path, include, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) stageRestore(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Path string `json:"path"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.StageRestore(r.Context(), b.Path, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) cancelRestore(w http.ResponseWriter, r *http.Request) {
	var b actorBody
	_ = json.NewDecoder(r.Body).Decode(&b) // body optional
	if err := s.st.CancelStagedRestore(r.Context(), b.Actor); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreStatus(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.RestoreStatus()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}
