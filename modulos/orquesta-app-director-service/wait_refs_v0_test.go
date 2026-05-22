package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestExistingDirectorLoopRequestV0DerivaWaitAgentRefsPorCohorte(t *testing.T) {
	runRef := "run-app-director-wait-cohort-001"
	first := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-app-director-wait-a", "wave-01", "cohort-a")
	second := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-app-director-wait-b", "wave-02", "cohort-b")
	run := serviceRunForWaitRefsTestV0(runRef, first.TaskID, second.TaskID)
	ports := StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(first, second),
	}
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:        runRef,
		WaitAgentRefs: []string{"agent-ref-explicit"},
		WaitCohortRef: " cohort-a ",
	})

	loop, err := existingDirectorLoopRequestV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	wantDerived := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(first.TaskID)
	if !serviceStringInSetV0(loop.WaitAgentRefs, "agent-ref-explicit") ||
		!serviceStringInSetV0(loop.WaitAgentRefs, wantDerived) ||
		serviceStringInSetV0(loop.WaitAgentRefs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(second.TaskID)) {
		t.Fatalf("wait_agent_refs=%v", loop.WaitAgentRefs)
	}
}

func TestExistingDirectorLoopRequestV0RegistraWaitStatePorOla(t *testing.T) {
	runRef := "run-app-director-wait-state-001"
	task := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-app-director-wait-state-a", "wave-01", "cohort-a")
	run := serviceRunForWaitRefsTestV0(runRef, task.TaskID)
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	ports := StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		WaitStateWriter:   waitStore,
	}
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-17T13:30:00Z",
		CorrelationID: "corr-app-director-wait-state-001",
		WaitWaveRef:   "wave-01",
	})

	loop, err := existingDirectorLoopRequestV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	if !loop.WaitScopeApplied {
		t.Fatalf("wait scope no aplicado: %+v", loop)
	}
	waitRef := appDirectorWaitRefV0(runRef, appDirectorWaitFilterV0{WaveRef: "wave-01"}, request.CorrelationID)
	state, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if state.WaveRef != "wave-01" ||
		len(state.AgentRefs) != 1 ||
		state.AgentRefs[0] != orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID) {
		t.Fatalf("state=%+v", state)
	}
}

func TestExistingDirectorLoopRequestV0DerivaWaitAgentRefsPorParentTask(t *testing.T) {
	runRef := "run-app-director-wait-parent-001"
	parentTaskRef := "task-app-director-wait-parent-root"
	child := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-app-director-wait-parent-child", "wave-01", "cohort-a")
	child.ParentTaskRef = parentTaskRef
	sibling := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-app-director-wait-parent-sibling", "wave-01", "cohort-a")
	sibling.ParentTaskRef = "task-app-director-wait-parent-other"
	run := serviceRunForWaitRefsTestV0(runRef, child.TaskID, sibling.TaskID)
	ports := StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(child, sibling),
	}
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:            runRef,
		WaitParentTaskRef: " " + parentTaskRef + " ",
	})

	loop, err := existingDirectorLoopRequestV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	wantDerived := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(child.TaskID)
	unwanted := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(sibling.TaskID)
	if !loop.WaitScopeApplied ||
		len(loop.WaitAgentRefs) != 1 ||
		loop.WaitAgentRefs[0] != wantDerived ||
		serviceStringInSetV0(loop.WaitAgentRefs, unwanted) {
		t.Fatalf("wait_agent_refs=%v want=%s unwanted=%s", loop.WaitAgentRefs, wantDerived, unwanted)
	}
}

func TestExistingDirectorLoopRequestV0RequiereTaskStoreConFiltroWait(t *testing.T) {
	runRef := "run-app-director-wait-store-missing-001"
	run := serviceRunForWaitRefsTestV0(runRef)
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:      runRef,
		WaitWaveRef: "wave-01",
	})

	_, err := existingDirectorLoopRequestV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	})
	if err == nil {
		t.Fatalf("expected missing director task store")
	}
	if issue, ok := err.(AppDirectorServiceIssueV0); !ok || issue.Field != "ports.director_task_store" {
		t.Fatalf("err=%T %+v", err, err)
	}
}

func TestExistingDirectorLoopRequestV0RequiereRunStoreConFiltroWait(t *testing.T) {
	runRef := "run-app-director-wait-run-store-missing-001"
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:      runRef,
		WaitWaveRef: "wave-01",
	})

	_, err := existingDirectorLoopRequestV0(context.Background(), request, StartAppDirectorPortsV0{
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
	})
	if err == nil {
		t.Fatalf("expected missing run store")
	}
	if issue, ok := err.(AppDirectorServiceIssueV0); !ok || issue.Field != "ports.run_store" {
		t.Fatalf("err=%T %+v", err, err)
	}
}

func TestExistingDirectorLoopRequestV0RefsExplicitosNoRequierenFiltro(t *testing.T) {
	runRef := "run-app-director-wait-explicit-001"
	run := serviceRunForWaitRefsTestV0(runRef)
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:        runRef,
		WaitAgentRefs: []string{" agent-ref-explicit ", "agent-ref-explicit"},
	})

	loop, err := existingDirectorLoopRequestV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
	})
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	if len(loop.WaitAgentRefs) != 1 || loop.WaitAgentRefs[0] != "agent-ref-explicit" {
		t.Fatalf("wait_agent_refs=%v", loop.WaitAgentRefs)
	}
}

func serviceWorkflowTaskForWaitRefsTestV0(
	runRef string,
	taskRef string,
	waveRef string,
	cohortRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea con cohorte",
		Summary:       "Trabajo compacto para probar espera por cohorte.",
		WriteSet:      []string{"docs/" + taskRef + ".md"},
		AcceptanceCriteria: []string{
			"Se conserva wave_ref y cohort_ref.",
		},
		WaveRef:   waveRef,
		CohortRef: cohortRef,
	}
}

func serviceRunForWaitRefsTestV0(
	runRef string,
	taskRefs ...string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-" + runRef,
		AppSpecRef:    "appspec-" + runRef,
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		}},
		Tasks: append([]string(nil), taskRefs...),
	}
}
