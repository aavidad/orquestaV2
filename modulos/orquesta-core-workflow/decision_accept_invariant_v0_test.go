package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestAcceptDecisionCommandV0RejectsMissingVoteRef(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	command := mustAcceptDecisionCommandV0(t, "cmd-decision-missing-vote", "idem-decision-missing-vote", "decision-missing-vote")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestArchitectureDecisionAcceptedEventV0RejectsMissingVoteRef(t *testing.T) {
	run := mustVoteActiveRunV0(t)
	event := mustArchitectureDecisionAcceptedEventV0(t, "evt-decision-missing-vote", run.LastSequence+1, "decision-missing-vote")

	_, err := ApplyEventV0(run, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrSecuenciaInvalidaV0)
	}
}
