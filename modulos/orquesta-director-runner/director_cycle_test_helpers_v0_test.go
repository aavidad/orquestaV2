package orquestadirectorrunner

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const runnerRunRefV0 = "run-runner-001"

func runnerValidInputV0(
	scheduler DirectorSchedulerPortV0,
	workflow WorkflowCommandPortV0,
) DirectorCycleInputV0 {
	return DirectorCycleInputV0{
		Scheduler:      scheduler,
		Workflow:       workflow,
		CycleRef:       "cycle-ref-runner-001",
		RunRef:         runnerRunRefV0,
		SchedulerInput: runnerSchedulerInputV0(),
		EvidenceRefs:   []string{"evidence-ref-runner-001"},
	}
}

func runnerSchedulerInputV0() orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	return orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:    "tick-ref-runner-001",
		RunRef:     runnerRunRefV0,
		OccurredAt: "2026-05-06T11:00:00Z",
		Snapshot: orquestadirectorscheduler.RunSchedulingSnapshotV0{
			RunRef:         runnerRunRefV0,
			CurrentPhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		},
		EvidenceRefs: []string{"evidence-ref-runner-scheduler-001"},
	}
}

func runnerSchedulerPlanV0(
	status orquestadirectorscheduler.DirectorSchedulerTickStatusV0,
	commands []orquestacoreworkflow.OrchestrationCommandV0,
) orquestadirectorscheduler.DirectorSchedulerTickPlanV0 {
	return orquestadirectorscheduler.DirectorSchedulerTickPlanV0{
		TickRef:      "tick-ref-runner-001",
		RunRef:       runnerRunRefV0,
		Status:       status,
		Commands:     commands,
		Summary:      "scheduler runner test",
		EvidenceRefs: []string{"evidence-ref-runner-plan-001"},
	}
}

func runnerCommandMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runnerRunRefV0,
		IdempotencyKey: "idem-runner-" + suffix,
		CorrelationID:  "corr-runner-001",
		RequestedBy:    "director-runner-test",
		OccurredAt:     "2026-05-06T11:00:00Z",
	}
}

func mustRunnerStartCommandV0(t *testing.T, suffix string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		runnerCommandMetaV0("cmd-runner-start-"+suffix, "start-"+suffix),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-runner-001",
			AppSpecRef: "appspec-ref-runner-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func mustRunnerOpenProgramacionCommandV0(t *testing.T, suffix string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		runnerCommandMetaV0("cmd-runner-open-"+suffix, "open-"+suffix),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion",
		},
	)
	if err != nil {
		t.Fatalf("open phase command: %v", err)
	}
	return command
}

func mustRunnerCapacityOutboxV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          "capacity-ref-runner-001",
		RunID:                      runnerRunRefV0,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    "task-ref-runner-001",
		ReasonCode:                 "programacion_siguiente_paso",
		Summary:                    "Decision compacta para siguiente paso.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		EvidenceRefs:               []string{"evidence-ref-runner-outbox-001"},
	})
	if err != nil {
		t.Fatalf("outbox payload: %v", err)
	}
	message := orquestacoreworkflow.OutboxMessageV0{
		MessageID:      "outbox-ref-runner-001",
		MessageType:    orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:          runnerRunRefV0,
		IdempotencyKey: "idem-runner-outbox-001",
		CorrelationID:  "corr-runner-001",
		TargetPort:     orquestacoreworkflow.OutboxTargetCapacityV0,
		PayloadVersion: orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:        payload,
	}
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("outbox message: %v", err)
	}
	return message
}
