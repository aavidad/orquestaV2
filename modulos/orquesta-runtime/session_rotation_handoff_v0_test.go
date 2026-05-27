package orquestaruntime

import "testing"

func TestEvaluateRuntimeSessionRotationV0NoRelanzaSinHandoffCompleto(t *testing.T) {
	request := validRuntimeSessionRotationRequestForTestV0()
	request.Handoff.ProgressSummary = ""

	decision := EvaluateRuntimeSessionRotationV0(request)

	if decision.Action != RuntimeSessionRotationActionRequestHandoffV0 ||
		decision.LaunchAllowed ||
		decision.HandoffAccepted ||
		!containsRuntimeSessionRotationEvidenceForTestV0(decision.EvidenceRefs, "evidence-ref-session-rotation-handoff-required") {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateRuntimeSessionRotationV0PideHandoffSiFalta(t *testing.T) {
	request := validRuntimeSessionRotationRequestForTestV0()
	request.Handoff = nil

	decision := EvaluateRuntimeSessionRotationV0(request)

	if decision.Action != RuntimeSessionRotationActionRequestHandoffV0 ||
		decision.LaunchAllowed ||
		decision.HandoffAccepted ||
		decision.StopCurrentAllowed ||
		decision.Reason != "session_rotation_handoff_insufficient" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateRuntimeSessionRotationV0PermiteRelevoOptInConHandoffAceptado(t *testing.T) {
	decision := EvaluateRuntimeSessionRotationV0(validRuntimeSessionRotationRequestForTestV0())

	if decision.Action != RuntimeSessionRotationActionLaunchNextV0 ||
		!decision.LaunchAllowed ||
		decision.StopCurrentAllowed ||
		!decision.HandoffAccepted ||
		decision.SessionEpoch != "session-epoch-002" ||
		decision.ComparisonMetric.BaselineAction != RuntimeSessionRotationActionContinueCurrentV0 ||
		decision.ComparisonMetric.CandidateAction != RuntimeSessionRotationActionLaunchNextV0 ||
		!containsRuntimeSessionRotationEvidenceForTestV0(decision.EvidenceRefs, "evidence-ref-session-rotation-handoff-accepted") {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateRuntimeSessionRotationV0NoActivaSinOptIn(t *testing.T) {
	request := validRuntimeSessionRotationRequestForTestV0()
	request.Mode = ""

	decision := EvaluateRuntimeSessionRotationV0(request)

	if decision.Action != RuntimeSessionRotationActionContinueCurrentV0 ||
		decision.LaunchAllowed ||
		decision.HandoffAccepted ||
		decision.Reason != "session_rotation_not_opt_in" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestValidateRuntimeSessionRotationHandoffV0RechazaRutaAbsoluta(t *testing.T) {
	handoff := validRuntimeSessionRotationHandoffForTestV0()
	handoff.TouchedFiles = []string{"/tmp/proyecto/secreto.go"}

	issues := ValidateRuntimeSessionRotationHandoffV0(handoff, "corr-session-rotation-001")

	if len(issues) == 0 || issues[0].Field != "handoff.touched_files[0]" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestEvaluateRuntimeSessionRotationV0PideCompletarCierreSaliente(t *testing.T) {
	request := validRuntimeSessionRotationRequestForTestV0()
	request.Handoff.PendingStatus = ""

	decision := EvaluateRuntimeSessionRotationV0(request)

	if decision.Action != RuntimeSessionRotationActionRequestHandoffV0 ||
		decision.LaunchAllowed ||
		decision.HandoffAccepted {
		t.Fatalf("decision=%+v", decision)
	}
}

func validRuntimeSessionRotationRequestForTestV0() RuntimeSessionRotationRequestV0 {
	handoff := validRuntimeSessionRotationHandoffForTestV0()
	return RuntimeSessionRotationRequestV0{
		SchemaVersion:      RuntimeSessionRotationRequestSchemaVersionV0,
		RequestRef:         "request-ref-session-rotation-001",
		CorrelationID:      "corr-session-rotation-001",
		RunRef:             "run-ref-session-rotation-001",
		TaskRef:            "task-ref-session-rotation-001",
		AgentRef:           "agent-ref-session-rotation-001",
		SessionRef:         "session-ref-session-rotation-001",
		SessionEpoch:       "session-epoch-002",
		ExperimentRef:      "experiment-ref-session-rotation-t207",
		Mode:               RuntimeSessionRotationModeOptInLowPriorityV0,
		RotationReason:     "context_budget_experiment",
		CurrentProcessLive: true,
		Handoff:            &handoff,
		EvidenceRefs:       []string{"evidence-ref-session-rotation-budget"},
	}
}

func validRuntimeSessionRotationHandoffForTestV0() RuntimeSessionRotationHandoffV0 {
	return RuntimeSessionRotationHandoffV0{
		SchemaVersion:     RuntimeSessionRotationHandoffSchemaVersionV0,
		HandoffRef:        "handoff-ref-session-rotation-001",
		OutgoingAckRef:    "ack-ref-session-rotation-001",
		OutgoingStatus:    RuntimeSessionRotationOutgoingStatusHandoffReadyV0,
		PendingStatus:     "pending_work_bounded",
		OriginalObjective: "mantener avance real sin perder contexto",
		ProgressSummary:   "analisis completado y siguiente cambio identificado",
		NextAction:        "lanzar relevo con contexto acotado",
		PendingWork:       []string{"implementar adaptador opt-in"},
		TouchedFiles:      []string{"modulos/orquesta-runtime/session_rotation_handoff_v0.go"},
		RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-runtime"},
		CausalRefs:        []string{"task-ref-session-rotation-001"},
		EvidenceRefs:      []string{"evidence-ref-session-rotation-handoff-durable"},
	}
}

func containsRuntimeSessionRotationEvidenceForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
