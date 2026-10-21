package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ReportEvidence struct {
	Evidence
	Custody      []CustodyEvent `json:"custody"`
	CustodyChain ChainResult    `json:"custodyChain"`
}

// Report is the full, source-linked record of a case at a point in time.
type Report struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	GeneratedBy string           `json:"generatedBy"`
	Case        Case             `json:"case"`
	Notes       []Note           `json:"notes"`
	Evidence    []ReportEvidence `json:"evidence"`
	Findings    []Finding        `json:"findings"`
	Labels      []Label          `json:"labels"`
	AuditChain  ChainResult      `json:"auditChain"`
}

// ReportBundle holds the exact exported bytes and their SHA-256 digests, so a
// recipient can verify a downloaded file with any standard sha256 tool.
type ReportBundle struct {
	GeneratedAt    time.Time `json:"generatedAt"`
	Markdown       string    `json:"markdown"`
	MarkdownSHA256 string    `json:"markdownSha256"`
	JSON           string    `json:"json"`
	JSONSHA256     string    `json:"jsonSha256"`
}

func sha(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (s *Store) BuildReport(ctx context.Context, caseID, actor string) (*Report, error) {
	c, err := s.GetCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	r := &Report{GeneratedAt: now(), GeneratedBy: actorOrDefault(actor), Case: *c}
	if r.Notes, err = s.ListNotes(ctx, caseID); err != nil {
		return nil, err
	}
	evs, err := s.ListEvidence(ctx, caseID)
	if err != nil {
		return nil, err
	}
	r.Evidence = []ReportEvidence{}
	for _, e := range evs {
		cu, err := s.ListCustody(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		ch, err := s.VerifyCustodyChain(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		r.Evidence = append(r.Evidence, ReportEvidence{Evidence: e, Custody: cu, CustodyChain: ch})
	}
	if r.Findings, err = s.ListFindings(ctx, caseID); err != nil {
		return nil, err
	}
	if r.Labels, err = s.ListLabels(ctx, caseID, ""); err != nil {
		return nil, err
	}
	if r.AuditChain, err = s.VerifyAuditChain(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\r", " ")), " ")
}

func utc(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") + " UTC" }

func RenderMarkdown(r *Report) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	w("# Case Report: %s", oneLine(r.Case.Title))
	w("")
	w("- Case ID: `%s`", r.Case.ID)
	w("- Status: %s", r.Case.Status)
	w("- Opened: %s", utc(r.Case.CreatedAt))
	w("- Report generated: %s by %s", utc(r.GeneratedAt), oneLine(r.GeneratedBy))
	w("- Application audit chain at generation: %s (%d entries)", chainWord(r.AuditChain), r.AuditChain.Count)
	if r.Case.Description != "" {
		w("\n## Description\n\n%s", r.Case.Description)
	}

	w("\n## Evidence (%d)", len(r.Evidence))
	if len(r.Evidence) == 0 {
		w("\nNo evidence imported.")
	}
	for _, e := range r.Evidence {
		w("\n### %s", oneLine(e.Name))
		w("- Evidence ID: `%s`", e.ID)
		w("- SHA-256 (at import): `%s`", e.SHA256)
		w("- Size: %d bytes · Status: %s · Imported: %s", e.SizeBytes, e.Status, utc(e.ImportedAt))
		if e.LastVerifiedAt != nil {
			w("- Last verified: %s", utc(*e.LastVerifiedAt))
		}
		w("- Custody chain: %s (%d events)", chainWord(e.CustodyChain), e.CustodyChain.Count)
		for _, c := range e.Custody {
			w("  - %s · %s · %s · %s", utc(c.Timestamp), c.Action, oneLine(c.Actor), oneLine(c.Detail))
		}
	}

	w("\n## Findings (%d)", len(r.Findings))
	if len(r.Findings) == 0 {
		w("\nNo findings recorded.")
	}
	for _, f := range r.Findings {
		w("\n### [%s] %s", f.Kind, oneLine(f.Title))
		w("- Recorded: %s", utc(f.CreatedAt))
		if f.RefID != "" {
			w("- Reference: `%s` (%s)", oneLine(f.RefID), f.RefType)
		}
		if f.Note != "" {
			w("- Note: %s", oneLine(f.Note))
		}
		if f.SourceJSON != "" && f.SourceJSON != "{}" {
			w("- Source snapshot: preserved in the JSON export (`findings[].sourceJson`)")
		}
	}

	w("\n## Entity labels (%d)", len(r.Labels))
	if len(r.Labels) == 0 {
		w("\nNo labels recorded.")
	} else {
		w("\nLabels are attributions with a stated basis; they are not proof of real-world ownership.\n")
	}
	for _, l := range r.Labels {
		w("- `%s` — **%s** (%s) · source: %s · confidence: %s · basis: %s", l.Address, oneLine(l.Name), l.Category, l.Source, l.Confidence, oneLine(l.Basis))
	}

	w("\n## Notes (%d)", len(r.Notes))
	for _, n := range r.Notes {
		w("\n- %s — %s", utc(n.CreatedAt), oneLine(n.Body))
	}

	w("\n---\nIntegrity: the SHA-256 of this Markdown file and of the companion JSON export are recorded in the Xolograph audit log at export time.")
	return b.String()
}

func chainWord(c ChainResult) string {
	if c.Valid {
		return "VALID"
	}
	return fmt.Sprintf("BROKEN at record #%d", c.BrokenAt)
}

// ExportReport builds the report, renders both formats, hashes the exact bytes
// and records the export (with both digests) in the audit log.
func (s *Store) ExportReport(ctx context.Context, caseID, actor string) (*ReportBundle, error) {
	r, err := s.BuildReport(ctx, caseID, actor)
	if err != nil {
		return nil, err
	}
	jb, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	bundle := &ReportBundle{GeneratedAt: r.GeneratedAt, Markdown: RenderMarkdown(r), JSON: string(jb)}
	bundle.MarkdownSHA256, bundle.JSONSHA256 = sha(bundle.Markdown), sha(bundle.JSON)
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		return s.appendAudit(ctx, tx, actor, "REPORT_EXPORTED", "case", caseID,
			"markdown_sha256="+bundle.MarkdownSHA256+" json_sha256="+bundle.JSONSHA256)
	})
	if err != nil {
		return nil, err
	}
	return bundle, nil
}
