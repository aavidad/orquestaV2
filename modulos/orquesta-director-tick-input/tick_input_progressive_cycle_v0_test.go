package orquestadirectortickinput

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildDirectorSchedulerTickInputV0CiclosProgresivosProgramacion(t *testing.T) {
	workflow := &tickInputMemoryWorkflowPortV0{run: tickInputProgramacionRunV0(t)}

	first := tickInputRunCycleV0(t, workflow, "tick-ref-progressive-001")
	if first.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		first.AppliedCommands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRequestCapacityV0 {
		t.Fatalf("first cycle unexpected: %+v", first)
	}
	if !tickInputContainsRefV0(workflow.run.CapacityRequests, "capacity-ref-tick-input-001") {
		t.Fatalf("capacity not reflected: %+v", workflow.run.CapacityRequests)
	}

	tickInputApplyWorkflowCommandV0(t, workflow, tickInputCapacityDecisionCommandV0(t))
	second := tickInputRunCycleV0(t, workflow, "tick-ref-progressive-002")
	if second.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		len(second.AppliedCommands) != 2 {
		t.Fatalf("second cycle unexpected: %+v", second)
	}
	if second.AppliedCommands[0].CommandType != orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0 ||
		second.AppliedCommands[1].CommandType != orquestacoreworkflow.OrchestrationCommandRequestAgentV0 {
		t.Fatalf("second commands unexpected: %+v", second.AppliedCommands)
	}
	if !tickInputContainsRefV0(workflow.run.Agents, "agent-ref-tick-input-001") {
		t.Fatalf("agent not reflected: %+v", workflow.run.Agents)
	}

	tickInputApplyWorkflowCommandV0(t, workflow, tickInputAgentStartedCommandV0(t))
	third := tickInputRunCycleV0(t, workflow, "tick-ref-progressive-003")
	if third.Status != orquestadirectorrunner.DirectorCycleStatusWaitingV0 ||
		len(third.AppliedCommands) != 0 ||
		len(third.WaitingReasons) != 1 ||
		third.WaitingReasons[0] != orquestadirectorscheduler.SchedulerWaitingAgentDeliveryPendingV0 {
		t.Fatalf("third cycle unexpected: %+v", third)
	}
}

func tickInputRunCycleV0(
	t *testing.T,
	workflow *tickInputMemoryWorkflowPortV0,
	tickRef string,
) orquestadirectorrunner.DirectorCycleResultV0 {
	t.Helper()
	schedulerInput, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:        tickRef,
		OccurredAt:     "2026-05-06T12:10:00Z",
		Run:            workflow.run,
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{tickInputWorkCandidateV0(workflow.run.RunID)},
		EvidenceRefs:   []string{"evidence-ref-progressive-001"},
	})
	if err != nil {
		t.Fatalf("build scheduler input: %v", err)
	}
	result, err := orquestadirectorrunner.RunDirectorCycleV0(context.Background(), orquestadirectorrunner.DirectorCycleInputV0{
		Scheduler:      orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		CycleRef:       "cycle-ref-" + tickRef,
		RunRef:         workflow.run.RunID,
		SchedulerInput: schedulerInput,
	})
	if err != nil {
		t.Fatalf("run cycle: %v", err)
	}
	return result
}

func tickInputApplyWorkflowCommandV0(
	t *testing.T,
	workflow *tickInputMemoryWorkflowPortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	if _, err := workflow.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		t.Fatalf("apply workflow command %s: %v", command.CommandType, err)
	}
}

func tickInputCapacityDecisionCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		tickInputCommandMetaV0("cmd-tick-input-capacity-decision", "capacity-decision"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: "capacity-ref-tick-input-001",
			DecisionRef:       "capacity-decision-ref-tick-input-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityMediumV0,
			Summary:           "Decision compacta de capacidad.",
			EvidenceRefs:      []string{"evidence-ref-capacity-decision-001"},
		},
	)
	if err != nil {
		t.Fatalf("capacity decision command: %v", err)
	}
	return command
}

func tickInputAgentStartedCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRegisterAgentStartedCommandV0(
		tickInputCommandMetaV0("cmd-tick-input-agent-started", "agent-started"),
		orquestacoreworkflow.RegisterAgentStartedCommandPayloadV0{
			AgentRequestID: "agent-ref-tick-input-001",
			LaunchRef:      "launch-ref-tick-input-001",
			AckRef:         "ack-ref-tick-input-001",
			ReadinessRef:   "readiness-ref-tick-input-001",
			EvidenceRefs:   []string{"evidence-ref-agent-started-001"},
		},
	)
	if err != nil {
		t.Fatalf("agent started command: %v", err)
	}
	return command
}
