package orquestadirector

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildPostLeaseActionV0StopAgentConstruyeComandosSeparados(t *testing.T) {
	h := newSupervisionHarnessWithAgentV0(t, "task-ref-post-lease-stop-001", "agent-request-ref-post-lease-stop-001", "post-lease-stop")

	result, err := BuildPostLeaseActionV0(validPostLeaseActionInputV0(
		h,
		"post-lease-stop",
		"agent-request-ref-post-lease-stop-001",
		orquestacoreworkflow.AgentLeaseActionStopAgentV0,
	))
	if err != nil {
		t.Fatalf("BuildPostLeaseActionV0: %v", err)
	}
	if result.FollowupStatus != PostLeaseFollowupStatusStopAgentV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	if result.StopAgentCommand == nil || result.AskDirectorCommand != nil {
		t.Fatalf("comandos secundarios inesperados: stop=%v ask=%v", result.StopAgentCommand, result.AskDirectorCommand)
	}
	assertCommandTypeV0(t, result.RegisterLeaseExpiredCommand, orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0)
	assertCommandTypeV0(t, *result.StopAgentCommand, orquestacoreworkflow.OrchestrationCommandStopAgentV0)

	lease := h.handle(result.RegisterLeaseExpiredCommand)
	if len(lease.Events) != 1 || len(lease.Outbox) != 0 {
		t.Fatalf("RegisterAgentLeaseExpired efectos inesperados: events=%v outbox=%v", lease.Events, lease.Outbox)
	}
	stop := h.handle(*result.StopAgentCommand)
	if len(stop.Events) != 1 || len(stop.Outbox) != 1 ||
		stop.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("StopAgent efectos inesperados: events=%v outbox=%v", stop.Events, stop.Outbox)
	}
}

func TestBuildPostLeaseActionV0AskDirectorConstruyeComandoSeparado(t *testing.T) {
	h := newSupervisionHarnessWithAgentV0(t, "task-ref-post-lease-ask-001", "agent-request-ref-post-lease-ask-001", "post-lease-ask")
	input := validPostLeaseActionInputV0(
		h,
		"post-lease-ask",
		"agent-request-ref-post-lease-ask-001",
		orquestacoreworkflow.AgentLeaseActionAskDirectorV0,
	)
	input.QuestionID = "question-ref-post-lease-ask-001"

	result, err := BuildPostLeaseActionV0(input)
	if err != nil {
		t.Fatalf("BuildPostLeaseActionV0: %v", err)
	}
	if result.FollowupStatus != PostLeaseFollowupStatusAskDirectorV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	if result.AskDirectorCommand == nil || result.StopAgentCommand != nil {
		t.Fatalf("comandos secundarios inesperados: ask=%v stop=%v", result.AskDirectorCommand, result.StopAgentCommand)
	}
	assertCommandTypeV0(t, result.RegisterLeaseExpiredCommand, orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0)
	assertCommandTypeV0(t, *result.AskDirectorCommand, orquestacoreworkflow.OrchestrationCommandAskDirectorV0)

	h.handle(result.RegisterLeaseExpiredCommand)
	asked := h.handle(*result.AskDirectorCommand)
	if len(asked.Outbox) != 1 ||
		asked.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0 {
		t.Fatalf("AskDirector outbox inesperado: %+v", asked.Outbox)
	}
}

func TestBuildPostLeaseActionV0AccionNoSoportadaQuedaNeedsDirector(t *testing.T) {
	h := newSupervisionHarnessWithAgentV0(t, "task-ref-post-lease-retry-001", "agent-request-ref-post-lease-retry-001", "post-lease-retry")

	result, err := BuildPostLeaseActionV0(validPostLeaseActionInputV0(
		h,
		"post-lease-retry",
		"agent-request-ref-post-lease-retry-001",
		orquestacoreworkflow.AgentLeaseActionRetryV0,
	))
	if err != nil {
		t.Fatalf("BuildPostLeaseActionV0: %v", err)
	}
	if result.FollowupStatus != PostLeaseFollowupStatusUnsupportedNeedsDirectorV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	if result.StopAgentCommand != nil || result.AskDirectorCommand != nil {
		t.Fatalf("no esperaba comando secundario: stop=%v ask=%v", result.StopAgentCommand, result.AskDirectorCommand)
	}
	handled := h.handle(result.RegisterLeaseExpiredCommand)
	if len(handled.Events) != 1 || len(handled.Outbox) != 0 {
		t.Fatalf("efectos inesperados: events=%v outbox=%v", handled.Events, handled.Outbox)
	}
}

func TestBuildPostLeaseActionV0ValidaInputYSaneaSalida(t *testing.T) {
	h := newProgressiveHarnessV0(t)
	input := validPostLeaseActionInputV0(
		h,
		"post-lease-invalid",
		"agent-request-ref-post-lease-invalid-001",
		orquestacoreworkflow.AgentLeaseActionAskDirectorV0,
	)
	input.QuestionID = ""

	_, err := BuildPostLeaseActionV0(input)
	var postLeaseErr PostLeaseActionErrorV0
	if !errors.As(err, &postLeaseErr) {
		t.Fatalf("error type=%T, want PostLeaseActionErrorV0", err)
	}
	if postLeaseErr.Code != ErrDirectorPostLeaseActionInvalidaV0 || postLeaseErr.Field != "question_id" {
		t.Fatalf("error inesperado: %+v", postLeaseErr)
	}

	input.RecommendedAction = orquestacoreworkflow.AgentLeaseActionStopAgentV0
	input.QuestionID = ""
	input.EvidenceRefs = []string{"post-lease-evidence-ref-safe-001", "$HOME/transcript-ref-001"}
	result, err := BuildPostLeaseActionV0(input)
	if err != nil {
		t.Fatalf("BuildPostLeaseActionV0 safe: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"home", "transcript", "provider", "oauth", "token", "prompt"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized result contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
}

func validPostLeaseActionInputV0(
	h *progressiveHarnessV0,
	suffix string,
	agentRequestID string,
	action orquestacoreworkflow.AgentLeaseRecommendedActionV0,
) PostLeaseActionInputV0 {
	return PostLeaseActionInputV0{
		CommandMeta:       h.meta(suffix),
		RunRef:            h.run.RunID,
		AgentRequestID:    agentRequestID,
		LeaseRef:          "lease-ref-" + suffix,
		ReasonCode:        "heartbeat_timeout",
		ObservedAt:        "2026-05-04T10:00:00Z",
		RecommendedAction: action,
		EvidenceRefs:      []string{"post-lease-evidence-ref-" + suffix},
	}
}

func assertCommandTypeV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	want string,
) {
	t.Helper()
	if command.CommandType != want {
		t.Fatalf("command_type=%q, want %q", command.CommandType, want)
	}
}
