package store

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const evContent = "hello-evidence-content"

func seedBackup(t *testing.T) (zipPath, evID string) {
	t.Helper()
	ctx := context.Background()
	a := newStore(t)
	c, _ := a.CreateCase(ctx, "Case A", "", "")
	src := filepath.Join(t.TempDir(), "e.txt")
	if err := os.WriteFile(src, []byte(evContent), 0o600); err != nil {
		t.Fatal(err)
	}
	ev, err := a.ImportEvidence(ctx, c.ID, src, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = a.AddNote(ctx, c.ID, "n", "")
	zipPath = filepath.Join(t.TempDir(), "backup.zip")
	res, err := a.Backup(ctx, zipPath, true, "tester")
	if err != nil || res.SHA256 == "" || res.Files < 3 || !res.IncludesEvidence {
		t.Fatalf("backup: %+v %v", res, err)
	}
	return zipPath, ev.ID
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	zipPath, evID := seedBackup(t)

	dir2 := t.TempDir()
	b, err := Open(dir2)
	if err != nil {
		t.Fatal(err)
	}
	st, err := b.StageRestore(ctx, zipPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if st.Cases != 1 || !st.AuditChainValid || !st.IncludesEvidence {
		t.Fatalf("stage result: %+v", st)
	}
	if status, _ := b.RestoreStatus(); status.Pending == nil {
		t.Fatal("staged restore must be reported as pending")
	}
	_ = b.Close()

	r, err := Open(dir2) // applies the staged restore
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if cases, _ := r.ListCases(ctx); len(cases) != 1 {
		t.Fatalf("want 1 restored case, got %d", len(cases))
	}
	ev, err := r.VerifyEvidence(ctx, evID, "")
	if err != nil || ev.Status != "VERIFIED" {
		t.Fatalf("restored evidence must verify: %+v %v", ev, err)
	}
	if chain, _ := r.VerifyAuditChain(ctx); !chain.Valid {
		t.Fatal("audit chain must be valid after restore")
	}
	entries, _ := r.ListAudit(ctx, 20) // newest first; VerifyEvidence above adds its own entry
	audited := false
	for _, e := range entries {
		if e.Action == "RESTORE_APPLIED" {
			audited = true
			break
		}
	}
	if !audited {
		t.Fatalf("restore must be audited: %+v", entries)
	}
	if _, err := os.Stat(filepath.Join(dir2, "pre-restore")); err != nil {
		t.Fatal("previous data must be kept under pre-restore/")
	}
	status, _ := r.RestoreStatus()
	if status.Pending != nil || status.Last == nil || !status.Last.Applied {
		t.Fatalf("status after restore: %+v", status)
	}
}

func TestRestoreRejectsTamperedArchive(t *testing.T) {
	zipPath, _ := seedBackup(t)
	data, _ := os.ReadFile(zipPath)
	i := bytes.Index(data, []byte(evContent))
	if i < 0 {
		t.Fatal("evidence bytes should be stored uncompressed in the archive")
	}
	data[i] ^= 0xFF
	bad := filepath.Join(t.TempDir(), "tampered.zip")
	_ = os.WriteFile(bad, data, 0o600)

	b := newStore(t)
	if _, err := b.StageRestore(context.Background(), bad, ""); err == nil {
		t.Fatal("tampered archive must be rejected")
	}
	if _, err := os.Stat(filepath.Join(b.DataDir(), stagingDirName)); !os.IsNotExist(err) {
		t.Fatal("failed staging must be cleaned up")
	}
}

func TestRestoreRejectsUnsafeArchive(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create(manifestName)
	_, _ = w.Write([]byte(`{"format":"xolograph-backup-v1","files":[{"path":"data/../../evil","size":1,"sha256":"` + string(bytes.Repeat([]byte("a"), 64)) + `"}]}`))
	w2, _ := zw.Create("data/../../evil")
	_, _ = w2.Write([]byte("x"))
	_ = zw.Close()
	p := filepath.Join(t.TempDir(), "evil.zip")
	_ = os.WriteFile(p, buf.Bytes(), 0o600)

	b := newStore(t)
	if _, err := b.StageRestore(context.Background(), p, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("path traversal must be rejected as invalid, got %v", err)
	}
}

func TestBackupAndRestoreRejectPathsInsideDataDir(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	if _, err := s.Backup(ctx, filepath.Join(s.DataDir(), "b.zip"), true, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("backup inside data dir must be rejected, got %v", err)
	}
	if _, err := s.StageRestore(ctx, filepath.Join(s.DataDir(), "b.zip"), ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("restore from inside data dir must be rejected, got %v", err)
	}
}
