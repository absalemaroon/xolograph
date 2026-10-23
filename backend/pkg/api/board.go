package api

import (
	"net/http"

	"github.com/xolograph/backend/pkg/store"
)

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.GetBoard(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addBoardNode(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Kind    string  `json:"kind"`
		RefID   string  `json:"refId"`
		Network string  `json:"network"`
		Title   string  `json:"title"`
		Body    string  `json:"body"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddBoardNode(r.Context(), r.PathValue("id"), store.BoardNodeInput{Kind: b.Kind, RefID: b.RefID, Network: b.Network, Title: b.Title, Body: b.Body, X: b.X, Y: b.Y}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) moveBoardNodes(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Positions []struct {
			ID string  `json:"id"`
			X  float64 `json:"x"`
			Y  float64 `json:"y"`
		} `json:"positions"`
	}
	if !decode(w, r, &b) {
		return
	}
	ids, xs, ys := make([]string, 0, len(b.Positions)), make([]float64, 0, len(b.Positions)), make([]float64, 0, len(b.Positions))
	for _, p := range b.Positions {
		ids, xs, ys = append(ids, p.ID), append(xs, p.X), append(ys, p.Y)
	}
	if err := s.st.MoveBoardNodes(r.Context(), r.PathValue("id"), ids, xs, ys); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, nil)
}

func (s *Server) patchBoardNode(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.UpdateBoardNode(r.Context(), r.PathValue("id"), b.Title, b.Body, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) deleteBoardNode(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.DeleteBoardNode(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addBoardLink(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		SourceID    string   `json:"sourceId"`
		TargetID    string   `json:"targetId"`
		Relation    string   `json:"relation"`
		Claim       string   `json:"claim"`
		Confidence  string   `json:"confidence"`
		SourceRef   string   `json:"sourceRef"`
		Note        string   `json:"note"`
		EvidenceIDs []string `json:"evidenceIds"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddBoardLink(r.Context(), r.PathValue("id"), store.BoardLinkInput{SourceID: b.SourceID, TargetID: b.TargetID, Relation: b.Relation,
		Claim: b.Claim, Confidence: b.Confidence, SourceRef: b.SourceRef, Note: b.Note, EvidenceIDs: b.EvidenceIDs}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) patchBoardLink(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Relation   string `json:"relation"`
		Claim      string `json:"claim"`
		Confidence string `json:"confidence"`
		SourceRef  string `json:"sourceRef"`
		Note       string `json:"note"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.UpdateBoardLink(r.Context(), r.PathValue("id"), b.Relation, b.Claim, b.Confidence, b.SourceRef, b.Note, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) deleteBoardLink(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.DeleteBoardLink(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addBoardLinkEvidence(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		EvidenceID  string `json:"evidenceId"`
		Description string `json:"description"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddBoardLinkEvidence(r.Context(), r.PathValue("id"), b.EvidenceID, b.Description, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) deleteBoardLinkEvidence(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.RemoveBoardLinkEvidence(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}
