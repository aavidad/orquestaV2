// Estas pruebas demuestran que el historial rechaza representaciones JSONL
// ambiguas sin alterar el formato producido por el escritor.
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProposalCodecRejectsAmbiguousOrNonCanonicalRecords(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	store.now = func() time.Time {
		return time.Date(2026, 7, 30, 1, 2, 3, 0, time.UTC)
	}
	if _, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("7", 64))); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	line := bytes.TrimSuffix(content, []byte{'\n'})
	withLF := func(raw []byte) []byte {
		return append(append([]byte(nil), raw...), '\n')
	}
	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"clave duplicada", func(raw []byte) []byte {
			return withLF(bytes.Replace(
				raw,
				[]byte(`"schema_version":1`),
				[]byte(`"schema_version":1,"schema_version":1`),
				1,
			))
		}},
		{"campo desconocido", func(raw []byte) []byte {
			changed := append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"campo_desconocido":true}`)...)
			return withLF(changed)
		}},
		{"JSON posterior", func(raw []byte) []byte {
			return withLF(append(append([]byte(nil), raw...), []byte(` {}`)...))
		}},
		{"representación no canónica", func(raw []byte) []byte {
			return withLF(append([]byte(" "), raw...))
		}},
		{"fecha equivalente no canónica", func(raw []byte) []byte {
			return withLF(bytes.Replace(
				raw,
				[]byte(`"created_at":"2026-07-30T01:02:03Z"`),
				[]byte(`"created_at":"2026-07-30T01:02:03+00:00"`),
				1,
			))
		}},
		{"LF final ausente", func(raw []byte) []byte {
			return append([]byte(nil), raw...)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := test.mutate(line)
			if bytes.Equal(mutated, content) {
				t.Fatal("la mutación no cambió el historial")
			}
			if err := os.WriteFile(filePath, mutated, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readProposals(filePath); err == nil {
				t.Fatal("el historial ambiguo o no canónico fue aceptado")
			}
		})
	}
}

func TestProposalCodecPreservesWriterFormat(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	request := validProposalRequest(item, 0, strings.Repeat("8", 64))
	request.Reason = "La razón conserva <etiquetas> y no cambia el formato válido."
	if _, err := store.submit(item, request); err != nil {
		t.Fatal(err)
	}
	proposals, err := readProposals(filePath)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 || !bytes.Contains(content, []byte("<etiquetas>")) ||
		bytes.Contains(content, []byte(`\u003c`)) {
		t.Fatalf("el formato válido cambió: propuestas=%d contenido=%q", len(proposals), content)
	}
}

func TestProposalStoreNormalizesBeforeValidationDigestAndPersistence(t *testing.T) {
	_, item := testInventory(t)
	filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
	store := newProposalStore(filePath)
	request := validProposalRequest(item, 0, strings.Repeat("9", 64))
	request.Reason = " \n" + request.Reason + "\t "
	request.FoundedSolution = "\t " + request.FoundedSolution + "\n"
	request.RuleNotes = "  " + request.RuleNotes + "\r\n"
	request.ActorRef = "\n " + request.ActorRef + "\t"
	request.ProjectRef = " \t" + request.ProjectRef + "\n"
	request.IdempotencyKey = "  " + request.IdempotencyKey + "\t"

	created, err := store.submit(item, request)
	if err != nil {
		t.Fatal(err)
	}
	normalized := request
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.FoundedSolution = strings.TrimSpace(normalized.FoundedSolution)
	normalized.RuleNotes = strings.TrimSpace(normalized.RuleNotes)
	normalized.ActorRef = strings.TrimSpace(normalized.ActorRef)
	normalized.ProjectRef = strings.TrimSpace(normalized.ProjectRef)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	if got := storedProposalRequest(created.Proposal); !equalProposalRequests(got, normalized) {
		t.Fatalf("la solicitud persistida no es la normalizada: %#v", got)
	}
	if created.Proposal.RequestDigest != digestRequest(normalized) {
		t.Fatal("el resumen no corresponde a la solicitud persistida")
	}

	proposals, err := readProposals(filePath)
	if err != nil {
		t.Fatalf("la propuesta válida no se pudo releer: %v", err)
	}
	if len(proposals) != 1 || proposals[0].ProposalRef != created.Proposal.ProposalRef {
		t.Fatalf("relectura inesperada: %#v", proposals)
	}

	replay, err := store.submit(item, request)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replay || replay.Proposal.ProposalRef != created.Proposal.ProposalRef {
		t.Fatalf("la repetición con espacios no fue idempotente: %#v", replay)
	}
	canonicalReplay, err := store.submit(item, normalized)
	if err != nil {
		t.Fatal(err)
	}
	if !canonicalReplay.Replay || canonicalReplay.Proposal.ProposalRef != created.Proposal.ProposalRef {
		t.Fatalf("la repetición normalizada no fue idempotente: %#v", canonicalReplay)
	}
}

func TestProposalCodecRejectsSemanticallyInvalidCanonicalHistory(t *testing.T) {
	_, item := testInventory(t)
	store := newProposalStore(filepath.Join(t.TempDir(), "origen.jsonl"))
	created, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*proposal)
	}{
		{"historial acolchado", func(item *proposal) {
			item.Reason = " " + item.Reason + "\t"
		}},
		{"clave histórica acolchada", func(item *proposal) {
			item.IdempotencyKey = " " + item.IdempotencyKey + "\t"
		}},
		{"identificador de elemento vacío", func(item *proposal) {
			item.ItemRef = ""
		}},
		{"identificador de elemento con espacio exterior", func(item *proposal) {
			item.ItemRef = " " + item.ItemRef
		}},
		{"identificador de elemento con control", func(item *proposal) {
			item.ItemRef = "conducta:\x00uno"
		}},
		{"identificador de elemento excesivo", func(item *proposal) {
			item.ItemRef = strings.Repeat("i", maxProposalReferenceBytes+1)
		}},
		{"revisión de elemento vacía", func(item *proposal) {
			item.ItemRevision = ""
		}},
		{"revisión de elemento malformada", func(item *proposal) {
			item.ItemRevision = "sha256:" + strings.Repeat("A", 64)
		}},
		{"revisión de elemento con 63 hexadecimales", func(item *proposal) {
			item.ItemRevision = "sha256:" + strings.Repeat("a", 63)
		}},
		{"revisión de elemento con 65 hexadecimales", func(item *proposal) {
			item.ItemRevision = "sha256:" + strings.Repeat("a", 65)
		}},
		{"revisión de elemento con prefijo incorrecto", func(item *proposal) {
			item.ItemRevision = "sha512:" + strings.Repeat("a", 64)
		}},
		{"actor inválido", func(item *proposal) {
			item.ActorRef = "actor no opaco"
		}},
		{"actor con control Unicode", func(item *proposal) {
			item.ActorRef = "actor:\x00no-opaco"
		}},
		{"proyecto inválido", func(item *proposal) {
			item.ProjectRef = "project\tno-opaco"
		}},
		{"proyecto con control Unicode", func(item *proposal) {
			item.ProjectRef = "project:\x01no-opaco"
		}},
		{"clave con treinta y un caracteres", func(item *proposal) {
			item.IdempotencyKey = strings.Repeat("k", 31)
		}},
		{"clave de treinta y dos espacios", func(item *proposal) {
			item.IdempotencyKey = strings.Repeat(" ", 32)
		}},
		{"clave con espacio interno", func(item *proposal) {
			item.IdempotencyKey = strings.Repeat("b", 32) + " " + strings.Repeat("c", 32)
		}},
		{"clave excesiva", func(item *proposal) {
			item.IdempotencyKey = strings.Repeat("d", maxProposalReferenceBytes+1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid := created.Proposal
			invalid.RuleCompliance = cloneRules(invalid.RuleCompliance)
			test.mutate(&invalid)
			refreshProposalIdentity(&invalid)
			filePath := filepath.Join(t.TempDir(), "propuestas.jsonl")
			if err := writeProposals(filePath, []proposal{invalid}); err != nil {
				t.Fatal(err)
			}
			if _, err := readProposals(filePath); err == nil {
				t.Fatal("el historial semánticamente inválido fue aceptado")
			}
		})
	}
}

func TestProposalStoreRejectsInvalidReferencesAndKeys(t *testing.T) {
	_, source := testInventory(t)
	tests := []struct {
		name   string
		mutate func(*inventoryItem, *proposalRequest)
	}{
		{"identificador de elemento vacío", func(item *inventoryItem, request *proposalRequest) {
			item.ID, request.ItemRef = "", ""
		}},
		{"identificador de elemento con espacio exterior", func(item *inventoryItem, request *proposalRequest) {
			item.ID, request.ItemRef = " "+item.ID, " "+request.ItemRef
		}},
		{"identificador de elemento con control", func(item *inventoryItem, request *proposalRequest) {
			item.ID, request.ItemRef = "conducta:\x00uno", "conducta:\x00uno"
		}},
		{"identificador de elemento excesivo", func(item *inventoryItem, request *proposalRequest) {
			value := strings.Repeat("i", maxProposalReferenceBytes+1)
			item.ID, request.ItemRef = value, value
		}},
		{"revisión de elemento vacía", func(item *inventoryItem, request *proposalRequest) {
			item.Revision, request.ItemRevision = "", ""
		}},
		{"revisión de elemento malformada", func(item *inventoryItem, request *proposalRequest) {
			value := "sha256:" + strings.Repeat("A", 64)
			item.Revision, request.ItemRevision = value, value
		}},
		{"revisión de elemento con 63 hexadecimales", func(item *inventoryItem, request *proposalRequest) {
			value := "sha256:" + strings.Repeat("a", 63)
			item.Revision, request.ItemRevision = value, value
		}},
		{"revisión de elemento con 65 hexadecimales", func(item *inventoryItem, request *proposalRequest) {
			value := "sha256:" + strings.Repeat("a", 65)
			item.Revision, request.ItemRevision = value, value
		}},
		{"revisión de elemento con prefijo incorrecto", func(item *inventoryItem, request *proposalRequest) {
			value := "sha512:" + strings.Repeat("a", 64)
			item.Revision, request.ItemRevision = value, value
		}},
		{"actor inválido", func(_ *inventoryItem, request *proposalRequest) {
			request.ActorRef = "actor no opaco"
		}},
		{"actor con control Unicode", func(_ *inventoryItem, request *proposalRequest) {
			request.ActorRef = "actor:\x00no-opaco"
		}},
		{"proyecto inválido", func(_ *inventoryItem, request *proposalRequest) {
			request.ProjectRef = "project\tno-opaco"
		}},
		{"proyecto con control Unicode", func(_ *inventoryItem, request *proposalRequest) {
			request.ProjectRef = "project:\x01no-opaco"
		}},
		{"clave con treinta y un caracteres", func(_ *inventoryItem, request *proposalRequest) {
			request.IdempotencyKey = strings.Repeat("k", 31)
		}},
		{"clave de treinta y dos espacios", func(_ *inventoryItem, request *proposalRequest) {
			request.IdempotencyKey = strings.Repeat(" ", 32)
		}},
		{"clave con espacio interno", func(_ *inventoryItem, request *proposalRequest) {
			request.IdempotencyKey = strings.Repeat("b", 32) + " " + strings.Repeat("c", 32)
		}},
		{"clave excesiva", func(_ *inventoryItem, request *proposalRequest) {
			request.IdempotencyKey = strings.Repeat("d", maxProposalReferenceBytes+1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := source
			request := validProposalRequest(item, 0, strings.Repeat("e", 64))
			test.mutate(&item, &request)
			store := newProposalStore(filepath.Join(t.TempDir(), "propuestas.jsonl"))
			if _, err := store.submit(item, request); !errors.Is(err, errInvalidProposal) {
				t.Fatalf("la solicitud semánticamente inválida produjo %v", err)
			}
		})
	}
}

func TestProposalStoreAllowsInternalSpacesInInventoryItemRef(t *testing.T) {
	_, item := testInventory(t)
	item.ID = "conducta:identificador con espacio"
	store := newProposalStore(filepath.Join(t.TempDir(), "propuestas.jsonl"))
	result, err := store.submit(item, validProposalRequest(item, 0, strings.Repeat("f", 64)))
	if err != nil {
		t.Fatal(err)
	}
	proposals, err := store.list()
	if err != nil || len(proposals) != 1 || proposals[0].ProposalRef != result.Proposal.ProposalRef {
		t.Fatalf("el identificador legítimo con espacio interno no se conservó: %#v %v", proposals, err)
	}
}
