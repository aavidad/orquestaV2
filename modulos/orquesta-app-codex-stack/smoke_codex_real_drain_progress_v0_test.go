package orquestaappcodexstack

import (
	"context"
	"testing"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackRealSmokeDrainHastaProgramacionToleraTasksVaciasConProgreso(t *testing.T) {
	runtime := newDelayedAckCodexStackRuntimeV0(50 * time.Millisecond)
	stack := mustBuildCodexStackForTestV0(t, runtime)
	source := &smokeDrainDelayedProgrammingDecisionSourceV0{}
	stack.Ports.DirectorDecisionSource = source

	director := postDirectorAPIV0(t, stack)
	initial := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)

	result := codexStackRealSmokeDrainUntilProgrammingDeliveredV0(
		t,
		context.Background(),
		stack,
		smokeDrainStoresFromStackForTestV0(t, stack),
		director.RunRef,
		t.TempDir(),
		100,
	)

	if source.calls < 2 {
		t.Fatalf("decision_source_calls=%d, want >=2", source.calls)
	}
	if len(source.progressCallTasks) != 0 {
		t.Fatalf("progress_call_tasks=%v, want []", source.progressCallTasks)
	}
	if source.progressCallSequence <= initial.LastSequence {
		t.Fatalf(
			"progress_call_sequence=%d initial=%d, want progreso antes de tareas",
			source.progressCallSequence,
			initial.LastSequence,
		)
	}
	if !codexStackRealSmokeAllRunTasksDeliveredV0(result.Run) {
		t.Fatalf("tasks=%v delivered=%v", result.Run.Tasks, result.Run.DeliveredTasks)
	}
}

type smokeDrainDelayedProgrammingDecisionSourceV0 struct {
	calls                int
	progressCallSequence int64
	progressCallTasks    []string
	progressSeen         bool
	decisionsSent        bool
}

func (source *smokeDrainDelayedProgrammingDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	if !source.progressSeen && smokeDrainRunHasDirectorArtifactsWithoutTasksV0(request.Run) {
		source.progressSeen = true
		source.progressCallSequence = request.Run.LastSequence
		source.progressCallTasks = append([]string(nil), request.Run.Tasks...)
		return nil, nil
	}
	if !source.progressSeen || source.decisionsSent {
		return nil, nil
	}
	source.decisionsSent = true
	return smokeDrainProgrammingDecisionsForTestV0(request.Run), nil
}

func smokeDrainRunHasDirectorArtifactsWithoutTasksV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactStringsV0(run.PhaseArtifacts)) > 0 &&
		len(compactStringsV0(run.Tasks)) == 0
}

func smokeDrainStoresFromStackForTestV0(
	t *testing.T,
	stack StackV0,
) codexStackRealSmokeStoresV0 {
	t.Helper()
	return codexStackRealSmokeStoresV0{
		RunStore:        stack.Stores.RunStore.(*orquestacionnucleoapp.InMemoryRunStoreV0),
		EventSink:       stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0),
		OutboxLedger:    stack.Stores.OutboxLedger.(*orquestacionnucleoapp.InMemoryOutboxLedgerV0),
		TaskStore:       stack.Stores.TaskStore.(*orquestacionnucleoapp.InMemoryWorkflowTaskStoreV0),
		AppChangeStore:  stack.Stores.AppChangeStore.(*orquestaappchange.InMemoryAppChangeStoreV0),
		ReceiptStore:    stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0),
		ProgressState:   stack.Stores.ProgressState.(*orquestaruntimecodexdelivery.InMemoryCodexProgressStateStoreV0),
		ProcessRegistry: stack.Stores.ProcessRegistry.(*orquestacionnucleoapp.InMemoryAgentProcessRegistryV0),
		RunMemory:       stack.Stores.RunControl.(*orquestarunmemory.RunMemoryStoreV0),
	}
}

func smokeDrainProgrammingDecisionsForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	brainstormRef := "brainstorm-ref-smoke-drain-progress"
	if len(run.Brainstorms) > 0 {
		brainstormRef = run.Brainstorms[0]
	}
	return codexStackDirectorDecisionsForTestV0(run.RunID, brainstormRef)
}
