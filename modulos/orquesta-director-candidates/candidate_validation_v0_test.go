package orquestadirectorcandidates

import (
	"errors"
	"testing"
)

func TestBuildSchedulableWorkCandidateV0RechazaIdempotencyVacia(t *testing.T) {
	input := validCandidateInputV0()
	input.Commands.AgentIdempotencyKey = " "

	_, err := BuildSchedulableWorkCandidateV0(input)
	if err == nil {
		t.Fatal("BuildSchedulableWorkCandidateV0() error = nil")
	}
	var candidateErr DirectorCandidateErrorV0
	if !errors.As(err, &candidateErr) {
		t.Fatalf("error type = %T", err)
	}
	if candidateErr.Field != "commands.agent_idempotency_key" {
		t.Fatalf("field = %q", candidateErr.Field)
	}
}

func TestBuildSchedulableWorkCandidateV0RechazaSinClaims(t *testing.T) {
	input := validCandidateInputV0()
	input.ScopeClaims = nil

	_, err := BuildSchedulableWorkCandidateV0(input)
	if err == nil {
		t.Fatal("BuildSchedulableWorkCandidateV0() error = nil")
	}
}
