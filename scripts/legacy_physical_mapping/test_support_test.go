// Este fichero genera candidatos completos sin copiar 382 filas a mano.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

const (
	v3Fixture       = "../../product/traceability/legacy_source_roots_2026-07-30.json"
	universeFixture = "../../product/traceability/legacy_physical_subject_universe_2026-07-30.json"
)

func validCandidate(t *testing.T) mappingCandidate {
	t.Helper()
	v3Raw := readTestFile(t, v3Fixture)
	universeRaw := readTestFile(t, universeFixture)
	facts, err := validateBases(v3Raw, universeRaw)
	if err != nil {
		t.Fatal(err)
	}
	context := mappingContext{
		ViewRef: opaqueValue("view", "vista"), ViewEvidenceSHA256: testDigest("vista-evidencia"),
		ViewState: "declared_stable", FenceRef: opaqueValue("fence", "cercado"),
		FenceGeneration: 7, FenceEvidenceSHA256: testDigest("cercado-evidencia"),
		FenceState: "declared_valid", PolicyRef: opaqueValue("policy", "politica"),
		PolicySHA256: testDigest("politica"), ConfigurationSHA256: testDigest("configuracion"),
		AttemptRef:      opaqueValue("attempt", "intento"),
		WindowStartedAt: "2026-07-30T04:00:00Z", WindowEndedAt: "2026-07-30T04:10:00Z",
	}
	candidate := mappingCandidate{
		DocumentKind: "orquesta_legacy_private_mapping_candidate", SchemaVersion: 1,
		SourceV3BytesSHA256: expectedV3SHA, UniverseBytesSHA256: expectedUniverseSHA,
		LogicalReferenceSetSHA256: expectedReferences, SubjectSetSHA256: expectedSubjects,
		Context: context, Observations: make([]mappingObservation, 0, 382),
		Ownership: make([]ownershipDeclaration, 0, 382),
	}
	for index, subject := range facts.subjects {
		identity := opaqueValue("identity", indexName("identidad", index))
		binding := opaqueValue("binding", indexName("vinculacion", index))
		candidate.Observations = append(candidate.Observations, mappingObservation{
			Subject: subject, ExistsAtV3Observation: facts.historical[index],
			PresentInStableView: true, ObservedType: observedType(index),
			ObservationOutcome: "present", ObservedAt: "2026-07-30T04:05:00Z",
			IdentityRef: identity, ReopenEvidenceSHA256: testDigest("reapertura-" + identity),
			BindingRef: binding,
		})
		candidate.Ownership = append(candidate.Ownership, ownershipDeclaration{
			BindingRef: binding, OwnerSubject: subject,
			AliasSubjects: []subjectRef{}, OverlapSubjects: []subjectRef{},
			PrunedFromSubjects: []subjectRef{},
		})
	}
	resealCandidate(t, &candidate)
	return candidate
}

func resealCandidate(t *testing.T, candidate *mappingCandidate) {
	t.Helper()
	digest, err := domainDigest(contextDigestDomain, candidate.Context)
	if err != nil {
		t.Fatal(err)
	}
	candidate.ContextSHA256 = digest
	candidate.SemanticIntegritySHA256 = ""
	digest, err = domainDigest(candidateDigestDomain, *candidate)
	if err != nil {
		t.Fatal(err)
	}
	candidate.SemanticIntegritySHA256 = digest
}

func encodeCandidate(t *testing.T, candidate mappingCandidate) []byte {
	t.Helper()
	raw, err := canonicalJSON(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testInputs(t *testing.T, candidate mappingCandidate) ([]byte, []byte, []byte) {
	t.Helper()
	return readTestFile(t, v3Fixture), readTestFile(t, universeFixture), encodeCandidate(t, candidate)
}

func writePrivateCandidate(t *testing.T, raw []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "candidato.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func opaqueValue(prefix, seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

func testDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func indexName(prefix string, index int) string {
	return prefix + "-" + string(rune(index/100+'0')) +
		string(rune((index/10)%10+'0')) + string(rune(index%10+'0'))
}

func observedType(index int) string {
	if index%2 == 0 {
		return "directory"
	}
	return "regular"
}

func makeAbsent(t *testing.T, candidate *mappingCandidate, index int) {
	t.Helper()
	observation := &candidate.Observations[index]
	binding := observation.BindingRef
	observation.PresentInStableView = false
	observation.ObservedType, observation.ObservationOutcome = "absent", "not_found"
	observation.IdentityRef, observation.ReopenEvidenceSHA256, observation.BindingRef = "", "", ""
	observation.AbsenceEvidenceRef = opaqueValue("absence", indexName("ausencia", index))
	observation.AbsenceEvidenceSHA256 = testDigest(indexName("ausencia-evidencia", index))
	for current, declaration := range candidate.Ownership {
		if declaration.BindingRef == binding {
			candidate.Ownership = append(candidate.Ownership[:current], candidate.Ownership[current+1:]...)
			break
		}
	}
	resealCandidate(t, candidate)
}
