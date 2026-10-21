package store

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	backupFormat   = "xolograph-backup-v1"
	manifestName   = "manifest.json"
	stagingDirName = "restore-pending"
	stageMarker    = "stage.json"
	resultFile     = "restore-result.json"
	maxManifest    = 16 << 20
)

type manifestFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type backupManifest struct {
	Format           string         `json:"format"`
	CreatedAt        time.Time      `json:"createdAt"`
	IncludesEvidence bool           `json:"includesEvidence"`
	Files            []manifestFile `json:"files"`
}

type BackupResult struct {
	Path             string    `json:"path"`
	SHA256           string    `json:"sha256"` // digest of the archive file itself
	SizeBytes        int64     `json:"sizeBytes"`
	Files            int       `json:"files"`
	IncludesEvidence bool      `json:"includesEvidence"`
	CreatedAt        time.Time `json:"createdAt"`
}

type StageResult struct {
	ArchiveSHA256    string    `json:"archiveSha256"`
	ArchiveCreatedAt time.Time `json:"archiveCreatedAt"`
	StagedAt         time.Time `json:"stagedAt"`
	Files            int       `json:"files"`
	Bytes            int64     `json:"bytes"`
	IncludesEvidence bool      `json:"includesEvidence"`
	Cases            int       `json:"cases"`
	AuditEntries     int       `json:"auditEntries"`
	AuditChainValid  bool      `json:"auditChainValid"`
	AuditBrokenAt    int64     `json:"auditBrokenAt,omitempty"`
}

type RestoreResult struct {
	Applied          bool      `json:"applied"`
	AppliedAt        time.Time `json:"appliedAt"`
	ArchiveSHA256    string    `json:"archiveSha256"`
	ArchiveCreatedAt time.Time `json:"archiveCreatedAt"`
	PreviousData     string    `json:"previousData,omitempty"`
	Error            string    `json:"error,omitempty"`
}

type RestoreStatus struct {
	Pending *StageResult   `json:"pending"`
	Last    *RestoreResult `json:"last"`
}

func (s *Store) insideDataDir(p string) bool {
	d, err := filepath.Abs(s.dataDir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(d, p)
	if err != nil {
		return false
	}
	return !(rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Backup writes a verifiable archive: a consistent database snapshot (VACUUM INTO),
// the evidence vault (optional) and a manifest with a SHA-256 for every file.
func (s *Store) Backup(ctx context.Context, dest string, includeEvidence bool, actor string) (*BackupResult, error) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return nil, fmt.Errorf("%w: destination path required", ErrInvalid)
	}
	dest, err := filepath.Abs(dest)
	if err != nil {
		return nil, fmt.Errorf("%w: bad destination", ErrInvalid)
	}
	if s.insideDataDir(dest) {
		return nil, fmt.Errorf("%w: choose a location outside the application data folder", ErrInvalid)
	}

	snap := filepath.Join(os.TempDir(), "xolograph-snapshot-"+newID("t")+".db")
	defer os.Remove(snap)
	if _, err := s.DB.ExecContext(ctx, `VACUUM INTO ?`, snap); err != nil {
		return nil, fmt.Errorf("snapshot database: %w", err)
	}

	part := dest + ".partial"
	out, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot write destination: %v", ErrInvalid, err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = out.Close()
			_ = os.Remove(part)
		}
	}()

	archiveHash := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(out, archiveHash))
	created := now()
	var files []manifestFile
	add := func(name, src string, method uint16) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: created})
		if err != nil {
			return err
		}
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(w, h), f)
		if err != nil {
			return err
		}
		files = append(files, manifestFile{Path: name, Size: n, SHA256: hex.EncodeToString(h.Sum(nil))})
		return nil
	}

	if err := add("data/xolograph.db", snap, zip.Deflate); err != nil {
		return nil, fmt.Errorf("add database: %w", err)
	}
	if includeEvidence {
		root := filepath.Join(s.dataDir, "evidence")
		walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(s.dataDir, p)
			if err != nil {
				return err
			}
			return add("data/"+filepath.ToSlash(rel), p, zip.Store)
		})
		if walkErr != nil {
			return nil, fmt.Errorf("add evidence: %w", walkErr)
		}
	}

	mb, err := json.MarshalIndent(backupManifest{Format: backupFormat, CreatedAt: created, IncludesEvidence: includeEvidence, Files: files}, "", "  ")
	if err != nil {
		return nil, err
	}
	mw, err := zw.CreateHeader(&zip.FileHeader{Name: manifestName, Method: zip.Deflate, Modified: created})
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write(mb); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if err := out.Sync(); err != nil {
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(part, dest); err != nil {
		return nil, err
	}
	ok = true

	res := &BackupResult{Path: dest, SHA256: hex.EncodeToString(archiveHash.Sum(nil)), Files: len(files) + 1, IncludesEvidence: includeEvidence, CreatedAt: created}
	if st, err := os.Stat(dest); err == nil {
		res.SizeBytes = st.Size()
	}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		return s.appendAudit(ctx, tx, actor, "BACKUP_CREATED", "system", "backup",
			fmt.Sprintf("path=%s sha256=%s files=%d evidence=%t", dest, res.SHA256, res.Files, includeEvidence))
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func validArchiveName(n string) bool {
	if strings.Contains(n, "\\") || strings.Contains(n, ":") || path.Clean(n) != n {
		return false
	}
	return n == "data/xolograph.db" || strings.HasPrefix(n, "data/evidence/")
}

func readManifest(zf *zip.File) (*backupManifest, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, maxManifest))
	if err != nil {
		return nil, err
	}
	var m backupManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func extractVerified(zf *zip.File, dst string, want manifestFile) error {
	rc, err := zf.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	defer rc.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, cerr := io.Copy(io.MultiWriter(out, h), io.LimitReader(rc, want.Size+1))
	if serr := out.Sync(); cerr == nil {
		cerr = serr
	}
	if clerr := out.Close(); cerr == nil {
		cerr = clerr
	}
	if cerr != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalid, want.Path, cerr)
	}
	if n != want.Size || hex.EncodeToString(h.Sum(nil)) != want.SHA256 {
		return fmt.Errorf("%w: integrity check failed for %s", ErrInvalid, want.Path)
	}
	if strings.HasPrefix(want.Path, "data/evidence/") {
		_ = os.Chmod(dst, 0o400)
	}
	return nil
}

