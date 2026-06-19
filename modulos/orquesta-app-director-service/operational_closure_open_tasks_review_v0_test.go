package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreReviewDeFollowupEntregadoTrasClosureNoProgress(t *testing.T) {
	runRef := "run-app-director-closure-no-progress-followup-review"
	planRef := "plan-ref-closure-no-progress-followup-review"
	parentTaskRef := "task-ref-closure-no-progress-parent"
	followupTask := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-closure-no-progress-followup"}
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskRef)
	followupDeliveryRef := "delivery-ref-closure-no-progress-followup"
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-closure-no-progress-followup",
		"cohort-closure-no-progress-followup",
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef},
		followupTask,
	)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-replan-or-close"
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{operationalClosureOpenTasksNoProgressReasonV0, "run.open_tasks"}
	state.ClosureReason = operationalClosureOpenTasksNoProgressReasonV0
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].BlockerRefs = nil
			state.Steps[index].Reason = "wait-subagents-consumed"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].DeliveryRefs = []string{"delivery-ref-closure-no-progress-parent"}
			state.Steps[index].ReviewResultRefs = []string{"review-result-ref-closure-no-progress-parent"}
			state.Steps[index].AcceptedReviewRefs = []string{"accepted-review-ref-closure-no-progress-parent"}
			state.Steps[index].Reason = "review-deliveries-accepted"
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].Reason = "required-tests-passed"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef, followupTask.TaskRef}
			state.Steps[index].BlockerRefs = []string{operationalClosureOpenTasksNoProgressReasonV0, "run.open_tasks"}
			state.Steps[index].Reason = operationalClosureOpenTasksNoProgressReasonV0
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "orquesta",
		AppSpecRef:      "app-spec-closure-no-progress-followup-review",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{parentTaskRef, followupTask.TaskRef},
		DeliveredAgents: []string{followupAgentRef},
		DeliveredTasks:  []string{followupTask.TaskRef},
		Deliveries:      []string{followupDeliveryRef},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-27T22:30:00Z",
		CorrelationID:              "corr-closure-no-progress-followup-review",
	}

	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: []orquestacoreworkflow.OrchestrationEventV0{serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{DeliveryRef: followupDeliveryRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), TaskID: followupTask.TaskRef, AgentRef: followupAgentRef, Summary: "Entrega followup pendiente de review.", EvidenceRefs: []string{"evidence-ref-followup-delivered-no-progress"}})}},
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
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-replan-or-close")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		recovered.ClosureReason != "" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		reviewStep.Reason != "closure-open-tasks-review-followups" ||
		!serviceStringInSetV0(reviewStep.TaskRefs, followupTask.TaskRef) ||
		serviceStringInSetV0(reviewStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, followupDeliveryRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(got.WaitAgentRefs, followupAgentRef) {
		t.Fatalf("got=%+v recovered=%+v review=%+v replan=%+v", got, recovered, reviewStep, replanStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreReviewDeFollowupEntregadoTrasClosureIssues(t *testing.T) {
	runRef := "run-app-director-closure-issues-followup-review"
	planRef := "plan-ref-closure-issues-followup-review"
	parentTaskRef := "task-ref-closure-issues-parent"
	followupTask := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-closure-issues-followup"}
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskRef)
	followupDeliveryRef := "delivery-ref-closure-issues-followup"
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-closure-issues-followup",
		"cohort-closure-issues-followup",
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef},
		followupTask,
	)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-replan-or-close"
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"operational-closure-issues", "delivery_ref", "nucleo_orquestacion_invalido"}
	state.ClosureReason = "operational-closure-issues"
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].BlockerRefs = nil
			state.Steps[index].Reason = "wait-subagents-consumed"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].DeliveryRefs = []string{"delivery-ref-closure-issues-parent"}
			state.Steps[index].ReviewResultRefs = []string{"review-result-ref-closure-issues-parent"}
			state.Steps[index].AcceptedReviewRefs = []string{"accepted-review-ref-closure-issues-parent"}
			state.Steps[index].Reason = "review-deliveries-accepted"
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].Reason = "required-tests-passed"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].BlockerRefs = []string{"operational-closure-issues", "delivery_ref"}
			state.Steps[index].Reason = "operational-closure-issues"
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "orquesta",
		AppSpecRef:      "app-spec-closure-issues-followup-review",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{parentTaskRef, followupTask.TaskRef},
		DeliveredAgents: []string{followupAgentRef},
		DeliveredTasks:  []string{followupTask.TaskRef},
		Deliveries:      []string{followupDeliveryRef},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-27T22:45:00Z",
		CorrelationID:              "corr-closure-issues-followup-review",
	}

	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: []orquestacoreworkflow.OrchestrationEventV0{serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{DeliveryRef: followupDeliveryRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), TaskID: followupTask.TaskRef, AgentRef: followupAgentRef, Summary: "Entrega followup pendiente de review.", EvidenceRefs: []string{"evidence-ref-followup-delivered-closure-issues"}})}},
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
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-replan-or-close")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		recovered.ClosureReason != "" ||
		len(recovered.BlockerRefs) != 0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, followupTask.TaskRef) ||
		serviceStringInSetV0(reviewStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, followupDeliveryRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(replanStep.BlockerRefs) != 0 ||
		!serviceStringInSetV0(got.WaitAgentRefs, followupAgentRef) {
		t.Fatalf("got=%+v recovered=%+v review=%+v replan=%+v", got, recovered, reviewStep, replanStep)
	}
}

