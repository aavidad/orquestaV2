// Estas pruebas fijan el límite simétrico de línea y la publicación durable.
// No crean otra ruta de escritura ni relajan la validación semántica.
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func TestProposalStoreAcceptsExactReferenceAndRevisionBoundaries(t *testing.T) {
	_, source := testInventory(t)
	tests := []struct {
		name, itemRef, itemRevision, idempotencyKey string
	}{
		{
			name: "identificador de elemento de 200 bytes", itemRef: strings.Repeat("i", 200),
			itemRevision: source.Revision, idempotencyKey: strings.Repeat("k", 64),
		},
		{
			name: "clave idempotente de 32 bytes", itemRef: source.ID,
			itemRevision: source.Revision, idempotencyKey: strings.Repeat("k", 32),
		},
		{
			name: "clave idempotente de 200 bytes", itemRef: source.ID,
			itemRevision: source.Revision, idempotencyKey: strings.Repeat("k", 200),
		},
		{
			name: "revisión con 64 hexadecimales minúsculos", itemRef: source.ID,
			itemRevision: "sha256:" + strings.Repeat("a", 64), idempotencyKey: strings.Repeat("k", 64),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := source
			item.ID, item.Revision = test.itemRef, test.itemRevision
			request := validProposalRequest(item, 0, test.idempotencyKey)
			store := newProposalStore(filepath.Join(t.TempDir(), "propuestas.jsonl"))
			result, err := store.submit(item, request)
			if err != nil {
				t.Fatal(err)
			}
			proposals, err := store.list()
			if err != nil || len(proposals) != 1 ||
				proposals[0].ProposalRef != result.Proposal.ProposalRef {
				t.Fatalf("la frontera válida no sobrevivió a la relectura: %#v %v", proposals, err)
			}
		})
	}
}

func TestProposalCodecAcceptsExactLineLimitAndRejectsNextByte(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	result, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("4", 64)))
	if err != nil {
		t.Fatal(err)
	}
	exact := proposalWithLineLength(t, result.Proposal, maxProposalLineBytes)
	if err := writeProposals(filePath, []proposal{exact}); err != nil {
		t.Fatalf("el escritor rechazó el límite exacto: %v", err)
	}
	if proposals, err := readProposals(filePath); err != nil || len(proposals) != 1 {
		t.Fatalf("el lector rechazó el límite exacto: propuestas=%d error=%v", len(proposals), err)
	}
	oversized := proposalWithLineLength(t, result.Proposal, maxProposalLineBytes+1)
	line, err := encodeCanonicalProposal(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readProposals(filePath); err == nil {
		t.Fatal("el lector aceptó una línea un byte mayor que el límite")
	}
}

func TestProposalWriterRejectsOversizeWithoutCorruptingHistory(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	result, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("5", 64)))
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	oversized := proposalWithLineLength(t, result.Proposal, maxProposalLineBytes+1)
	if err := writeProposals(filePath, []proposal{oversized}); err == nil {
		t.Fatal("el escritor aceptó una línea un byte mayor que el límite")
	}
	after, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("el rechazo de tamaño corrompió el historial publicado")
	}
}

func TestProposalStoreRecoversExactPublicationAfterDirectorySyncFailure(t *testing.T) {
	_, item := testInventory(t)
	directory := filepath.Join(t.TempDir(), "solo-escritura")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(directory, 0o700)
	})
	if err := os.Chmod(directory, 0o300); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(directory, "propuestas.jsonl")
	store := newProposalStore(filePath)
	request := validProposalRequest(item, 0, strings.Repeat("6", 64))
	if _, err := store.submit(item, request); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("submit no informó el fallo real de fsync del directorio: %v", err)
	}
	if info, err := os.Lstat(filePath); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("el rename no publicó el historial antes del fallo de fsync: info=%v error=%v", info, err)
	}
	lock, err := os.OpenFile(filePath+".lock", os.O_RDWR|syscall.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		t.Fatalf("submit no liberó el candado tras el fallo de fsync: %v", err)
	}
	unlockFile(lock)

	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	persisted, err := readProposals(filePath)
	if err != nil || len(persisted) != 1 || persisted[0].ProposalRevision != 1 {
		t.Fatalf("la publicación ambigua no se recuperó: propuestas=%#v error=%v", persisted, err)
	}
	restarted := newProposalStore(filePath)
	replay, replayErr := restarted.submit(item, request)
	if replayErr != nil || !replay.Replay || !reflect.DeepEqual(replay.Proposal, persisted[0]) {
		t.Fatalf("el reinicio no recuperó la publicación exacta: resultado=%#v error=%v", replay, replayErr)
	}
	if proposals, readErr := restarted.list(); readErr != nil || len(proposals) != 1 ||
		latestRevision(proposals, item.ID) != 1 || !reflect.DeepEqual(proposals[0], persisted[0]) {
		t.Fatalf("la repetición alteró el historial: propuestas=%#v error=%v", proposals, readErr)
	}
}

func proposalWithLineLength(t *testing.T, source proposal, desired int) proposal {
	t.Helper()
	result := source
	result.Reason = strings.Repeat("r", 10)
	refreshProposalIdentity(&result)
	line, err := encodeCanonicalProposal(result)
	if err != nil {
		t.Fatal(err)
	}
	padding := desired - len(line)
	if padding < 0 {
		t.Fatalf("el registro base supera el tamaño deseado: base=%d deseado=%d", len(line), desired)
	}
	result.Reason += strings.Repeat("r", padding)
	refreshProposalIdentity(&result)
	line, err = encodeCanonicalProposal(result)
	if err != nil || len(line) != desired {
		t.Fatalf("no se construyó la frontera exacta: longitud=%d error=%v", len(line), err)
	}
	return result
}

func refreshProposalIdentity(item *proposal) {
	request := storedProposalRequest(*item)
	item.RequestDigest = digestRequest(request)
	item.ProposalRef = proposalReference(item.ItemRef, item.ProposalRevision, item.RequestDigest)
}
