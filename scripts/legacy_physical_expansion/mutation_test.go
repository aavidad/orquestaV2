// Estas pruebas mutan el V3 para impedir duplicados, deriva y falsas presencias.
package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPinnedV3RejectsByteAndStructuralDrift(t *testing.T) {
	raw := readFixture(t)
	mutations := map[string][]byte{
		"espaciado":       append([]byte(" "), raw...),
		"BOM":             append([]byte{0xef, 0xbb, 0xbf}, raw...),
		"clave extra":     bytes.Replace(raw, []byte(`"document_kind":`), []byte(`"extra":true,"document_kind":`), 1),
		"clave duplicada": bytes.Replace(raw, []byte(`"document_kind":`), []byte(`"document_kind":"duplicado","document_kind":`), 1),
		"escape alterno":  bytes.Replace(raw, []byte(`"cache_global"`), []byte(`"cache_glob\u0061l"`), 1),
	}
	for name, mutated := range mutations {
		t.Run(name, func(t *testing.T) {
			if _, err := generate(mutated); err == nil || errorCode(err) != "v3_no_fijado" {
				t.Fatalf("la huella no cortó antes de interpretar: %v", err)
			}
		})
	}
}

func TestRejectsHeaderAndPendingReferenceMutations(t *testing.T) {
	for name, mutate := range map[string]func(*sourceDocument){
		"kind":    func(value *sourceDocument) { value.DocumentKind = "otro" },
		"version": func(value *sourceDocument) { value.SchemaVersion++ },
		"closed":  func(value *sourceDocument) { value.Closed = true },
		"omitida": func(value *sourceDocument) {
			value.BlockingPhysicalCensusRootIDs = value.BlockingPhysicalCensusRootIDs[:111]
		},
		"añadida": func(value *sourceDocument) {
			value.BlockingPhysicalCensusRootIDs = append(value.BlockingPhysicalCensusRootIDs, "cache_global")
		},
		"duplicada": func(value *sourceDocument) {
			value.BlockingPhysicalCensusRootIDs[1] = value.BlockingPhysicalCensusRootIDs[0]
		},
		"reordenada": func(value *sourceDocument) {
			value.BlockingPhysicalCensusRootIDs[0], value.BlockingPhysicalCensusRootIDs[1] =
				value.BlockingPhysicalCensusRootIDs[1], value.BlockingPhysicalCensusRootIDs[0]
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := decodeFixture(t)
			mutate(&value)
			requireBuildFailure(t, value)
		})
	}
}

func TestRejectsCollectionAndMembershipMutations(t *testing.T) {
	for name, mutate := range map[string]func(*sourceDocument){
		"colección omitida": func(value *sourceDocument) {
			value.Collections = value.Collections[:14]
		},
		"colección duplicada": func(value *sourceDocument) {
			value.Collections = append(value.Collections, value.Collections[0])
		},
		"huella corrupta": func(value *sourceDocument) {
			value.Collections[0].MembersSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		},
		"miembros reordenados": func(value *sourceDocument) {
			value.Collections[0].Members[0], value.Collections[0].Members[1] =
				value.Collections[0].Members[1], value.Collections[0].Members[0]
		},
		"alias duplicado": func(value *sourceDocument) {
			value.Collections[0].Members[1].PathAlias = value.Collections[0].Members[0].PathAlias
			refreshMembersDigest(t, &value.Collections[0])
		},
		"381 sujetos": func(value *sourceDocument) {
			value.Collections[0].Members = value.Collections[0].Members[:len(value.Collections[0].Members)-1]
			refreshMembersDigest(t, &value.Collections[0])
		},
		"383 sujetos": func(value *sourceDocument) {
			member := value.Collections[0].Members[0]
			member.PathAlias = "miembro-adicional"
			value.Collections[0].Members = append(value.Collections[0].Members, member)
			refreshMembersDigest(t, &value.Collections[0])
		},
		"presencia histórica": func(value *sourceDocument) {
			value.Collections[0].Members[0].ExistsAtObservation =
				!value.Collections[0].Members[0].ExistsAtObservation
			refreshMembersDigest(t, &value.Collections[0])
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := decodeFixture(t)
			mutate(&value)
			requireBuildFailure(t, value)
		})
	}
}

func TestStrictDecoderRejectsCurrentPresence(t *testing.T) {
	raw := readFixture(t)
	mutated := bytes.Replace(
		raw,
		[]byte(`"exists_at_observation": true,`),
		[]byte(`"exists_at_observation": true,"present_in_stable_view":true,`),
		1,
	)
	if _, err := decodeSource(mutated); err == nil {
		t.Fatal("el decodificador admitió presencia de una vista no observada")
	}
}

func refreshMembersDigest(t *testing.T, collection *sourceCollection) {
	t.Helper()
	encoded, err := json.Marshal(collection.Members)
	if err != nil {
		t.Fatal(err)
	}
	collection.MembersSHA256 = rawSHA256(append(encoded, '\n'))
}

func requireBuildFailure(t *testing.T, document sourceDocument) {
	t.Helper()
	if _, err := buildExpansion(document, "sha256:"+expectedSourceSHA256); err == nil {
		t.Fatal("la mutación produjo una expansión")
	}
}
