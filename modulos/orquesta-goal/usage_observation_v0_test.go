package orquestagoal

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGoalUsageObservationV0ZeroValueCompatible(t *testing.T) {
	observation := NormalizeGoalUsageObservationV0(GoalUsageObservationV0{})
	if !GoalUsageObservationEmptyV0(observation) {
		t.Fatalf("observation=%+v", observation)
	}
	if issues := ValidateGoalUsageObservationV0(observation); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	result := NormalizeGoalWorkResultV0(GoalWorkResultV0{Status: GoalStatusCompleteV0, GoalRef: "goal-ref-usage-zero-001"})
	if issues := ValidateGoalWorkResultV0(result); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"usage_observation"`) {
		t.Fatalf("zero usage_observation serialized: %s", encoded)
	}
}

func TestNormalizeGoalUsageObservationV0CanonicalizesAndDoesNotMutateInput(t *testing.T) {
	input := GoalUsageObservationV0{
		TokensAccumulated: 42,
		RuntimeSeconds:    7,
		ObservedAt:        " 2026-07-11T12:34:56+02:00 ",
		EvidenceRefs:      []string{" evidence-ref-usage-001 ", "evidence-ref-usage-001", "evidence-ref-usage-002"},
		SourceRef:         " usage-source-ref-001 ",
	}
	normalized := NormalizeGoalUsageObservationV0(input)
	if normalized.ObservedAt != "2026-07-11T10:34:56Z" || normalized.SourceRef != "usage-source-ref-001" {
		t.Fatalf("normalized=%+v", normalized)
	}
	if len(normalized.EvidenceRefs) != 2 || normalized.EvidenceRefs[0] != "evidence-ref-usage-001" {
		t.Fatalf("evidence=%v", normalized.EvidenceRefs)
	}
	normalized.EvidenceRefs[0] = "changed"
	if input.EvidenceRefs[0] != " evidence-ref-usage-001 " {
		t.Fatalf("input mutated: %+v", input)
	}
	if issues := ValidateGoalUsageObservationV0(NormalizeGoalUsageObservationV0(input)); len(issues) != 0 {
		t.Fatalf("issues=%v", issues)
	}
}

func TestValidateGoalUsageObservationV0RejectsAdversarialValues(t *testing.T) {
	cases := []struct {
		name        string
		observation GoalUsageObservationV0
		field       string
		code        string
	}{
		{"negative tokens", GoalUsageObservationV0{TokensAccumulated: -1, ObservedAt: "2026-07-11T10:34:56Z", EvidenceRefs: []string{"evidence-ref-usage-001"}, SourceRef: "source-ref-usage-001"}, "usage_observation.tokens_accumulated", ErrGoalUsageObservationInvalidV0},
		{"negative runtime", GoalUsageObservationV0{RuntimeSeconds: -1, ObservedAt: "2026-07-11T10:34:56Z", EvidenceRefs: []string{"evidence-ref-usage-001"}, SourceRef: "source-ref-usage-001"}, "usage_observation.runtime_seconds", ErrGoalUsageObservationInvalidV0},
		{"invalid timestamp", GoalUsageObservationV0{ObservedAt: "not-a-time", EvidenceRefs: []string{"evidence-ref-usage-001"}, SourceRef: "source-ref-usage-001"}, "usage_observation.observed_at", ErrGoalUsageObservationTimestampInvalidV0},
		{"timestamp only", GoalUsageObservationV0{ObservedAt: "2026-07-11T10:34:56Z"}, "usage_observation.evidence_refs", ErrGoalUsageObservationEvidenceRequiredV0},
		{"evidence only", GoalUsageObservationV0{EvidenceRefs: []string{"evidence-ref-usage-001"}}, "usage_observation.observed_at", ErrGoalUsageObservationInvalidV0},
		{"source only", GoalUsageObservationV0{SourceRef: "source-ref-usage-001"}, "usage_observation.observed_at", ErrGoalUsageObservationInvalidV0},
		{"positive tokens missing observation", GoalUsageObservationV0{TokensAccumulated: 1, EvidenceRefs: []string{"evidence-ref-usage-001"}, SourceRef: "source-ref-usage-001"}, "usage_observation.observed_at", ErrGoalUsageObservationInvalidV0},
		{"positive runtime missing evidence", GoalUsageObservationV0{RuntimeSeconds: 1, ObservedAt: "2026-07-11T10:34:56Z", SourceRef: "source-ref-usage-001"}, "usage_observation.evidence_refs", ErrGoalUsageObservationEvidenceRequiredV0},
		{"missing source", GoalUsageObservationV0{ObservedAt: "2026-07-11T10:34:56Z", EvidenceRefs: []string{"evidence-ref-usage-001"}}, "usage_observation.source_ref", ErrGoalUsageObservationSourceRequiredV0},
		{"invalid evidence ref", GoalUsageObservationV0{ObservedAt: "2026-07-11T10:34:56Z", EvidenceRefs: []string{"/tmp/evidence"}, SourceRef: "source-ref-usage-001"}, "usage_observation.evidence_refs", ErrGoalRefFieldInvalidV0},
		{"invalid source ref", GoalUsageObservationV0{ObservedAt: "2026-07-11T10:34:56Z", EvidenceRefs: []string{"evidence-ref-usage-001"}, SourceRef: "/tmp/source"}, "usage_observation.source_ref", ErrGoalRefFieldInvalidV0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := ValidateGoalUsageObservationV0(tc.observation)
			if !hasGoalIssueFieldCodeV0(issues, tc.field, tc.code) {
				t.Fatalf("issues=%v", issues)
			}
		})
	}
}

func TestValidateGoalWorkResultV0IncludesUsageObservationValidation(t *testing.T) {
	issues := ValidateGoalWorkResultV0(GoalWorkResultV0{
		Status:  GoalStatusCompleteV0,
		GoalRef: "goal-ref-usage-result-001",
		UsageObservation: GoalUsageObservationV0{
			TokensAccumulated: 1,
			ObservedAt:        "2026-07-11T10:34:56Z",
			SourceRef:         "source-ref-usage-result-001",
		},
	})
	if !hasGoalIssueFieldCodeV0(issues, "usage_observation.evidence_refs", ErrGoalUsageObservationEvidenceRequiredV0) {
		t.Fatalf("issues=%v", issues)
	}
}
