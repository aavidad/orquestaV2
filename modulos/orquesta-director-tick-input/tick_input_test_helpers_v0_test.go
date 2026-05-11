package orquestadirectortickinput

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const tickInputRunRefV0 = "run-tick-input-001"

func tickInputProgramacionRunV0(t *testing.T) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = tickInputApplyCommandV0(t, run, tickInputStartCommandV0(t))
	return tickInputApplyCommandV0(t, run, tickInputOpenProgramacionCommandV0(t))
}

func tickInputApplyCommandV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle command %s: %v", command.CommandType, err)
	}
	for _, event := range result.Events {
		run, err = orquestacoreworkflow.ApplyEventV0(run, event)
		if err != nil {
			t.Fatalf("apply event %s: %v", event.EventType, err)
		}
	}
	return run
}

func tickInputStartCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		tickInputCommandMetaV0("cmd-tick-input-start", "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-tick-input-001",
			AppSpecRef: "appspec-ref-tick-input-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func tickInputOpenProgramacionCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		tickInputCommandMetaV0("cmd-tick-input-open-programacion", "open-programacion"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar programacion",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func tickInputCommandMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          tickInputRunRefV0,
		IdempotencyKey: "idem-tick-input-" + suffix,
		CorrelationID:  "corr-tick-input-001",
		RequestedBy:    "director-tick-input-test",
		OccurredAt:     "2026-05-06T12:00:00Z",
	}
}

type tickInputMemoryWorkflowPortV0 struct {
	run orquestacoreworkflow.OrchestrationRunV0
}

func (port *tickInputMemoryWorkflowPortV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(port.run, command)
	if err != nil {
		return result, err
	}
	for _, event := range result.Events {
		port.run, err = orquestacoreworkflow.ApplyEventV0(port.run, event)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
