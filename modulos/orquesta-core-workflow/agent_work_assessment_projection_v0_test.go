package orquestacoreworkflow

import "testing"

func TestAgentAssessmentProjectionV0ConservaCamposMinimos(t *testing.T) {
	projection := AgentAssessmentProjectionRefV0(AgentWorkAssessedPayloadV0{
		AssessmentRef:  "assessment-ref-projection-001",
		PhaseID:        string(OrchestrationPhaseProgramacionV0),
		AgentRequestID: "agent-ref-projection-001",
		TaskRef:        "task-ref-projection-001",
		DeliveryRef:    "delivery-ref-projection-001",
		Verdict:        AgentAssessmentVerdictLoopDetectedV0,
		Action:         AgentAssessmentActionStopAgentV0,
		Severity:       AgentAssessmentSeverityCriticalV0,
	})

	got, ok := ParseAgentAssessmentProjectionV0(projection)
	if !ok {
		t.Fatalf("projection no parseable: %q", projection)
	}
	if got.AssessmentRef != "assessment-ref-projection-001" ||
		got.PhaseID != string(OrchestrationPhaseProgramacionV0) ||
		got.AgentRequestID != "agent-ref-projection-001" ||
		got.TaskRef != "task-ref-projection-001" ||
		got.DeliveryRef != "delivery-ref-projection-001" ||
		got.Verdict != AgentAssessmentVerdictLoopDetectedV0 ||
		got.Action != AgentAssessmentActionStopAgentV0 ||
		got.Severity != AgentAssessmentSeverityCriticalV0 {
		t.Fatalf("projection=%+v raw=%q", got, projection)
	}
}

func TestAgentAssessmentProjectionV0AceptaRefLegacy(t *testing.T) {
	got, ok := ParseAgentAssessmentProjectionV0("assessment-ref-legacy-001")
	if !ok || got.AssessmentRef != "assessment-ref-legacy-001" {
		t.Fatalf("projection legacy=%+v ok=%v", got, ok)
	}
}
