package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Evidence struct {
	ID             string     `json:"id"`
	CaseID         string     `json:"caseId"`
	Name           string     `json:"name"`
	StoredPath     string     `json:"-"` // never exposed to the frontend
	SizeBytes      int64      `json:"sizeBytes"`
	SHA256         string     `json:"sha256"`
	Status         string     `json:"status"` // IMPORTED, VERIFIED, FAILED
	ImportedAt     time.Time  `json:"importedAt"`
	LastVerifiedAt *time.Time `json:"lastVerifiedAt,omitempty"`
}

const evSelect = `SELECT id,case_id,name,stored_path,size_bytes,sha256,status,imported_at,last_verified_at FROM evidence`

func scanEvidence(sc interface{ Scan(...any) error }) (*Evidence, error) {
	var e Evidence
	var ia string
	var lv sql.NullString
	if err := sc.Scan(&e.ID, &e.CaseID, &e.Name, &e.StoredPath, &e.SizeBytes, &e.SHA256, &e.Status, &ia, &lv); err != nil {
		return nil, err
	}
	e.ImportedAt = parseTS(ia)
	if lv.Valid {
		t := parseTS(lv.String)
		e.LastVerifiedAt = &t
	}
	return &e, nil
}

func sanitizeName(p string) string {
	n := filepath.Base(p)
	var b strings.Builder
	for _, r := range n {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), ". ")
	if out == "" {
		return "evidence.bin"
	}
	return out
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// ImportEvidence copies a local file into the controlled vault with streaming
// SHA-256 hashing, re-hashes the stored copy, then records custody + audit.
func (s *Store) ImportEvidence(ctx context.Context, caseID, srcPath, actor string) (*Evidence, error) {
	if _, err := s.GetCase(ctx, caseID); err != nil {
		return nil, err
	}
	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read source: %v", ErrInvalid, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: source is not a regular file", ErrInvalid)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot open source: %v", ErrInvalid, err)
	}
	defer src.Close()

	id := newID("ev")
	dir := filepath.Join(s.dataDir, "evidence", caseID, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	name := sanitizeName(srcPath)
	dstPath := filepath.Join(dir, name)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(dst, h), src)
	if err == nil {
		err = dst.Sync()
	}
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if reSum, _, herr := hashFile(dstPath); herr != nil || reSum != sum {
		_ = os.RemoveAll(dir)
		return nil, errors.New("stored copy failed integrity check; import aborted")
	}
	_ = os.Chmod(dstPath, 0o400)
	relPath := filepath.ToSlash(filepath.Join("evidence", caseID, id, name)) // relative: survives backup/restore

	ev := &Evidence{ID: id, CaseID: caseID, Name: name, StoredPath: dstPath, SizeBytes: size,
		SHA256: sum, Status: "IMPORTED", ImportedAt: now()}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO evidence(id,case_id,name,stored_path,size_bytes,sha256,status,imported_at)
			VALUES(?,?,?,?,?,?,'IMPORTED',?)`, ev.ID, caseID, name, relPath, size, sum, ts(ev.ImportedAt)); err != nil {
			return err
		}
		detail := fmt.Sprintf("source=%s size=%d", srcPath, size)
		if err := s.appendCustody(ctx, tx, ev.ID, caseID, actor, "IMPORTED", detail, sum); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "EVIDENCE_IMPORTED", "evidence", ev.ID, name+" sha256="+sum)
	})
	if err != nil {
		_ = os.Chmod(dstPath, 0o600)
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return ev, nil
}

func (s *Store) ListEvidence(ctx context.Context, caseID string) ([]Evidence, error) {
	rows, err := s.DB.QueryContext(ctx, evSelect+` WHERE case_id=? ORDER BY imported_at DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Evidence{}
	for rows.Next() {
		e, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (s *Store) GetEvidence(ctx context.Context, id string) (*Evidence, error) {
	e, err := scanEvidence(s.DB.QueryRowContext(ctx, evSelect+` WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

// VerifyEvidence re-hashes the stored file, compares to the recorded digest,
// and logs the outcome in the custody chain and audit log.
func (s *Store) VerifyEvidence(ctx context.Context, id, actor string) (*Evidence, error) {
	ev, err := s.GetEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	status, detail, got := "VERIFIED", "digest matches recorded SHA-256", ""
	sum, _, herr := hashFile(s.evidencePath(ev.StoredPath))
	switch {
	case herr != nil:
		status, detail = "FAILED", "stored file unreadable: "+herr.Error()
	case sum != ev.SHA256:
		status, got = "FAILED", sum
		detail = "digest mismatch: expected " + ev.SHA256 + " got " + sum
	default:
		got = sum
	}
	action := "VERIFIED"
	if status == "FAILED" {
		action = "INTEGRITY_FAILURE"
	}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE evidence SET status=?, last_verified_at=? WHERE id=?`, status, ts(now()), id); err != nil {
			return err
		}
		if err := s.appendCustody(ctx, tx, id, ev.CaseID, actor, action, detail, got); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "EVIDENCE_"+action, "evidence", id, detail)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEvidence(ctx, id)
}

// evidencePath resolves a stored path. New records are relative to the data dir;
// absolute paths from older versions are re-based onto the current data dir if needed.
func (s *Store) evidencePath(stored string) string {
	slash := filepath.ToSlash(stored)
	if !filepath.IsAbs(stored) && !strings.HasPrefix(slash, "/") {
		rel := filepath.Clean(filepath.FromSlash(slash))
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "" // never read outside the data directory
		}
		return filepath.Join(s.dataDir, rel)
	}
	if _, err := os.Stat(stored); err == nil {
		return stored
	}
	if i := strings.LastIndex(slash, "/evidence/"); i >= 0 {
		return filepath.Join(s.dataDir, filepath.FromSlash(slash[i+1:]))
	}
	return stored
}

// EvidenceFile returns the on-disk path and recorded SHA-256 of a vault item so it can be inspected read-only.
// It does not change the item's status or custody record.
func (s *Store) EvidenceFile(ctx context.Context, id string) (path, recordedSHA256 string, err error) {
	ev, err := s.GetEvidence(ctx, id)
	if err != nil {
		return "", "", err
	}
	p := s.evidencePath(ev.StoredPath)
	if p == "" {
		return "", "", fmt.Errorf("%w: stored path is outside the data directory", ErrInvalid)
	}
	return p, ev.SHA256, nil
}
