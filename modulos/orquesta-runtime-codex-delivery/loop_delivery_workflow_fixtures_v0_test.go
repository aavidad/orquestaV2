package orquestaruntimecodexdelivery

import (
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func codexDeliveryLoopWorkCandidateV0(
	runRef string,
	agentRef string,
	taskRef string,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-ref-receipt-loop-001",
		SubjectClaimRefs: []string{"claim-ref-receipt-loop-001"},
		Claims: []orquestacoreconcurrency.WorksetClaimV0{{
			SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
			ClaimRef:       "claim-ref-receipt-loop-001",
			RunRef:         runRef,
			TaskRef:        taskRef,
			AgentRequestID: agentRef,
			WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "app/main.go"}},
			EvidenceRefs:   []string{"evidence-ref-receipt-claim-001"},
		}},
		CapacityCandidate: &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
			CommandMeta: codexDeliveryLoopCommandMetaV0(runRef, "cmd-receipt-capacity-001", "idem-receipt-capacity-001"),
			Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
				CapacityRequestID:          "capacity-ref-receipt-loop-001",
				PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:                    taskRef,
				ReasonCode:                 "programacion_siguiente_paso",
				Summary:                    "Capacidad para validar entrega por receipt.",
				MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
				EvidenceRefs:               []string{"evidence-ref-receipt-capacity-request-001"},
			},
		},
		AgentCandidate: &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
			ClaimRef:    "claim-ref-receipt-loop-001",
			CommandMeta: codexDeliveryLoopCommandMetaV0(runRef, "cmd-receipt-agent-001", "idem-receipt-agent-001"),
			Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
				AgentRequestID:     agentRef,
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:            taskRef,
				CapacityRequestRef: "capacity-ref-receipt-loop-001",
				Role:               "implementacion",
				Summary:            "Agente para validar entrega por receipt.",
				EvidenceRefs:       []string{"evidence-ref-receipt-agent-request-001"},
			},
		},
		GateCommandMeta:  codexDeliveryLoopCommandMetaV0(runRef, "cmd-receipt-gate-001", "idem-receipt-gate-001"),
		GateEvidenceRefs: []string{"evidence-ref-receipt-gate-001"},
		EvidenceRefs:     []string{"evidence-ref-receipt-candidate-001"},
	}
}

func codexDeliveryLoopRunForTestV0(
	t *testing.T,
	runRef string,
	taskRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = codexDeliveryLoopApplyCommandV0(t, run, codexDeliveryLoopStartCommandV0(t, runRef))
	run = codexDeliveryLoopApplyCommandV0(t, run, codexDeliveryLoopOpenProgrammingCommandV0(t, runRef))
	run.Tasks = append(run.Tasks, taskRef)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run invalido: %+v", issues)
	}
	return run
}

func codexDeliveryLoopApplyCommandV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
	}
	next := run
	for _, event := range result.Events {
		next, err = orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			t.Fatalf("ApplyEventV0 %s: %v", event.EventType, err)
		}
	}
	return next
}

func codexDeliveryLoopStartCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		codexDeliveryLoopCommandMetaV0(runRef, "cmd-receipt-start-001", "idem-receipt-start-001"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-receipt-001",
			AppSpecRef: "appspec-ref-receipt-001",
		},
	)
	if err != nil {
		t.Fatalf("NewStartRunCommandV0: %v", err)
	}
	return command
}

func codexDeliveryLoopOpenProgrammingCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		codexDeliveryLoopCommandMetaV0(runRef, "cmd-receipt-open-001", "idem-receipt-open-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion.",
		},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	return command
}

func codexDeliveryLoopCommandMetaV0(
	runRef string,
	commandID string,
	idempotencyKey string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-receipt-loop-001",
		RequestedBy:    "orquesta-receipt-loop-test",
		OccurredAt:     "2026-05-09T18:00:00Z",
	}
}

func codexDeliveryLoopContainsRefV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func codexDeliveryLoopHasEventV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) bool {
	return codexDeliveryLoopEventCountV0(events, eventType) > 0
}

func codexDeliveryLoopEventCountV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}
