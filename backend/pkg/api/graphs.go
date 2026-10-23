package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) listGraphSnapshots(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.ListGraphSnapshots(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) saveGraphSnapshot(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Name      string          `json:"name"`
		Seeds     []string        `json:"seeds"`
		NodeCount int             `json:"nodeCount"`
		EdgeCount int             `json:"edgeCount"`
		Data      json.RawMessage `json:"data"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.SaveGraphSnapshot(r.Context(), b.Name, b.Seeds, b.NodeCount, b.EdgeCount, b.Data, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) getGraphSnapshot(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.GetGraphSnapshot(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) deleteGraphSnapshot(w http.ResponseWriter, r *http.Request) {
	if err := s.st.DeleteGraphSnapshot(r.Context(), r.PathValue("id"), optionalActor(r)); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, nil)
}
