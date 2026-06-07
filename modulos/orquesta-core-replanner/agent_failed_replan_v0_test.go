package orquestacorereplanner

import (
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAgentFailedToReplanProposalV0ReplaceAgentValid(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionReplaceAgentV0)

	proposal := mustAgentFailedReplanProposalV0(t, input)
	if proposal.RecommendedAction != ReplanActionReplaceAgentV0 {
		t.Fatalf("recommended_action=%q", proposal.RecommendedAction)
	}
	if proposal.ReplacementRole != input.ReplacementRole {
		t.Fatalf("replacement_role=%q, want %q", proposal.ReplacementRole, input.ReplacementRole)
	}
	assertAgentFailedReplanProposalTraceV0(t, proposal, input)
}

func TestAgentFailedToReplanProposalV0AskDirectorValid(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionAskDirectorV0)
	input.ReasonRef = ""
	input.ReplacementRole = ""

	proposal := mustAgentFailedReplanProposalV0(t, input)
	if proposal.RecommendedAction != ReplanActionAskDirectorV0 {
		t.Fatalf("recommended_action=%q", proposal.RecommendedAction)
	}
	if proposal.ReplacementRole != "" {
		t.Fatalf("replacement_role must stay empty: %#v", proposal)
	}
	if proposal.ReasonCode != "agent_failed_launch_failed" {
		t.Fatalf("reason_code=%q", proposal.ReasonCode)
	}
	assertAgentFailedReplanProposalTraceV0(t, proposal, input)
}

func TestAgentFailedToReplanProposalV0RejectsMissingReplacementRole(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionReplaceAgentV0)
	input.ReplacementRole = " "

	_, err := AgentFailedToReplanProposalV0(input)
	assertAgentFailedReplanSignalErrorV0(t, err, ErrAgentFailedReplanSignalInvalidoV0, "replacement_role")
}

func TestAgentFailedToReplanProposalV0RejectsUnsupportedAction(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionRetryTaskV0)

	_, err := AgentFailedToReplanProposalV0(input)
	assertAgentFailedReplanSignalErrorV0(t, err, ErrAgentFailedReplanActionNoSoportadaV0, "requested_action")
}

func TestAgentFailedToReplanProposalV0AllowsOperationalLabels(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionAskDirectorV0)
	input.Summary = "Incluye runtime provider ref externo."

	_, err := AgentFailedToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("AgentFailedToReplanProposalV0: %v", err)
	}
}

func TestAgentFailedToReplanProposalV0RejectsSensitiveDetails(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionAskDirectorV0)
	input.Summary = "client_secret=valor"

	_, err := AgentFailedToReplanProposalV0(input)
	assertAgentFailedReplanSignalErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func TestAgentFailedToReplanProposalV0RejectsFailedAgentAsReplacement(t *testing.T) {
	input := validAgentFailedReplanInputV0(ReplanActionReplaceAgentV0)
	input.ReplacementRole = input.AgentFailed.AgentRequestID

	_, err := AgentFailedToReplanProposalV0(input)
	assertAgentFailedReplanSignalErrorV0(t, err, ErrAgentFailedReplanSignalInvalidoV0, "replacement_role")
}

func validAgentFailedReplanInputV0(action ReplanRecommendedActionV0) AgentFailedReplanInputV0 {
	return AgentFailedReplanInputV0{
		ReplanRef:       "replan-agent-failed-001",
		SignalRef:       "agent-failed-signal-001",
		RunRef:          "run-001",
		TaskRef:         "task-001",
		RequestedAction: action,
		ReplacementRole: "implementer",
		ReasonRef:       "agent_failed_replacement",
		Summary:         "Fallo compacto de lanzamiento requiere replanificacion.",
		EvidenceRefs:    []string{"decision-context-001"},
		AgentFailed: orquestacoreworkflow.AgentFailedPayloadV0{
			AgentRequestID: "agent-request-001",
			LaunchRef:      "launch-001",
			ReasonCode:     "launch_failed",
			Retryable:      true,
			EvidenceRefs:   []string{"launch-evidence-001"},
		},
	}
}

func mustAgentFailedReplanProposalV0(t *testing.T, input AgentFailedReplanInputV0) ReplanProposalV0 {
	t.Helper()
	proposal, err := AgentFailedToReplanProposalV0(input)
	if err != nil {
		t.Fatalf("AgentFailedToReplanProposalV0: %v", err)
	}
	return proposal
}

func assertAgentFailedReplanProposalTraceV0(t *testing.T, proposal ReplanProposalV0, input AgentFailedReplanInputV0) {
	t.Helper()
	if proposal.ReplanRef != input.ReplanRef ||
		proposal.RunRef != input.RunRef ||
		proposal.TaskRef != input.TaskRef ||
		proposal.SourceRef != input.AgentFailed.AgentRequestID {
		t.Fatalf("proposal trace mismatch: %#v input=%#v", proposal, input)
	}
	if proposal.CapacityRequestRef != "" {
		t.Fatalf("capacity_request_ref must stay empty: %#v", proposal)
	}
	if len(proposal.EvidenceRefs) != 4 || proposal.EvidenceRefs[0] != input.SignalRef {
		t.Fatalf("evidence_refs=%v", proposal.EvidenceRefs)
	}
}

func assertAgentFailedReplanSignalErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr AgentFailedReplanSignalErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected AgentFailedReplanSignalErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}
