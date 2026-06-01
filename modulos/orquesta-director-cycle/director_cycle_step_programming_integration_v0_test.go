package orquestadirectorcycle

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestExecuteDirectorCycleStepV0ConCandidatesRealesProgramacion(t *testing.T) {
	workflow := newCycleStepWorkflowV0(t)
	ledger := newCycleStepLedgerAdapterV0()
	first := cycleStepProgrammingCandidateV0(workflow.run.RunID, "001", "modulos/demo/main.go")
	second := cycleStepProgrammingCandidateV0(workflow.run.RunID, "002", "modulos/demo/worker.go")
	cycleStepSeedCapacityDecisionForCandidateV0(t, workflow, first, "001")
	cycleStepSeedCapacityDecisionForCandidateV0(t, workflow, second, "002")
	input := validCycleStepInputV0(workflow, ledger, "cycle-ref-step-programming", "tick-ref-step-programming")
	input.WorkCandidates = []orquestadirectorscheduler.SchedulableWorkCandidateV0{first, second}
	input.WorkClaims = append([]orquestacoreconcurrency.WorksetClaimV0{}, first.Claims...)
	input.WorkClaims = append(input.WorkClaims, second.Claims...)
	input.MaxCommands = 4
	input.MaxOutbox = 2

	result, err := ExecuteDirectorCycleStepV0(context.Background(), input)
	if err != nil {
		t.Fatalf("cycle step programming candidates: %v", err)
	}

	if result.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		result.SchedulerStatus != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 {
		t.Fatalf("status inesperado: %+v", result)
	}
	assertCycleStepAppliedCommandTypesV0(t, result,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	if result.EventsCount != 4 || result.OutboxSavedCount != 2 || result.OutboxPendingAfterCount != 2 {
		t.Fatalf("counters: %+v", result)
	}
	if !cycleStepContainsRefV0(workflow.run.Agents, "agent-ref-cycle-step-programming-001") ||
		!cycleStepContainsRefV0(workflow.run.Agents, "agent-ref-cycle-step-programming-002") {
		t.Fatalf("agents no reflejados: %+v", workflow.run.Agents)
	}
	if len(workflow.run.ConcurrencyGates) != 2 {
		t.Fatalf("concurrency_gates=%+v", workflow.run.ConcurrencyGates)
	}

	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: workflow.run.RunID,
	})
	if len(issues) != 0 {
		t.Fatalf("ledger issues: %+v", issues)
	}
	if len(pending) != 2 {
		t.Fatalf("pending=%d want 2: %+v", len(pending), pending)
	}
	for _, message := range pending {
		if message.MessageType != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 ||
			message.TargetPort != orquestacoreworkflow.OutboxTargetAgentLauncherV0 {
			t.Fatalf("outbox inesperado: %+v", message)
		}
		var payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			t.Fatalf("decode launch payload: %v", err)
		}
		if !cycleStepContainsRefV0([]string{"agent-ref-cycle-step-programming-001", "agent-ref-cycle-step-programming-002"}, payload.AgentRequestID) {
			t.Fatalf("agent_request_id inesperado: %+v", payload)
		}
	}
	if !reflect.DeepEqual(result.PendingOutboxAfterRefs, []string{
		"outbox-launchruntimeagent-idem-cycle-step-agent-programming-001",
		"outbox-launchruntimeagent-idem-cycle-step-agent-programming-002",
	}) {
		t.Fatalf("pending refs=%+v", result.PendingOutboxAfterRefs)
	}
}

