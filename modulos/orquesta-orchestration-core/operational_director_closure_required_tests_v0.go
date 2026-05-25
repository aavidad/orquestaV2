package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func (closer OperationalDirectorClosureV0) operationalDirectorClosureRequiredTestIssuesV0(
	ctx context.Context,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	trace operationalDirectorClosureTraceV0,
	request OperationalDirectorClosureRequestV0,
) ([]ErrorV0, error) {
	requiredTests := operationalDirectorClosureRequiredTestsForTaskV0(tasks, request.TaskID)
	if len(requiredTests) == 0 {
		return nil, nil
	}
	if len(request.RequiredTestEvidenceRefs) == 0 {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_refs",
			"evidencia durable de tests requerida",
		)}, nil
	}
	if closer.RequiredTestEvidenceStore == nil {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_store",
			"store de evidencias de tests requerido",
		)}, nil
	}
	accepted := trace.AcceptedReviews[request.AcceptedReviewRef]
	reviewResult, ok := operationalDirectorClosureAcceptedResultV0(trace, accepted, request.DeliveryRef)
	if !ok {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"review_result_ref",
			"resultado aceptado no encontrado para tests requeridos",
		)}, nil
	}
	evidence, err := closer.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(
		ctx,
		request.RunRef,
		request.RequiredTestEvidenceRefs,
	)
	if err != nil {
		return nil, err
	}
	if !operationalDirectorClosureRequiredTestsSatisfiedV0(requiredTests, evidence, reviewResult, request) {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"required_test_evidence_refs",
			"tests requeridos sin evidencia passed causal",
		)}, nil
	}
	return nil, nil
}

func operationalDirectorClosureRequiredTestsForTaskV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) []string {
	taskID = strings.TrimSpace(taskID)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskID {
			return compactStringsV0(task.RequiredTests)
		}
	}
	return nil
}

func operationalDirectorClosureRequiredTestsSatisfiedV0(
	requiredTests []string,
	evidence []RequiredTestEvidenceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	request OperationalDirectorClosureRequestV0,
) bool {
	for _, required := range compactStringsV0(requiredTests) {
		if !operationalDirectorClosureRequiredTestSatisfiedV0(required, evidence, reviewResult, request) {
			return false
		}
	}
	return len(requiredTests) > 0
}

func operationalDirectorClosureRequiredTestSatisfiedV0(
	required string,
	evidence []RequiredTestEvidenceV0,
	reviewResult orquestacoreworkflow.ReviewResultV0,
	request OperationalDirectorClosureRequestV0,
) bool {
	required = strings.TrimSpace(required)
	for _, item := range evidence {
		if item.Status == RequiredTestEvidenceStatusPassedV0 &&
			strings.TrimSpace(item.RunRef) == request.RunRef &&
			operationalDirectorClosureReflectedV0(request.RequiredTestEvidenceRefs, item.EvidenceRef) &&
			strings.TrimSpace(item.TaskRef) == request.TaskID &&
			strings.TrimSpace(item.TestCommand) == required &&
			strings.TrimSpace(item.DeliveryRef) == request.DeliveryRef &&
			strings.TrimSpace(item.ReviewRequestID) == strings.TrimSpace(reviewResult.ReviewRequestID) &&
			strings.TrimSpace(item.ReviewResultRef) == strings.TrimSpace(reviewResult.ReviewResultRef) &&
			strings.TrimSpace(item.AcceptedReviewRef) == request.AcceptedReviewRef {
			return true
		}
	}
	return false
}
