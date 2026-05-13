package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestHeuristicAutonomousDirectorPolicyV0RecomiendaReplanPorQualityYReview(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-nucleo-autonomous-quality-001")
	run.QualityGates = []string{
		"quality-gate-ref-auto-001#decision:rework_required#subject:delivery-ref-auto-001",
	}
	run.ReviewResults = []string{
		"review-result-ref-auto-001#review_result:changes_requested#review_request:review-request-ref-auto-001#delivery:delivery-ref-auto-001",
	}

	decision := mustAutonomousQualityDecisionForTestV0(t, run, BuildDirectorRunStatsV0(run))

	assertAutonomousRecommendationForTestV0(
		t,
		decision.QualityRecommendations,
		autonomousQualityKindQualityGateV0,
		"delivery-ref-auto-001",
		AutonomousQualityActionReplanV0,
		"quality_gate_rework_required",
	)
	assertAutonomousRecommendationForTestV0(
		t,
		decision.QualityRecommendations,
		autonomousQualityKindDeliveryV0,
		"delivery-ref-auto-001",
		AutonomousQualityActionReplanV0,
		"review_changes_requested",
	)
}

func TestHeuristicAutonomousDirectorPolicyV0RecomiendaStopPorOverBudgetNoActivity(t *testing.T) {
	runRef := "run-nucleo-autonomous-quality-budget-001"
	agentRef := "agent-ref-auto-quality-budget-001"
	taskRef := "task-ref-auto-quality-budget-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-auto-quality-budget-001",
		SessionRef:     "session-ref-auto-quality-budget-001",
		LaunchRef:      "launch-ref-auto-quality-budget-001",
		ReadinessRef:   "readiness-ref-auto-quality-budget-001",
		EvidenceRefs:   []string{"evidence-ref-auto-quality-budget-process-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}
	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)
	ApplyDirectorProgressObservationsV0(&stats, []AgentProgressObservationV0{
		directorBudgetProgressObservationForTestV0(
			runRef,
			agentRef,
			taskRef,
			orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		),
	}, nil)

	decision := mustAutonomousQualityDecisionForTestV0(t, run, stats)

	assertAutonomousRecommendationForTestV0(
		t,
		decision.QualityRecommendations,
		autonomousQualityKindAgentV0,
		agentRef,
		AutonomousQualityActionStopAgentV0,
		"over_budget_no_activity",
	)
}

func TestHeuristicAutonomousDirectorPolicyV0ConsultaPorLoopSinControlDeParada(t *testing.T) {
	runRef := "run-nucleo-autonomous-quality-loop-001"
	agentRef := "agent-ref-auto-quality-loop-001"
	taskRef := "task-ref-auto-quality-loop-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{
		directorStatsProgressObservationForTestV0(
			runRef,
			agentRef,
			taskRef,
			orquestaruntime.AgentLoopDetectedV0,
		),
	}, nil)

	decision := mustAutonomousQualityDecisionForTestV0(t, run, stats)

	assertAutonomousRecommendationForTestV0(
		t,
		decision.QualityRecommendations,
		autonomousQualityKindAgentV0,
		agentRef,
		AutonomousQualityActionAskDirectorV0,
		"agent_loop_detected",
	)
}

func mustAutonomousQualityDecisionForTestV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	stats DirectorRunStatsV0,
) AutonomousDirectorDecisionV0 {
	t.Helper()
	decision, err := HeuristicAutonomousDirectorPolicyV0{}.DecideAutonomousDirectorV0(
		context.Background(),
		AutonomousDirectorDecisionInputV0{Run: run, Stats: stats},
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	return decision
}

func assertAutonomousRecommendationForTestV0(
	t *testing.T,
	recommendations []AutonomousQualityRecommendationV0,
	kind string,
	subjectRef string,
	action string,
	reason string,
) {
	t.Helper()
	for _, recommendation := range recommendations {
		if recommendation.SubjectKind == kind &&
			recommendation.SubjectRef == subjectRef &&
			recommendation.RecommendedAction == action &&
			recommendation.ReasonCode == reason {
			return
		}
	}
	t.Fatalf("recommendation no encontrada kind=%s subject=%s action=%s reason=%s en %+v",
		kind, subjectRef, action, reason, recommendations)
}
