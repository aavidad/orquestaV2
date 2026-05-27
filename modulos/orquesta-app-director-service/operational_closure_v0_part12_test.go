package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0NoCierraConPlanStateFueraDeReplanOrCloseRunning(t *testing.T) {
	type closureBlockedStep struct {
		Name   string
		Kind   orquestadirectoroperativo.OperationalDirectorStepKindV0
		Status orquestadirectoroperativo.OperationalDirectorStepStatusV0
	}
	for _, tc := range []closureBlockedStep{
		{Name: "launch-running", Kind: orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "gather-context-running", Kind: orquestadirectoroperativo.OperationalDirectorStepGatherContextV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "split-work-running", Kind: orquestadirectoroperativo.OperationalDirectorStepSplitWorkV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "govern-delegation-running", Kind: orquestadirectoroperativo.OperationalDirectorStepGovernDelegationV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "replan-or-close-pending", Kind: orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0, Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			runRef := "run-service-operational-closure-plan-state-" + tc.Name
			planRef := "plan-ref-service-operational-closure-plan-state-" + tc.Name
			run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
			run.Tasks = []string{"task-ref-service-operational-closure-001"}
			run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
			run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
			source := &serviceOperationalClosureSourceForTestV0{
				Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
					TaskID:                   "task-ref-service-operational-closure-001",
					DeliveryRef:              "delivery-ref-service-operational-closure-001",
					AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
					ValidationRef:            "validation-ref-service-operational-closure-plan-state-" + tc.Name,
					ClosureRef:               "closure-ref-service-operational-closure-plan-state-" + tc.Name,
					RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
					EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
				},
			}
			state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
			state.ActiveStepID = "step-active-before-replan"
			state.Steps = []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
				StepID:    "step-active-before-replan",
				Kind:      tc.Kind,
				Status:    tc.Status,
				TaskRefs:  []string{"task-ref-service-operational-closure-001"},
				AgentRefs: []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
			}}
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
			loop, issues, err := maybeCloseOperationalDirectorV0(
				context.Background(),
				ContinueAppDirectorRequestV0{
					RunRef:                     runRef,
					OccurredAt:                 "2026-05-22T11:10:00Z",
					OperationalDirectorPlanRef: planRef,
				},
				StartAppDirectorPortsV0{
					RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
					EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
					EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
					DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
					RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
					OperationalClosureSource:   source,
					OperationalPlanStateStore:  planStateStore,
					OperationalPlanStateWriter: planStateStore,
				},
				orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
					Run:    run,
				},
				orquestacionnucleoapp.ProgressiveLoopRequestV0{},
			)
			if err != nil || len(issues) != 0 {
				t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
			}
			if source.Called {
				t.Fatalf("closure source no debe invocarse con active step %s/%s", tc.Kind, tc.Status)
			}
			if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
				t.Fatalf("run cerrado con active step %s/%s: %+v", tc.Kind, tc.Status, loop.Run)
			}
		})
	}
}

type serviceOperationalClosureSourceForTestV0 struct {
	Request     orquestacionnucleoapp.OperationalDirectorClosureRequestV0
	LastRequest AppDirectorOperationalClosureRequestV0
	Called      bool
	Reject      bool
}

func (source *serviceOperationalClosureSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	if source.Reject {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
	}
	return source.Request, true, nil
}

type serviceOperationalClosureOpenTaskSourceForTestV0 struct {
	Requests  map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0
	LastInput AppDirectorOperationalClosureRequestV0
	TaskCalls []string
}

func (source *serviceOperationalClosureOpenTaskSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.LastInput = request
	for _, taskRef := range request.Run.Tasks {
		if serviceStringInSetV0(request.Run.ClosedTasks, taskRef) {
			continue
		}
		closureRequest, ok := source.Requests[taskRef]
		if !ok {
			return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
		}
		source.TaskCalls = append(source.TaskCalls, taskRef)
		return closureRequest, true, nil
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

type serviceOperationalClosureTaskRefsForTestV0 struct {
	TaskRef         string
	DeliveryRef     string
	ReviewRequestID string
	ReviewResultRef string
	AcceptedRef     string
	TestEvidenceRef string
	ValidationRef   string
	ClosureRef      string
}

func serviceOperationalClosureRequestForTaskRefsV0(
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorClosureRequestV0 {
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   refs.TaskRef,
		DeliveryRef:              refs.DeliveryRef,
		AcceptedReviewRef:        refs.AcceptedRef,
		ValidationRef:            refs.ValidationRef,
		ClosureRef:               refs.ClosureRef,
		RequiredTestEvidenceRefs: []string{refs.TestEvidenceRef},
		EvidenceRefs:             []string{refs.DeliveryRef, refs.TestEvidenceRef},
	}
}

func serviceOperationalClosureReviewResultProjectionForTestV0(
	refs serviceOperationalClosureTaskRefsForTestV0,
) string {
	return refs.ReviewResultRef +
		"#review_result:" + string(orquestacoreworkflow.ReviewResultStatusAcceptedV0) +
		"#review_request:" + refs.ReviewRequestID +
		"#delivery:" + refs.DeliveryRef
}

func serviceCountEventsByTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func serviceCountStringV0(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func serviceFirstAgentFollowupFromReplanProjectionForTestV0(t *testing.T, projection string) string {
	t.Helper()
	for _, part := range strings.Split(projection, "#") {
		if !strings.HasPrefix(part, "followups:") {
			continue
		}
		for _, ref := range strings.Split(strings.TrimPrefix(part, "followups:"), "+") {
			if strings.HasPrefix(ref, "agent-ref-") {
				return ref
			}
		}
	}
	t.Fatalf("proyeccion sin followup agent: %s", projection)
	return ""
}

func serviceOperationalClosurePlanStateReadyForCloseV0(
	runRef string,
	planRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-" + planRef,
		PlanRef:             planRef,
		RequestRef:          "request-ref-" + planRef,
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-replan-or-close",
		RequiredTestRefs:    []string{"go test ./..."},
		ObservedAt:          "2026-05-17T16:00:00Z",
		ActiveWaveRef:       "wave-service-operational-closure-plan-state",
		ActiveCohortRef:     "cohort-service-operational-closure-plan-state",
		ActiveParentTaskRef: "parent-task-service-operational-closure-plan-state",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				DeliveryRefs:       []string{"delivery-ref-service-operational-closure-001"},
				ReviewResultRefs:   []string{"review-result-ref-service-operational-closure-001"},
				AcceptedReviewRefs: []string{"accepted-review-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-run-required-tests",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-replan-or-close",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:                  "wave-service-operational-closure-plan-state",
				CohortRef:                "cohort-service-operational-closure-plan-state",
				ParentTaskRef:            "parent-task-service-operational-closure-plan-state",
				TaskRefs:                 []string{"task-ref-service-operational-closure-001"},
				AgentRefs:                []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
				DeliveryRefs:             []string{"delivery-ref-service-operational-closure-001"},
				ReviewResultRefs:         []string{"review-result-ref-service-operational-closure-001"},
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
		},
	}
}
