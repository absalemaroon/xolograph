package store

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Label is an analyst- or source-attributed name for an address. It is never a
// statement of real-world ownership: every label carries its source, confidence and basis.
type Label struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"caseId"`
	CaseTitle  string    `json:"caseTitle"`
	Address    string    `json:"address"`
	Network    string    `json:"network"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Source     string    `json:"source"`
	Confidence string    `json:"confidence"`
	Basis      string    `json:"basis"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

var ethAddrRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func (s *Store) AddLabel(ctx context.Context, l Label, actor string) (*Label, error) {
	l.Address = strings.ToLower(strings.TrimSpace(l.Address))
	l.Name = strings.TrimSpace(l.Name)
	l.Basis = strings.TrimSpace(l.Basis)
	l.Category = strings.ToUpper(strings.TrimSpace(l.Category))
	l.Source = strings.ToUpper(strings.TrimSpace(l.Source))
	l.Confidence = strings.ToUpper(strings.TrimSpace(l.Confidence))
	if l.Category == "" {
		l.Category = "UNCATEGORIZED"
	}
	switch {
	case !ethAddrRe.MatchString(l.Address):
		return nil, fmt.Errorf("%w: invalid Ethereum address", ErrInvalid)
	case l.Name == "" || len(l.Name) > 120:
		return nil, fmt.Errorf("%w: label name must be 1-120 characters", ErrInvalid)
	case len(l.Basis) < 3:
		return nil, fmt.Errorf("%w: a basis (supporting evidence or source) is required", ErrInvalid)
	case l.Source != "ANALYST_ASSERTION" && l.Source != "THIRD_PARTY_PROVIDER" && l.Source != "VERIFIED_SOURCE":
		return nil, fmt.Errorf("%w: unknown label source", ErrInvalid)
	case l.Confidence != "LOW" && l.Confidence != "MEDIUM" && l.Confidence != "HIGH":
		return nil, fmt.Errorf("%w: confidence must be LOW, MEDIUM or HIGH", ErrInvalid)
	}
	if _, err := s.GetCase(ctx, l.CaseID); err != nil {
		return nil, err
	}
	l.ID, l.Network, l.CreatedBy, l.CreatedAt = newID("lbl"), "ethereum-mainnet", actorOrDefault(actor), now()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entity_labels(id,case_id,address,network,name,category,source,confidence,basis,created_by,created_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, l.ID, l.CaseID, l.Address, l.Network, l.Name, l.Category, l.Source, l.Confidence, l.Basis, l.CreatedBy, ts(l.CreatedAt)); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, l.CaseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "LABEL_ADDED", "case", l.CaseID, l.ID+" "+l.Address+" "+l.Name+" ["+l.Source+"/"+l.Confidence+"]")
	})
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ListLabels filters by case and/or address; empty filters mean "all".
func (s *Store) ListLabels(ctx context.Context, caseID, address string) ([]Label, error) {
	q := `SELECT l.id,l.case_id,c.title,l.address,l.network,l.name,l.category,l.source,l.confidence,l.basis,l.created_by,l.created_at
		FROM entity_labels l JOIN cases c ON c.id=l.case_id WHERE 1=1`
	args := []any{}
	if caseID != "" {
		q += ` AND l.case_id=?`
		args = append(args, caseID)
	}
	if address != "" {
		q += ` AND l.address=?`
		args = append(args, strings.ToLower(strings.TrimSpace(address)))
	}
	rows, err := s.DB.QueryContext(ctx, q+` ORDER BY l.created_at DESC LIMIT 1000`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Label{}
	for rows.Next() {
		var l Label
		var t string
		if err := rows.Scan(&l.ID, &l.CaseID, &l.CaseTitle, &l.Address, &l.Network, &l.Name, &l.Category, &l.Source, &l.Confidence, &l.Basis, &l.CreatedBy, &t); err != nil {
			return nil, err
		}
		l.CreatedAt = parseTS(t)
		out = append(out, l)
	}
	return out, rows.Err()
}
