package store

import (
	"context"
	"strings"
)

// SearchHit is one match in the user's own records.
type SearchHit struct {
	Kind     string `json:"kind"` // ENTITY | CASE | EVIDENCE | LABEL
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	CaseID   string `json:"caseId,omitempty"`
}

const searchLimit = 8

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// SearchLocal looks through entities, cases, evidence and case labels. It never queries a blockchain.
func (s *Store) SearchLocal(ctx context.Context, q string) ([]SearchHit, error) {
	q = strings.TrimSpace(q)
	hits := []SearchHit{}
	if len(q) < 2 || len(q) > 200 {
		return hits, nil
	}
	like := "%" + escapeLike(strings.ToLower(q)) + "%"

	ents, err := s.ListEntities(ctx, q)
	if err != nil {
		return nil, err
	}
	for i, e := range ents {
		if i == searchLimit {
			break
		}
		hits = append(hits, SearchHit{Kind: "ENTITY", ID: e.ID, Title: e.Name, Subtitle: e.Type + " · " + e.Status})
	}

	type q4 struct {
		kind, sql string
		args      []any
	}
	for _, x := range []q4{
		{"CASE", `SELECT id,title,status,'' FROM cases WHERE lower(title) LIKE ? ESCAPE '\' ORDER BY updated_at DESC LIMIT ?`, []any{like, searchLimit}},
		{"EVIDENCE", `SELECT v.id,v.name,c.title,v.case_id FROM evidence v JOIN cases c ON c.id=v.case_id WHERE lower(v.name) LIKE ? ESCAPE '\' OR lower(v.sha256) LIKE ? ESCAPE '\' ORDER BY v.imported_at DESC LIMIT ?`, []any{like, like, searchLimit}},
		{"LABEL", `SELECT l.id,l.name,l.address,l.case_id FROM entity_labels l WHERE lower(l.address) LIKE ? ESCAPE '\' OR lower(l.name) LIKE ? ESCAPE '\' ORDER BY l.created_at DESC LIMIT ?`, []any{like, like, searchLimit}},
	} {
		rows, err := s.DB.QueryContext(ctx, x.sql, x.args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			h := SearchHit{Kind: x.kind}
			if err := rows.Scan(&h.ID, &h.Title, &h.Subtitle, &h.CaseID); err != nil {
				rows.Close()
				return nil, err
			}
			if x.kind == "CASE" {
				h.CaseID = h.ID
			}
			hits = append(hits, h)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return hits, nil
}
