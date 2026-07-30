// Este fichero prueba la única escritura, incluidos conflictos y repetición.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProposalStoreIsAtomicIdempotentAndRevisioned(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	store.now = func() time.Time {
		return time.Date(2026, 7, 30, 1, 2, 3, 0, time.UTC)
	}
	request := validProposalRequest(item, 0, strings.Repeat("a", 64))
	first, err := store.submit(item, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replay || first.Proposal.ProposalRevision != 1 ||
		first.Proposal.ExpectedRevision != 0 || first.Proposal.CreatedAt != "2026-07-30T01:02:03Z" {
		t.Fatalf("primera propuesta incorrecta: %#v", first)
	}
	replay, err := store.submit(item, request)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replay || replay.Proposal.ProposalRef != first.Proposal.ProposalRef {
		t.Fatalf("la repetición no devolvió el mismo hecho: %#v", replay)
	}
	changed := request
	changed.Reason = "Una razón deliberadamente distinta y suficientemente larga."
	if _, err := store.submit(item, changed); !errors.Is(err, errIdempotencyConflict) {
		t.Fatalf("se esperaba conflicto idempotente; recibido %v", err)
	}
	proposals, err := readProposals(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 || proposalFileMode(t, filePath) != 0o600 {
		t.Fatalf("historial o permisos incorrectos: len=%d modo=%o", len(proposals), proposalFileMode(t, filePath))
	}
}

func TestProposalStoreSerializesConcurrentExpectedRevision(t *testing.T) {
	_, item := testInventory(t)
	store := newProposalStore(filepath.Join(t.TempDir(), "propuestas.jsonl"))
	if _, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("0", 64))); err != nil {
		t.Fatal(err)
	}
	requests := []proposalRequest{
		validProposalRequest(item, 1, strings.Repeat("1", 64)),
		validProposalRequest(item, 1, strings.Repeat("2", 64)),
	}
	var wait sync.WaitGroup
	results := make(chan error, len(requests))
	for _, request := range requests {
		wait.Add(1)
		go func(request proposalRequest) {
			defer wait.Done()
			_, err := store.submit(item, request)
			results <- err
		}(request)
	}
	wait.Wait()
	close(results)
	var successes, conflicts int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, errRevisionConflict):
			conflicts++
		default:
			t.Fatalf("resultado concurrente inesperado: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrencia no serializada: éxitos=%d conflictos=%d", successes, conflicts)
	}
	proposals, err := store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 2 || latestRevision(proposals, item.ID) != 2 {
		t.Fatalf("historial concurrente incorrecto: %#v", proposals)
	}
}

func TestProposalValidationRejectsIncompleteOrNonCompliantAdmission(t *testing.T) {
	_, item := testInventory(t)
	base := validProposalRequest(item, 0, strings.Repeat("a", 64))
	base.Disposition = dispositionAdmitted
	base.RuleCompliance["i18n"] = false
	if err := validateProposalRequest(item, base); !errors.Is(err, errInvalidProposal) {
		t.Fatalf("se admitió una propuesta incompatible: %v", err)
	}
	base.Disposition = dispositionStudy
	base.RuleNotes = ""
	if err := validateProposalRequest(item, base); !errors.Is(err, errInvalidProposal) {
		t.Fatalf("se aceptó una duda sin explicación: %v", err)
	}
	delete(base.RuleCompliance, "i18n")
	base.RuleNotes = "Existe una duda de internacionalización que debe estudiarse."
	if err := validateProposalRequest(item, base); !errors.Is(err, errInvalidProposal) {
		t.Fatalf("se aceptó un conjunto de reglas incompleto: %v", err)
	}
}

func TestProposalStoreRejectsSymlinkAndMalformedHistory(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "real.jsonl")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "propuestas.jsonl")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readProposals(link); err == nil {
		t.Fatal("se siguió un enlace simbólico de propuestas")
	}

	malformed := filepath.Join(directory, "malformado.jsonl")
	if err := os.WriteFile(malformed, []byte("{no-json}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readProposals(malformed); err == nil {
		t.Fatal("se aceptó un historial malformado")
	}
}

func TestProposalStoreRejectsTamperedCanonicalFields(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	if _, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("9", 64))); err != nil {
		t.Fatal(err)
	}
	proposals, err := readProposals(filePath)
	if err != nil {
		t.Fatal(err)
	}
	proposals[0].Disposition = disposition("inventada")
	if err := writeProposals(filePath, proposals); err != nil {
		t.Fatal(err)
	}
	if _, err := readProposals(filePath); err == nil {
		t.Fatal("se aceptó una disposición manipulada sin actualizar sus huellas")
	}
}

func TestInventoryRevisionDetectsExternalChange(t *testing.T) {
	inventory, _ := testInventory(t)
	if err := inventory.verifyUnchanged(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory.path, []byte("{\"id\":\"otro\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := inventory.verifyUnchanged(); !errors.Is(err, errInventoryChanged) {
		t.Fatalf("no se detectó el cambio del inventario: %v", err)
	}
}
