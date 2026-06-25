package main

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func directorCycleResidentConfigV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	return config
}

type residentCycleWorkflowV0 struct {
	run     orquestacoreworkflow.OrchestrationRunV0
	counter int
}

func newResidentCycleWorkflowV0(t *testing.T) *residentCycleWorkflowV0 {
	t.Helper()
	port := &residentCycleWorkflowV0{}
	port.mustApplyV0(t, residentCycleStartCommandV0(t))
	port.mustApplyV0(t, residentCycleOpenPhaseCommandV0(t))
	return port
}

func (port *residentCycleWorkflowV0) HandleWorkflowCommandV0(
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

func (port *residentCycleWorkflowV0) mustApplyV0(t *testing.T, command orquestacoreworkflow.OrchestrationCommandV0) {
	t.Helper()
	if _, err := port.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
}

func residentCycleInputV0(
	workflow *residentCycleWorkflowV0,
	ledger orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0,
	cycleRef string,
	tickRef string,
) orquestadirectorcycle.DirectorCycleStepInputV0 {
	workflow.counter++
	return orquestadirectorcycle.DirectorCycleStepInputV0{
		Scheduler:    orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:     workflow,
		OutboxLedger: ledger,
		CycleRef:     cycleRef,
		TickRef:      tickRef,
		RunRef:       workflow.run.RunID,
		OccurredAt:   "2026-05-25T16:20:00Z",
		Run:          workflow.run,
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
			residentCycleWorkCandidateV0(workflow.run.RunID, workflow.counter),
		},
		EvidenceRefs: []string{"evidence-ref-resident-cycle"},
	}
}

func residentCycleWorkCandidateV0(
	runRef string,
	sequence int,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	suffix := strconv.Itoa(sequence)
	claimRef := "claim-ref-resident-cycle-" + suffix
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-ref-resident-cycle-" + suffix,
		SubjectClaimRefs: []string{claimRef},
		Claims: []orquestacoreconcurrency.WorksetClaimV0{
			residentCycleClaimV0(runRef, claimRef, sequence),
		},
		CapacityCandidate: residentCycleCapacityCandidateV0(sequence),
		GateCommandMeta:   residentCycleCommandMetaV0("cmd-resident-cycle-gate-"+suffix, "gate-"+suffix),
		EvidenceRefs:      []string{"evidence-ref-resident-cycle-candidate"},
	}
}

func residentCycleClaimV0(
	runRef string,
	claimRef string,
	sequence int,
) orquestacoreconcurrency.WorksetClaimV0 {
	suffix := strconv.Itoa(sequence)
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         runRef,
		TaskRef:        "task-ref-resident-cycle-" + suffix,
		AgentRequestID: "agent-ref-resident-cycle-" + suffix,
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "cmd/orquesta-server"}},
		EvidenceRefs:   []string{"evidence-ref-resident-cycle-claim"},
	}
}

func residentCycleCapacityCandidateV0(
	sequence int,
) *orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0 {
	suffix := strconv.Itoa(sequence)
	return &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
		CommandMeta: residentCycleCommandMetaV0("cmd-resident-cycle-capacity-"+suffix, "capacity-"+suffix),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-resident-cycle-" + suffix,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-resident-cycle-" + suffix,
			ReasonCode:                 "programacion_siguiente_paso",
			Summary:                    "Capacidad para smoke residente del ciclo.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			EvidenceRefs:               []string{"evidence-ref-resident-cycle-capacity"},
		},
	}
}

func residentCycleStartCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		residentCycleCommandMetaV0("cmd-resident-cycle-start", "start"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-resident-cycle",
			AppSpecRef: "appspec-ref-resident-cycle",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func residentCycleOpenPhaseCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		residentCycleCommandMetaV0("cmd-resident-cycle-open", "open"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Preparar smoke residente.",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func residentCycleCommandMetaV0(commandID string, suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          "run-ref-resident-cycle",
		IdempotencyKey: "idem-resident-cycle-" + suffix,
		CorrelationID:  "corr-resident-cycle",
		RequestedBy:    "orquesta-server-resident-cycle-smoke",
		OccurredAt:     "2026-05-25T16:20:00Z",
	}
}

type residentCycleDispatchExecutorV0 struct {
	calls int
}

func (executor *residentCycleDispatchExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	executor.calls++
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  "dispatch-ref-" + intent.MessageID,
		EvidenceRefs: []string{"evidence-ref-resident-cycle-dispatch"},
	}, nil
}

type residentCycleRuntimeDirectorV0 struct {
	workflow *residentCycleWorkflowV0
	ledger   orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	calls    int
	results  []orquestadirectorcycle.DirectorCycleStepResultV0
}

func (director *residentCycleRuntimeDirectorV0) RunResidentDirectorV0(
	ctx context.Context,
	command orquestaserver.ResidentDirectorCommandV0,
) (orquestaserver.ResidentDirectorResultV0, error) {
	director.calls++
	step, err := orquestadirectorcycle.ExecuteDirectorCycleStepV0(
		ctx,
		residentCycleInputV0(
			director.workflow,
			director.ledger,
			"cycle-ref-runtime-resident-"+strconv.Itoa(director.calls),
			"tick-ref-runtime-resident-"+strconv.Itoa(director.calls),
		),
	)
	director.results = append(director.results, step)
	return orquestaserver.ResidentDirectorResultV0{
		Status:          string(step.Status),
		RunRef:          step.RunRef,
		ExecutedActions: residentCycleRuntimeExecutedActionsV0(step),
		EvidenceRefs: []string{
			"evidence-ref-runtime-resident-director-cycle",
			command.CorrelationID,
		},
	}, err
}

func residentCycleRuntimeExecutedActionsV0(
	step orquestadirectorcycle.DirectorCycleStepResultV0,
) int {
	if step.OutboxSavedCount > 0 {
		return step.OutboxSavedCount
	}
	return len(step.AppliedCommands)
}

func waitRuntimeResidentDirectorTicksForTestV0(
	t *testing.T,
	runtime *orquestaserver.RuntimeV0,
	minTicks int,
) orquestaserver.StateV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := runtime.StateV0()
		if state.ResidentDirectorTicks >= minTicks && !state.ResidentDirectorTickActive {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime sin ticks residentes suficientes: state=%+v want>=%d", runtime.StateV0(), minTicks)
	return orquestaserver.StateV0{}
}

func waitRuntimeDoneForCycleResidentTestV0(
	t *testing.T,
	done <-chan error,
) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatalf("runtime no paro")
		return nil
	}
}
