package orquestaappdirectorservice

import (
	"context"
	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func serviceOperationalDirectorWorkflowTaskForFixtureV0(
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             fixture.TaskRef,
		RunID:              fixture.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Tarea del ciclo integrado del Director Operativo",
		Summary:            "Trabajo de programacion con review, runner requerido y cierre causal.",
		WriteSet:           []string{"modulos/orquesta-app-director-service"},
		AcceptanceCriteria: []string{"Review aceptada, test requerido evidenciado y cierre causal."},
		RequiredTests:      []string{fixture.RequiredTest},
		ParentTaskRef:      fixture.ParentTaskRef,
		CohortRef:          fixture.CohortRef,
		WaveRef:            fixture.WaveRef,
		DelegationDepth:    1,
		MaxChildAgents:     0,
	}
}

type serviceOperationalDirectorClosureSourceFromRequestForTestV0 struct {
	Fixture     serviceOperationalDirectorPlanStateReviewFixtureV0
	LastRequest AppDirectorOperationalClosureRequestV0
	Called      bool
}

func (source *serviceOperationalDirectorClosureSourceFromRequestForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   source.Fixture.TaskRef,
		DeliveryRef:              source.Fixture.DeliveryRef,
		AcceptedReviewRef:        source.Fixture.AcceptedReviewRef,
		ValidationRef:            "validation-ref-service-review-runner-close-001",
		ClosureRef:               "closure-ref-service-review-runner-close-001",
		RequiredTestEvidenceRefs: append([]string(nil), request.RequiredTestEvidenceRefs...),
		EvidenceRefs:             compactServiceRefsV0(append(request.EvidenceRefs, "evidence-ref-service-review-runner-close-001")),
	}, true, nil
}

type serviceOperationalDirectorClosureSourceRefsForTestV0 struct {
	TaskRef           string
	DeliveryRef       string
	AcceptedReviewRef string
	ValidationRef     string
	ClosureRef        string
	LastRequest       AppDirectorOperationalClosureRequestV0
	Called            bool
}

func (source *serviceOperationalDirectorClosureSourceRefsForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   source.TaskRef,
		DeliveryRef:              source.DeliveryRef,
		AcceptedReviewRef:        source.AcceptedReviewRef,
		ValidationRef:            source.ValidationRef,
		ClosureRef:               source.ClosureRef,
		RequiredTestEvidenceRefs: append([]string(nil), request.RequiredTestEvidenceRefs...),
		EvidenceRefs:             compactServiceRefsV0(append(request.EvidenceRefs, "evidence-ref-service-closure-source-refs-test-v0")),
	}, true, nil
}

type serviceOperationalDirectorSplitReviewReworkReplanSourceV0 struct {
	Fixture    serviceOperationalDirectorPlanStateReviewFixtureV0
	SplitTasks []orquestacoreworkflow.WorkflowTaskV0
}

func (source serviceOperationalDirectorSplitReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	if !serviceRunProjectionHasRefPrefixV0(request.Run.ReworkRequests, source.Fixture.ReworkRequestRef, "#review_result:") ||
		serviceRunHasAnyRefV0(request.Run.Agents, serviceWorkflowTaskAgentRefsV0(source.SplitTasks)...) {
		return nil, nil
	}
	return []orquestacionnucleoapp.ReviewReworkReplanPlanV0{{
		CandidateRef:     "candidate-ref-app-director-review-rework-durable-split",
		ReplanRef:        source.Fixture.ReplanDecisionRef,
		SignalRef:        "signal-ref-app-director-review-rework-durable-split",
		ReworkRequestRef: source.Fixture.ReworkRequestRef,
		TaskRef:          source.Fixture.TaskRef,
		ReasonRef:        "review-rework-durable-split-required",
		RequestedAction:  orquestacorereplanner.ReplanActionSplitTaskV0,
		Summary:          "Dividir retrabajo de review negativa en microtareas durables.",
		EvidenceRefs:     []string{"evidence-ref-app-director-review-rework-durable-split"},
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: source.Fixture.ReviewResultRef,
			ReviewRequestID: source.Fixture.ReviewRequestID,
			DeliveryRef:     source.Fixture.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "Review negativa requiere split.",
			EvidenceRefs:    []string{"evidence-ref-review-result-durable-split"},
		},
		SplitTasks: append([]orquestacoreworkflow.WorkflowTaskV0(nil), source.SplitTasks...),
	}}, nil
}

