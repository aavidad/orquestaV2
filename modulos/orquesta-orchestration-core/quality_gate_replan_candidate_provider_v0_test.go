package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestQualityGateReplanCandidateProviderV0BuildsFollowupFromRunProjections(t *testing.T) {
	run := qualityGateReplanRunV0()
	run.QualityGates = []string{
		qualityGateReplanGateProjectionV0("quality-gate-ref-nucleo-001", "task-ref-nucleo-001", orquestacoreworkflow.QualityGateDecisionBlockedV0),
	}
	run.ReplanDecisions = []string{
		qualityGateReplanDecisionProjectionV0(
			"replan-ref-nucleo-quality-gate-001",
			"quality-gate-ref-nucleo-001",
			"task-ref-nucleo-001",
			orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			[]string{"capacity-ref-nucleo-quality-gate-001", "agent-ref-nucleo-quality-gate-001"},
		),
	}
	base := StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
		EvidenceRefs: []string{"evidence-ref-base-001"},
	}}
	provider := QualityGateReplanCandidateProviderV0{
		Base:        base,
		RequestedBy: "orquesta-nucleo-test",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-21T10:00:00Z",
		CorrelationID: "corr-quality-gate-replan-001",
		EvidenceRefs:  []string{"evidence-ref-quality-gate-replan-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan candidates=%d", len(candidates.ReplanFollowupCandidates))
	}
	if !qualityGateReplanContainsStringV0(candidates.EvidenceRefs, "evidence-ref-base-001") ||
		!qualityGateReplanContainsStringV0(candidates.EvidenceRefs, "evidence-ref-quality-gate-replan-001") {
		t.Fatalf("base evidence not preserved: %v", candidates.EvidenceRefs)
	}

	input := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput
	if input.SourceKind != orquestadirector.ReplanFollowupSourceQualityGateBlockedV0 {
		t.Fatalf("source_kind=%q", input.SourceKind)
	}
	if input.DecisionPayload.ReplanRef != "replan-ref-nucleo-quality-gate-001" ||
		input.DecisionPayload.SourceRef != "quality-gate-ref-nucleo-001" ||
		input.DecisionPayload.TaskRef != "task-ref-nucleo-001" ||
		input.DecisionPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("decision payload=%+v", input.DecisionPayload)
	}
	if input.CapacityCandidate == nil ||
		input.CapacityCandidate.Payload.CapacityRequestID != "capacity-ref-nucleo-quality-gate-001" {
		t.Fatalf("capacity candidate=%+v", input.CapacityCandidate)
	}
	if input.AgentCandidate == nil ||
		input.AgentCandidate.Payload.AgentRequestID != "agent-ref-nucleo-quality-gate-001" ||
		input.AgentCandidate.Payload.CapacityRequestRef != "capacity-ref-nucleo-quality-gate-001" {
		t.Fatalf("agent candidate=%+v", input.AgentCandidate)
	}
	if _, err := orquestadirector.BuildReplanFollowupsV0(input); err != nil {
		t.Fatalf("BuildReplanFollowupsV0 reconstructed input: %v", err)
	}
}

func TestQualityGateReplanCandidateProviderV0DoesNothingWithoutBlockedGateReplanPair(t *testing.T) {
	provider := QualityGateReplanCandidateProviderV0{RequestedBy: "orquesta-nucleo-test"}
	run := qualityGateReplanRunV0()
	run.QualityGates = []string{
		qualityGateReplanGateProjectionV0("quality-gate-ref-nucleo-accepted-001", "task-ref-nucleo-001", orquestacoreworkflow.QualityGateDecisionAcceptedV0),
	}
	run.ReplanDecisions = []string{
		qualityGateReplanDecisionProjectionV0(
			"replan-ref-nucleo-quality-gate-accepted-001",
			"quality-gate-ref-nucleo-accepted-001",
			"task-ref-nucleo-001",
			orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			[]string{"capacity-ref-nucleo-quality-gate-accepted-001", "agent-ref-nucleo-quality-gate-accepted-001"},
		),
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-21T10:01:00Z",
		CorrelationID: "corr-quality-gate-replan-accepted-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0 accepted: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 0 {
		t.Fatalf("expected no candidates for accepted gate, got %d", len(candidates.ReplanFollowupCandidates))
	}

	run = qualityGateReplanRunV0()
	run.QualityGates = []string{
		qualityGateReplanGateProjectionV0("quality-gate-ref-nucleo-002", "task-ref-nucleo-001", orquestacoreworkflow.QualityGateDecisionBlockedV0),
	}
	run.ReplanDecisions = []string{
		qualityGateReplanDecisionProjectionV0(
			"replan-ref-nucleo-quality-gate-unsafe-001",
			"quality-gate-ref-nucleo-002",
			"task-ref-nucleo-001",
			orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			[]string{"capacity-ref-nucleo-quality-gate-unsafe-001"},
		),
	}

	candidates, err = provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-21T10:02:00Z",
		CorrelationID: "corr-quality-gate-replan-unsafe-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0 unsafe: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 0 {
		t.Fatalf("expected no candidates for missing agent followup, got %d", len(candidates.ReplanFollowupCandidates))
	}
}

func qualityGateReplanRunV0() orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "run-nucleo-quality-gate-replan-001",
		ProjectRef:    "project-ref-nucleo-quality-gate-replan-001",
		AppSpecRef:    "app-spec-ref-nucleo-quality-gate-replan-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-nucleo-001"},
	}
}

func qualityGateReplanGateProjectionV0(
	gateRef string,
	subjectRef string,
	decision orquestacoreworkflow.QualityGateDecisionV0,
) string {
	return gateRef +
		qualityGateReplanGateDecisionSeparatorV0 + string(decision) +
		qualityGateReplanGateSubjectSeparatorV0 + subjectRef
}

func qualityGateReplanDecisionProjectionV0(
	replanRef string,
	sourceRef string,
	taskRef string,
	action orquestacoreworkflow.ReplanDecisionActionV0,
	followupRefs []string,
) string {
	return replanRef +
		qualityGateReplanDecisionSourceSeparatorV0 + sourceRef +
		qualityGateReplanDecisionTaskSeparatorV0 + taskRef +
		qualityGateReplanDecisionActionSeparatorV0 + string(action) +
		qualityGateReplanDecisionFollowupsSeparatorV0 + stringsJoinQualityGateReplanV0(followupRefs)
}

func stringsJoinQualityGateReplanV0(values []string) string {
	result := ""
	for _, value := range values {
		if result != "" {
			result += qualityGateReplanDecisionFollowupsJoinerV0
		}
		result += value
	}
	return result
}

func qualityGateReplanContainsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
