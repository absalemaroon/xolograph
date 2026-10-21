// Package store is the Xolograph persistence engine: SQLite (pure Go, WAL),
// versioned migrations, hash-chained audit and custody logs, and a controlled
// evidence vault.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
)

const defaultActor = "local-analyst"

type Store struct {
	DB      *sql.DB
	dataDir string
}

type dbtx interface {
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}

// Open creates the data directory layout, opens the database and applies migrations.
func Open(dataDir string) (*Store, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("%w: data dir required", ErrInvalid)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	restore := applyPendingRestore(dataDir) // must run before the database is opened
	if err := os.MkdirAll(filepath.Join(dataDir, "evidence"), 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(filepath.Join(dataDir, "xolograph.db")) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Single connection: serialises writes so hash chains cannot interleave.
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, dataDir: dataDir}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if restore != nil && restore.Applied {
		_ = s.inTx(context.Background(), func(tx *sql.Tx) error {
			return s.appendAudit(context.Background(), tx, "system", "RESTORE_APPLIED", "system", "restore",
				"archive_sha256="+restore.ArchiveSHA256+" previous_data="+restore.PreviousData)
		})
	}
	return s, nil
}

func (s *Store) Close() error   { return s.DB.Close() }
func (s *Store) DataDir() string { return s.dataDir }

func (s *Store) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func newID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func now() time.Time { return time.Now().UTC() }
func ts(t time.Time) string { return t.Format(time.RFC3339Nano) }
func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func chainHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func actorOrDefault(a string) string {
	if a == "" {
		return defaultActor
	}
	return a
}

// ---- migrations -----------------------------------------------------------

var migrations = []string{
	// v1: core schema
	`CREATE TABLE cases (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','CLOSED','ARCHIVED')),
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE notes (
		id TEXT PRIMARY KEY,
		case_id TEXT NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
		body TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_notes_case ON notes(case_id, created_at);
	CREATE TABLE evidence (
		id TEXT PRIMARY KEY,
		case_id TEXT NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
		name TEXT NOT NULL,
		stored_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		sha256 TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'IMPORTED' CHECK (status IN ('IMPORTED','VERIFIED','FAILED')),
		imported_at TEXT NOT NULL,
		last_verified_at TEXT
	);
	CREATE INDEX idx_evidence_case ON evidence(case_id);
	CREATE TABLE custody_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		evidence_id TEXT NOT NULL REFERENCES evidence(id) ON DELETE RESTRICT,
		case_id TEXT NOT NULL,
		ts TEXT NOT NULL,
		actor TEXT NOT NULL,
		action TEXT NOT NULL,
		detail TEXT NOT NULL DEFAULT '',
		sha256 TEXT NOT NULL DEFAULT '',
		prev_hash TEXT NOT NULL,
		event_hash TEXT NOT NULL
	);
	CREATE INDEX idx_custody_evidence ON custody_events(evidence_id, id);
	CREATE TABLE audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts TEXT NOT NULL,
		actor TEXT NOT NULL,
		action TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		detail TEXT NOT NULL DEFAULT '',
		prev_hash TEXT NOT NULL,
		entry_hash TEXT NOT NULL
	);
	CREATE TABLE findings (
		id TEXT PRIMARY KEY,
		case_id TEXT NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
		kind TEXT NOT NULL,
		ref_type TEXT NOT NULL DEFAULT '',
		ref_id TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL,
		note TEXT NOT NULL DEFAULT '',
		source_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_findings_case ON findings(case_id, created_at);
	CREATE TABLE preferences (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
	// v2: analyst entity labels (always with a stated basis and source)
	`CREATE TABLE entity_labels (
		id TEXT PRIMARY KEY,
		case_id TEXT NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
		address TEXT NOT NULL,
		network TEXT NOT NULL DEFAULT 'ethereum-mainnet',
		name TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'UNCATEGORIZED',
		source TEXT NOT NULL CHECK (source IN ('ANALYST_ASSERTION','THIRD_PARTY_PROVIDER','VERIFIED_SOURCE')),
		confidence TEXT NOT NULL CHECK (confidence IN ('LOW','MEDIUM','HIGH')),
		basis TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_labels_address ON entity_labels(address);
	CREATE INDEX idx_labels_case ON entity_labels(case_id);`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	var cur int
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&cur); err != nil {
		return err
	}
	for i := cur; i < len(migrations); i++ {
		v := i + 1
		err := s.inTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?,?)`, v, ts(now()))
			return err
		})
		if err != nil {
			return fmt.Errorf("migration v%d: %w", v, err)
		}
	}
	return nil
}