// StageRestore fully verifies a backup (every file against its manifest digest, the
// database opens, the audit chain is checked) and stages it. It is applied the next
// time the engine starts; the current data is kept under pre-restore/.
func (s *Store) StageRestore(ctx context.Context, archivePath, actor string) (res *StageResult, err error) {
	src, aerr := filepath.Abs(strings.TrimSpace(archivePath))
	if aerr != nil || archivePath == "" {
		return nil, fmt.Errorf("%w: backup path required", ErrInvalid)
	}
	if s.insideDataDir(src) {
		return nil, fmt.Errorf("%w: choose a backup file outside the application data folder", ErrInvalid)
	}
	archiveSum, _, herr := hashFile(src)
	if herr != nil {
		return nil, fmt.Errorf("%w: cannot read backup: %v", ErrInvalid, herr)
	}
	zr, zerr := zip.OpenReader(src)
	if zerr != nil {
		return nil, fmt.Errorf("%w: not a valid backup archive", ErrInvalid)
	}
	defer zr.Close()

	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if _, dup := byName[f.Name]; dup {
			return nil, fmt.Errorf("%w: duplicate entry %q", ErrInvalid, f.Name)
		}
		byName[f.Name] = f
	}
	mf, found := byName[manifestName]
	if !found {
		return nil, fmt.Errorf("%w: manifest missing", ErrInvalid)
	}
	m, merr := readManifest(mf)
	if merr != nil || m.Format != backupFormat {
		return nil, fmt.Errorf("%w: unsupported or damaged manifest", ErrInvalid)
	}
	if len(byName)-1 != len(m.Files) {
		return nil, fmt.Errorf("%w: archive contents do not match the manifest", ErrInvalid)
	}
	hasDB := false
	var total int64
	for _, f := range m.Files {
		if !validArchiveName(f.Path) || f.Size < 0 || len(f.SHA256) != 64 {
			return nil, fmt.Errorf("%w: unsafe or malformed entry %q", ErrInvalid, f.Path)
		}
		if f.Path == "data/xolograph.db" {
			hasDB = true
		}
		total += f.Size
	}
	if !hasDB {
		return nil, fmt.Errorf("%w: backup contains no database", ErrInvalid)
	}

	pending := filepath.Join(s.dataDir, stagingDirName)
	_ = os.RemoveAll(pending)
	defer func() {
		if err != nil {
			_ = os.RemoveAll(pending)
		}
	}()
	for _, f := range m.Files {
		zf, ok := byName[f.Path]
		if !ok {
			return nil, fmt.Errorf("%w: %s listed but missing", ErrInvalid, f.Path)
		}
		if xerr := extractVerified(zf, filepath.Join(pending, filepath.FromSlash(f.Path)), f); xerr != nil {
			return nil, xerr
		}
	}

	staged, oerr := Open(filepath.Join(pending, "data"))
	if oerr != nil {
		return nil, fmt.Errorf("%w: backup database is unusable: %v", ErrInvalid, oerr)
	}
	cases, cerr := staged.ListCases(ctx)
	chain, verr := staged.VerifyAuditChain(ctx)
	_ = staged.Close()
	if cerr != nil || verr != nil {
		return nil, fmt.Errorf("%w: backup database could not be read", ErrInvalid)
	}

	res = &StageResult{
		ArchiveSHA256: archiveSum, ArchiveCreatedAt: m.CreatedAt, StagedAt: now(),
		Files: len(m.Files), Bytes: total, IncludesEvidence: m.IncludesEvidence,
		Cases: len(cases), AuditEntries: chain.Count, AuditChainValid: chain.Valid, AuditBrokenAt: chain.BrokenAt,
	}
	mb, _ := json.MarshalIndent(res, "", "  ")
	if werr := os.WriteFile(filepath.Join(pending, stageMarker), mb, 0o600); werr != nil {
		return nil, werr
	}
	if terr := s.inTx(ctx, func(tx *sql.Tx) error {
		return s.appendAudit(ctx, tx, actor, "RESTORE_STAGED", "system", "restore", "archive_sha256="+archiveSum)
	}); terr != nil {
		return nil, terr
	}
	return res, nil
}

