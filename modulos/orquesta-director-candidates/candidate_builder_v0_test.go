package orquestadirectorcandidates

import (
	"testing"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildSchedulableWorkCandidateV0DesdeScopes(t *testing.T) {
	candidate, err := BuildSchedulableWorkCandidateV0(validCandidateInputV0())
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidateV0() error = %v", err)
	}

	if candidate.CandidateRef != "candidate-work-001" {
		t.Fatalf("candidate_ref = %q", candidate.CandidateRef)
	}
	if candidate.CapacityCandidate.Payload.CapacityRequestID != "capacity-candidates-001" {
		t.Fatalf("capacity_request_id = %q", candidate.CapacityCandidate.Payload.CapacityRequestID)
	}
	if candidate.AgentCandidate.Payload.AgentRequestID != "agent-candidates-001" {
		t.Fatalf("agent_request_id = %q", candidate.AgentCandidate.Payload.AgentRequestID)
	}
	if len(candidate.AgentCandidate.Payload.SkillRefs) != 1 ||
		candidate.AgentCandidate.Payload.SkillRefs[0] != "skill-ref-orquesta-programacion-autonoma-v0" {
		t.Fatalf("skill_refs = %+v", candidate.AgentCandidate.Payload.SkillRefs)
	}
	if got := candidate.Claims[0].WriteSet[0].Ref; got != "modulos/demo/main.go" {
		t.Fatalf("write scope = %q", got)
	}
}

func TestBuildSchedulableWorkCandidateV0ValidaScheduler(t *testing.T) {
	candidate, err := BuildSchedulableWorkCandidateV0(validCandidateInputV0())
	if err != nil {
		t.Fatalf("BuildSchedulableWorkCandidateV0() error = %v", err)
	}

	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(schedulerInputV0(candidate))
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0() error = %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
		t.Fatalf("status = %q", plan.Status)
	}
	if len(plan.Commands) != 1 || plan.Commands[0].CommandType != "RequestCapacity" {
		t.Fatalf("commands = %#v", plan.Commands)
	}
}

func schedulerInputV0(
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	return orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:    "tick-candidates-001",
		RunRef:     "run-candidates-001",
		OccurredAt: "2026-05-06T11:00:00Z",
		Snapshot: orquestadirectorscheduler.RunSchedulingSnapshotV0{
			RunRef:         "run-candidates-001",
			CurrentPhaseID: candidate.CapacityCandidate.Payload.PhaseID,
		},
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{candidate},
		EvidenceRefs:   []string{"evidence-tick-001"},
	}
}
