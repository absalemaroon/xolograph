package store

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestBoardFactsNeedSupport(t *testing.T) {
	ctx, s := context.Background(), newStore(t)
	c, _ := s.CreateCase(ctx, "Board", "", "")
	other, _ := s.CreateCase(ctx, "Other", "", "")
	file := t.TempDir() + "/doc.txt"
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	ev, err := s.ImportEvidence(ctx, c.ID, file, "")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.ImportEvidence(ctx, other.ID, file, "")
	if err != nil {
		t.Fatal(err)
	}
	ent, _ := s.CreateEntity(ctx, EntityInput{Name: "Acme"}, "")

	b, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "entity", RefID: ent.ID, X: 1, Y: 2}, "")
	if err != nil || len(b.Nodes) != 1 || b.Nodes[0].Title != "Acme" || b.Nodes[0].LiveName != "Acme" {
		t.Fatalf("entity node: %+v %v", b, err)
	}
	if _, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "ENTITY", RefID: ent.ID}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate entity node accepted")
	}
	if _, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "ENTITY", RefID: "ent_missing"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("missing entity accepted")
	}
	if _, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "EVIDENCE", RefID: foreign.ID}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("evidence from another case accepted")
	}
	if _, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "NOTE"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("untitled note accepted")
	}
	if _, err := s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "WALLET", RefID: "0x12"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad wallet accepted")
	}
	b, _ = s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "WALLET", RefID: addrA}, "")
	b, err = s.AddBoardNode(ctx, c.ID, BoardNodeInput{Kind: "NOTE", Title: "Observation", Body: "text"}, "")
	if err != nil || len(b.Nodes) != 3 {
		t.Fatalf("nodes: %+v %v", b, err)
	}
	byKind := map[string]string{}
	for _, n := range b.Nodes {
		byKind[n.Kind] = n.ID
	}
	entN, walN, noteN := byKind["ENTITY"], byKind["WALLET"], byKind["NOTE"]

	// A FACT with no evidence and no source is refused; a HYPOTHESIS is fine.
	if _, err := s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: entN, TargetID: walN, Relation: "controls", Claim: "FACT"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("unsupported FACT accepted")
	}
	if _, err := s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: entN, TargetID: entN, Relation: "x", Claim: "HYPOTHESIS"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("self link accepted")
	}
	if _, err := s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: entN, TargetID: walN, Relation: "controls", Claim: "FACT", EvidenceIDs: []string{foreign.ID}}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("foreign evidence accepted")
	}
	b, err = s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: entN, TargetID: walN, Relation: "may control", Claim: "hypothesis"}, "")
	if err != nil || len(b.Links) != 1 || b.Links[0].Claim != "HYPOTHESIS" || b.Links[0].Confidence != "LOW" {
		t.Fatalf("hypothesis: %+v %v", b, err)
	}
	hyp := b.Links[0].ID
	// Promoting to FACT needs support first.
	if _, err := s.UpdateBoardLink(ctx, hyp, "controls", "FACT", "HIGH", "", "", ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("promotion without support accepted")
	}
	b, err = s.AddBoardLinkEvidence(ctx, hyp, ev.ID, "signed message", "")
	if err != nil || len(b.Links[0].Evidence) != 1 || b.Links[0].Evidence[0].SHA256 != helloSHA {
		t.Fatalf("link evidence: %+v %v", b, err)
	}
	if _, err = s.UpdateBoardLink(ctx, hyp, "controls", "FACT", "HIGH", "", "", ""); err != nil {
		t.Fatalf("promotion with evidence: %v", err)
	}
	// The last support cannot be removed from a FACT without a source.
	b, _ = s.GetBoard(ctx, c.ID)
	if _, err := s.RemoveBoardLinkEvidence(ctx, b.Links[0].Evidence[0].ID, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("removed the last support of a FACT")
	}
	// A FACT can instead rest on a stated source.
	b, err = s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: walN, TargetID: noteN, Relation: "documented in", Claim: "FACT", SourceRef: "https://example.org/report"}, "")
	if err != nil || len(b.Links) != 2 {
		t.Fatalf("sourced fact: %+v %v", b, err)
	}
	if _, err := s.AddBoardLink(ctx, c.ID, BoardLinkInput{SourceID: walN, TargetID: noteN, Relation: "documented in", Claim: "HYPOTHESIS"}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate link accepted")
	}

	// Moving nodes persists layout; deleting a node removes its links but nothing it references.
	if err := s.MoveBoardNodes(ctx, c.ID, []string{entN}, []float64{10}, []float64{-5}); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveBoardNodes(ctx, c.ID, []string{entN}, []float64{1e12}, []float64{0}); !errors.Is(err, ErrInvalid) {
		t.Fatal("absurd coordinates accepted")
	}
	b, err = s.DeleteBoardNode(ctx, walN, "")
	if err != nil || len(b.Nodes) != 2 || len(b.Links) != 0 {
		t.Fatalf("delete node: %+v %v", b, err)
	}
	for _, n := range b.Nodes {
		if n.ID == entN && n.X != 10 {
			t.Fatalf("position not saved: %+v", n)
		}
	}
	if got, _ := s.GetEntity(ctx, ent.ID); got == nil {
		t.Fatal("entity must survive board deletion")
	}
	if list, _ := s.ListEvidence(ctx, c.ID); len(list) != 1 {
		t.Fatal("evidence must survive board deletion")
	}
}
