package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

const (
	addrA = "0xabcdef1111111111111111111111111111111111"
	addrB = "0x2222222222222222222222222222222222222222"
	solA  = "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM"
)

func TestEntityMultiWalletAndAttribution(t *testing.T) {
	ctx, s := context.Background(), newStore(t)

	// An entity needs no wallet at all.
	bare, err := s.CreateEntity(ctx, EntityInput{Name: " Acme Corp ", Type: "company", Tags: []string{"OTC", "otc", " fund "}}, "")
	if err != nil || bare.Name != "Acme Corp" || bare.Type != "COMPANY" || bare.Status != "UNKNOWN" || len(bare.Tags) != 2 || bare.AddressCount != 0 {
		t.Fatalf("bare entity: %+v %v", bare, err)
	}

	// Strong claims must cite a source.
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Address: addrA, Status: "VERIFIED"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("VERIFIED without source must be rejected: %v", err)
	}
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Address: addrA, Status: "INVESTIGATOR_ATTRIBUTED"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("investigator attribution without basis must be rejected: %v", err)
	}
	// A suspicion may be recorded without a source.
	e, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Address: "0x" + strings.ToUpper(addrA[2:]), Status: "suspected"}, "")
	if err != nil || e.AddressCount != 1 || e.Addresses[0].Address != addrA || e.Addresses[0].Network != "ethereum-mainnet" {
		t.Fatalf("suspected: %+v %v", e, err)
	}
	// Same address twice, invalid addresses and unknown networks are refused.
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Address: addrA}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate association: %v", err)
	}
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Address: "0x123"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad address accepted")
	}
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Network: "dogechain", Address: addrB}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("unknown network accepted")
	}
	// Multiple chains, including Solana.
	if _, err := s.AddEntityAddress(ctx, bare.ID, AddressInput{Network: "base", Address: addrA, Status: "PUBLICLY_REPORTED", SourceRef: "https://example.org/docs"}, ""); err != nil {
		t.Fatal(err)
	}
	e, err = s.AddEntityAddress(ctx, bare.ID, AddressInput{Network: "solana", Address: solA, Status: "UNKNOWN"}, "")
	if err != nil || e.AddressCount != 3 {
		t.Fatalf("multi-chain: %+v %v", e, err)
	}

	// Status changes are recorded in history; upgrading to VERIFIED needs a source.
	id := ""
	for _, a := range e.Addresses {
		if a.Network == "ethereum-mainnet" {
			id = a.ID
		}
	}
	if id == "" {
		t.Fatal("ethereum association missing")
	}
	if _, err := s.UpdateEntityAddress(ctx, id, AddressUpdate{Status: "VERIFIED"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("upgrade without source accepted")
	}
	e, err = s.UpdateEntityAddress(ctx, id, AddressUpdate{Status: "VERIFIED", Confidence: "HIGH", SourceRef: "https://example.org/proof", ChangeNote: "signed message"}, "alice")
	if err != nil {
		t.Fatal(err)
	}
	var found *EntityAddress
	for i := range e.Addresses {
		if e.Addresses[i].ID == id {
			found = &e.Addresses[i]
		}
	}
	if found == nil || found.Status != "VERIFIED" || len(found.History) != 2 || found.History[1].FromStatus != "SUSPECTED" || found.History[1].Actor != "alice" {
		t.Fatalf("history: %+v", found)
	}

	// Lookup by address finds the entity with only the matching association(s).
	hits, err := s.EntitiesForAddress(ctx, "ethereum-mainnet", "0x"+strings.ToUpper(addrA[2:]))
	if err != nil || len(hits) != 1 || len(hits[0].Addresses) != 1 || hits[0].Addresses[0].Network != "ethereum-mainnet" {
		t.Fatalf("lookup: %+v %v", hits, err)
	}
	if hits, _ = s.EntitiesForAddress(ctx, "ethereum-mainnet", addrB); len(hits) != 0 {
		t.Fatal("unassociated address must not match")
	}

	// Removing an association leaves the entity and its other wallets intact.
	e, err = s.RemoveEntityAddress(ctx, id, "")
	if err != nil || e.AddressCount != 2 {
		t.Fatalf("remove: %+v %v", e, err)
	}
	if _, err := s.RemoveEntityAddress(ctx, id, ""); !errors.Is(err, ErrNotFound) {
		t.Fatal("second remove must be not found")
	}
}