func (s *Store) CancelStagedRestore(ctx context.Context, actor string) error {
	if err := os.RemoveAll(filepath.Join(s.dataDir, stagingDirName)); err != nil {
		return err
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		return s.appendAudit(ctx, tx, actor, "RESTORE_CANCELLED", "system", "restore", "")
	})
}

func (s *Store) RestoreStatus() (*RestoreStatus, error) {
	st := &RestoreStatus{}
	if b, err := os.ReadFile(filepath.Join(s.dataDir, stagingDirName, stageMarker)); err == nil {
		var sr StageResult
		if json.Unmarshal(b, &sr) == nil {
			st.Pending = &sr
		}
	}
	if b, err := os.ReadFile(filepath.Join(s.dataDir, resultFile)); err == nil {
		var rr RestoreResult
		if json.Unmarshal(b, &rr) == nil {
			st.Last = &rr
		}
	}
	return st, nil
}

func writeRestoreResult(dataDir string, r *RestoreResult) {
	if b, err := json.MarshalIndent(r, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(dataDir, resultFile), b, 0o600)
	}
}

// applyPendingRestore swaps a staged restore into place. Current data is moved (not
// deleted) to pre-restore/<timestamp>; on any failure everything is rolled back and
// the app starts with its existing data.
func applyPendingRestore(dataDir string) *RestoreResult {
	pending := filepath.Join(dataDir, stagingDirName)
	b, err := os.ReadFile(filepath.Join(pending, stageMarker))
	if err != nil {
		return nil // nothing (completely) staged
	}
	var st StageResult
	res := &RestoreResult{AppliedAt: now()}
	fail := func(msg string) *RestoreResult {
		res.Error = msg
		writeRestoreResult(dataDir, res)
		_ = os.RemoveAll(pending)
		return res
	}
	if json.Unmarshal(b, &st) != nil {
		return fail("staged restore marker is corrupt")
	}
	res.ArchiveSHA256, res.ArchiveCreatedAt = st.ArchiveSHA256, st.ArchiveCreatedAt

	staged := filepath.Join(pending, "data")
	if _, err := os.Stat(filepath.Join(staged, "xolograph.db")); err != nil {
		return fail("staged database is missing")
	}
	prev := filepath.Join(dataDir, "pre-restore", now().Format("20060102-150405"))
	if err := os.MkdirAll(prev, 0o700); err != nil {
		return fail("cannot create pre-restore folder: " + err.Error())
	}
	names := []string{"xolograph.db", "xolograph.db-wal", "xolograph.db-shm", "evidence"}
	var moved, installed []string
	rollback := func() {
		for _, n := range installed {
			_ = os.Rename(filepath.Join(dataDir, n), filepath.Join(staged, n))
		}
		for _, n := range moved {
			_ = os.Rename(filepath.Join(prev, n), filepath.Join(dataDir, n))
		}
	}
	for _, n := range names {
		cur := filepath.Join(dataDir, n)
		if _, err := os.Lstat(cur); err != nil {
			continue
		}
		if err := os.Rename(cur, filepath.Join(prev, n)); err != nil {
			rollback()
			return fail("could not set current data aside: " + err.Error())
		}
		moved = append(moved, n)
	}
	for _, n := range names {
		from := filepath.Join(staged, n)
		if _, err := os.Lstat(from); err != nil {
			continue
		}
		if err := os.Rename(from, filepath.Join(dataDir, n)); err != nil {
			rollback()
			return fail("could not install restored data: " + err.Error())
		}
		installed = append(installed, n)
	}
	_ = os.RemoveAll(pending)
	res.Applied, res.PreviousData = true, prev
	writeRestoreResult(dataDir, res)
	return res
}

