package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func neutralProcessStopCleanupV0(
	runtime *orquestaruntime.ProcessRuntimeConnectorV0,
	processRef string,
) {
	if runtime == nil || strings.TrimSpace(processRef) == "" {
		return
	}
	_, _ = runtime.StopV0(context.Background(), processRef)
}

func neutralProcessStopDispatchV0(
	t *testing.T,
	ctx context.Context,
	stack orquestaappcodexstack.StackV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaoutboxdispatch.RunOutboxDispatchOnceResultV0 {
	t.Helper()
	result, err := orquestaoutboxdispatch.RunOutboxDispatchOnceV0(
		orquestaoutboxdispatch.RunOutboxDispatchOnceInputV0{
			RunID:       t66RunRef,
			TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
			Reader:      stack.Stores.OutboxLedger,
			Claimer:     stack.Stores.OutboxLedger,
			Executor: orquestacionnucleoapp.AgentStopperExecutorV0{
				RunStore:    stack.Stores.RunStore,
				EventSink:   stack.Stores.EventSink,
				Stopper:     orquestacionnucleoapp.ProcessAgentStopperV0{Registry: stack.Stores.ProcessRegistry, Runtime: processRuntime},
				ObservedAt:  t66OccurredAt,
				RequestedBy: "director",
				Summary:     "Stop neutral confirmado por runtime de proceso.",
				EvidenceRefs: []string{
					"evidence-ref-neutral-process-stop-dispatch",
				},
			},
			Acker: stack.Stores.OutboxLedger,
		},
	)
	if err != nil {
		t.Fatalf("dispatch stop: %v result=%+v", err, result)
	}
	return result
}

func neutralProcessStopAssertStoppedV0(
	t *testing.T,
	runtime *orquestaruntime.ProcessRuntimeConnectorV0,
	processRef string,
) {
	t.Helper()
	snapshot, err := runtime.SnapshotV0(processRef)
	if err != nil {
		t.Fatalf("SnapshotV0: %v", err)
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeStoppedV0 ||
		strings.TrimSpace(snapshot.StopRef) == "" {
		t.Fatalf("snapshot not stopped: %+v", snapshot)
	}
}

type neutralProcessStopNoopExecutorV0 struct{}

type neutralProcessStopStoresV0 struct {
	RunStore        orquestacionnucleoapp.RunStorePortV0
	EventSink       orquestacionnucleoapp.EventSinkPortV0
	ProcessRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0
	OutboxLedger    orquestaappcodexstack.OutboxLedgerPortV0
}

func neutralProcessStopDispatchWithStoresV0(
	t *testing.T,
	stores neutralProcessStopStoresV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) orquestaoutboxdispatch.RunOutboxDispatchOnceResultV0 {
	t.Helper()
	result, err := orquestaoutboxdispatch.RunOutboxDispatchOnceV0(
		orquestaoutboxdispatch.RunOutboxDispatchOnceInputV0{
			RunID:       t66RunRef,
			TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
			Reader:      stores.OutboxLedger,
			Claimer:     stores.OutboxLedger,
			Executor: orquestacionnucleoapp.AgentStopperExecutorV0{
				RunStore:    stores.RunStore,
				EventSink:   stores.EventSink,
				Stopper:     orquestacionnucleoapp.ProcessAgentStopperV0{Registry: stores.ProcessRegistry, Runtime: processRuntime},
				ObservedAt:  t66OccurredAt,
				RequestedBy: "director",
				Summary:     "Stop neutral confirmado por runtime de proceso.",
				EvidenceRefs: []string{
					"evidence-ref-neutral-process-stop-dispatch",
				},
			},
			Acker: stores.OutboxLedger,
		},
	)
	if err != nil {
		t.Fatalf("dispatch stop: %v result=%+v", err, result)
	}
	return result
}

func (neutralProcessStopNoopExecutorV0) ExecuteOutboxDispatchV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) (orquestaoutboxdispatch.OutboxDispatchExecutionResultV0, error) {
	return orquestaoutboxdispatch.OutboxDispatchExecutionResultV0{
		DispatchRef:  "dispatch-ref-noop-" + strings.TrimSpace(intent.MessageID),
		EvidenceRefs: []string{"evidence-ref-neutral-process-stop-noop"},
	}, nil
}

func neutralProcessStopAssertRunConfirmedV0(
	t *testing.T,
	ctx context.Context,
	store orquestacionnucleoapp.RunStorePortV0,
) {
	t.Helper()
	run, err := store.LoadRunV0(ctx, t66RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !containsStringV0(run.StoppedAgents, t66AgentRef) ||
		!containsStringV0(run.ConfirmedStoppedAgents, t66AgentRef) {
		t.Fatalf("stop not confirmed: stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
}

func neutralProcessStopAssertNoSensitiveStateV0(t *testing.T, stateDir string, forbidden string) {
	t.Helper()
	forbidden = strings.TrimSpace(forbidden)
	if forbidden == "" {
		return
	}
	err := filepath.WalkDir(stateDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("state contains forbidden process command path: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk state: %v", err)
	}
}

func neutralProcessStopProgressReportV0(
	t *testing.T,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) orquestaruntime.AgentProgressReportV0 {
	t.Helper()
	return orquestaruntime.AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-neutral-process-stop",
		RunID:               t66RunRef,
		AgentRequestID:      t66AgentRef,
		Status:              orquestaruntime.AgentLoopDetectedV0,
		NoProgressTicks:     6,
		RepeatedActionCount: 3,
		Summary:             "Bucle operativo confirmado; detener agente neutral.",
		EvidenceRefs: []string{
			"evidence-ref-neutral-process-stop-progress",
			snapshot.ProcessRef,
		},
	}
}

type neutralProcessStoredWorkflowV0 struct {
	store orquestacionnucleoapp.RunStorePortV0
}

func (workflow neutralProcessStoredWorkflowV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	return orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, workflow.store, nil, command)
}

func neutralProcessCommandMetaV0(commandID string, idempotencyKey string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          t66RunRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-neutral-process-stop",
		RequestedBy:    "director",
		OccurredAt:     t66OccurredAt,
	}
}

func containsStringV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
