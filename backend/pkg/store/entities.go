package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// An Entity is a structured investigative record. It is NOT a wallet with a name:
// wallet ownership is a per-address claim (EntityAddress) that carries its own status,
// confidence, source and history. Observed transfers never create or merge associations.

var (
	entityTypes = set("INDIVIDUAL", "ORGANIZATION", "COMPANY", "EXCHANGE", "PROTOCOL", "DAPP", "TOKEN_PROJECT", "FUND", "WALLET_OPERATOR", "OTHER")
	// Attribution status, strongest to weakest claim:
	//   VERIFIED                 confirmed by the controlling party or a directly verifiable primary source
	//   PUBLICLY_REPORTED        stated by a public source (official docs, explorer label, reliable reporting)
	//   INVESTIGATOR_ATTRIBUTED  concluded by the analyst from stated evidence
	//   SUSPECTED                plausible but unconfirmed
	//   DISPUTED                 conflicting claims exist
	//   REJECTED                 considered and ruled out (kept so it is not re-proposed)
	//   UNKNOWN                  no assessment
	attributionStatuses = set("VERIFIED", "PUBLICLY_REPORTED", "INVESTIGATOR_ATTRIBUTED", "SUSPECTED", "DISPUTED", "REJECTED", "UNKNOWN")
	confidences         = set("LOW", "MEDIUM", "HIGH")
	identPlatforms      = set("WEBSITE", "X", "TELEGRAM", "DISCORD", "FACEBOOK", "LINKEDIN", "GITHUB", "EMAIL", "OTHER")
	identStatuses       = set("UNVERIFIED", "PUBLICLY_LISTED", "VERIFIED", "DISPUTED")
	evmNetworks         = set("ethereum-mainnet", "base", "arbitrum", "optimism", "polygon", "bsc", "gnosis", "avalanche", "scroll", "celo")
	solanaRe            = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)
	btcAddrRe           = regexp.MustCompile(`^([13][a-km-zA-HJ-NP-Z1-9]{25,34}|(?i:bc1)[a-zA-HJ-NP-Z0-9]{11,87})$`)
)

func set(v ...string) map[string]bool {
	m := make(map[string]bool, len(v))
	for _, s := range v {
		m[s] = true
	}
	return m
}

type AddressHistory struct {
	Timestamp  time.Time `json:"timestamp"`
	Actor      string    `json:"actor"`
	FromStatus string    `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	Note       string    `json:"note"`
}

type EntityAddress struct {
	ID         string           `json:"id"`
	EntityID   string           `json:"entityId"`
	Network    string           `json:"network"`
	Address    string           `json:"address"`
	Label      string           `json:"label"`
	Role       string           `json:"role"`
	Status     string           `json:"status"`
	Confidence string           `json:"confidence"`
	SourceType string           `json:"sourceType"`
	SourceRef  string           `json:"sourceRef"`
	Basis      string           `json:"basis"`
	Notes      string           `json:"notes"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
	History    []AddressHistory `json:"history,omitempty"`
}

type EntityIdentifier struct {
	ID         string    `json:"id"`
	EntityID   string    `json:"entityId"`
	Platform   string    `json:"platform"`
	Value      string    `json:"value"`
	Status     string    `json:"status"`
	SourceRef  string    `json:"sourceRef"`
	Confidence string    `json:"confidence"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"createdAt"`
}

type EntityEvidence struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entityId"`
	EvidenceID  string    `json:"evidenceId"`
	CaseID      string    `json:"caseId"`
	CaseTitle   string    `json:"caseTitle"`
	Name        string    `json:"name"`
	SHA256      string    `json:"sha256"`
	SizeBytes   int64     `json:"sizeBytes"`
	AddressID   string    `json:"addressId"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Entity struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Description  string             `json:"description"`
	Country      string             `json:"country"`
	Tags         []string           `json:"tags"`
	Status       string             `json:"status"`
	Confidence   string             `json:"confidence"`
	Notes        string             `json:"notes"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	AddressCount int                `json:"addressCount"`
	Addresses    []EntityAddress    `json:"addresses,omitempty"`
	Identifiers  []EntityIdentifier `json:"identifiers,omitempty"`
	Evidence     []EntityEvidence   `json:"evidence,omitempty"`
}