func TestOperationalDirectorPlanStateRequiredTestsPassedRefrescaScopeDeCierreV0(t *testing.T) {
	runRef := "run-app-director-required-tests-refresh-closure-scope"
	planRef := "plan-ref-required-tests-refresh-closure-scope"
	parentTaskRef := "task-ref-required-tests-refresh-parent"
	followupTask := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-required-tests-refresh-followup"}
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskRef)
	followupDeliveryRef := "delivery-ref-required-tests-refresh-followup"
	followupReviewResultRef := "review-result-ref-required-tests-refresh-followup"
	passedEvidenceRef := "evidence-ref-required-tests-refresh-followup-passed"
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-required-tests-refresh",
		"cohort-required-tests-refresh",
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef},
		followupTask,
	)
	state.ActiveStepID = "step-run-required-tests"
	state.PendingAgentRefs = nil
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].BlockerRefs = nil
			state.Steps[index].Reason = "wait-subagents-consumed"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{followupTask.TaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].AcceptedReviewRefs = []string{"accepted-review-ref-required-tests-refresh-followup"}
			state.Steps[index].Reason = "review-deliveries-accepted"
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{followupTask.TaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].BlockerRefs = []string{"required-tests-pending"}
			state.Steps[index].Reason = "required-tests-pending"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].AgentRefs = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parentTaskRef)}
			state.Steps[index].DeliveryRefs = []string{"delivery-ref-required-tests-refresh-parent"}
			state.Steps[index].ReviewResultRefs = []string{"review-result-ref-required-tests-refresh-parent"}
			state.Steps[index].Reason = "stale-closure-scope"
		}
	}
	activeStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")

	recovered, changed, err := operationalDirectorPlanStateWithRequiredTestsPassedV0(
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-28T00:10:00Z",
			CorrelationID: "corr-required-tests-refresh-closure-scope",
		},
		state,
		activeStep,
		[]string{passedEvidenceRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStateWithRequiredTestsPassedV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-replan-or-close")
	if !changed ||
		recovered.ActiveStepID != "step-replan-or-close" ||
		!serviceStringInSetV0(replanStep.TaskRefs, followupTask.TaskRef) ||
		serviceStringInSetV0(replanStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(replanStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(replanStep.DeliveryRefs, followupDeliveryRef) ||
		!serviceStringInSetV0(replanStep.ReviewResultRefs, followupReviewResultRef) ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, passedEvidenceRef) {
		t.Fatalf("recovered=%+v replan=%+v", recovered, replanStep)
	}
}

func TestContinueOperationalDirectorPlanStateRefreshStaleClosureScopeV0(t *testing.T) {
	runRef := "run-app-director-refresh-stale-closure-scope"
	planRef := "plan-ref-refresh-stale-closure-scope"
	parentTaskRef := "task-ref-refresh-stale-closure-parent"
	followupTask := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-ref-refresh-stale-closure-followup"}
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskRef)
	followupDeliveryRef := "delivery-ref-refresh-stale-closure-followup"
	followupReviewRequestRef := "review-request-ref-refresh-stale-closure-followup"
	followupReviewResultRef := "review-result-ref-refresh-stale-closure-followup"
	followupAcceptedReviewRef := "accepted-review-ref-refresh-stale-closure-followup"
	otherTaskRef := "task-ref-refresh-stale-closure-other-scope"
	otherAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(otherTaskRef)
	otherDeliveryRef := "delivery-ref-refresh-stale-closure-other-scope"
	otherReviewRequestRef := "review-request-ref-refresh-stale-closure-other-scope"
	otherReviewResultRef := "review-result-ref-refresh-stale-closure-other-scope"
	otherAcceptedReviewRef := "accepted-review-ref-refresh-stale-closure-other-scope"
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-refresh-stale-closure",
		"cohort-refresh-stale-closure",
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef},
		followupTask,
	)
	state.ActiveStepID = "step-replan-or-close"
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.BlockerRefs = []string{"operational-closure-issues", "delivery_ref"}
	state.ClosureReason = "operational-closure-issues"
	state.PendingAgentRefs = nil
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].BlockerRefs = nil
			state.Steps[index].Reason = "wait-subagents-consumed"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{followupTask.TaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].AcceptedReviewRefs = []string{followupAcceptedReviewRef}
			state.Steps[index].Reason = "review-deliveries-accepted"
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{followupTask.TaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].RequiredTestEvidenceRefs = []string{"evidence-ref-refresh-stale-closure-followup-test"}
			state.Steps[index].Reason = "required-tests-passed"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].AgentRefs = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parentTaskRef)}
			state.Steps[index].DeliveryRefs = []string{"delivery-ref-refresh-stale-closure-parent"}
			state.Steps[index].ReviewResultRefs = []string{"review-result-ref-refresh-stale-closure-parent"}
			state.Steps[index].AcceptedReviewRefs = []string{"accepted-review-ref-refresh-stale-closure-parent"}
			state.Steps[index].BlockerRefs = []string{"operational-closure-issues", "delivery_ref"}
			state.Steps[index].Reason = "stale-closure-scope"
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "orquesta",
		AppSpecRef:      "app-spec-refresh-stale-closure-scope",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{parentTaskRef, followupTask.TaskRef},
		DeliveredAgents: []string{followupAgentRef, otherAgentRef},
		DeliveredTasks:  []string{followupTask.TaskRef, otherTaskRef},
		Deliveries:      []string{followupDeliveryRef, otherDeliveryRef},
		Reviews:         []string{followupReviewRequestRef, otherReviewRequestRef},
		ReviewResults:   []string{followupReviewResultRef, otherReviewResultRef},
		AcceptedReviews: []string{followupAcceptedReviewRef, otherAcceptedReviewRef},
		ReplanDecisions: []string{"replan-ref-refresh-stale-closure#source:rework-request-ref-refresh-stale-closure#task:" + parentTaskRef + "#action:split_task#followups:" + followupTask.TaskRef},
	}
	run.Tasks = append(run.Tasks, otherTaskRef)
	parentTask := serviceOperationalClosureTaskWithRefsForTestV0(runRef, "", serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef})
	parentTask.WaveRef = "wave-refresh-stale-closure"
	parentTask.CohortRef = "cohort-refresh-stale-closure"
	followupWorkflowTask := serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, followupTask)
	followupWorkflowTask.WaveRef = "wave-refresh-stale-closure"
	followupWorkflowTask.CohortRef = "cohort-refresh-stale-closure"
	otherWorkflowTask := serviceOperationalClosureTaskWithRefsForTestV0(runRef, "task-ref-refresh-stale-closure-other-parent", serviceOperationalClosureTaskRefsForTestV0{TaskRef: otherTaskRef})
	otherWorkflowTask.WaveRef = "wave-refresh-stale-closure-other"
	otherWorkflowTask.CohortRef = "cohort-refresh-stale-closure-other"
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-28T00:20:00Z",
		CorrelationID:              "corr-refresh-stale-closure-scope",
	}

	recovered, changed, err := continueOperationalDirectorPlanStateRefreshStaleClosureScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parentTask, followupWorkflowTask, otherWorkflowTask),
			EventReader: serviceOperationalClosureEventReaderForTestV0{Events: []orquestacoreworkflow.OrchestrationEventV0{
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{DeliveryRef: followupDeliveryRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), TaskID: followupTask.TaskRef, AgentRef: followupAgentRef, Summary: "Entrega followup aceptada.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-delivery"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{ReviewRequestID: followupReviewRequestRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), DeliveryRef: followupDeliveryRef, Summary: "Revisar followup.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-review-request"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{ReviewResultRef: followupReviewResultRef, ReviewRequestID: followupReviewRequestRef, DeliveryRef: followupDeliveryRef, Status: orquestacoreworkflow.ReviewResultStatusAcceptedV0, Summary: "Aceptado.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-review-result"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{AcceptedReviewRef: followupAcceptedReviewRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), ReviewRequestID: followupReviewRequestRef, DeliveryRef: followupDeliveryRef, Summary: "Aceptado por director.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-accepted"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{ReplanRef: "replan-ref-refresh-stale-closure", SourceRef: "rework-request-ref-refresh-stale-closure", RunRef: runRef, TaskRef: parentTaskRef, AcceptedAction: orquestacoreworkflow.ReplanDecisionActionSplitTaskV0, FollowupRefs: []string{followupTask.TaskRef}, Summary: "Split parent.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-replan"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 6, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{DeliveryRef: otherDeliveryRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), TaskID: otherTaskRef, AgentRef: otherAgentRef, Summary: "Entrega de otra ola.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-other-delivery"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 7, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{ReviewRequestID: otherReviewRequestRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), DeliveryRef: otherDeliveryRef, Summary: "Revisar otra ola.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-other-review-request"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 8, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{ReviewResultRef: otherReviewResultRef, ReviewRequestID: otherReviewRequestRef, DeliveryRef: otherDeliveryRef, Status: orquestacoreworkflow.ReviewResultStatusAcceptedV0, Summary: "Otra ola aceptada.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-other-review-result"}}),
				serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 9, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{AcceptedReviewRef: otherAcceptedReviewRef, PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), ReviewRequestID: otherReviewRequestRef, DeliveryRef: otherDeliveryRef, Summary: "Otra ola aceptada por director.", EvidenceRefs: []string{"evidence-ref-refresh-stale-closure-other-accepted"}}),
			}},
		},
		state,
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStateRefreshStaleClosureScopeV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-replan-or-close")
	if !changed ||
		recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.TaskRefs, followupTask.TaskRef) ||
		serviceStringInSetV0(replanStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(replanStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(replanStep.DeliveryRefs, followupDeliveryRef) ||
		!serviceStringInSetV0(replanStep.ReviewResultRefs, followupReviewResultRef) ||
		!serviceStringInSetV0(replanStep.AcceptedReviewRefs, followupAcceptedReviewRef) ||
		serviceStringInSetV0(replanStep.AcceptedReviewRefs, "accepted-review-ref-refresh-stale-closure-parent") ||
		serviceStringInSetV0(replanStep.TaskRefs, otherTaskRef) ||
		serviceStringInSetV0(replanStep.AgentRefs, otherAgentRef) ||
		serviceStringInSetV0(replanStep.DeliveryRefs, otherDeliveryRef) ||
		serviceStringInSetV0(replanStep.ReviewResultRefs, otherReviewResultRef) ||
		replanStep.Reason != "closure-scope-refreshed-from-open-reviewed-tasks" {
		t.Fatalf("changed=%v recovered=%+v replan=%+v", changed, recovered, replanStep)
	}
}

func TestOperationalDirectorPlanStateWithRequiredTestsPassedV0SustituyeScopeStaleDeCierre(t *testing.T) {
	runRef := "run-app-director-required-tests-refresh-replan-scope"
	planRef := "plan-ref-required-tests-refresh-replan-scope"
	parentTaskRef := "task-ref-required-tests-refresh-parent"
	followupTaskRef := "task-ref-required-tests-refresh-followup"
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTaskRef)
	followupDeliveryRef := "delivery-ref-required-tests-refresh-followup"
	followupReviewResultRef := "review-result-ref-required-tests-refresh-followup"
	passedEvidenceRef := "required-test-evidence-ref-required-tests-refresh-followup"
	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef,
		planRef,
		parentTaskRef,
		"wave-required-tests-refresh",
		"cohort-required-tests-refresh",
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: parentTaskRef},
		serviceOperationalClosureTaskRefsForTestV0{TaskRef: followupTaskRef},
	)
	state.ActiveStepID = "step-run-required-tests"
	state.PendingAgentRefs = nil
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].Reason = "wait-subagents-consumed"
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].TaskRefs = []string{followupTaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].AcceptedReviewRefs = []string{"accepted-review-ref-required-tests-refresh-followup"}
			state.Steps[index].Reason = "review-deliveries-accepted"
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{followupTaskRef}
			state.Steps[index].AgentRefs = []string{followupAgentRef}
			state.Steps[index].DeliveryRefs = []string{followupDeliveryRef}
			state.Steps[index].ReviewResultRefs = []string{followupReviewResultRef}
			state.Steps[index].Reason = "required-tests-running"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{parentTaskRef}
			state.Steps[index].AgentRefs = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parentTaskRef)}
			state.Steps[index].DeliveryRefs = []string{"delivery-ref-required-tests-refresh-parent"}
			state.Steps[index].ReviewResultRefs = []string{"review-result-ref-required-tests-refresh-parent"}
			state.Steps[index].RequiredTestEvidenceRefs = []string{"required-test-evidence-ref-required-tests-refresh-parent"}
			state.Steps[index].Reason = "stale-parent-scope"
		}
	}
	activeStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")

	next, changed, err := operationalDirectorPlanStateWithRequiredTestsPassedV0(
		ContinueAppDirectorRequestV0{
			RunRef:     runRef,
			OccurredAt: "2026-05-27T23:10:00Z",
		},
		state,
		activeStep,
		[]string{passedEvidenceRef},
	)
	if err != nil || !changed {
		t.Fatalf("operationalDirectorPlanStateWithRequiredTestsPassedV0 changed=%v err=%v", changed, err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-replan-or-close")
	if next.ActiveStepID != "step-replan-or-close" ||
		!serviceStringInSetV0(replanStep.TaskRefs, followupTaskRef) ||
		serviceStringInSetV0(replanStep.TaskRefs, parentTaskRef) ||
		!serviceStringInSetV0(replanStep.AgentRefs, followupAgentRef) ||
		!serviceStringInSetV0(replanStep.DeliveryRefs, followupDeliveryRef) ||
		!serviceStringInSetV0(replanStep.ReviewResultRefs, followupReviewResultRef) ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, passedEvidenceRef) ||
		serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, "required-test-evidence-ref-required-tests-refresh-parent") {
		t.Fatalf("replanStep=%+v next=%+v", replanStep, next)
	}
}
