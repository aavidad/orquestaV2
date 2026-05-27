package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestOperationalDirectorPlanStateV0BloqueaWaitTerminalSinDeliveryV0(t *testing.T) {
	runRef := "run-app-director-operational-plan-terminal-wait"
	planRef := "plan-ref-terminal-wait"
	task := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-terminal-wait"}
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		"task-ref-parent-terminal-wait",
		"wave-terminal-wait",
		"cohort-terminal-wait",
		task,
	)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef)
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "orquesta",
		AppSpecRef:    "app-spec-terminal-wait",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{task.TaskRef},
		StartedAgents: []string{agentRef},
		StoppedAgents: []string{agentRef},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-terminal-wait",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: agentRef,
			TaskRef:        task.TaskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		})},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-25T18:10:00Z",
		CorrelationID:              "corr-terminal-wait",
	}
	changed, err := applyOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	})
	if err != nil || !changed {
		t.Fatalf("applyOperationalDirectorPlanStateAfterLoopV0 changed=%v err=%v", changed, err)
	}
	blocked, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blocked, "step-wait-subagents")
	if blocked.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		blocked.ActiveStepID != "step-wait-subagents" ||
		len(blocked.PendingAgentRefs) != 0 ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason != "wait-subagents-terminal-without-delivery" ||
		len(waitStep.PendingAgentRefs) != 0 {
		t.Fatalf("blocked=%+v waitStep=%+v", blocked, waitStep)
	}
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{
		OperationalPlanStateStore: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 blocked terminal wait: %v", err)
	}
	if len(got.WaitAgentRefs) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RecuperaWaitTerminalConTareaEntregadaV0(t *testing.T) {
	runRef := "run-app-director-operational-plan-terminal-wait-delivered"
	planRef := "plan-ref-terminal-wait-delivered"
	task := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-terminal-wait-delivered"}
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		"task-ref-parent-terminal-wait-delivered",
		"wave-terminal-wait-delivered",
		"cohort-terminal-wait-delivered",
		task,
	)
	originalAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef)
	replacementAgentRef := "agent-ref-assessment-task-ref-terminal-wait-delivered-001"
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"wait-subagents-terminal-without-delivery"}
	state.ClosureReason = "wait-subagents-terminal-without-delivery"
	for index := range state.Steps {
		if state.Steps[index].StepID != "step-wait-subagents" {
			continue
		}
		state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
		state.Steps[index].PendingAgentRefs = nil
		state.Steps[index].BlockerRefs = []string{"wait-subagents-terminal-without-delivery"}
		state.Steps[index].Reason = "wait-subagents-terminal-without-delivery"
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "orquesta",
		AppSpecRef:      "app-spec-terminal-wait-delivered",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{task.TaskRef},
		StartedAgents:   []string{originalAgentRef, replacementAgentRef},
		LostAgents:      []string{originalAgentRef},
		DeliveredAgents: []string{replacementAgentRef},
		DeliveredTasks:  []string{task.TaskRef},
		Deliveries:      []string{"delivery-ref-terminal-wait-delivered"},
		AcceptedReviews: []string{"accepted-review-ref-terminal-wait-delivered"},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-27T09:20:00Z",
		CorrelationID:              "corr-terminal-wait-delivered",
	}

	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	recovered, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.AgentRefs, replacementAgentRef) ||
		!serviceStringInSetV0(got.WaitAgentRefs, replacementAgentRef) ||
		serviceStringInSetV0(got.WaitAgentRefs, originalAgentRef) {
		t.Fatalf("got=%+v recovered=%+v wait=%+v review=%+v", got, recovered, waitStep, reviewStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RecuperaWaitTerminalDeReworkEntregadoV0(t *testing.T) {
	runRef := "run-app-director-operational-plan-terminal-rework-delivered"
	planRef := "plan-ref-terminal-rework-delivered"
	parentTaskRef := "task-ref-terminal-rework-parent"
	followupTask := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-terminal-rework-followup"}
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-terminal-rework-delivered",
		"cohort-terminal-rework-delivered",
		followupTask,
	)
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"wait-subagents-terminal-without-delivery"}
	state.ClosureReason = "wait-subagents-terminal-without-delivery"
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].BlockerRefs = []string{"wait-subagents-terminal-without-delivery", "wait-subagents-replan-followups"}
			state.Steps[index].Reason = "wait-subagents-terminal-without-delivery"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].BlockerRefs = []string{"review-rework-replan-recorded"}
			state.Steps[index].Reason = "review-rework-replan-recorded"
			state.Steps[index].ReworkRequestRefs = []string{"rework-request-ref-terminal-rework-delivered"}
			state.Steps[index].ReplanDecisionRefs = []string{"replan-decision-ref-terminal-rework-delivered"}
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "orquesta",
		AppSpecRef:      "app-spec-terminal-rework-delivered",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{parentTaskRef, followupTask.TaskRef},
		DeliveredAgents: []string{followupAgentRef},
		DeliveredTasks:  []string{followupTask.TaskRef},
		Deliveries:      []string{"delivery-ref-terminal-rework-delivered"},
		AcceptedReviews: []string{"accepted-review-ref-terminal-rework-delivered"},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-27T10:05:00Z",
		CorrelationID:              "corr-terminal-rework-delivered",
	}

	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	recovered, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, followupTask.TaskRef) ||
		serviceStringInSetV0(reviewStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(got.WaitAgentRefs, followupAgentRef) {
		t.Fatalf("got=%+v recovered=%+v wait=%+v review=%+v", got, recovered, waitStep, reviewStep)
	}
}
