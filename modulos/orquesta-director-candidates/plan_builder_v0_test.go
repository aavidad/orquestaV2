package orquestadirectorcandidates

import (
	"errors"
	"testing"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildSchedulableWorkCandidatesFromPlanV0DosMicrotareas(t *testing.T) {
	candidates, err := BuildSchedulableWorkCandidatesFromPlanV0(validPlanInputV0())
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidatesFromPlanV0() error = %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("candidates=%d, want 2", len(candidates))
	}
	if candidates[0].CandidateRef != "candidate-work-001" ||
		candidates[1].CandidateRef != "candidate-work-002" {
		t.Fatalf("candidate refs = %q %q", candidates[0].CandidateRef, candidates[1].CandidateRef)
	}
	if candidates[0].Claims[0].WriteSet[0].Ref == candidates[1].Claims[0].WriteSet[0].Ref {
		t.Fatalf("write scopes no independientes: %#v", candidates)
	}
}

func TestBuildSchedulableWorkCandidatesFromPlanV0RechazaMissingIDs(t *testing.T) {
	input := validPlanInputV0()
	input.WorkItems[0].Capacity.CapacityRequestID = " "

	_, err := BuildSchedulableWorkCandidatesFromPlanV0(input)
	assertCandidateFieldErrorV0(t, err, "capacity.capacity_request_id")
}

func TestBuildSchedulableWorkCandidatesFromPlanV0RechazaScopesInvalidos(t *testing.T) {
	input := validPlanInputV0()
	input.WorkItems[0].ScopeClaims[0].WriteScopes = []string{"../fuera"}

	_, err := BuildSchedulableWorkCandidatesFromPlanV0(input)
	assertCandidateFieldErrorV0(t, err, "scope_claims.write_scopes")
}

func TestBuildSchedulableWorkCandidatesFromPlanV0ValidaScheduler(t *testing.T) {
	candidates, err := BuildSchedulableWorkCandidatesFromPlanV0(validPlanInputV0())
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidatesFromPlanV0() error = %v", err)
	}
	candidate := candidates[0]
	input := schedulerInputV0(candidate)
	input.WorkCandidates = candidates

	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(input)
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0() error = %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
		t.Fatalf("status=%q", plan.Status)
	}
	if len(plan.Commands) != 2 {
		t.Fatalf("commands=%d, want 2", len(plan.Commands))
	}
}

func validPlanInputV0() CompactBacklogPlanCandidatesInputV0 {
	base := validCandidateInputV0()
	return CompactBacklogPlanCandidatesInputV0{
		RunRef:  base.RunRef,
		PhaseID: base.PhaseID,
		WorkItems: []CompactBacklogPlanWorkItemV0{
			planItemFromCandidateInputV0(base),
			planItemFromCandidateInputV0(planCandidateVariantV0(
				validCandidateInputV0(),
				"002",
				"modulos/demo/worker.go",
			)),
		},
	}
}

func planCandidateVariantV0(
	base SchedulableWorkCandidateInputV0,
	suffix string,
	scope string,
) SchedulableWorkCandidateInputV0 {
	base.CandidateRef = "candidate-work-" + suffix
	base.TaskRef = "task-candidates-" + suffix
	base.SubjectClaimRefs = []string{"claim-candidates-" + suffix}
	base.ScopeClaims[0].ClaimRef = "claim-candidates-" + suffix
	base.ScopeClaims[0].AgentRequestID = "agent-candidates-" + suffix
	base.ScopeClaims[0].WriteScopes = []string{scope}
	base.Commands.CapacityCommandID = "cmd-capacity-candidates-" + suffix
	base.Commands.CapacityIdempotencyKey = "idem-capacity-candidates-" + suffix
	base.Commands.GateCommandID = "cmd-gate-candidates-" + suffix
	base.Commands.GateIdempotencyKey = "idem-gate-candidates-" + suffix
	base.Commands.AgentCommandID = "cmd-agent-candidates-" + suffix
	base.Commands.AgentIdempotencyKey = "idem-agent-candidates-" + suffix
	base.Capacity.CapacityRequestID = "capacity-candidates-" + suffix
	base.Agent.AgentRequestID = "agent-candidates-" + suffix
	base.Agent.ClaimRef = "claim-candidates-" + suffix
	return base
}

func planItemFromCandidateInputV0(input SchedulableWorkCandidateInputV0) CompactBacklogPlanWorkItemV0 {
	return CompactBacklogPlanWorkItemV0{
		CandidateRef:     input.CandidateRef,
		TaskRef:          input.TaskRef,
		SubjectClaimRefs: input.SubjectClaimRefs,
		ScopeClaims:      input.ScopeClaims,
		Commands:         input.Commands,
		Capacity:         input.Capacity,
		Agent:            input.Agent,
		GateEvidenceRefs: input.GateEvidenceRefs,
		EvidenceRefs:     input.EvidenceRefs,
	}
}

func assertCandidateFieldErrorV0(t *testing.T, err error, field string) {
	t.Helper()
	var candidateErr DirectorCandidateErrorV0
	if !errors.As(err, &candidateErr) {
		t.Fatalf("error type = %T", err)
	}
	if candidateErr.Field != field {
		t.Fatalf("field=%q, want %q", candidateErr.Field, field)
	}
}
