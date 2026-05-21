package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
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

func TestDrainRunV0AceptaACKTardioDeOlaInicialMientrasDirectorAvanza(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	ctx := context.Background()

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)
	initial := codexStackDescriptorsForTestV0(t, stack)
	for _, target := range []string{
		"orquesta-app-stack-director",
		"orquesta-app-stack-web",
		"orquesta-app-stack-persistencia",
	} {
		descriptor := codexStackDescriptorByTargetModuleForTestV0(t, initial, target)
		if err := writeCodexStackAckForDescriptorV0(t, descriptor); err != nil {
			t.Fatalf("write ACK %s: %v", target, err)
		}
	}
	api := codexStackDescriptorByTargetModuleForTestV0(t, initial, "orquesta-app-stack-api")

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-late-initial-wave-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 avance parcial: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("director no avanzo con ACKs parciales: phase=%s launches=%d artifacts=%v",
			run.CurrentPhase,
			runtime.launchCountV0(),
			run.PhaseArtifacts,
		)
	}
	apiObservation := drainObservationFromDescriptorForTestV0(api)
	if drainObservationAlreadyRegisteredV0(run, apiObservation) {
		t.Fatalf("ACK api registrado antes de existir: artifacts=%v", run.PhaseArtifacts)
	}
	if err := writeCodexStackAckForDescriptorV0(t, api); err != nil {
		t.Fatalf("write late ACK api: %v", err)
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-late-initial-wave-002",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 ACK tardio api: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !drainObservationAlreadyRegisteredV0(run, apiObservation) {
		t.Fatalf("ACK tardio api no registrado: artifacts=%v observation=%+v", run.PhaseArtifacts, apiObservation)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s want=%s", run.CurrentPhase, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
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

func codexStackDescriptorByTargetModuleForTestV0(
	t *testing.T,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	targetModule string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	for _, descriptor := range descriptors {
		if descriptor.Spec.AgentPacket.TargetModule == targetModule {
			return descriptor
		}
	}
	t.Fatalf("descriptor target=%s no encontrado en %v", targetModule, codexStackRealSmokeDescriptorAgentsV0(descriptors))
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}
