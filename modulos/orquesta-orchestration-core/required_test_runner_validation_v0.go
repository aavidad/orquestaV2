package orquestacionnucleoapp

import "strings"

func validateRequiredTestExistingEvidenceForRequestV0(
	evidence RequiredTestEvidenceV0,
	request RequiredTestExecutionRequestV0,
	command string,
) []ErrorV0 {
	if err := ValidateRequiredTestEvidenceV0(evidence); err != nil {
		if issue, ok := err.(ErrorV0); ok {
			return []ErrorV0{issue}
		}
		return []ErrorV0{errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test_evidence", err.Error())}
	}
	if evidence.EvidenceRef != requiredTestEvidenceRefForCommandV0(request, command) ||
		evidence.RunRef != request.RunRef ||
		evidence.TaskRef != request.TaskRef ||
		evidence.TestCommand != command ||
		evidence.DeliveryRef != request.DeliveryRef ||
		evidence.ReviewRequestID != request.ReviewRequestID ||
		evidence.ReviewResultRef != request.ReviewResultRef ||
		evidence.AcceptedReviewRef != request.AcceptedReviewRef {
		return []ErrorV0{errorV0(
			ErrNucleoOrquestacionStoreV0,
			"required_test_evidence",
			"evidencia de test existente no corresponde al request causal",
		)}
	}
	return nil
}

func normalizeRequiredTestExecutionRequestV0(
	request RequiredTestExecutionRequestV0,
) RequiredTestExecutionRequestV0 {
	return RequiredTestExecutionRequestV0{
		RunRef:            strings.TrimSpace(request.RunRef),
		TaskRef:           strings.TrimSpace(request.TaskRef),
		TestCommands:      compactStringsV0(request.TestCommands),
		DeliveryRef:       strings.TrimSpace(request.DeliveryRef),
		ReviewRequestID:   strings.TrimSpace(request.ReviewRequestID),
		ReviewResultRef:   strings.TrimSpace(request.ReviewResultRef),
		AcceptedReviewRef: strings.TrimSpace(request.AcceptedReviewRef),
		OccurredAt:        strings.TrimSpace(request.OccurredAt),
		CorrelationID:     strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:      compactStringsV0(request.EvidenceRefs),
	}
}

func validateRequiredTestExecutionRequestV0(
	request RequiredTestExecutionRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for field, value := range map[string]string{
		"run_ref":             request.RunRef,
		"task_ref":            request.TaskRef,
		"delivery_ref":        request.DeliveryRef,
		"review_request_id":   request.ReviewRequestID,
		"review_result_ref":   request.ReviewResultRef,
		"accepted_review_ref": request.AcceptedReviewRef,
		"occurred_at":         request.OccurredAt,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, field, field+" requerido"))
		}
	}
	if len(request.TestCommands) == 0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "test_commands", "test_commands requerido"))
	}
	return issues
}

func validateRequiredTestExecutionPortsV0(
	runner RequiredTestRunnerV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if runner.Executor == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test_executor", "required_test_executor requerido"))
	}
	if runner.EvidenceWriter == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test_evidence_writer", "required_test_evidence_writer requerido"))
	}
	return issues
}

func normalizeRequiredTestCommandExecutionResultV0(
	result RequiredTestCommandExecutionResultV0,
) RequiredTestCommandExecutionResultV0 {
	return RequiredTestCommandExecutionResultV0{
		Status:       RequiredTestEvidenceStatusV0(strings.TrimSpace(string(result.Status))),
		EvidenceRefs: compactStringsV0(result.EvidenceRefs),
	}
}

func validateRequiredTestCommandExecutionResultV0(
	result RequiredTestCommandExecutionResultV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if result.Status != RequiredTestEvidenceStatusPassedV0 &&
		result.Status != RequiredTestEvidenceStatusFailedV0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test.status", "status invalido"))
	}
	if len(result.EvidenceRefs) == 0 {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test.evidence_refs", "evidence_refs requerido"))
	}
	return issues
}
