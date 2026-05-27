package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strconv"
	"testing"
)

func serviceOperationalClosurePlanStateReadyForCloseTasksV0(
	runRef string,
	planRef string,
	parentTaskRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.ActiveParentTaskRef = parentTaskRef
	taskRefs := make([]string, 0, len(tasks))
	agentRefs := make([]string, 0, len(tasks))
	deliveryRefs := make([]string, 0, len(tasks))
	reviewResultRefs := make([]string, 0, len(tasks))
	acceptedRefs := make([]string, 0, len(tasks))
	testEvidenceRefs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskRefs = append(taskRefs, task.TaskRef)
		agentRefs = append(agentRefs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef))
		deliveryRefs = append(deliveryRefs, task.DeliveryRef)
		reviewResultRefs = append(reviewResultRefs, task.ReviewResultRef)
		acceptedRefs = append(acceptedRefs, task.AcceptedRef)
		testEvidenceRefs = append(testEvidenceRefs, task.TestEvidenceRef)
	}
	state.Steps = []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:             "step-review-deliveries",
			Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
			Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			TaskRefs:           taskRefs,
			DeliveryRefs:       deliveryRefs,
			ReviewResultRefs:   reviewResultRefs,
			AcceptedReviewRefs: acceptedRefs,
		},
		{
			StepID:                   "step-run-required-tests",
			Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
			Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			TaskRefs:                 taskRefs,
			RequiredTestEvidenceRefs: testEvidenceRefs,
		},
		{
			StepID:                   "step-replan-or-close",
			Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:                  "wave-service-operational-closure-plan-state",
			CohortRef:                "cohort-service-operational-closure-plan-state",
			ParentTaskRef:            parentTaskRef,
			TaskRefs:                 taskRefs,
			AgentRefs:                agentRefs,
			DeliveryRefs:             deliveryRefs,
			ReviewResultRefs:         reviewResultRefs,
			RequiredTestEvidenceRefs: testEvidenceRefs,
		},
	}
	return state
}

type serviceOperationalClosureEventReaderForTestV0 struct {
	Events []orquestacoreworkflow.OrchestrationEventV0
}

func (reader serviceOperationalClosureEventReaderForTestV0) LoadRunEventsV0(
	_ context.Context,
	_ string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	return append([]orquestacoreworkflow.OrchestrationEventV0(nil), reader.Events...), nil
}

func serviceOperationalClosureEventsForTasksForTestV0(
	t *testing.T,
	runRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(tasks)*4)
	sequence := int64(1)
	for _, task := range tasks {
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
				DeliveryRef:  task.DeliveryRef,
				PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskID:       task.TaskRef,
				AgentRef:     orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef),
				Summary:      "Entrega para cierre operativo multitarea.",
				EvidenceRefs: []string{task.DeliveryRef},
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
				ReviewRequestID: task.ReviewRequestID,
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     task.DeliveryRef,
				Summary:         "Review para cierre operativo multitarea.",
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
				ReviewResultRef: task.ReviewResultRef,
				ReviewRequestID: task.ReviewRequestID,
				DeliveryRef:     task.DeliveryRef,
				Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
				Summary:         "Review aceptada.",
				EvidenceRefs:    []string{task.TestEvidenceRef},
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
				AcceptedReviewRef: task.AcceptedRef,
				PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				ReviewRequestID:   task.ReviewRequestID,
				DeliveryRef:       task.DeliveryRef,
				Summary:           "Review aceptada.",
			}),
		)
		sequence++
	}
	return events
}

func newServiceOperationalClosureEventReaderForTestV0(
	t *testing.T,
	runRef string,
) serviceOperationalClosureEventReaderForTestV0 {
	t.Helper()
	return serviceOperationalClosureEventReaderForTestV0{Events: []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalClosureEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  "delivery-ref-service-operational-closure-001",
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       "task-ref-service-operational-closure-001",
			AgentRef:     orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001"),
			Summary:      "Entrega para cierre operativo.",
			EvidenceRefs: []string{"delivery-ref-service-operational-closure-001"},
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: "review-request-ref-service-operational-closure-001",
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     "delivery-ref-service-operational-closure-001",
			Summary:         "Review para cierre operativo.",
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-ref-service-operational-closure-001",
			ReviewRequestID: "review-request-ref-service-operational-closure-001",
			DeliveryRef:     "delivery-ref-service-operational-closure-001",
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review aceptada.",
			EvidenceRefs:    []string{"test-evidence-ref-service-operational-closure-001"},
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   "review-request-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			Summary:           "Review aceptada.",
		}),
	}}
}

func serviceOperationalClosureEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	eventType string,
	payload any,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestacoreworkflow.OrchestrationEventV0{
		EventID:        "evt-service-operational-closure-" + eventType + "-" + strconv.FormatInt(sequence, 10),
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}

func serviceOperationalClosureTaskForTestV0(runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-ref-service-operational-closure-001",
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo service",
		WriteSet:           []string{"internal/module"},
		AcceptanceCriteria: []string{"entrega revisada"},
		RequiredTests:      []string{"go test ./..."},
	}
}

func serviceOperationalClosureTaskWithRefsForTestV0(
	runRef string,
	parentTaskRef string,
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             refs.TaskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo multitarea",
		WriteSet:           []string{"internal/module"},
		AcceptanceCriteria: []string{"entrega revisada"},
		RequiredTests:      []string{"go test ./..."},
		ParentTaskRef:      parentTaskRef,
		CohortRef:          "cohort-service-operational-closure-plan-state",
		WaveRef:            "wave-service-operational-closure-plan-state",
		DelegationDepth:    1,
		MaxChildAgents:     6,
	}
}

func serviceOperationalClosureRequiredTestEvidenceForTestV0(
	runRef string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-service-operational-closure-001",
		RunRef:            runRef,
		TaskRef:           "task-ref-service-operational-closure-001",
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       "delivery-ref-service-operational-closure-001",
		ReviewRequestID:   "review-request-ref-service-operational-closure-001",
		ReviewResultRef:   "review-result-ref-service-operational-closure-001",
		AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
		OccurredAt:        "2026-05-17T15:45:00Z",
		EvidenceRefs:      []string{"evidence-ref-service-operational-closure-001"},
	}
}
