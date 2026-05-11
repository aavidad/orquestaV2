package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexStackV0FlujoProgresivoAPIMCPProgramaYCambiaSinIntervencionManual(t *testing.T) {
	ctx := context.Background()
	runtime := newDecisionWritingFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-progressive-decision-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 decisiones: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	programmingDescriptors := codexStackRealSmokeProgrammingReceiptDescriptorsV0(
		codexStackDescriptorsForTestV0(t, stack),
	)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(programmingDescriptors) == 0 {
		t.Fatalf("phase=%s programming_descriptors=%v", run.CurrentPhase, programmingDescriptors)
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-progressive-programming-delivery",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 entregas programacion: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(run.Deliveries, programmingDescriptors) {
		t.Fatalf("deliveries=%v programming=%v", run.Deliveries, codexStackRealSmokeProgrammingDescriptorsV0(programmingDescriptors))
	}
	tasksBeforeChange := len(compactStringsV0(run.Tasks))
	startedBeforeChange := len(compactStringsV0(run.StartedAgents))
	deliveriesBeforeChange := len(compactStringsV0(run.Deliveries))

	change := progressiveCodexStackAppChangeRequestV0(director.RunRef)
	mustPostCodexStackAppChangeV0(t, stack, change)
	if !codexStackStringInSetForTestV0(
		mustLoadCodexStackRunForTestV0(t, stack, director.RunRef).DirectorQuestions,
		"question-ref-app-change-"+change.ChangeRef,
	) {
		t.Fatalf("cambio no quedo como pregunta durable")
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-progressive-change-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 cambio: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	changeTaskRef := codexStackTaskIDByWriteSetForTestV0(
		t,
		stack,
		director.RunRef,
		run.Tasks,
		"docs/change-request-midrun.md",
	)
	changeAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(changeTaskRef)
	if len(compactStringsV0(run.Tasks)) <= tasksBeforeChange ||
		len(compactStringsV0(run.StartedAgents)) <= startedBeforeChange ||
		!codexStackStringInSetForTestV0(run.StartedAgents, changeAgentRef) {
		t.Fatalf("tasks=%v started=%v change_agent=%s", run.Tasks, run.StartedAgents, changeAgentRef)
	}
	if !codexStackStringInSetForTestV0(
		run.DirectorAnsweredQuestions,
		"question-ref-app-change-"+change.ChangeRef,
	) {
		t.Fatalf("director_answered_questions=%v", run.DirectorAnsweredQuestions)
	}

	changeDescriptor, ok := codexStackRealSmokeDescriptorByWriteSetV0(
		codexStackDescriptorsForTestV0(t, stack),
		"docs/change-request-midrun.md",
	)
	if !ok {
		t.Fatalf("descriptor de cambio no encontrado: %v", codexStackDescriptorWriteSetsV0(t, stack))
	}
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-progressive-change-delivery",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 entrega cambio: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if len(compactStringsV0(run.Deliveries)) <= deliveriesBeforeChange ||
		!codexStackStringInSetForTestV0(run.Deliveries, changeDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef) {
		t.Fatalf("deliveries=%v change_ack=%s", run.Deliveries, changeDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	}
}

func progressiveCodexStackAppChangeRequestV0(runRef string) orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		RunRef:             runRef,
		AppRef:             "agenda-equipo",
		ChangeRef:          "change-ref-stack-progressive-midrun-001",
		UserIntent:         "Anadir una nota de cambio de vista mensual durante el run.",
		TargetArea:         "docs",
		CurrentStateRefs:   []string{"delivery-ref-programacion-001"},
		AcceptanceCriteria: []string{"documentar vista mensual en el fichero solicitado"},
		AllowedWriteSet:    []string{"docs/change-request-midrun.md"},
	}
}

func mustLoadCodexStackRunForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	return run
}

func codexStackDescriptorsForTestV0(
	t *testing.T,
	stack StackV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	store := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	return codexStackRealSmokeDescriptorsV0(t, store)
}

func codexStackTaskIDByWriteSetForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	taskRefs []string,
	path string,
) string {
	t.Helper()
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(context.Background(), runRef, taskRefs)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	for _, task := range tasks {
		for _, entry := range task.WriteSet {
			if strings.TrimSpace(entry) == path {
				return task.TaskID
			}
		}
	}
	t.Fatalf("task write_set %q no encontrado en %+v", path, tasks)
	return ""
}