func cycleStepProgrammingCandidateV0(
	runRef string,
	suffix string,
	writeRef string,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	claimRef := "claim-ref-cycle-step-programming-" + suffix
	agentRef := "agent-ref-cycle-step-programming-" + suffix
	capacityRef := "capacity-ref-cycle-step-programming-" + suffix
	taskRef := "task-ref-cycle-step-programming-" + suffix
	claim := cycleStepClaimWithRefsV0(runRef, claimRef, taskRef, agentRef, writeRef)
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-ref-cycle-step-programming-" + suffix,
		SubjectClaimRefs: []string{claimRef},
		Claims:           []orquestacoreconcurrency.WorksetClaimV0{claim},
		CapacityCandidate: &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
			CommandMeta: cycleStepCommandMetaV0("cmd-cycle-step-capacity-programming-"+suffix, "capacity-programming-"+suffix),
			Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
				CapacityRequestID:          capacityRef,
				PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:                    taskRef,
				ReasonCode:                 "programacion_siguiente_paso",
				Summary:                    "Capacidad explicita para programacion.",
				MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
				EvidenceRefs:               []string{"evidence-ref-cycle-step-capacity-programming-" + suffix},
			},
		},
		AgentCandidate: &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
			ClaimRef:    claimRef,
			CommandMeta: cycleStepCommandMetaV0("cmd-cycle-step-agent-programming-"+suffix, "agent-programming-"+suffix),
			Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
				AgentRequestID:     agentRef,
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:            taskRef,
				CapacityRequestRef: capacityRef,
				Role:               "implementacion",
				Summary:            "Agente explicito para programacion.",
				EvidenceRefs:       []string{"evidence-ref-cycle-step-agent-programming-" + suffix},
			},
		},
		GateCommandMeta:  cycleStepCommandMetaV0("cmd-cycle-step-gate-programming-"+suffix, "gate-programming-"+suffix),
		GateEvidenceRefs: []string{"evidence-ref-cycle-step-gate-programming-" + suffix},
		EvidenceRefs:     []string{"evidence-ref-cycle-step-programming-" + suffix},
	}
}

func cycleStepClaimWithRefsV0(
	runRef string,
	claimRef string,
	taskRef string,
	agentRef string,
	writeRef string,
) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         runRef,
		TaskRef:        taskRef,
		AgentRequestID: agentRef,
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: writeRef}},
		EvidenceRefs:   []string{"evidence-ref-" + claimRef},
	}
}

func cycleStepSeedCapacityDecisionForCandidateV0(
	t *testing.T,
	workflow *cycleStepWorkflowPortV0,
	candidate orquestadirectorscheduler.SchedulableWorkCandidateV0,
	suffix string,
) {
	t.Helper()
	if candidate.CapacityCandidate == nil {
		t.Fatal("candidate sin capacity")
	}
	capacity, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		candidate.CapacityCandidate.CommandMeta,
		candidate.CapacityCandidate.Payload,
	)
	if err != nil {
		t.Fatalf("capacity command: %v", err)
	}
	workflow.mustApplyCommandV0(t, capacity)
	decision, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		cycleStepCommandMetaV0("cmd-cycle-step-capacity-decision-programming-"+suffix, "capacity-decision-programming-"+suffix),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: candidate.CapacityCandidate.Payload.CapacityRequestID,
			DecisionRef:       "decision-ref-cycle-step-programming-" + suffix,
			Tier:              orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityMediumV0,
			Summary:           "Decision de capacidad para programacion.",
			EvidenceRefs:      []string{"evidence-ref-cycle-step-capacity-decision-programming-" + suffix},
		},
	)
	if err != nil {
		t.Fatalf("capacity decision command: %v", err)
	}
	workflow.mustApplyCommandV0(t, decision)
}

func assertCycleStepAppliedCommandTypesV0(
	t *testing.T,
	result DirectorCycleStepResultV0,
	want ...string,
) {
	t.Helper()
	if len(result.AppliedCommands) != len(want) {
		t.Fatalf("applied=%d want %d: %+v", len(result.AppliedCommands), len(want), result.AppliedCommands)
	}
	for index, commandType := range want {
		if result.AppliedCommands[index].CommandType != commandType {
			t.Fatalf("applied[%d]=%s want %s", index, result.AppliedCommands[index].CommandType, commandType)
		}
	}
}
