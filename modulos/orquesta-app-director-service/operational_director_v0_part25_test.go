package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func serviceOperationalDirectorPlanStateReviewFixtureForTestV0(
	t *testing.T,
	withRequiredTests bool,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	runRef := "run-app-director-operational-plan-state-review-accepted"
	planRef := "plan-ref-app-director-operational-plan-state-review-accepted"
	taskRef := "task-ref-app-director-operational-plan-state-review-accepted"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	waveRef := "wave-app-director-operational-plan-state-review-accepted"
	cohortRef := "cohort-app-director-operational-plan-state-review-accepted"
	parentTaskRef := "parent-task-ref-app-director-operational-plan-state-review-accepted"
	deliveryRef := "delivery-ref-app-director-operational-plan-state-review-accepted"
	reviewRequestID := "review-request-ref-app-director-operational-plan-state-review-accepted"
	reviewResultRef := "review-result-ref-app-director-operational-plan-state-review-accepted"
	acceptedReviewRef := "accepted-review-ref-app-director-operational-plan-state-review-accepted"
	requiredTest := "go test -count=1 ./modulos/orquesta-app-director-service"
	requiredTestEvidenceRef := "test-evidence-ref-" + deliveryRef
	requiredTests := []string(nil)
	reviewResultEvidenceRefs := []string{"evidence-ref-review-result-accepted"}
	if withRequiredTests {
		requiredTests = []string{requiredTest}
		reviewResultEvidenceRefs = append(reviewResultEvidenceRefs, requiredTestEvidenceRef)
	}
	run := serviceRunForWaitRefsTestV0(runRef, taskRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{reviewRequestID}
	run.ReviewResults = []string{reviewResultRef + "#review_result:accepted#review_request:" + reviewRequestID + "#delivery:" + deliveryRef}
	run.AcceptedReviews = []string{acceptedReviewRef}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-app-director-operational-plan-state-review-accepted",
		PlanRef:             planRef,
		RequestRef:          "request-ref-app-director-operational-plan-state-review-accepted",
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-review-deliveries",
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		RequiredTestRefs:    requiredTests,
		ObservedAt:          "2026-05-17T14:19:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:        "step-launch-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-wait-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
				WaitRefs:      []string{"wait-ref-app-director-operational-plan-state-review-accepted"},
			},
			{
				StepID:        "step-review-deliveries",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-run-required-tests",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-replan-or-close",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
		},
	}
	events := []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  deliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       taskRef,
			AgentRef:     agentRef,
			Summary:      "Entrega del scope activo.",
			EvidenceRefs: []string{"evidence-ref-delivery-review-accepted"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestID,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Review del scope activo.",
			EvidenceRefs:    []string{"evidence-ref-review-requested"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestID,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review aceptada.",
			EvidenceRefs:    reviewResultEvidenceRefs,
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestID,
			DeliveryRef:       deliveryRef,
			Summary:           "Review aceptada.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted"},
		}),
	}
	return serviceOperationalDirectorPlanStateReviewFixtureV0{
		RunRef:                  runRef,
		PlanRef:                 planRef,
		TaskRef:                 taskRef,
		AgentRef:                agentRef,
		WaveRef:                 waveRef,
		CohortRef:               cohortRef,
		ParentTaskRef:           parentTaskRef,
		DeliveryRef:             deliveryRef,
		ReviewRequestID:         reviewRequestID,
		ReviewResultRef:         reviewResultRef,
		AcceptedReviewRef:       acceptedReviewRef,
		RequiredTest:            requiredTest,
		RequiredTestEvidenceRef: requiredTestEvidenceRef,
		Run:                     run,
		State:                   state,
		Events:                  events,
	}
}

func serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
	t *testing.T,
	status orquestacoreworkflow.ReviewResultStatusV0,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.ReviewResultRef = "review-result-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.AcceptedReviewRef = ""
	fixture.ReworkRequestRef = "rework-request-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.ReplanDecisionRef = "replan-decision-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.Run.ReviewResults = []string{
		fixture.ReviewResultRef + "#review_result:" + string(status) +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.AcceptedReviews = nil
	fixture.Run.ReworkRequests = []string{
		fixture.ReworkRequestRef + "#review_result:" + fixture.ReviewResultRef +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + fixture.TaskRef,
	}
	fixture.Events = []orquestacoreworkflow.OrchestrationEventV0{
		fixture.Events[0],
		fixture.Events[1],
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: fixture.ReviewResultRef,
			ReviewRequestID: fixture.ReviewRequestID,
			DeliveryRef:     fixture.DeliveryRef,
			Status:          status,
			Summary:         "Review pide cambios.",
			EvidenceRefs:    []string{"evidence-ref-review-result-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 4, orquestacoreworkflow.OrchestrationEventReworkRequestedV0, orquestacoreworkflow.ReworkRequestedPayloadV0{
			ReworkRequestRef: fixture.ReworkRequestRef,
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  fixture.ReviewResultRef,
			ReviewRequestID:  fixture.ReviewRequestID,
			DeliveryRef:      fixture.DeliveryRef,
			Summary:          "Rework requerido por review negativa.",
			EvidenceRefs:     []string{"evidence-ref-rework-requested-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      fixture.ReplanDecisionRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      fixture.ReworkRequestRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{fixture.TaskRef},
			Summary:        "Replan por review negativa.",
			EvidenceRefs:   []string{"evidence-ref-replan-decision-" + string(status)},
		}),
	}
	return fixture
}

func serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       fixture.RequiredTestEvidenceRef,
		RunRef:            fixture.RunRef,
		TaskRef:           fixture.TaskRef,
		TestCommand:       fixture.RequiredTest,
		Status:            status,
		DeliveryRef:       fixture.DeliveryRef,
		ReviewRequestID:   fixture.ReviewRequestID,
		ReviewResultRef:   fixture.ReviewResultRef,
		AcceptedReviewRef: fixture.AcceptedReviewRef,
		OccurredAt:        "2026-05-17T14:20:00Z",
		EvidenceRefs:      []string{"test-output-ref-" + fixture.DeliveryRef},
	}
}