// AddressInput creates an address association. Status defaults to UNKNOWN.
type AddressInput struct {
	Network, Address, Label, Role, Status, Confidence, SourceType, SourceRef, Basis, Notes string
}

type EntityInput struct {
	Name, Type, Description, Country, Status, Confidence, Notes string
	Tags                                                         []string
	Addresses                                                    []AddressInput
}

func normTags(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || len(t) > 40 || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) == 20 {
			break
		}
	}
	return out
}

func upper(s, def string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return def
	}
	return s
}

// NormalizeAddress validates an address for its network and returns the canonical form
// (EVM addresses lower-cased; Solana addresses are case-sensitive and kept as given).
func NormalizeAddress(network, addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	switch {
	case network == "bitcoin":
		if !btcAddrRe.MatchString(addr) {
			return "", fmt.Errorf("%w: invalid Bitcoin address", ErrInvalid)
		}
		if strings.HasPrefix(strings.ToLower(addr), "bc1") {
			addr = strings.ToLower(addr)
		}
		return addr, nil
	case network == "solana":
		if !solanaRe.MatchString(addr) {
			return "", fmt.Errorf("%w: invalid Solana address", ErrInvalid)
		}
		return addr, nil
	case evmNetworks[network]:
		if !ethAddrRe.MatchString(addr) {
			return "", fmt.Errorf("%w: invalid EVM address", ErrInvalid)
		}
		return strings.ToLower(addr), nil
	}
	return "", fmt.Errorf("%w: unsupported network %q", ErrInvalid, network)
}

func validateEntityFields(e *EntityInput) error {
	e.Name = strings.TrimSpace(e.Name)
	e.Type = upper(e.Type, "OTHER")
	e.Status = upper(e.Status, "UNKNOWN")
	e.Confidence = upper(e.Confidence, "LOW")
	e.Description = strings.TrimSpace(e.Description)
	e.Country = strings.TrimSpace(e.Country)
	switch {
	case e.Name == "" || len(e.Name) > 120:
		return fmt.Errorf("%w: entity name must be 1-120 characters", ErrInvalid)
	case !entityTypes[e.Type]:
		return fmt.Errorf("%w: unknown entity type", ErrInvalid)
	case !attributionStatuses[e.Status]:
		return fmt.Errorf("%w: unknown attribution status", ErrInvalid)
	case !confidences[e.Confidence]:
		return fmt.Errorf("%w: confidence must be LOW, MEDIUM or HIGH", ErrInvalid)
	case len(e.Description) > 4000 || len(e.Notes) > 20000 || len(e.Country) > 80:
		return fmt.Errorf("%w: field too long", ErrInvalid)
	}
	return nil
}

// validateAddressClaim enforces that a strong claim is traceable to a source.
func validateAddressClaim(a *AddressInput) error {
	a.Network = strings.TrimSpace(a.Network)
	if a.Network == "" {
		a.Network = "ethereum-mainnet"
	}
	norm, err := NormalizeAddress(a.Network, a.Address)
	if err != nil {
		return err
	}
	a.Address = norm
	a.Status = upper(a.Status, "UNKNOWN")
	a.Confidence = upper(a.Confidence, "LOW")
	a.Label, a.Role = strings.TrimSpace(a.Label), strings.TrimSpace(a.Role)
	a.SourceType, a.SourceRef, a.Basis = strings.TrimSpace(a.SourceType), strings.TrimSpace(a.SourceRef), strings.TrimSpace(a.Basis)
	switch {
	case !attributionStatuses[a.Status]:
		return fmt.Errorf("%w: unknown attribution status", ErrInvalid)
	case !confidences[a.Confidence]:
		return fmt.Errorf("%w: confidence must be LOW, MEDIUM or HIGH", ErrInvalid)
	case len(a.Label) > 120 || len(a.Role) > 80 || len(a.Basis) > 4000 || len(a.SourceRef) > 2000 || len(a.Notes) > 4000:
		return fmt.Errorf("%w: field too long", ErrInvalid)
	case (a.Status == "VERIFIED" || a.Status == "PUBLICLY_REPORTED") && a.SourceRef == "":
		return fmt.Errorf("%w: %s attribution requires a source (URL or citation)", ErrInvalid, a.Status)
	case a.Status == "INVESTIGATOR_ATTRIBUTED" && a.SourceRef == "" && len(a.Basis) < 3:
		return fmt.Errorf("%w: investigator attribution requires a basis or source", ErrInvalid)
	}
	return nil
}

