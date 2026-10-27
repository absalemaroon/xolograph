package api

import (
	"net/http"

	"github.com/xolograph/backend/pkg/inspect"
)

// inspectFile reads a local file read-only and reports its type, hashes and metadata. Nothing is stored or audited:
// it is an analysis aid; attaching a result to a case is a separate, explicit finding.
func (s *Server) inspectFile(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Path         string `json:"path"`
		ExpectedHash string `json:"expectedHash"`
	}
	if !decode(w, r, &b) {
		return
	}
	rep, err := inspect.File(r.Context(), b.Path, b.ExpectedHash)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rep)
}

// inspectEvidence inspects a vault item and compares it with the digest recorded at import (unless another is given).
// It does not alter the item's verification status or its chain of custody.
func (s *Server) inspectEvidence(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ExpectedHash string `json:"expectedHash"`
	}
	_ = decodeOptional(r, &b)
	path, recorded, err := s.st.EvidenceFile(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	expected := b.ExpectedHash
	if expected == "" {
		expected = recorded
	}
	rep, err := inspect.File(r.Context(), path, expected)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rep)
}