func TestEntityIdentifiersAreNotProofOfIdentity(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	e, _ := s.CreateEntity(ctx, EntityInput{Name: "Someone", Type: "INDIVIDUAL"}, "")
	// Default status is UNVERIFIED.
	e, err := s.AddEntityIdentifier(ctx, e.ID, IdentifierInput{Platform: "x", Value: "@someone"}, "")
	if err != nil || e.Identifiers[0].Status != "UNVERIFIED" {
		t.Fatalf("default status: %+v %v", e, err)
	}
	if _, err := s.AddEntityIdentifier(ctx, e.ID, IdentifierInput{Platform: "x", Value: "@other", Status: "VERIFIED"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("VERIFIED identifier without source accepted")
	}
	if _, err := s.AddEntityIdentifier(ctx, e.ID, IdentifierInput{Platform: "WEBSITE", Value: "javascript:alert(1)"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("non-http URL accepted")
	}
	if _, err := s.AddEntityIdentifier(ctx, e.ID, IdentifierInput{Platform: "x", Value: "@someone"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate identifier accepted")
	}
}

func TestEntityEvidenceLinkAndDelete(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	c, _ := s.CreateCase(ctx, "Case", "", "")
	file := t.TempDir() + "/proof.txt"
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	ev, err := s.ImportEvidence(ctx, c.ID, file, "")
	if err != nil {
		t.Fatal(err)
	}
	e, _ := s.CreateEntity(ctx, EntityInput{Name: "Linked", Addresses: []AddressInput{{Address: addrA, Status: "SUSPECTED"}}}, "")
	e, err = s.LinkEntityEvidence(ctx, e.ID, ev.ID, e.Addresses[0].ID, "screenshot", "")
	if err != nil || len(e.Evidence) != 1 || e.Evidence[0].SHA256 != helloSHA || e.Evidence[0].CaseTitle != "Case" {
		t.Fatalf("link: %+v %v", e, err)
	}
	if _, err := s.LinkEntityEvidence(ctx, e.ID, ev.ID, "", "", ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate link accepted")
	}
	if _, err := s.LinkEntityEvidence(ctx, e.ID, "ev_missing", "", "", ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("missing evidence accepted")
	}
	// Deleting the entity removes the record but never the evidence or the case.
	if err := s.DeleteEntity(ctx, e.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEntity(ctx, e.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("entity still present")
	}
	if list, err := s.ListEvidence(ctx, c.ID); err != nil || len(list) != 1 {
		t.Fatalf("evidence must survive entity deletion: %v %v", list, err)
	}
}

func TestV1LabelsMigrateConservatively(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	// Simulate v1 data by rolling the schema version back, inserting labels, and re-running v3.
	c, _ := s.CreateCase(ctx, "Old", "", "")
	if _, err := s.AddLabel(ctx, Label{CaseID: c.ID, Address: addrA, Name: "Exchange X", Source: "VERIFIED_SOURCE", Confidence: "HIGH", Basis: "published list"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddLabel(ctx, Label{CaseID: c.ID, Address: addrB, Name: "exchange x", Source: "ANALYST_ASSERTION", Confidence: "LOW", Basis: "seen in flow"}, ""); err != nil {
		t.Fatal(err)
	}
	// Roll the schema back to v2 (labels only) so migration v3 runs over real v1-style label data.
	// Every table created by v3 or later must be dropped here, children first, and their versions forgotten;
	// otherwise migrate() sees a newer version and has nothing to do.
	for _, tbl := range []string{"board_link_evidence", "board_links", "board_nodes", "graph_snapshots", "entity_evidence", "entity_identifiers", "entity_address_history", "entity_addresses", "entities"} {
		if _, err := s.DB.ExecContext(ctx, `DROP TABLE `+tbl); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version > 2`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(ctx); err != nil {
		t.Fatalf("re-running migrations v3+ over existing labels: %v", err)
	}
	list, err := s.ListEntities(ctx, "")
	if err != nil || len(list) != 1 || list[0].AddressCount != 2 || list[0].Status != "UNKNOWN" {
		t.Fatalf("migrated entities: %+v %v", list, err)
	}
	e, _ := s.GetEntity(ctx, list[0].ID)
	got := map[string]string{}
	for _, a := range e.Addresses {
		got[a.Address] = a.Status
	}
	// A "verified source" label is carried forward only as publicly reported, never as VERIFIED.
	if got[addrA] != "PUBLICLY_REPORTED" || got[addrB] != "INVESTIGATOR_ATTRIBUTED" {
		t.Fatalf("status mapping: %+v", got)
	}
	if labels, _ := s.ListLabels(ctx, "", ""); len(labels) != 2 {
		t.Fatal("original labels must be preserved")
	}
}

func TestLookupAddressesAndGraphSnapshots(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	e, _ := s.CreateEntity(ctx, EntityInput{Name: "Mixer-ish", Addresses: []AddressInput{
		{Address: addrA, Status: "SUSPECTED"},
		{Address: addrB, Status: "REJECTED", Basis: "ruled out"},
	}}, "")
	got, err := s.LookupAddresses(ctx, "", []string{"0x" + strings.ToUpper(addrA[2:]), addrB, "0xnotanaddress", "0x3333333333333333333333333333333333333333"})
	if err != nil || len(got) != 2 || got[addrA][0].EntityID != e.ID || got[addrB][0].Status != "REJECTED" {
		t.Fatalf("lookup: %+v %v", got, err)
	}

	data := json.RawMessage(`{"nodes":[{"id":"a","x":1,"y":2}],"edges":[]}`)
	if _, err := s.SaveGraphSnapshot(ctx, " ", []string{addrA}, 1, 0, data, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("blank name accepted")
	}
	if _, err := s.SaveGraphSnapshot(ctx, "x", nil, 1, 0, data, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("no seeds accepted")
	}
	if _, err := s.SaveGraphSnapshot(ctx, "x", []string{addrA}, 1, 0, json.RawMessage(`{bad`), ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid JSON accepted")
	}
	g, err := s.SaveGraphSnapshot(ctx, "First look", []string{addrA, addrB}, 1, 0, data, "alice")
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.ListGraphSnapshots(ctx)
	if err != nil || len(list) != 1 || list[0].Data != nil || len(list[0].Seeds) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	full, err := s.GetGraphSnapshot(ctx, g.ID)
	if err != nil || string(full.Data) != string(data) {
		t.Fatalf("get: %+v %v", full, err)
	}
	if err := s.DeleteGraphSnapshot(ctx, g.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetGraphSnapshot(ctx, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("snapshot still present")
	}
}

func TestMoreNetworksAndLocalSearch(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	const btc = "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"
	e, err := s.CreateEntity(ctx, EntityInput{Name: "Cold Storage 100%", Addresses: []AddressInput{
		{Network: "bitcoin", Address: strings.ToUpper(btc), Status: "SUSPECTED"},
		{Network: "gnosis", Address: addrA, Status: "SUSPECTED"},
		{Network: "solana", Address: solA, Status: "SUSPECTED"},
	}}, "")
	if err != nil || e.AddressCount != 3 {
		t.Fatalf("multi-network entity: %+v %v", e, err)
	}
	for _, a := range e.Addresses {
		if a.Network == "bitcoin" && a.Address != btc {
			t.Fatalf("bech32 not normalised: %s", a.Address)
		}
	}
	if _, err := s.AddEntityAddress(ctx, e.ID, AddressInput{Network: "bitcoin", Address: addrA}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("EVM address accepted as bitcoin")
	}

	c, _ := s.CreateCase(ctx, "Cold storage case", "", "")
	if hits, err := s.SearchLocal(ctx, "x"); err != nil || len(hits) != 0 {
		t.Fatalf("too-short query: %+v %v", hits, err)
	}
	hits, err := s.SearchLocal(ctx, "cold storage")
	kinds := map[string]bool{}
	for _, h := range hits {
		kinds[h.Kind] = true
		if h.Kind == "CASE" && h.CaseID != c.ID {
			t.Fatalf("case hit must carry its id: %+v", h)
		}
	}
	if err != nil || !kinds["ENTITY"] || !kinds["CASE"] {
		t.Fatalf("search: %+v %v", hits, err)
	}
	// LIKE wildcards in the query are literal, so "%" does not match everything.
	if hits, _ := s.SearchLocal(ctx, "zz%"); len(hits) != 0 {
		t.Fatalf("wildcard leaked: %+v", hits)
	}
}