func (s *Store) CreateEntity(ctx context.Context, in EntityInput, actor string) (*Entity, error) {
	if err := validateEntityFields(&in); err != nil {
		return nil, err
	}
	for i := range in.Addresses {
		if err := validateAddressClaim(&in.Addresses[i]); err != nil {
			return nil, err
		}
	}
	id, t := newID("ent"), now()
	tags, _ := json.Marshal(normTags(in.Tags))
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entities(id,name,type,description,country,tags,status,confidence,notes,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, in.Name, in.Type, in.Description, in.Country, string(tags), in.Status, in.Confidence, in.Notes, ts(t), ts(t)); err != nil {
			return err
		}
		for _, a := range in.Addresses {
			if _, err := s.insertAddress(ctx, tx, id, a, actor); err != nil {
				return err
			}
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_CREATED", "entity", id, in.Name+" ["+in.Type+"/"+in.Status+"]")
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, id)
}

func (s *Store) insertAddress(ctx context.Context, tx *sql.Tx, entityID string, a AddressInput, actor string) (string, error) {
	id, t := newID("ea"), now()
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_addresses(id,entity_id,network,address,label,role,status,confidence,source_type,source_ref,basis,notes,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, entityID, a.Network, a.Address, a.Label, a.Role, a.Status, a.Confidence, a.SourceType, a.SourceRef, a.Basis, a.Notes, ts(t), ts(t)); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return "", fmt.Errorf("%w: address already associated with this entity", ErrInvalid)
		}
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_address_history(entity_address_id,ts,actor,from_status,to_status,note) VALUES(?,?,?,?,?,?)`,
		id, ts(t), actorOrDefault(actor), "", a.Status, "association created"); err != nil {
		return "", err
	}
	return id, s.appendAudit(ctx, tx, actor, "ENTITY_ADDRESS_ADDED", "entity", entityID, a.Network+" "+a.Address+" ["+a.Status+"/"+a.Confidence+"]")
}

func scanEntity(row interface{ Scan(...any) error }) (*Entity, error) {
	var e Entity
	var tags, c, u string
	if err := row.Scan(&e.ID, &e.Name, &e.Type, &e.Description, &e.Country, &tags, &e.Status, &e.Confidence, &e.Notes, &c, &u, &e.AddressCount); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tags), &e.Tags); err != nil || e.Tags == nil {
		e.Tags = []string{}
	}
	e.CreatedAt, e.UpdatedAt = parseTS(c), parseTS(u)
	return &e, nil
}

const entitySelect = `SELECT e.id,e.name,e.type,e.description,e.country,e.tags,e.status,e.confidence,e.notes,e.created_at,e.updated_at,
	(SELECT COUNT(*) FROM entity_addresses a WHERE a.entity_id=e.id) FROM entities e`

// ListEntities filters by free text (name, tag, address) and optionally by address.
func (s *Store) ListEntities(ctx context.Context, q string) ([]Entity, error) {
	q = strings.TrimSpace(q)
	query, args := entitySelect, []any{}
	if q != "" {
		like := "%" + escapeLike(strings.ToLower(q)) + "%" // % and _ in the query are literal characters
		query += ` WHERE lower(e.name) LIKE ? ESCAPE '\' OR lower(e.tags) LIKE ? ESCAPE '\' OR e.id IN (SELECT entity_id FROM entity_addresses WHERE lower(address) LIKE ? ESCAPE '\' OR lower(label) LIKE ? ESCAPE '\')`
		args = append(args, like, like, like, like)
	}
	rows, err := s.DB.QueryContext(ctx, query+` ORDER BY e.updated_at DESC LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entity{}
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// EntitiesForAddress returns the entities that have a claimed association with an address,
// each with only the matching association(s). REJECTED associations are included so the
// explorer can show that a claim was considered and ruled out.
func (s *Store) EntitiesForAddress(ctx context.Context, network, address string) ([]Entity, error) {
	if network == "" {
		network = "ethereum-mainnet"
	}
	norm, err := NormalizeAddress(network, address)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, entitySelect+` WHERE e.id IN (SELECT entity_id FROM entity_addresses WHERE network=? AND address=?) ORDER BY e.name`, network, norm)
	if err != nil {
		return nil, err
	}
	var ents []Entity
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		ents = append(ents, *e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range ents {
		addrs, err := s.entityAddresses(ctx, ents[i].ID, false)
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			if a.Network == network && a.Address == norm {
				ents[i].Addresses = append(ents[i].Addresses, a)
			}
		}
	}
	if ents == nil {
		ents = []Entity{}
	}
	return ents, nil
}

func (s *Store) entityAddresses(ctx context.Context, entityID string, withHistory bool) ([]EntityAddress, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,entity_id,network,address,label,role,status,confidence,source_type,source_ref,basis,notes,created_at,updated_at
		FROM entity_addresses WHERE entity_id=? ORDER BY network, created_at`, entityID)
	if err != nil {
		return nil, err
	}
	out := []EntityAddress{}
	for rows.Next() {
		var a EntityAddress
		var c, u string
		if err := rows.Scan(&a.ID, &a.EntityID, &a.Network, &a.Address, &a.Label, &a.Role, &a.Status, &a.Confidence, &a.SourceType, &a.SourceRef, &a.Basis, &a.Notes, &c, &u); err != nil {
			rows.Close()
			return nil, err
		}
		a.CreatedAt, a.UpdatedAt = parseTS(c), parseTS(u)
		out = append(out, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !withHistory {
		return out, nil
	}
	for i := range out {
		hr, err := s.DB.QueryContext(ctx, `SELECT ts,actor,from_status,to_status,note FROM entity_address_history WHERE entity_address_id=? ORDER BY id`, out[i].ID)
		if err != nil {
			return nil, err
		}
		for hr.Next() {
			var h AddressHistory
			var t string
			if err := hr.Scan(&t, &h.Actor, &h.FromStatus, &h.ToStatus, &h.Note); err != nil {
				hr.Close()
				return nil, err
			}
			h.Timestamp = parseTS(t)
			out[i].History = append(out[i].History, h)
		}
		hr.Close()
	}
	return out, nil
}

func (s *Store) GetEntity(ctx context.Context, id string) (*Entity, error) {
	e, err := scanEntity(s.DB.QueryRowContext(ctx, entitySelect+` WHERE e.id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if e.Addresses, err = s.entityAddresses(ctx, id, true); err != nil {
		return nil, err
	}
	e.Identifiers = []EntityIdentifier{}
	ir, err := s.DB.QueryContext(ctx, `SELECT id,entity_id,platform,value,status,source_ref,confidence,notes,created_at FROM entity_identifiers WHERE entity_id=? ORDER BY platform, created_at`, id)
	if err != nil {
		return nil, err
	}
	for ir.Next() {
		var x EntityIdentifier
		var c string
		if err := ir.Scan(&x.ID, &x.EntityID, &x.Platform, &x.Value, &x.Status, &x.SourceRef, &x.Confidence, &x.Notes, &c); err != nil {
			ir.Close()
			return nil, err
		}
		x.CreatedAt = parseTS(c)
		e.Identifiers = append(e.Identifiers, x)
	}
	ir.Close()
	e.Evidence = []EntityEvidence{}
	er, err := s.DB.QueryContext(ctx, `SELECT l.id,l.entity_id,l.evidence_id,v.case_id,c.title,v.name,v.sha256,v.size_bytes,l.address_id,l.description,l.created_at
		FROM entity_evidence l JOIN evidence v ON v.id=l.evidence_id JOIN cases c ON c.id=v.case_id WHERE l.entity_id=? ORDER BY l.created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer er.Close()
	for er.Next() {
		var x EntityEvidence
		var c string
		if err := er.Scan(&x.ID, &x.EntityID, &x.EvidenceID, &x.CaseID, &x.CaseTitle, &x.Name, &x.SHA256, &x.SizeBytes, &x.AddressID, &x.Description, &c); err != nil {
			return nil, err
		}
		x.CreatedAt = parseTS(c)
		e.Evidence = append(e.Evidence, x)
	}
	return e, er.Err()
}

// UpdateEntity replaces the editable profile fields. Addresses and identifiers are managed separately.
func (s *Store) UpdateEntity(ctx context.Context, id string, in EntityInput, actor string) (*Entity, error) {
	if err := validateEntityFields(&in); err != nil {
		return nil, err
	}
	tags, _ := json.Marshal(normTags(in.Tags))
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE entities SET name=?,type=?,description=?,country=?,tags=?,status=?,confidence=?,notes=?,updated_at=? WHERE id=?`,
			in.Name, in.Type, in.Description, in.Country, string(tags), in.Status, in.Confidence, in.Notes, ts(now()), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_UPDATED", "entity", id, in.Name+" ["+in.Type+"/"+in.Status+"/"+in.Confidence+"]")
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, id)
}

// DeleteEntity removes the entity record and its associations, identifiers and evidence links.
// It never deletes evidence files, cases, case labels or any blockchain data.
func (s *Store) DeleteEntity(ctx context.Context, id, actor string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var name string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM entities WHERE id=?`, id).Scan(&name); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM entities WHERE id=?`, id); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_DELETED", "entity", id, name)
	})
}

func (s *Store) touchEntity(ctx context.Context, tx dbtx, id string) error {
	_, err := tx.ExecContext(ctx, `UPDATE entities SET updated_at=? WHERE id=?`, ts(now()), id)
	return err
}

func (s *Store) AddEntityAddress(ctx context.Context, entityID string, a AddressInput, actor string) (*Entity, error) {
	if err := validateAddressClaim(&a); err != nil {
		return nil, err
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM entities WHERE id=?`, entityID).Scan(&one); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := s.insertAddress(ctx, tx, entityID, a, actor); err != nil {
			return err
		}
		return s.touchEntity(ctx, tx, entityID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

// AddressUpdate changes the claim on an existing association; a status change is recorded in its history.
type AddressUpdate struct {
	Label, Role, Status, Confidence, SourceType, SourceRef, Basis, Notes, ChangeNote string
}

func (s *Store) UpdateEntityAddress(ctx context.Context, addressID string, u AddressUpdate, actor string) (*Entity, error) {
	var entityID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var cur EntityAddress
		err := tx.QueryRowContext(ctx, `SELECT entity_id,network,address,status FROM entity_addresses WHERE id=?`, addressID).Scan(&cur.EntityID, &cur.Network, &cur.Address, &cur.Status)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		entityID = cur.EntityID
		in := AddressInput{Network: cur.Network, Address: cur.Address, Label: u.Label, Role: u.Role, Status: u.Status, Confidence: u.Confidence,
			SourceType: u.SourceType, SourceRef: u.SourceRef, Basis: u.Basis, Notes: u.Notes}
		if err := validateAddressClaim(&in); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE entity_addresses SET label=?,role=?,status=?,confidence=?,source_type=?,source_ref=?,basis=?,notes=?,updated_at=? WHERE id=?`,
			in.Label, in.Role, in.Status, in.Confidence, in.SourceType, in.SourceRef, in.Basis, in.Notes, ts(now()), addressID); err != nil {
			return err
		}
		if in.Status != cur.Status {
			if _, err := tx.ExecContext(ctx, `INSERT INTO entity_address_history(entity_address_id,ts,actor,from_status,to_status,note) VALUES(?,?,?,?,?,?)`,
				addressID, ts(now()), actorOrDefault(actor), cur.Status, in.Status, strings.TrimSpace(u.ChangeNote)); err != nil {
				return err
			}
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_ADDRESS_UPDATED", "entity", entityID, cur.Network+" "+cur.Address+" "+cur.Status+" -> "+in.Status)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

// RemoveEntityAddress deletes only the association (and its status history). Evidence links that
// pointed at it are detached; nothing else is affected.
func (s *Store) RemoveEntityAddress(ctx context.Context, addressID, actor string) (*Entity, error) {
	var entityID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var network, address string
		err := tx.QueryRowContext(ctx, `SELECT entity_id,network,address FROM entity_addresses WHERE id=?`, addressID).Scan(&entityID, &network, &address)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE entity_evidence SET address_id='' WHERE address_id=?`, addressID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_addresses WHERE id=?`, addressID); err != nil {
			return err
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_ADDRESS_REMOVED", "entity", entityID, network+" "+address)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

type IdentifierInput struct {
	Platform, Value, Status, SourceRef, Confidence, Notes string
}

// AddEntityIdentifier records a public identifier. A matching display name or avatar is never proof of
// identity, so VERIFIED / PUBLICLY_LISTED require a source and the default is UNVERIFIED.
func (s *Store) AddEntityIdentifier(ctx context.Context, entityID string, in IdentifierInput, actor string) (*Entity, error) {
	in.Platform = upper(in.Platform, "OTHER")
	in.Status = upper(in.Status, "UNVERIFIED")
	in.Confidence = upper(in.Confidence, "LOW")
	in.Value, in.SourceRef = strings.TrimSpace(in.Value), strings.TrimSpace(in.SourceRef)
	switch {
	case !identPlatforms[in.Platform]:
		return nil, fmt.Errorf("%w: unknown platform", ErrInvalid)
	case !identStatuses[in.Status]:
		return nil, fmt.Errorf("%w: unknown identifier status", ErrInvalid)
	case !confidences[in.Confidence]:
		return nil, fmt.Errorf("%w: confidence must be LOW, MEDIUM or HIGH", ErrInvalid)
	case in.Value == "" || len(in.Value) > 500 || len(in.Notes) > 2000 || len(in.SourceRef) > 2000:
		return nil, fmt.Errorf("%w: identifier value must be 1-500 characters", ErrInvalid)
	case (in.Status == "VERIFIED" || in.Status == "PUBLICLY_LISTED") && in.SourceRef == "":
		return nil, fmt.Errorf("%w: %s identifier requires a source", ErrInvalid, in.Status)
	}
	if in.Platform == "WEBSITE" || in.Platform == "LINKEDIN" || in.Platform == "GITHUB" || in.Platform == "FACEBOOK" {
		if u, err := url.Parse(in.Value); err == nil && u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("%w: only http(s) URLs are accepted", ErrInvalid)
		}
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM entities WHERE id=?`, entityID).Scan(&one); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO entity_identifiers(id,entity_id,platform,value,status,source_ref,confidence,notes,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
			newID("eid"), entityID, in.Platform, in.Value, in.Status, in.SourceRef, in.Confidence, in.Notes, ts(now())); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: identifier already recorded", ErrInvalid)
			}
			return err
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_IDENTIFIER_ADDED", "entity", entityID, in.Platform+" "+in.Value+" ["+in.Status+"]")
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

func (s *Store) RemoveEntityIdentifier(ctx context.Context, identifierID, actor string) (*Entity, error) {
	var entityID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var platform, value string
		err := tx.QueryRowContext(ctx, `SELECT entity_id,platform,value FROM entity_identifiers WHERE id=?`, identifierID).Scan(&entityID, &platform, &value)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_identifiers WHERE id=?`, identifierID); err != nil {
			return err
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_IDENTIFIER_REMOVED", "entity", entityID, platform+" "+value)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

// LinkEntityEvidence attaches an existing vault item (it stays in its case; nothing is copied).
func (s *Store) LinkEntityEvidence(ctx context.Context, entityID, evidenceID, addressID, description, actor string) (*Entity, error) {
	description = strings.TrimSpace(description)
	if len(description) > 2000 {
		return nil, fmt.Errorf("%w: description too long", ErrInvalid)
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM entities WHERE id=?`, entityID).Scan(&one); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM evidence WHERE id=?`, evidenceID).Scan(&one); err == sql.ErrNoRows {
			return fmt.Errorf("%w: evidence not found", ErrInvalid)
		} else if err != nil {
			return err
		}
		if addressID != "" {
			if err := tx.QueryRowContext(ctx, `SELECT 1 FROM entity_addresses WHERE id=? AND entity_id=?`, addressID, entityID).Scan(&one); err == sql.ErrNoRows {
				return fmt.Errorf("%w: address does not belong to this entity", ErrInvalid)
			} else if err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO entity_evidence(id,entity_id,evidence_id,address_id,description,created_at) VALUES(?,?,?,?,?,?)`,
			newID("eev"), entityID, evidenceID, addressID, description, ts(now())); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("%w: evidence already linked to this entity", ErrInvalid)
			}
			return err
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_EVIDENCE_LINKED", "entity", entityID, evidenceID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

