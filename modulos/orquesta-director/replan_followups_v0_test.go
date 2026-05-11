package orquestadirector

import (
	"encoding/json"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildReplanFollowupsV0RetryConstruyeDecisionCapacidadYAgente(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0, "retry")
	input.CapacityCandidate = validReplanCapacityCandidateV0("retry", input.DecisionCommandMeta.RunID)
	input.AgentCandidate = validReplanAgentCandidateV0("retry", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusCapacityAndAgentRequestedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertCommandTypeV0(t, result.RecordReplanDecisionCommand, orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0)
	assertOptionalCommandTypeV0(t, result.RequestCapacityCommand, orquestacoreworkflow.OrchestrationCommandRequestCapacityV0)
	assertOptionalCommandTypeV0(t, result.RequestAgentCommand, orquestacoreworkflow.OrchestrationCommandRequestAgentV0)
	if result.AskDirectorCommand != nil {
		t.Fatalf("ask_director inesperado: %+v", result.AskDirectorCommand)
	}

	var decision orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0
	mustDecodeCommandPayloadV0(t, result.RecordReplanDecisionCommand, &decision)
	if decision.ReplanRef != "replan-ref-retry" || decision.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("decision inesperada: %+v", decision)
	}
}

func TestBuildReplanFollowupsV0RetryConstruyeDecisionOpenPhaseYCapacidad(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0, "retry-open")
	input.OpenPhaseCandidate = validReplanOpenPhaseCandidateV0("retry-open", input.DecisionCommandMeta.RunID)
	input.CapacityCandidate = validReplanCapacityCandidateV0("retry-open", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}

	assertCommandTypeV0(t, result.RecordReplanDecisionCommand, orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0)
	assertOptionalCommandTypeV0(t, result.OpenPhaseCommand, orquestacoreworkflow.OrchestrationCommandOpenPhaseV0)
	assertOptionalCommandTypeV0(t, result.RequestCapacityCommand, orquestacoreworkflow.OrchestrationCommandRequestCapacityV0)
	if result.RequestAgentCommand != nil {
		t.Fatalf("request_agent inesperado: %+v", result.RequestAgentCommand)
	}

	var openPhase orquestacoreworkflow.OpenPhaseCommandPayloadV0
	mustDecodeCommandPayloadV0(t, *result.OpenPhaseCommand, &openPhase)
	if openPhase.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		t.Fatalf("open_phase inesperado: %+v", openPhase)
	}
}

func TestBuildReplanFollowupsV0NoInventaFollowupsSinCandidates(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0, "missing")

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusNeedsDirectorUnsupportedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertCommandTypeV0(t, result.RecordReplanDecisionCommand, orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0)
	if result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil || result.AskDirectorCommand != nil {
		t.Fatalf("no esperaba comandos secundarios: capacity=%v agent=%v ask=%v",
			result.RequestCapacityCommand, result.RequestAgentCommand, result.AskDirectorCommand)
	}
}

func TestBuildReplanFollowupsV0RejectsBlockedAgentCandidate(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0, "blocked")
	input.CapacityCandidate = validReplanCapacityCandidateV0("blocked", input.DecisionCommandMeta.RunID)
	input.AgentCandidate = validReplanAgentCandidateV0("blocked", input.DecisionCommandMeta.RunID)
	input.BlockedAgentRefs = []string{"agent-request-ref-replan-blocked"}

	_, err := BuildReplanFollowupsV0(input)
	var replanErr ReplanFollowupsErrorV0
	if !errors.As(err, &replanErr) {
		t.Fatalf("error type=%T, want ReplanFollowupsErrorV0", err)
	}
	if replanErr.Code != ErrDirectorReplanFollowupsInvalidoV0 ||
		replanErr.Field != "agent_candidate.payload.agent_request_id" {
		t.Fatalf("error inesperado: %+v", replanErr)
	}
}

func TestBuildReplanFollowupsV0AllowsReplacementAgentWhenBlockedDiffers(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0, "replacement")
	input.CapacityCandidate = validReplanCapacityCandidateV0("replacement", input.DecisionCommandMeta.RunID)
	input.AgentCandidate = validReplanAgentCandidateV0("replacement", input.DecisionCommandMeta.RunID)
	input.BlockedAgentRefs = []string{"agent-request-ref-replan-failed"}

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0 replacement: %v", err)
	}
	assertOptionalCommandTypeV0(t, result.RequestAgentCommand, orquestacoreworkflow.OrchestrationCommandRequestAgentV0)
}

func TestBuildReplanFollowupsV0EscalateCapacitySoloConstruyeCapacitySiPayloadVieneDado(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0, "capacity")
	input.CapacityCandidate = validReplanCapacityCandidateV0("capacity", input.DecisionCommandMeta.RunID)
	input.AgentCandidate = validReplanAgentCandidateV0("capacity", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusCapacityRequestedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertOptionalCommandTypeV0(t, result.RequestCapacityCommand, orquestacoreworkflow.OrchestrationCommandRequestCapacityV0)
	if result.RequestAgentCommand != nil || result.AskDirectorCommand != nil {
		t.Fatalf("comandos secundarios inesperados: agent=%v ask=%v", result.RequestAgentCommand, result.AskDirectorCommand)
	}
}

func TestBuildReplanFollowupsV0AskDirectorSoloSiPayloadVieneDado(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionAskDirectorV0, "ask")
	input.AskDirectorCandidate = validReplanAskDirectorCandidateV0("ask", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusAskDirectorV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertOptionalCommandTypeV0(t, result.AskDirectorCommand, orquestacoreworkflow.OrchestrationCommandAskDirectorV0)
	if result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil {
		t.Fatalf("comandos secundarios inesperados: capacity=%v agent=%v",
			result.RequestCapacityCommand, result.RequestAgentCommand)
	}
}

