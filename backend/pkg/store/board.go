package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

// The case board is an investigator's working map of how things relate. Two rules keep it honest:
//  1. A link is either a FACT or a HYPOTHESIS. A FACT must be supported by at least one linked evidence item
//     or a stated source; anything weaker is a hypothesis and is always shown as one.
//  2. Chain activity never creates links. Every link is drawn by a person and carries their stated basis.

var (
	boardKinds  = set("ENTITY", "WALLET", "EVIDENCE", "NOTE", "EVENT")
	boardClaims = set("FACT", "HYPOTHESIS")
)

type BoardNode struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"caseId"`
	Kind      string    `json:"kind"`
	RefID     string    `json:"refId"`
	Network   string    `json:"network"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Live values joined from the referenced record (empty for NOTE/EVENT/WALLET).
	LiveName      string `json:"liveName,omitempty"`
	EntityStatus  string `json:"entityStatus,omitempty"`
	EvidenceHash  string `json:"evidenceHash,omitempty"`
	EvidenceState string `json:"evidenceState,omitempty"`
}

type BoardLinkEvidence struct {
	ID          string    `json:"id"`
	LinkID      string    `json:"linkId"`
	EvidenceID  string    `json:"evidenceId"`
	Name        string    `json:"name"`
	SHA256      string    `json:"sha256"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type BoardLink struct {
	ID         string              `json:"id"`
	CaseID     string              `json:"caseId"`
	SourceID   string              `json:"sourceId"`
	TargetID   string              `json:"targetId"`
	Relation   string              `json:"relation"`
	Claim      string              `json:"claim"`
	Confidence string              `json:"confidence"`
	SourceRef  string              `json:"sourceRef"`
	Note       string              `json:"note"`
	Evidence   []BoardLinkEvidence `json:"evidence"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
}

type Board struct {
	Nodes []BoardNode `json:"nodes"`
	Links []BoardLink `json:"links"`
}

type BoardNodeInput struct {
	Kind, RefID, Network, Title, Body string
	X, Y                              float64
}

type BoardLinkInput struct {
	SourceID, TargetID, Relation, Claim, Confidence, SourceRef, Note string
	EvidenceIDs                                                      []string
}

func validCoord(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) <= 1e6 }

func (s *Store) GetBoard(ctx context.Context, caseID string) (*Board, error) {
	if _, err := s.GetCase(ctx, caseID); err != nil {
		return nil, err
	}
	b := &Board{Nodes: []BoardNode{}, Links: []BoardLink{}}
	nr, err := s.DB.QueryContext(ctx, `SELECT n.id,n.case_id,n.kind,n.ref_id,n.network,n.title,n.body,n.x,n.y,n.created_at,n.updated_at,
			COALESCE(e.name,''),COALESCE(e.status,''),COALESCE(v.sha256,''),COALESCE(v.status,'')
		FROM board_nodes n
		LEFT JOIN entities e ON n.kind='ENTITY' AND e.id=n.ref_id
		LEFT JOIN evidence v ON n.kind='EVIDENCE' AND v.id=n.ref_id
		WHERE n.case_id=? ORDER BY n.created_at, n.id`, caseID)
	if err != nil {
		return nil, err
	}
	for nr.Next() {
		var n BoardNode
		var c, u string
		if err := nr.Scan(&n.ID, &n.CaseID, &n.Kind, &n.RefID, &n.Network, &n.Title, &n.Body, &n.X, &n.Y, &c, &u, &n.LiveName, &n.EntityStatus, &n.EvidenceHash, &n.EvidenceState); err != nil {
			nr.Close()
			return nil, err
		}
		n.CreatedAt, n.UpdatedAt = parseTS(c), parseTS(u)
		b.Nodes = append(b.Nodes, n)
	}
	nr.Close()
	if err := nr.Err(); err != nil {
		return nil, err
	}
	lr, err := s.DB.QueryContext(ctx, `SELECT id,case_id,source_id,target_id,relation,claim,confidence,source_ref,note,created_at,updated_at FROM board_links WHERE case_id=? ORDER BY created_at, id`, caseID)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for lr.Next() {
		var l BoardLink
		var c, u string
		if err := lr.Scan(&l.ID, &l.CaseID, &l.SourceID, &l.TargetID, &l.Relation, &l.Claim, &l.Confidence, &l.SourceRef, &l.Note, &c, &u); err != nil {
			lr.Close()
			return nil, err
		}
		l.CreatedAt, l.UpdatedAt = parseTS(c), parseTS(u)
		l.Evidence = []BoardLinkEvidence{}
		idx[l.ID] = len(b.Links)
		b.Links = append(b.Links, l)
	}
	lr.Close()
	if err := lr.Err(); err != nil {
		return nil, err
	}
	er, err := s.DB.QueryContext(ctx, `SELECT x.id,x.link_id,x.evidence_id,v.name,v.sha256,v.status,x.description,x.created_at
		FROM board_link_evidence x JOIN evidence v ON v.id=x.evidence_id JOIN board_links l ON l.id=x.link_id WHERE l.case_id=? ORDER BY x.created_at, x.id`, caseID)
	if err != nil {
		return nil, err
	}
	defer er.Close()
	for er.Next() {
		var x BoardLinkEvidence
		var c string
		if err := er.Scan(&x.ID, &x.LinkID, &x.EvidenceID, &x.Name, &x.SHA256, &x.Status, &x.Description, &c); err != nil {
			return nil, err
		}
		x.CreatedAt = parseTS(c)
		if i, ok := idx[x.LinkID]; ok {
			b.Links[i].Evidence = append(b.Links[i].Evidence, x)
		}
	}
	return b, er.Err()
}

