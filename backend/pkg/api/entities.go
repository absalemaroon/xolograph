package api

import (
	"encoding/json"
	"net/http"

	"github.com/xolograph/backend/pkg/store"
)

type addressBody struct {
	Network    string `json:"network"`
	Address    string `json:"address"`
	Label      string `json:"label"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Confidence string `json:"confidence"`
	SourceType string `json:"sourceType"`
	SourceRef  string `json:"sourceRef"`
	Basis      string `json:"basis"`
	Notes      string `json:"notes"`
}

func (a addressBody) input() store.AddressInput {
	return store.AddressInput{Network: a.Network, Address: a.Address, Label: a.Label, Role: a.Role, Status: a.Status,
		Confidence: a.Confidence, SourceType: a.SourceType, SourceRef: a.SourceRef, Basis: a.Basis, Notes: a.Notes}
}

type entityBody struct {
	actorBody
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Country     string        `json:"country"`
	Tags        []string      `json:"tags"`
	Status      string        `json:"status"`
	Confidence  string        `json:"confidence"`
	Notes       string        `json:"notes"`
	Addresses   []addressBody `json:"addresses"`
}

func (b entityBody) input() store.EntityInput {
	in := store.EntityInput{Name: b.Name, Type: b.Type, Description: b.Description, Country: b.Country, Tags: b.Tags,
		Status: b.Status, Confidence: b.Confidence, Notes: b.Notes}
	for _, a := range b.Addresses {
		in.Addresses = append(in.Addresses, a.input())
	}
	return in
}

// optionalActor reads an optional JSON body (used by DELETE requests) for the actor name.
func optionalActor(r *http.Request) string {
	var b actorBody
	_ = json.NewDecoder(r.Body).Decode(&b)
	return b.Actor
}

func (s *Server) listEntities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if addr := q.Get("address"); addr != "" {
		v, err := s.st.EntitiesForAddress(r.Context(), q.Get("network"), addr)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, 200, v)
		return
	}
	v, err := s.st.ListEntities(r.Context(), q.Get("q"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) createEntity(w http.ResponseWriter, r *http.Request) {
	var b entityBody
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.CreateEntity(r.Context(), b.input(), b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) getEntity(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.GetEntity(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) patchEntity(w http.ResponseWriter, r *http.Request) {
	var b entityBody
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.UpdateEntity(r.Context(), r.PathValue("id"), b.input(), b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) deleteEntity(w http.ResponseWriter, r *http.Request) {
	if err := s.st.DeleteEntity(r.Context(), r.PathValue("id"), optionalActor(r)); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, nil)
}

func (s *Server) addEntityAddress(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		addressBody
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddEntityAddress(r.Context(), r.PathValue("id"), b.addressBody.input(), b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) patchEntityAddress(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Label      string `json:"label"`
		Role       string `json:"role"`
		Status     string `json:"status"`
		Confidence string `json:"confidence"`
		SourceType string `json:"sourceType"`
		SourceRef  string `json:"sourceRef"`
		Basis      string `json:"basis"`
		Notes      string `json:"notes"`
		ChangeNote string `json:"changeNote"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.UpdateEntityAddress(r.Context(), r.PathValue("id"), store.AddressUpdate{Label: b.Label, Role: b.Role, Status: b.Status,
		Confidence: b.Confidence, SourceType: b.SourceType, SourceRef: b.SourceRef, Basis: b.Basis, Notes: b.Notes, ChangeNote: b.ChangeNote}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) deleteEntityAddress(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.RemoveEntityAddress(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) addEntityIdentifier(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		Platform   string `json:"platform"`
		Value      string `json:"value"`
		Status     string `json:"status"`
		SourceRef  string `json:"sourceRef"`
		Confidence string `json:"confidence"`
		Notes      string `json:"notes"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.AddEntityIdentifier(r.Context(), r.PathValue("id"), store.IdentifierInput{Platform: b.Platform, Value: b.Value,
		Status: b.Status, SourceRef: b.SourceRef, Confidence: b.Confidence, Notes: b.Notes}, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) deleteEntityIdentifier(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.RemoveEntityIdentifier(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) linkEntityEvidence(w http.ResponseWriter, r *http.Request) {
	var b struct {
		actorBody
		EvidenceID  string `json:"evidenceId"`
		AddressID   string `json:"addressId"`
		Description string `json:"description"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.LinkEntityEvidence(r.Context(), r.PathValue("id"), b.EvidenceID, b.AddressID, b.Description, b.Actor)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) unlinkEntityEvidence(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.UnlinkEntityEvidence(r.Context(), r.PathValue("id"), optionalActor(r))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) lookupEntities(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Network   string   `json:"network"`
		Addresses []string `json:"addresses"`
	}
	if !decode(w, r, &b) {
		return
	}
	v, err := s.st.LookupAddresses(r.Context(), b.Network, b.Addresses)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	v, err := s.st.SearchLocal(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

// decodeOptional reads an optional JSON body, ignoring an empty one.
func decodeOptional(r *http.Request, v any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(v)
}