func TestBuildReplanFollowupsV0AccionSoportadaNoAutomaticaQuedaNeedsDirector(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0, "split")
	input.CapacityCandidate = validReplanCapacityCandidateV0("split", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusNeedsDirectorUnsupportedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	if result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil || result.AskDirectorCommand != nil {
		t.Fatalf("no esperaba comandos secundarios: %+v", result)
	}
}

func TestBuildReplanFollowupsV0ReviewReworkSplitPreguntaAlDirector(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0, "review-split")
	input.SourceKind = ReplanFollowupSourceReviewReworkV0
	input.AskDirectorCandidate = validReplanAskDirectorCandidateV0("review-split", input.DecisionCommandMeta.RunID)

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0 review rework: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusAskDirectorV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertOptionalCommandTypeV0(t, result.AskDirectorCommand, orquestacoreworkflow.OrchestrationCommandAskDirectorV0)
	if result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil {
		t.Fatalf("comandos secundarios inesperados: capacity=%v agent=%v",
			result.RequestCapacityCommand, result.RequestAgentCommand)
	}
}

func TestBuildReplanFollowupsV0ValidaRunRefsAntesDeConstruir(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0, "invalid")
	input.DecisionPayload.RunRef = "run-ref-distinto"

	_, err := BuildReplanFollowupsV0(input)
	var replanErr ReplanFollowupsErrorV0
	if !errors.As(err, &replanErr) {
		t.Fatalf("error type=%T, want ReplanFollowupsErrorV0", err)
	}
	if replanErr.Code != ErrDirectorReplanFollowupsInvalidoV0 ||
		replanErr.Field != "decision_payload.run_ref" {
		t.Fatalf("error inesperado: %+v", replanErr)
	}
}

func validReplanFollowupsInputV0(
	action orquestacoreworkflow.ReplanDecisionActionV0,
	suffix string,
) ReplanFollowupsInputV0 {
	runRef := "run-ref-replan-" + suffix
	return ReplanFollowupsInputV0{
		DecisionCommandMeta: validReplanCommandMetaV0("decision-"+suffix, runRef),
		DecisionPayload: orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      "replan-ref-" + suffix,
			RunRef:         runRef,
			TaskRef:        "task-ref-replan-" + suffix,
			SourceRef:      "rework-ref-replan-" + suffix,
			AcceptedAction: action,
			FollowupRefs:   []string{"followup-ref-replan-" + suffix},
			Summary:        "decision de replan " + suffix,
			EvidenceRefs:   []string{"evidence-ref-replan-" + suffix},
		},
	}
}

func validReplanOpenPhaseCandidateV0(suffix string, runRef string) *ReplanOpenPhaseCandidateV0 {
	return &ReplanOpenPhaseCandidateV0{
		CommandMeta: validReplanCommandMetaV0("open-phase-"+suffix, runRef),
		Payload: orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "replan followup " + suffix,
		},
	}
}

func validReplanCapacityCandidateV0(suffix string, runRef string) *ReplanCapacityCandidateV0 {
	return &ReplanCapacityCandidateV0{
		CommandMeta: validReplanCommandMetaV0("capacity-"+suffix, runRef),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-request-ref-replan-" + suffix,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-replan-" + suffix,
			ReasonCode:                 "replan_followup",
			Summary:                    "capacidad explicita para replan " + suffix,
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"capacity-evidence-ref-replan-" + suffix},
		},
	}
}

func validReplanAgentCandidateV0(suffix string, runRef string) *ReplanAgentCandidateV0 {
	return &ReplanAgentCandidateV0{
		CommandMeta: validReplanCommandMetaV0("agent-"+suffix, runRef),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-request-ref-replan-" + suffix,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-replan-" + suffix,
			CapacityRequestRef: "capacity-request-ref-replan-" + suffix,
			Role:               "implementador",
			Summary:            "agente explicito para replan " + suffix,
			EvidenceRefs:       []string{"agent-evidence-ref-replan-" + suffix},
		},
	}
}

func validReplanAskDirectorCandidateV0(suffix string, runRef string) *ReplanAskDirectorCandidateV0 {
	return &ReplanAskDirectorCandidateV0{
		CommandMeta: validReplanCommandMetaV0("ask-"+suffix, runRef),
		Payload: orquestacoreworkflow.AskDirectorCommandPayloadV0{
			QuestionID:   "question-ref-replan-" + suffix,
			SourceGroup:  "orquesta-director",
			Summary:      "consulta explicita para replan " + suffix,
			Options:      []string{"aprobar", "rechazar"},
			EvidenceRefs: []string{"ask-evidence-ref-replan-" + suffix},
			Blocking:     true,
		},
	}
}

func validReplanCommandMetaV0(suffix string, runRef string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-replan-" + suffix,
		RunID:          runRef,
		IdempotencyKey: "idem-replan-" + suffix,
		CorrelationID:  "corr-replan-" + suffix,
		RequestedBy:    "orquesta-director",
		OccurredAt:     "2026-05-06T10:00:00Z",
	}
}

func assertOptionalCommandTypeV0(
	t *testing.T,
	command *orquestacoreworkflow.OrchestrationCommandV0,
	want string,
) {
	t.Helper()
	if command == nil {
		t.Fatalf("command nil, want %q", want)
	}
	assertCommandTypeV0(t, *command, want)
}

func mustDecodeCommandPayloadV0(t *testing.T, command orquestacoreworkflow.OrchestrationCommandV0, dest any) {
	t.Helper()
	if err := json.Unmarshal(command.Payload, dest); err != nil {
		t.Fatalf("decode payload %s: %v", command.CommandType, err)
	}
}