func (s *Store) UnlinkEntityEvidence(ctx context.Context, linkID, actor string) (*Entity, error) {
	var entityID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var evID string
		err := tx.QueryRowContext(ctx, `SELECT entity_id,evidence_id FROM entity_evidence WHERE id=?`, linkID).Scan(&entityID, &evID)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_evidence WHERE id=?`, linkID); err != nil {
			return err
		}
		if err := s.touchEntity(ctx, tx, entityID); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "ENTITY_EVIDENCE_UNLINKED", "entity", entityID, evID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetEntity(ctx, entityID)
}

// AddressAttribution is one entity's claim on one address, as shown on graph nodes.
type AddressAttribution struct {
	EntityID   string `json:"entityId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Confidence string `json:"confidence"`
}

// LookupAddresses returns, for each requested address on a network, every entity claim on it (rejected claims
// included, so the caller can show that a claim was ruled out). Addresses with no claims are absent from the map.
func (s *Store) LookupAddresses(ctx context.Context, network string, addrs []string) (map[string][]AddressAttribution, error) {
	if network == "" {
		network = "ethereum-mainnet"
	}
	if len(addrs) > 1000 {
		return nil, fmt.Errorf("%w: at most 1000 addresses per lookup", ErrInvalid)
	}
	out := map[string][]AddressAttribution{}
	if len(addrs) == 0 {
		return out, nil
	}
	args := []any{network}
	ph := make([]string, 0, len(addrs))
	for _, a := range addrs {
		norm, err := NormalizeAddress(network, a)
		if err != nil {
			continue // unparseable addresses simply have no claims
		}
		ph = append(ph, "?")
		args = append(args, norm)
	}
	if len(ph) == 0 {
		return out, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT a.address,e.id,e.name,e.type,a.status,a.confidence FROM entity_addresses a JOIN entities e ON e.id=a.entity_id
		WHERE a.network=? AND a.address IN (`+strings.Join(ph, ",")+`) ORDER BY e.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var addr string
		var x AddressAttribution
		if err := rows.Scan(&addr, &x.EntityID, &x.Name, &x.Type, &x.Status, &x.Confidence); err != nil {
			return nil, err
		}
		out[addr] = append(out[addr], x)
	}
	return out, rows.Err()
}
