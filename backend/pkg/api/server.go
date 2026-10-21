// Package api exposes the store over a token-protected loopback HTTP API.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/xolograph/backend/pkg/store"
)

type Server struct {
	st    *store.Store
	token string
}

func New(st *store.Store, token string) *Server { return &Server{st: st, token: token} }

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
	m.HandleFunc("GET /api/cases/{id}/notes", s.listNotes)
	m.HandleFunc("POST /api/cases/{id}/notes", s.addNote)
	m.HandleFunc("GET /api/cases/{id}/evidence", s.listEvidence)
	m.HandleFunc("POST /api/cases/{id}/evidence", s.importEvidence)
	m.HandleFunc("GET /api/cases/{id}/findings", s.listFindings)
	m.HandleFunc("POST /api/cases/{id}/findings", s.addFinding)
	m.HandleFunc("POST /api/evidence/{id}/verify", s.verifyEvidence)
	m.HandleFunc("GET /api/evidence/{id}/custody", s.custody)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
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
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, 404, "not found")
	case errors.Is(err, store.ErrInvalid):
		writeErr(w, 400, err.Error())
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
