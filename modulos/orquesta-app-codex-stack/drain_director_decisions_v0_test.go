package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-delayed-decisions-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", run.CurrentPhase)
	}
	if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
		t.Fatalf("tasks=%v missing=%s", run.Tasks, taskRef)
	}
	if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
		t.Fatalf("started_agents=%v missing=%s", run.StartedAgents, agentRef)
	}
	if runtime.launchCountV0() < 5 {
		t.Fatalf("launches=%d want>=5", runtime.launchCountV0())
	}
}

func codexStackWriteDelayedDirectorDecisionsForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	descriptors := codexStackRealSmokeDescriptorsV0(t, store)
	for _, descriptor := range descriptors {
		if descriptor.Spec.AgentPacket.TargetModule != "orquesta-app-stack-director" {
			continue
		}
		writeDelayedDirectorDecisionFileForTestV0(t, descriptor, runRef)
		return
	}
	t.Fatalf("descriptor director no encontrado")
}

func writeDelayedDirectorDecisionFileForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	runRef string,
) {
	t.Helper()
	brainstormRef := codexStackObjectiveValueForTestV0(
		descriptor.Spec.AgentPacket.Task.Objective,
		"BrainstormRef inicial:",
	)
	if brainstormRef == "" {
		t.Fatalf("brainstorm ref vacio en objetivo director")
	}
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     codexStackDirectorDecisionsForTestV0(runRef, brainstormRef),
	})
	if err != nil {
		t.Fatalf("marshal delayed decisions: %v", err)
	}
	path := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexDirectorDecisionsFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write delayed decisions: %v", err)
	}
}
