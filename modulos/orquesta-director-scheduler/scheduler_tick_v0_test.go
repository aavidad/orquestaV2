package orquestadirectorscheduler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0RequestsCapacityFirst(t *testing.T) {
	input := validSchedulerTickInputV0()

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	if plan.Commands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRequestCapacityV0 {
		t.Fatalf("command_type=%s", plan.Commands[0].CommandType)
	}
}

func TestBuildDirectorSchedulerTickV0WaitsWhenCapacityPending(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.CapacityRequests = []string{"capacity-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingCapacityPendingV0)
}

func TestBuildDirectorSchedulerTickV0BuildsGateAndAgentAfterCapacity(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	if plan.Commands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0 {
		t.Fatalf("first command=%s", plan.Commands[0].CommandType)
	}
	if plan.Commands[1].CommandType != orquestacoreworkflow.OrchestrationCommandRequestAgentV0 {
		t.Fatalf("second command=%s", plan.Commands[1].CommandType)
	}
}

func TestBuildDirectorSchedulerTickV0DoesNotRepeatRecordedGate(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001"}
	firstPlan := mustSchedulerTickPlanV0(t, input)
	input.Snapshot.ConcurrencyGates = []string{schedulerGateRefFromPlanV0(t, firstPlan)}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	if plan.Commands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRequestAgentV0 {
		t.Fatalf("command_type=%s", plan.Commands[0].CommandType)
	}
}

func TestBuildDirectorSchedulerTickV0WaitsWithPendingOutbox(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.PendingOutboxRefs = []string{"outbox-ref-launch-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingOutboxPendingV0)
}

func TestBuildDirectorSchedulerTickV0DoesNotRepeatRequestedAgent(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001"}
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentLifecyclePendingV0)
}

func TestBuildDirectorSchedulerTickV0WaitsWhenStartedAgentHasNoDelivery(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentDeliveryPendingV0)
}

func TestBuildDirectorSchedulerTickV0QuiescentWhenStartedAgentDelivered(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001"}
	input.Snapshot.Deliveries = []string{"delivery-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0BlocksConflictingClaim(t *testing.T) {
	input := validSchedulerTickInputWithConflictV0()
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	if plan.Status != SchedulerTickStatusBlockedV0 {
		t.Fatalf("status=%s", plan.Status)
	}
	if len(plan.Commands) != 1 || plan.Commands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0 {
		t.Fatalf("commands=%v", plan.Commands)
	}
	if len(plan.BlockedRefs) == 0 {
		t.Fatalf("blocked_refs empty")
	}
}

func TestBuildDirectorSchedulerTickV0AcceptsOpaqueEvidenceRefsWithOperationalTokens(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.EvidenceRefs = []string{"runtime/process/session-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
}

func TestBuildDirectorSchedulerTickV0AcceptsOpaqueRefsWithAppTokens(t *testing.T) {
	input := validSchedulerTickInputV0()
	claimRef := "claim-ref-db-admin/model-viewer-001"
	agentRef := "agent-ref-db-admin/model-viewer-001"
	input.WorkCandidates[0].CandidateRef = "candidate-ref-db-admin/model-viewer-001"
	input.WorkCandidates[0].SubjectClaimRefs = []string{claimRef}
	input.WorkCandidates[0].Claims[0].ClaimRef = claimRef
	input.WorkCandidates[0].Claims[0].AgentRequestID = agentRef
	input.WorkCandidates[0].AgentCandidate.ClaimRef = claimRef
	input.WorkCandidates[0].AgentCandidate.Payload.AgentRequestID = agentRef

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
}

func TestBuildDirectorSchedulerTickV0AceptaSetentaAgentesCompactos(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	for i := 0; i < 70; i++ {
		ref := "agent-ref-scheduler-parallel-" + strconv.Itoa(i)
		input.Snapshot.Agents = append(input.Snapshot.Agents, ref)
		input.Snapshot.StartedAgents = append(input.Snapshot.StartedAgents, ref)
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentDeliveryPendingV0)
}

func TestBuildDirectorSchedulerTickV0AceptaRefsDeArtefactosSinCortar(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.EvidenceRefs = []string{
		"README.md",
		"docs/arquitectura.md",
		"artifact-ref-" + strings.Repeat("x", 900),
	}
	input.Snapshot.PendingOutboxRefs = []string{"README.md"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingOutboxPendingV0)
}

func TestBuildDirectorSchedulerTickV0RejectsForbiddenPayloadDetails(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*DirectorSchedulerTickInputV0)
	}{
		{
			name: "summary_provider",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].CapacityCandidate.Payload.Summary = "provider operativo seleccionado"
			},
		},
		{
			name: "summary_model",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].AgentCandidate.Payload.Summary = "model operativo seleccionado"
			},
		},
		{
			name: "reason_home",
			mutate: func(input *DirectorSchedulerTickInputV0) {
				input.WorkCandidates[0].CapacityCandidate.Payload.ReasonCode = "home requerido"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validSchedulerTickInputV0()
			tt.mutate(&input)

			_, err := BuildDirectorSchedulerTickV0(input)
			assertSchedulerTickErrorV0(t, err, "payload")
		})
	}
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunCandidate(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates[0].AgentCandidate.CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)
	assertSchedulerTickErrorV0(t, err, "work_candidates.agent_candidate.command_meta.run_id")
}

func mustSchedulerTickPlanV0(t *testing.T, input DirectorSchedulerTickInputV0) DirectorSchedulerTickPlanV0 {
	t.Helper()
	plan, err := BuildDirectorSchedulerTickV0(input)
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0: %v", err)
	}
	return plan
}

func assertSchedulerPlanV0(t *testing.T, plan DirectorSchedulerTickPlanV0, status DirectorSchedulerTickStatusV0, commands int) {
	t.Helper()
	if plan.Status != status {
		t.Fatalf("status=%s, want %s", plan.Status, status)
	}
	if len(plan.Commands) != commands {
		t.Fatalf("commands=%d, want %d", len(plan.Commands), commands)
	}
}

func assertSchedulerWaitingV0(t *testing.T, plan DirectorSchedulerTickPlanV0, reason SchedulerWaitingReasonV0) {
	t.Helper()
	if len(plan.WaitingReasons) != 1 || plan.WaitingReasons[0] != reason {
		t.Fatalf("waiting_reasons=%v, want %s", plan.WaitingReasons, reason)
	}
}

func assertSchedulerTickErrorV0(t *testing.T, err error, field string) {
	t.Helper()
	var publicErr DirectorSchedulerTickErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected DirectorSchedulerTickErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != ErrDirectorSchedulerTickInvalidoV0 || publicErr.Field != field {
		t.Fatalf("error=%+v", publicErr)
	}
}

func schedulerGateRefFromPlanV0(t *testing.T, plan DirectorSchedulerTickPlanV0) string {
	t.Helper()
	var payload orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0
	if err := json.Unmarshal(plan.Commands[0].Payload, &payload); err != nil {
		t.Fatalf("decode gate payload: %v", err)
	}
	return payload.GateRef
}
