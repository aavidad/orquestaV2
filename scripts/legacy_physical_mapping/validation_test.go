// Estas pruebas fijan el veredicto honesto, el orden y la repetición determinista.
package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestValidCandidateProducesRedactedDeterministicVerdict(t *testing.T) {
	candidate := validCandidate(t)
	v3Raw, universeRaw, candidateRaw := testInputs(t, candidate)
	first, err := validateMapping(v3Raw, universeRaw, candidateRaw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := validateMapping(v3Raw, universeRaw, candidateRaw)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("la repetición no produjo los mismos bytes")
	}
	var verdict validationVerdict
	if err := json.Unmarshal(first, &verdict); err != nil {
		t.Fatal(err)
	}
	if verdict.Decision != "candidate_semantically_coherent" ||
		verdict.Authority != "semantic_validator_only" ||
		verdict.CandidateBytesSHA256 != rawSHA256(candidateRaw) ||
		verdict.CandidateSemanticIntegritySHA256 != candidate.SemanticIntegritySHA256 ||
		verdict.Counts.Subjects != 382 || verdict.Counts.StablePresent != 382 ||
		verdict.Counts.Bindings != 382 || verdict.Counts.HistoricalPresent != 364 ||
		verdict.Counts.HistoricalAbsent != 18 {
		t.Fatalf("veredicto inesperado: %#v", verdict)
	}
	for _, forbidden := range [][]byte{
		[]byte("/home/"), []byte(`"path"`), []byte("descriptor"), []byte("inode"),
		[]byte(`"uid"`), []byte(`"gid"`), []byte(candidate.Context.FenceRef),
		[]byte(candidate.Context.ViewRef), []byte(candidate.Observations[0].IdentityRef),
	} {
		if bytes.Contains(first, forbidden) {
			t.Fatalf("el veredicto filtró %q", forbidden)
		}
	}
}

func TestHistoricalAndCurrentPresenceRemainIndependent(t *testing.T) {
	candidate := validCandidate(t)
	index := -1
	for current, observation := range candidate.Observations {
		if observation.ExistsAtV3Observation {
			index = current
			break
		}
	}
	if index < 0 {
		t.Fatal("fixture sin presencia histórica")
	}
	makeAbsent(t, &candidate, index)
	v3Raw, universeRaw, candidateRaw := testInputs(t, candidate)
	output, err := validateMapping(v3Raw, universeRaw, candidateRaw)
	if err != nil {
		t.Fatal(err)
	}
	var verdict validationVerdict
	if err := json.Unmarshal(output, &verdict); err != nil {
		t.Fatal(err)
	}
	if verdict.Counts.HistoricalPresent != 364 || verdict.Counts.StableAbsent != 1 ||
		verdict.Counts.StablePresent != 381 {
		t.Fatalf("las presencias se confundieron: %#v", verdict.Counts)
	}
}

func TestAliasAndNestedPruneKeepAllLogicalSubjects(t *testing.T) {
	candidate := validCandidate(t)
	first, second := &candidate.Observations[0], &candidate.Observations[1]
	second.IdentityRef = first.IdentityRef
	second.BindingRef = first.BindingRef
	second.ReopenEvidenceSHA256 = first.ReopenEvidenceSHA256
	second.ObservedType = first.ObservedType
	candidate.Ownership[0].AliasSubjects = []subjectRef{second.Subject}
	candidate.Ownership = append(candidate.Ownership[:1], candidate.Ownership[2:]...)
	candidate.Ownership[1].OverlapSubjects = []subjectRef{first.Subject}
	candidate.Ownership[1].PrunedFromSubjects = []subjectRef{first.Subject}
	resealCandidate(t, &candidate)
	v3Raw, universeRaw, raw := testInputs(t, candidate)
	output, err := validateMapping(v3Raw, universeRaw, raw)
	if err != nil {
		t.Fatal(err)
	}
	var verdict validationVerdict
	if err := json.Unmarshal(output, &verdict); err != nil {
		t.Fatal(err)
	}
	if verdict.Counts.Subjects != 382 || verdict.Counts.Bindings != 381 ||
		verdict.Counts.Aliases != 1 || verdict.Counts.Overlaps != 1 ||
		verdict.Counts.Prunes != 1 {
		t.Fatalf("reconciliación inesperada: %#v", verdict.Counts)
	}
}
