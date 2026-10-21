package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Case struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	EvidenceCount int       `json:"evidenceCount"`
	FindingCount  int       `json:"findingCount"`
}

type Note struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"caseId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type Finding struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"caseId"`
	Kind       string    `json:"kind"`    // ADDRESS, TRANSACTION, ENTITY, PATH, OBSERVATION
	RefType    string    `json:"refType"` // source record type
	RefID      string    `json:"refId"`   // source record id (tx hash, address, ...)
	Title      string    `json:"title"`
	Note       string    `json:"note"`
	SourceJSON string    `json:"sourceJson"` // provenance snapshot of the underlying record
	CreatedAt  time.Time `json:"createdAt"`
}

const caseSelect = `SELECT c.id,c.title,c.description,c.status,c.created_at,c.updated_at,
	(SELECT COUNT(*) FROM evidence e WHERE e.case_id=c.id),
	(SELECT COUNT(*) FROM findings f WHERE f.case_id=c.id) FROM cases c`

func scanCase(sc interface{ Scan(...any) error }) (*Case, error) {
	var c Case
	var ca, ua string
	if err := sc.Scan(&c.ID, &c.Title, &c.Description, &c.Status, &ca, &ua, &c.EvidenceCount, &c.FindingCount); err != nil {
		return nil, err
	}
	c.CreatedAt, c.UpdatedAt = parseTS(ca), parseTS(ua)
	return &c, nil
}

func (s *Store) CreateCase(ctx context.Context, title, description, actor string) (*Case, error) {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 200 {
		return nil, fmt.Errorf("%w: title must be 1-200 characters", ErrInvalid)
	}
	t := now()
	id := newID("case")
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO cases(id,title,description,status,created_at,updated_at)
			VALUES(?,?,?,'OPEN',?,?)`, id, title, description, ts(t), ts(t)); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "CASE_CREATED", "case", id, title)
	})
	if err != nil {
		return nil, err
	}
	return s.GetCase(ctx, id)
}

func (s *Store) ListCases(ctx context.Context) ([]Case, error) {
	rows, err := s.DB.QueryContext(ctx, caseSelect+` ORDER BY c.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Case{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *Store) GetCase(ctx context.Context, id string) (*Case, error) {
	c, err := scanCase(s.DB.QueryRowContext(ctx, caseSelect+` WHERE c.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *Store) SetCaseStatus(ctx context.Context, id, status, actor string) (*Case, error) {
	status = strings.ToUpper(status)
	if status != "OPEN" && status != "CLOSED" && status != "ARCHIVED" {
		return nil, fmt.Errorf("%w: status must be OPEN, CLOSED or ARCHIVED", ErrInvalid)
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE cases SET status=?, updated_at=? WHERE id=?`, status, ts(now()), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return s.appendAudit(ctx, tx, actor, "CASE_STATUS_CHANGED", "case", id, status)
	})
	if err != nil {
		return nil, err
	}
	return s.GetCase(ctx, id)
}

func (s *Store) touchCase(ctx context.Context, tx dbtx, id string) error {
	_, err := tx.ExecContext(ctx, `UPDATE cases SET updated_at=? WHERE id=?`, ts(now()), id)
	return err
}

func (s *Store) AddNote(ctx context.Context, caseID, body, actor string) (*Note, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: note body required", ErrInvalid)
	}
	if _, err := s.GetCase(ctx, caseID); err != nil {
		return nil, err
	}
	n := &Note{ID: newID("note"), CaseID: caseID, Body: body, CreatedAt: now()}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notes(id,case_id,body,created_at) VALUES(?,?,?,?)`,
			n.ID, caseID, body, ts(n.CreatedAt)); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "NOTE_ADDED", "case", caseID, n.ID)
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Store) ListNotes(ctx context.Context, caseID string) ([]Note, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,case_id,body,created_at FROM notes WHERE case_id=? ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		var t string
		if err := rows.Scan(&n.ID, &n.CaseID, &n.Body, &t); err != nil {
			return nil, err
		}
		n.CreatedAt = parseTS(t)
		out = append(out, n)
	}
	return out, rows.Err()
}

// AddFinding attaches an analysis finding (with a provenance snapshot) to a case.
func (s *Store) AddFinding(ctx context.Context, f Finding, actor string) (*Finding, error) {
	f.Title = strings.TrimSpace(f.Title)
	f.Kind = strings.ToUpper(strings.TrimSpace(f.Kind))
	if f.Title == "" || f.Kind == "" {
		return nil, fmt.Errorf("%w: kind and title required", ErrInvalid)
	}
	if f.SourceJSON == "" {
		f.SourceJSON = "{}"
	}
	if _, err := s.GetCase(ctx, f.CaseID); err != nil {
		return nil, err
	}
	f.ID, f.CreatedAt = newID("find"), now()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO findings(id,case_id,kind,ref_type,ref_id,title,note,source_json,created_at)
			VALUES(?,?,?,?,?,?,?,?,?)`, f.ID, f.CaseID, f.Kind, f.RefType, f.RefID, f.Title, f.Note, f.SourceJSON, ts(f.CreatedAt)); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, f.CaseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "FINDING_ADDED", "case", f.CaseID, f.ID+" "+f.Kind+" "+f.RefID)
	})
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) ListFindings(ctx context.Context, caseID string) ([]Finding, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,case_id,kind,ref_type,ref_id,title,note,source_json,created_at
		FROM findings WHERE case_id=? ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Finding{}
	for rows.Next() {
		var f Finding
		var t string
		if err := rows.Scan(&f.ID, &f.CaseID, &f.Kind, &f.RefType, &f.RefID, &f.Title, &f.Note, &f.SourceJSON, &t); err != nil {
			return nil, err
		}
		f.CreatedAt = parseTS(t)
		out = append(out, f)
	}
	return out, rows.Err()
}
