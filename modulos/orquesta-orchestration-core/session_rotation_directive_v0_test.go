package orquestacionnucleoapp

import (
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildSessionRotationDirectiveV0PideHandoffCompletoSinRelanzar(t *testing.T) {
	request := sessionRotationDirectiveRequestForTestV0()
	request.Handoff.NextAction = ""

	directive := BuildSessionRotationDirectiveV0(request)

	if directive.Directive != SessionRotationDirectiveAskHandoffV0 ||
		directive.CanLaunchReplacement ||
		directive.CanStopCurrent ||
		directive.DirectorQuestion != "complete_handoff_or_continue_current_session" {
		t.Fatalf("directive=%+v", directive)
	}
}

func TestBuildSessionRotationDirectiveV0ConservaRelevoOptInSinStop(t *testing.T) {
	directive := BuildSessionRotationDirectiveV0(sessionRotationDirectiveRequestForTestV0())

	if directive.Directive != SessionRotationDirectiveLaunchNextV0 ||
		!directive.CanLaunchReplacement ||
		directive.CanStopCurrent ||
		directive.SessionEpoch != "session-epoch-003" ||
		directive.HandoffRef != "handoff-ref-session-rotation-003" ||
		directive.ComparisonMetric.CandidateAction != orquestaruntime.RuntimeSessionRotationActionLaunchNextV0 {
		t.Fatalf("directive=%+v", directive)
	}
}

func sessionRotationDirectiveRequestForTestV0() orquestaruntime.RuntimeSessionRotationRequestV0 {
	handoff := orquestaruntime.RuntimeSessionRotationHandoffV0{
		SchemaVersion:     orquestaruntime.RuntimeSessionRotationHandoffSchemaVersionV0,
		HandoffRef:        "handoff-ref-session-rotation-003",
		OutgoingAckRef:    "ack-ref-session-rotation-003",
		OutgoingStatus:    orquestaruntime.RuntimeSessionRotationOutgoingStatusHandoffReadyV0,
		PendingStatus:     "pending_work_bounded",
		OriginalObjective: "probar rotacion opt-in",
		ProgressSummary:   "contrato revisado y pendiente de relevo",
		NextAction:        "arrancar sesion nueva con contexto acotado",
		PendingWork:       []string{"validar relanzamiento experimental"},
		TouchedFiles:      []string{"docs/runbooks/session_rotation_handoff_experimental_2026-05-27.md"},
		CausalRefs:        []string{"task-ref-session-rotation-003"},
		EvidenceRefs:      []string{"evidence-ref-session-rotation-003"},
	}
	return orquestaruntime.RuntimeSessionRotationRequestV0{
		SchemaVersion:      orquestaruntime.RuntimeSessionRotationRequestSchemaVersionV0,
		RequestRef:         "request-ref-session-rotation-003",
		CorrelationID:      "corr-session-rotation-003",
		RunRef:             "run-ref-session-rotation-003",
		TaskRef:            "task-ref-session-rotation-003",
		AgentRef:           "agent-ref-session-rotation-003",
		SessionRef:         "session-ref-session-rotation-003",
		SessionEpoch:       "session-epoch-003",
		ExperimentRef:      "experiment-ref-session-rotation-t207",
		Mode:               orquestaruntime.RuntimeSessionRotationModeOptInLowPriorityV0,
		RotationReason:     "context_budget_experiment",
		CurrentProcessLive: true,
		Handoff:            &handoff,
		EvidenceRefs:       []string{"evidence-ref-session-rotation-budget"},
	}
}
