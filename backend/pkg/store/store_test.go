package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const helloSHA = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" // sha256("hello")

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCaseLifecycleAndPersistence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateCase(ctx, "  Bridge exploit  ", "desc", "")
	if err != nil || c.Title != "Bridge exploit" || c.Status != "OPEN" {
		t.Fatalf("create: %+v %v", c, err)
	}
	if _, err := s.CreateCase(ctx, "   ", "", ""); err == nil {
		t.Fatal("empty title must be rejected")
	}
	if _, err := s.AddNote(ctx, c.ID, "first note", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetCaseStatus(ctx, c.ID, "bogus", ""); err == nil {
		t.Fatal("invalid status must be rejected")
	}
	_ = s.Close()

	s2, err := Open(dir) // restart: data must persist, migrations idempotent
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	list, _ := s2.ListCases(ctx)
	notes, _ := s2.ListNotes(ctx, c.ID)
	if len(list) != 1 || len(notes) != 1 {
		t.Fatalf("persistence failed: %d cases, %d notes", len(list), len(notes))
	}
}

func TestEvidenceImportVerifyAndTamper(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	src := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	ev, err := s.ImportEvidence(ctx, c.ID, src, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if ev.SHA256 != helloSHA || ev.Status != "IMPORTED" {
		t.Fatalf("bad import: %+v", ev)
	}
	v, err := s.VerifyEvidence(ctx, ev.ID, "tester")
	if err != nil || v.Status != "VERIFIED" {
		t.Fatalf("verify: %+v %v", v, err)
	}
	// tamper with the stored copy
	_ = os.Chmod(ev.StoredPath, 0o600)
	_ = os.WriteFile(ev.StoredPath, []byte("hellO"), 0o600)
	v, err = s.VerifyEvidence(ctx, ev.ID, "tester")
	if err != nil || v.Status != "FAILED" {
		t.Fatalf("tamper not detected: %+v %v", v, err)
	}
	chain, _ := s.VerifyCustodyChain(ctx, ev.ID)
	events, _ := s.ListCustody(ctx, ev.ID)
	if !chain.Valid || len(events) != 3 || events[2].Action != "INTEGRITY_FAILURE" {
		t.Fatalf("custody wrong: %+v %+v", chain, events)
	}
}

func TestImportRejectsBadSource(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	if _, err := s.ImportEvidence(ctx, c.ID, t.TempDir(), ""); err == nil {
		t.Fatal("directory must be rejected")
	}
	if _, err := s.ImportEvidence(ctx, c.ID, "/no/such/file", ""); err == nil {
		t.Fatal("missing file must be rejected")
	}
	if _, err := s.ImportEvidence(ctx, "case_missing", "x", ""); err != ErrNotFound {
		t.Fatalf("unknown case must be ErrNotFound, got %v", err)
	}
}

func TestSanitizeName(t *testing.T) {
	for in, want := range map[string]string{"../../etc/passwd": "passwd", "a:b*c.txt": "a_b_c.txt", "..": "evidence.bin"} {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestAuditChainDetectsTampering(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	_, _ = s.AddNote(ctx, c.ID, "n", "")
	r, err := s.VerifyAuditChain(ctx)
	if err != nil || !r.Valid || r.Count != 2 {
		t.Fatalf("clean chain: %+v %v", r, err)
	}
	if _, err := s.DB.ExecContext(ctx, `UPDATE audit_log SET detail='forged' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	r, _ = s.VerifyAuditChain(ctx)
	if r.Valid || r.BrokenAt != 1 {
		t.Fatalf("tamper not detected: %+v", r)
	}
}

func TestFindings(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	if _, err := s.AddFinding(ctx, Finding{CaseID: c.ID, Kind: "address", RefID: "0xabc", Title: "Suspect wallet"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddFinding(ctx, Finding{CaseID: "nope", Kind: "x", Title: "y"}, ""); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	got, _ := s.GetCase(ctx, c.ID)
	if got.FindingCount != 1 {
		t.Fatalf("finding count %d", got.FindingCount)
	}
}

func TestLabelsRequireBasisAndFilter(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	addr := "0xAB" + strings.Repeat("cd", 19)
	ok := Label{CaseID: c.ID, Address: addr, Name: "Mixer deposit", Source: "ANALYST_ASSERTION", Confidence: "LOW", Basis: "funds flow from seized wallet"}
	bad := ok
	bad.Basis = ""
	if _, err := s.AddLabel(ctx, bad, ""); err == nil {
		t.Fatal("label without basis must be rejected")
	}
	bad = ok
	bad.Source = "TRUST_ME"
	if _, err := s.AddLabel(ctx, bad, ""); err == nil {
		t.Fatal("unknown source must be rejected")
	}
	if _, err := s.AddLabel(ctx, ok, ""); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListLabels(ctx, "", addr)
	if len(list) != 1 || list[0].Address != strings.ToLower(addr) || list[0].CaseTitle != "Case" {
		t.Fatalf("labels: %+v", list)
	}
	if other, _ := s.ListLabels(ctx, "", "0x"+strings.Repeat("0", 40)); len(other) != 0 {
		t.Fatal("filter by address failed")
	}
}

func TestReportExportHashesAndAudit(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	c, _ := s.CreateCase(ctx, "Report case", "d", "")
	src := filepath.Join(t.TempDir(), "e.txt")
	_ = os.WriteFile(src, []byte("hello"), 0o600)
	_, _ = s.ImportEvidence(ctx, c.ID, src, "")
	_, _ = s.AddNote(ctx, c.ID, "a note", "")
	_, _ = s.AddFinding(ctx, Finding{CaseID: c.ID, Kind: "address", RefID: "0xabc", Title: "Wallet", SourceJSON: `{"x":1}`}, "")
	b, err := s.ExportReport(ctx, c.ID, "analyst-1")
	if err != nil {
		t.Fatal(err)
	}
	if b.MarkdownSHA256 != sha(b.Markdown) || b.JSONSHA256 != sha(b.JSON) {
		t.Fatal("digests must match exported bytes")
	}
	for _, want := range []string{"# Case Report: Report case", helloSHA, "Custody chain: VALID", "[ADDRESS] Wallet", "analyst-1"} {
		if !strings.Contains(b.Markdown, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
	entries, _ := s.ListAudit(ctx, 1)
	if len(entries) != 1 || entries[0].Action != "REPORT_EXPORTED" || !strings.Contains(entries[0].Detail, b.MarkdownSHA256) {
		t.Fatalf("export not audited: %+v", entries)
	}
	if r, _ := s.VerifyAuditChain(ctx); !r.Valid {
		t.Fatal("audit chain must stay valid")
	}
}
