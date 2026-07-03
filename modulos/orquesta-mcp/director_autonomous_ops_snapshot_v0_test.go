package orquestamcp

import (
	"testing"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestDirectorOpsDecisionFromAutoprogrammingActionableRunV0RespetaSeveridadV0(t *testing.T) {
	blocked, ok := directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(MCPAutoprogrammingActionableRunV0{
		Code:              MCPGoalFirstQAFailedPublicTextV0,
		Severity:          "blocked",
		RunRef:            "run-ref-ops-snapshot-qa-001",
		RecommendedAction: MCPGoalFirstReworkPublicTextActionV0,
		EvidenceRefs:      []string{mcpAutoprogrammingEvidenceQAFailedPublicTextV0},
	})
	if !ok ||
		blocked.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		blocked.RunRef != "run-ref-ops-snapshot-qa-001" ||
		blocked.ReasonCode != MCPGoalFirstQAFailedPublicTextV0 ||
		!blocked.Attention ||
		!containsStringMCPV0(blocked.EvidenceRefs, mcpAutoprogrammingEvidenceQAFailedPublicTextV0) {
		t.Fatalf("blocked decision=%+v ok=%v", blocked, ok)
	}

	warning, ok := directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(MCPAutoprogrammingActionableRunV0{
		Code:              mcpAutoprogrammingActionThreadOutputSanitizedV0,
		Severity:          "warning",
		RunRef:            "run-ref-ops-snapshot-output-001",
		RecommendedAction: mcpQueueGlobalStatusActionReplanNarrowContextV0,
		EvidenceRefs:      []string{mcpAutoprogrammingEvidenceThreadOutputSanitizedV0},
	})
	if !ok ||
		warning.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		warning.RunRef != "run-ref-ops-snapshot-output-001" ||
		warning.ReasonCode != mcpAutoprogrammingActionThreadOutputSanitizedV0 ||
		!warning.Attention ||
		!containsStringMCPV0(warning.EvidenceRefs, mcpAutoprogrammingEvidenceThreadOutputSanitizedV0) {
		t.Fatalf("warning decision=%+v ok=%v", warning, ok)
	}

	if decision, ok := directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(MCPAutoprogrammingActionableRunV0{
		Code:              mcpAutoprogrammingActionActiveTimeoutCheckpointRecentV0,
		Severity:          "info",
		RunRef:            "run-ref-ops-snapshot-info-001",
		RecommendedAction: mcpQueueGlobalStatusActionObserveGoalWaitForCheckpointV0,
	}); ok {
		t.Fatalf("info no debe convertirse en decision de atencion: %+v", decision)
	}
}

func TestDirectorOpsDecisionFromAutoprogrammingActionableRunV0NormalizaSummaryKeyPorRunV0(t *testing.T) {
	runRef := "run-ref-ops-snapshot-repair-receipt-001"
	decision, ok := directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(MCPAutoprogrammingActionableRunV0{
		Code:              MCPGoalFirstArtifactPathsOmittedMaterializedV0,
		Severity:          "blocked",
		RunRef:            runRef,
		RecommendedAction: MCPGoalFirstRepairReceiptActionV0 + ":run:" + runRef,
		EvidenceRefs:      []string{mcpAutoprogrammingEvidenceArtifactPathsOmittedV0},
	})

	if !ok ||
		decision.Action != orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0 ||
		decision.RunRef != runRef ||
		decision.SummaryKey != "director.ops.decision."+MCPGoalFirstRepairReceiptActionV0 ||
		!decision.Attention ||
		!containsStringMCPV0(decision.EvidenceRefs, mcpAutoprogrammingEvidenceArtifactPathsOmittedV0) {
		t.Fatalf("decision=%+v ok=%v", decision, ok)
	}
}
