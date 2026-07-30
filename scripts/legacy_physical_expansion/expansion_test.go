// Estas pruebas fijan la expansión exacta, su orden y la salida canónica sin raíces.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const sourceFixture = "../../product/traceability/legacy_source_roots_2026-07-30.json"

func TestGenerateExactPhysicalSubjectUniverse(t *testing.T) {
	raw := readFixture(t)
	output, err := generate(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(output) || len(output) == 0 || output[len(output)-1] != '\n' {
		t.Fatal("la salida no es JSON canónico terminado en LF")
	}
	if bytes.HasPrefix(output, []byte{0xef, 0xbb, 0xbf}) ||
		bytes.Contains(output, []byte(`\u003c`)) || bytes.Contains(output, []byte(`\u003e`)) {
		t.Fatal("la salida contiene BOM o escape HTML")
	}
	var result expansionDocument
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	if result.SourceV3.BytesSHA256 != "sha256:"+expectedSourceSHA256 ||
		result.LogicalReferenceSetSHA256 != expectedPendingSHA256 ||
		result.SubjectSetSHA256 != "sha256:405f5b68f0e886a40bb69751fbb62f60567a15d6879393e9f42c4d1b5a458cc4" {
		t.Fatalf("huellas inesperadas: %#v", result)
	}
	if result.Counts != (expansionCounts{
		SourceRoots: 122, LogicalReferences: 112, SimpleReferences: 97,
		CollectionReferences: 15, CollectionMembers: 285, PhysicalSubjects: 382,
		HistoricalPresent: 364, HistoricalAbsent: 18,
	}) || result.Decisions != (decisionCounts{Include: 36, Exclude: 49, Pending: 27}) {
		t.Fatalf("recuentos inesperados: %#v/%#v", result.Counts, result.Decisions)
	}
	if len(result.MembershipSeals) != 15 || len(result.Subjects) != 382 ||
		result.Subjects[0] != (subjectRef{Kind: "single", RootID: "cache_global"}) ||
		result.Subjects[len(result.Subjects)-1] != (subjectRef{
			Kind: "member", CollectionRootID: "municipal_scripts_orquestacion", PathAlias: "claude.sh",
		}) {
		t.Fatal("el orden V3 de sujetos no quedó sellado")
	}
	for _, forbidden := range []string{
		"present_in_stable_view", "receipt", "logical_root", "relative_path",
		"/home/", "Codex", "device", "inode",
	} {
		if bytes.Contains(output, []byte(forbidden)) {
			t.Fatalf("la salida publica el campo prohibido %q", forbidden)
		}
	}
}

func TestGenerateIsDeterministicAndDigestIsNotSelfReferential(t *testing.T) {
	raw := readFixture(t)
	first, err := generate(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate(raw)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("dos ejecuciones sobre el mismo V3 difieren")
	}
	var result expansionDocument
	if err := json.Unmarshal(first, &result); err != nil {
		t.Fatal(err)
	}
	got, err := domainSeparatedDigest(subjectDigestDomain, result.Subjects)
	if err != nil || got != result.SubjectSetSHA256 {
		t.Fatal("la huella no se deriva únicamente de sujetos y dominio")
	}
	result.SubjectSetSHA256 = strings.Repeat("x", len(result.SubjectSetSHA256))
	again, err := domainSeparatedDigest(subjectDigestDomain, result.Subjects)
	if err != nil || again != got {
		t.Fatal("la huella de expansión es autorreferente")
	}
}

func TestAliasRepeatedBetweenCollectionsKeepsTypedIdentity(t *testing.T) {
	result := buildFixture(t)
	var matches []subjectRef
	for _, subject := range result.Subjects {
		if subject.PathAlias == "orquestador-perfil-01" {
			matches = append(matches, subject)
		}
	}
	if len(matches) != 2 || matches[0].CollectionRootID == matches[1].CollectionRootID {
		t.Fatalf("un alias intercolección válido colisionó: %#v", matches)
	}
}

func readFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(sourceFixture)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func decodeFixture(t *testing.T) sourceDocument {
	t.Helper()
	document, err := decodeSource(readFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func buildFixture(t *testing.T) expansionDocument {
	t.Helper()
	result, err := buildExpansion(decodeFixture(t), "sha256:"+expectedSourceSHA256)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
