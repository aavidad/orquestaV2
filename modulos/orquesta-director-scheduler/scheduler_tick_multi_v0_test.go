package orquestadirectorscheduler

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0RequestsCapacityForMultipleCandidates(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{
		validSchedulableCandidateV0(),
		schedulableCandidateVariantV0("002", "modulos/demo/worker.go"),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
	assertSchedulerCapacityRefV0(t, plan.Commands[0], "capacity-ref-scheduler-001")
	assertSchedulerCapacityRefV0(t, plan.Commands[1], "capacity-ref-scheduler-002")
}

func TestBuildDirectorSchedulerTickV0BuildsAgentsForMultipleDecidedCandidates(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{
		validSchedulableCandidateV0(),
		schedulableCandidateVariantV0("002", "modulos/demo/worker.go"),
	}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001", "capacity-ref-scheduler-002"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 4)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[1], "agent-ref-scheduler-001")
	assertSchedulerAgentRefV0(t, plan.Commands[3], "agent-ref-scheduler-002")
}

func TestBuildDirectorSchedulerTickV0SchedulesReadyCandidateWhileOtherAgentInFlight(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{
		schedulableCandidateVariantV0("002", "modulos/demo/worker.go"),
	}
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-002"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[1], "agent-ref-scheduler-002")
}

func TestBuildDirectorSchedulerTickV0SkipsPendingCandidateAndKeepsReadyCommands(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{
		validSchedulableCandidateV0(),
		schedulableCandidateVariantV0("002", "modulos/demo/worker.go"),
	}
	input.Snapshot.CapacityRequests = []string{"capacity-ref-scheduler-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-002"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[1], "agent-ref-scheduler-002")
	if len(plan.WaitingReasons) != 0 {
		t.Fatalf("waiting_reasons=%v", plan.WaitingReasons)
	}
}

func TestBuildDirectorSchedulerTickV0ConflictDoesNotBlockSafeCandidate(t *testing.T) {
	input := validSchedulerTickInputWithConflictV0()
	input.WorkCandidates = append(input.WorkCandidates, schedulableCandidateVariantV0("002", "modulos/demo/worker.go"))
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001", "capacity-ref-scheduler-002"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 3)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[2], "agent-ref-scheduler-002")
	if len(plan.BlockedRefs) == 0 {
		t.Fatalf("blocked_refs empty")
	}
}

func TestBuildDirectorSchedulerTickV0UsaWorkClaimsCompartidosParaConflictos(t *testing.T) {
	first := validSchedulableCandidateV0()
	second := schedulableCandidateVariantV0("conflict", "modulos/demo")
	input := validSchedulerTickInputV0()
	input.WorkClaims = append(first.Claims, second.Claims...)
	input.WorkCandidates = []SchedulableWorkCandidateV0{first, second}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-001", "capacity-ref-scheduler-conflict"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusBlockedV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
	)
	if len(plan.BlockedRefs) != 2 {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}
}

func assertSchedulerCommandTypesV0(
	t *testing.T,
	plan DirectorSchedulerTickPlanV0,
	want ...string,
) {
	t.Helper()
	if len(plan.Commands) != len(want) {
		t.Fatalf("commands=%d, want %d", len(plan.Commands), len(want))
	}
	for index, commandType := range want {
		if plan.Commands[index].CommandType != commandType {
			t.Fatalf("command[%d]=%s, want %s", index, plan.Commands[index].CommandType, commandType)
		}
	}
}

func assertSchedulerCapacityRefV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	want string,
) {
	t.Helper()
	var payload orquestacoreworkflow.RequestCapacityCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode capacity payload: %v", err)
	}
	if payload.CapacityRequestID != want {
		t.Fatalf("capacity_request_id=%s, want %s", payload.CapacityRequestID, want)
	}
}

func assertSchedulerAgentRefV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	want string,
) {
	t.Helper()
	var payload orquestacoreworkflow.RequestAgentCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode agent payload: %v", err)
	}
	if payload.AgentRequestID != want {
		t.Fatalf("agent_request_id=%s, want %s", payload.AgentRequestID, want)
	}
}