func (s *Store) AddBoardNode(ctx context.Context, caseID string, in BoardNodeInput, actor string) (*Board, error) {
	in.Kind = upper(in.Kind, "")
	in.RefID, in.Network = strings.TrimSpace(in.RefID), strings.TrimSpace(in.Network)
	in.Title, in.Body = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if !boardKinds[in.Kind] {
		return nil, fmt.Errorf("%w: unknown node kind", ErrInvalid)
	}
	if !validCoord(in.X) || !validCoord(in.Y) {
		return nil, fmt.Errorf("%w: invalid position", ErrInvalid)
	}
	if len(in.Title) > 120 || len(in.Body) > 4000 {
		return nil, fmt.Errorf("%w: title or body too long", ErrInvalid)
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM cases WHERE id=?`, caseID).Scan(&one); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		switch in.Kind {
		case "ENTITY":
			var name string
			if err := tx.QueryRowContext(ctx, `SELECT name FROM entities WHERE id=?`, in.RefID).Scan(&name); err == sql.ErrNoRows {
				return fmt.Errorf("%w: entity not found", ErrInvalid)
			} else if err != nil {
				return err
			}
			in.Network = ""
			if in.Title == "" {
				in.Title = name
			}
		case "WALLET":
			if in.Network == "" {
				in.Network = "ethereum-mainnet"
			}
			norm, err := NormalizeAddress(in.Network, in.RefID)
			if err != nil {
				return err
			}
			in.RefID = norm
			if in.Title == "" {
				in.Title = norm[:6] + "…" + norm[len(norm)-4:]
			}
		case "EVIDENCE":
			var name string
			if err := tx.QueryRowContext(ctx, `SELECT name FROM evidence WHERE id=? AND case_id=?`, in.RefID, caseID).Scan(&name); err == sql.ErrNoRows {
				return fmt.Errorf("%w: evidence not found in this case", ErrInvalid)
			} else if err != nil {
				return err
			}
			in.Network = ""
			if in.Title == "" {
				in.Title = name
			}
		default: // NOTE, EVENT
			in.RefID, in.Network = "", ""
			if in.Title == "" {
				return fmt.Errorf("%w: a %s needs a title", ErrInvalid, strings.ToLower(in.Kind))
			}
		}
		id, t := newID("bn"), now()
		if _, err := tx.ExecContext(ctx, `INSERT INTO board_nodes(id,case_id,kind,ref_id,network,title,body,x,y,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			id, caseID, in.Kind, in.RefID, in.Network, in.Title, in.Body, in.X, in.Y, ts(t), ts(t)); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: this item is already on the board", ErrInvalid)
			}
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_NODE_ADDED", "case", caseID, in.Kind+" "+in.Title)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

func (s *Store) UpdateBoardNode(ctx context.Context, nodeID, title, body, actor string) (*Board, error) {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if title == "" || len(title) > 120 || len(body) > 4000 {
		return nil, fmt.Errorf("%w: title is required (max 120 chars); body max 4000", ErrInvalid)
	}
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT case_id FROM board_nodes WHERE id=?`, nodeID).Scan(&caseID); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE board_nodes SET title=?,body=?,updated_at=? WHERE id=?`, title, body, ts(now()), nodeID); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_NODE_UPDATED", "case", caseID, title)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

// MoveBoardNodes saves layout positions. Layout is not an investigative claim, so it is not audited.
func (s *Store) MoveBoardNodes(ctx context.Context, caseID string, ids []string, xs, ys []float64) error {
	if len(ids) != len(xs) || len(ids) != len(ys) || len(ids) > 500 {
		return fmt.Errorf("%w: malformed positions", ErrInvalid)
	}
	for i := range ids {
		if !validCoord(xs[i]) || !validCoord(ys[i]) {
			return fmt.Errorf("%w: invalid position", ErrInvalid)
		}
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for i, id := range ids {
			if _, err := tx.ExecContext(ctx, `UPDATE board_nodes SET x=?,y=? WHERE id=? AND case_id=?`, xs[i], ys[i], id, caseID); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBoardNode removes the node and every link attached to it. Referenced entities, wallets and evidence are untouched.
func (s *Store) DeleteBoardNode(ctx context.Context, nodeID, actor string) (*Board, error) {
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var title string
		if err := tx.QueryRowContext(ctx, `SELECT case_id,title FROM board_nodes WHERE id=?`, nodeID).Scan(&caseID, &title); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM board_nodes WHERE id=?`, nodeID); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_NODE_REMOVED", "case", caseID, title)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

// checkFactSupport enforces rule 1: a FACT needs evidence or a stated source.
func checkFactSupport(claim, sourceRef string, evidenceCount int) error {
	if claim == "FACT" && strings.TrimSpace(sourceRef) == "" && evidenceCount == 0 {
		return fmt.Errorf("%w: a fact needs at least one linked evidence item or a stated source; otherwise mark it as a hypothesis", ErrInvalid)
	}
	return nil
}

func validateLinkFields(relation, claim, confidence *string, sourceRef, note string) error {
	*relation, *claim, *confidence = strings.TrimSpace(*relation), upper(*claim, "HYPOTHESIS"), upper(*confidence, "LOW")
	switch {
	case *relation == "" || len(*relation) > 80:
		return fmt.Errorf("%w: relation must be 1-80 characters (e.g. \"funded\", \"same operator (suspected)\")", ErrInvalid)
	case !boardClaims[*claim]:
		return fmt.Errorf("%w: claim must be FACT or HYPOTHESIS", ErrInvalid)
	case !confidences[*confidence]:
		return fmt.Errorf("%w: confidence must be LOW, MEDIUM or HIGH", ErrInvalid)
	case len(sourceRef) > 2000 || len(note) > 4000:
		return fmt.Errorf("%w: field too long", ErrInvalid)
	}
	return nil
}

func (s *Store) AddBoardLink(ctx context.Context, caseID string, in BoardLinkInput, actor string) (*Board, error) {
	in.SourceRef, in.Note = strings.TrimSpace(in.SourceRef), strings.TrimSpace(in.Note)
	if err := validateLinkFields(&in.Relation, &in.Claim, &in.Confidence, in.SourceRef, in.Note); err != nil {
		return nil, err
	}
	if in.SourceID == in.TargetID {
		return nil, fmt.Errorf("%w: a link needs two different items", ErrInvalid)
	}
	evIDs := dedupe(in.EvidenceIDs)
	if err := checkFactSupport(in.Claim, in.SourceRef, len(evIDs)); err != nil {
		return nil, err
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_nodes WHERE case_id=? AND id IN (?,?)`, caseID, in.SourceID, in.TargetID).Scan(&n); err != nil {
			return err
		}
		if n != 2 {
			return fmt.Errorf("%w: both items must be on this case's board", ErrInvalid)
		}
		for _, ev := range evIDs {
			if err := tx.QueryRowContext(ctx, `SELECT 1 FROM evidence WHERE id=? AND case_id=?`, ev, caseID).Scan(&n); err == sql.ErrNoRows {
				return fmt.Errorf("%w: evidence %s is not in this case", ErrInvalid, ev)
			} else if err != nil {
				return err
			}
		}
		id, t := newID("bl"), now()
		if _, err := tx.ExecContext(ctx, `INSERT INTO board_links(id,case_id,source_id,target_id,relation,claim,confidence,source_ref,note,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			id, caseID, in.SourceID, in.TargetID, in.Relation, in.Claim, in.Confidence, in.SourceRef, in.Note, ts(t), ts(t)); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: this exact link already exists", ErrInvalid)
			}
			return err
		}
		for _, ev := range evIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO board_link_evidence(id,link_id,evidence_id,description,created_at) VALUES(?,?,?,?,?)`, newID("ble"), id, ev, "", ts(t)); err != nil {
				return err
			}
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_LINK_ADDED", "case", caseID, fmt.Sprintf("%s [%s/%s, %d evidence]", in.Relation, in.Claim, in.Confidence, len(evIDs)))
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func (s *Store) UpdateBoardLink(ctx context.Context, linkID, relation, claim, confidence, sourceRef, note, actor string) (*Board, error) {
	sourceRef, note = strings.TrimSpace(sourceRef), strings.TrimSpace(note)
	if err := validateLinkFields(&relation, &claim, &confidence, sourceRef, note); err != nil {
		return nil, err
	}
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var oldClaim string
		if err := tx.QueryRowContext(ctx, `SELECT case_id,claim FROM board_links WHERE id=?`, linkID).Scan(&caseID, &oldClaim); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		var evN int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_link_evidence WHERE link_id=?`, linkID).Scan(&evN); err != nil {
			return err
		}
		if err := checkFactSupport(claim, sourceRef, evN); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE board_links SET relation=?,claim=?,confidence=?,source_ref=?,note=?,updated_at=? WHERE id=?`,
			relation, claim, confidence, sourceRef, note, ts(now()), linkID); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: this exact link already exists", ErrInvalid)
			}
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_LINK_UPDATED", "case", caseID, fmt.Sprintf("%s: %s -> %s [%s]", relation, oldClaim, claim, confidence))
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

func (s *Store) DeleteBoardLink(ctx context.Context, linkID, actor string) (*Board, error) {
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var rel string
		if err := tx.QueryRowContext(ctx, `SELECT case_id,relation FROM board_links WHERE id=?`, linkID).Scan(&caseID, &rel); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM board_links WHERE id=?`, linkID); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_LINK_REMOVED", "case", caseID, rel)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

func (s *Store) AddBoardLinkEvidence(ctx context.Context, linkID, evidenceID, description, actor string) (*Board, error) {
	description = strings.TrimSpace(description)
	if len(description) > 2000 {
		return nil, fmt.Errorf("%w: description too long", ErrInvalid)
	}
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT case_id FROM board_links WHERE id=?`, linkID).Scan(&caseID); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM evidence WHERE id=? AND case_id=?`, evidenceID, caseID).Scan(&one); err == sql.ErrNoRows {
			return fmt.Errorf("%w: evidence is not in this case", ErrInvalid)
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO board_link_evidence(id,link_id,evidence_id,description,created_at) VALUES(?,?,?,?,?)`, newID("ble"), linkID, evidenceID, description, ts(now())); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: evidence already linked", ErrInvalid)
			}
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_LINK_EVIDENCE_ADDED", "case", caseID, linkID+" <- "+evidenceID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}

// RemoveBoardLinkEvidence refuses to strip the last support from a FACT that has no stated source.
func (s *Store) RemoveBoardLinkEvidence(ctx context.Context, linkEvidenceID, actor string) (*Board, error) {
	var caseID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var linkID, evID, claim, sourceRef string
		err := tx.QueryRowContext(ctx, `SELECT x.link_id,x.evidence_id,l.case_id,l.claim,l.source_ref FROM board_link_evidence x JOIN board_links l ON l.id=x.link_id WHERE x.id=?`, linkEvidenceID).
			Scan(&linkID, &evID, &caseID, &claim, &sourceRef)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_link_evidence WHERE link_id=?`, linkID).Scan(&n); err != nil {
			return err
		}
		if err := checkFactSupport(claim, sourceRef, n-1); err != nil {
			return fmt.Errorf("%w (change the link to a hypothesis first, or add a source)", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM board_link_evidence WHERE id=?`, linkEvidenceID); err != nil {
			return err
		}
		if err := s.touchCase(ctx, tx, caseID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "BOARD_LINK_EVIDENCE_REMOVED", "case", caseID, linkID+" -x "+evID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBoard(ctx, caseID)
}
