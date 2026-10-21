package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type AuditEntry struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityId"`
	Detail     string    `json:"detail"`
	EntryHash  string    `json:"entryHash"`
}

type CustodyEvent struct {
	ID         int64     `json:"id"`
	EvidenceID string    `json:"evidenceId"`
	CaseID     string    `json:"caseId"`
	Timestamp  time.Time `json:"timestamp"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	Detail     string    `json:"detail"`
	SHA256     string    `json:"sha256"`
	EventHash  string    `json:"eventHash"`
}

// ChainResult reports tamper-evidence verification of a hash chain.
type ChainResult struct {
	Valid    bool  `json:"valid"`
	Count    int   `json:"count"`
	BrokenAt int64 `json:"brokenAt,omitempty"` // id of first inconsistent record
}

const genesis = "GENESIS"

func (s *Store) appendAudit(ctx context.Context, tx dbtx, actor, action, etype, eid, detail string) error {
	prev := genesis
	var last string
	err := tx.QueryRowContext(ctx, `SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&last)
	switch {
	case err == nil:
		prev = last
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	actor = actorOrDefault(actor)
	t := ts(now())
	h := chainHash(prev, t, actor, action, etype, eid, detail)
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(ts,actor,action,entity_type,entity_id,detail,prev_hash,entry_hash)
		VALUES(?,?,?,?,?,?,?,?)`, t, actor, action, etype, eid, detail, prev, h)
	return err
}

func (s *Store) appendCustody(ctx context.Context, tx dbtx, evID, caseID, actor, action, detail, sha string) error {
	prev := genesis
	var last string
	err := tx.QueryRowContext(ctx, `SELECT event_hash FROM custody_events WHERE evidence_id=? ORDER BY id DESC LIMIT 1`, evID).Scan(&last)
	switch {
	case err == nil:
		prev = last
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	actor = actorOrDefault(actor)
	t := ts(now())
	h := chainHash(prev, evID, t, actor, action, detail, sha)
	_, err = tx.ExecContext(ctx, `INSERT INTO custody_events(evidence_id,case_id,ts,actor,action,detail,sha256,prev_hash,event_hash)
		VALUES(?,?,?,?,?,?,?,?,?)`, evID, caseID, t, actor, action, detail, sha, prev, h)
	return err
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,ts,actor,action,entity_type,entity_id,detail,entry_hash
		FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var t string
		if err := rows.Scan(&e.ID, &t, &e.Actor, &e.Action, &e.EntityType, &e.EntityID, &e.Detail, &e.EntryHash); err != nil {
			return nil, err
		}
		e.Timestamp = parseTS(t)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) VerifyAuditChain(ctx context.Context) (ChainResult, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,ts,actor,action,entity_type,entity_id,detail,prev_hash,entry_hash
		FROM audit_log ORDER BY id ASC`)
	if err != nil {
		return ChainResult{}, err
	}
	defer rows.Close()
	res := ChainResult{Valid: true}
	prev := genesis
	for rows.Next() {
		var id int64
		var t, actor, action, et, eid, detail, ph, eh string
		if err := rows.Scan(&id, &t, &actor, &action, &et, &eid, &detail, &ph, &eh); err != nil {
			return ChainResult{}, err
		}
		res.Count++
		if ph != prev || chainHash(prev, t, actor, action, et, eid, detail) != eh {
			return ChainResult{Valid: false, Count: res.Count, BrokenAt: id}, nil
		}
		prev = eh
	}
	return res, rows.Err()
}

func (s *Store) ListCustody(ctx context.Context, evidenceID string) ([]CustodyEvent, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,evidence_id,case_id,ts,actor,action,detail,sha256,event_hash
		FROM custody_events WHERE evidence_id=? ORDER BY id ASC`, evidenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CustodyEvent{}
	for rows.Next() {
		var e CustodyEvent
		var t string
		if err := rows.Scan(&e.ID, &e.EvidenceID, &e.CaseID, &t, &e.Actor, &e.Action, &e.Detail, &e.SHA256, &e.EventHash); err != nil {
			return nil, err
		}
		e.Timestamp = parseTS(t)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) VerifyCustodyChain(ctx context.Context, evidenceID string) (ChainResult, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,ts,actor,action,detail,sha256,prev_hash,event_hash
		FROM custody_events WHERE evidence_id=? ORDER BY id ASC`, evidenceID)
	if err != nil {
		return ChainResult{}, err
	}
	defer rows.Close()
	res := ChainResult{Valid: true}
	prev := genesis
	for rows.Next() {
		var id int64
		var t, actor, action, detail, sha, ph, eh string
		if err := rows.Scan(&id, &t, &actor, &action, &detail, &sha, &ph, &eh); err != nil {
			return ChainResult{}, err
		}
		res.Count++
		if ph != prev || chainHash(prev, evidenceID, t, actor, action, detail, sha) != eh {
			return ChainResult{Valid: false, Count: res.Count, BrokenAt: id}, nil
		}
		prev = eh
	}
	return res, rows.Err()
}