type changesRequestedServiceReviewGateSourceV0 struct {
	Fixture serviceOperationalDirectorPlanStateReviewFixtureV0
	Called  bool
}

func (source *changesRequestedServiceReviewGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	_ orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	source.Called = true
	return []orquestacionnucleoapp.ReviewGateObservationV0{{
		ReviewRequestID:  source.Fixture.ReviewRequestID,
		ReviewResultRef:  source.Fixture.ReviewResultRef,
		ReworkRequestRef: source.Fixture.ReworkRequestRef,
		DeliveryRef:      source.Fixture.DeliveryRef,
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:           orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		Summary:          "Review pide split duradero de followups.",
		EvidenceRefs: []string{
			"evidence-ref-service-review-gate-changes-requested",
		},
	}}, nil
}

type acceptedServiceReviewGateSourceV0 struct {
	Fixture serviceOperationalDirectorPlanStateReviewFixtureV0
	Called  bool
}

func (source *acceptedServiceReviewGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	_ orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	source.Called = true
	return []orquestacionnucleoapp.ReviewGateObservationV0{{
		ReviewRequestID:   source.Fixture.ReviewRequestID,
		ReviewResultRef:   source.Fixture.ReviewResultRef,
		AcceptedReviewRef: source.Fixture.AcceptedReviewRef,
		DeliveryRef:       source.Fixture.DeliveryRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:            orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:           "Review aceptada por fuente inyectada.",
		EvidenceRefs: []string{
			"evidence-ref-service-review-gate-accepted",
			source.Fixture.RequiredTestEvidenceRef,
		},
	}}, nil
}

func serviceWorkflowTaskAgentRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactServiceRefsV0(refs)
}

func serviceRunHasAnyRefV0(values []string, refs ...string) bool {
	for _, ref := range refs {
		if serviceStringInSetV0(values, ref) {
			return true
		}
	}
	return false
}

func serviceRunProjectionHasRefPrefixV0(values []string, ref string, marker string) bool {
	ref = strings.TrimSpace(ref)
	marker = strings.TrimSpace(marker)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == ref || strings.HasPrefix(value, ref+marker) {
			return true
		}
	}
	return false
}

type serviceContinueWaiterForTestV0 struct {
	Continue bool
	Calls    int
}

func (waiter *serviceContinueWaiterForTestV0) WaitExternalProgressV0(
	_ context.Context,
	_ orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	waiter.Calls++
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     waiter.Continue,
		EvidenceRefs: []string{"evidence-ref-service-continue-waiter-v0"},
	}, nil
}

func serviceOperationalDirectorReplanSplitTaskForTestV0(
	runRef string,
	taskRef string,
	parentTaskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Tarea split de replan",
		Summary:       "Corregir una parte acotada tras review negativa.",
		WriteSet:      []string{"app/replan_split_" + taskRef + ".go"},
		AcceptanceCriteria: []string{
			"Entrega corregida y trazable para nueva review.",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-director-service"},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:rework-split:v0", FunctionName: "NewWorkflowTaskV0"},
		},
		ParentTaskRef:   parentTaskRef,
		CohortRef:       "cohort-ref-app-director-replan-split-followups",
		WaveRef:         "wave-ref-app-director-replan-split-followups",
		DelegationDepth: 1,
		MaxChildAgents:  0,
	}
}

type fakeServiceRequiredTestCommandExecutorV0 struct {
	results  map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0
	commands []string
}

func (executor *fakeServiceRequiredTestCommandExecutorV0) RunRequiredTestCommandV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	executor.commands = append(executor.commands, request.TestCommand)
	return executor.results[request.TestCommand], nil
}
