package orquestacionnucleoapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type RequiredTestRunnerV0 struct {
	Executor       RequiredTestCommandExecutorPortV0
	EvidenceWriter RequiredTestEvidenceWriterPortV0
}

type RequiredTestExecutionRequestV0 struct {
	RunRef            string
	TaskRef           string
	TestCommands      []string
	DeliveryRef       string
	ReviewRequestID   string
	ReviewResultRef   string
	AcceptedReviewRef string
	OccurredAt        string
	CorrelationID     string
	EvidenceRefs      []string
}

type RequiredTestCommandExecutionRequestV0 struct {
	RunRef        string
	TaskRef       string
	TestCommand   string
	CorrelationID string
	EvidenceRefs  []string
}

type RequiredTestCommandExecutionResultV0 struct {
	Status       RequiredTestEvidenceStatusV0
	EvidenceRefs []string
}

type RequiredTestExecutionResultV0 struct {
	EvidenceRefs       []string
	PassedEvidenceRefs []string
	FailedEvidenceRefs []string
	Issues             []ErrorV0
}

type RequiredTestRunnerPortV0 interface {
	RunRequiredTestsV0(
		context.Context,
		RequiredTestExecutionRequestV0,
	) (RequiredTestExecutionResultV0, error)
}

type RequiredTestCommandExecutorPortV0 interface {
	RunRequiredTestCommandV0(
		context.Context,
		RequiredTestCommandExecutionRequestV0,
	) (RequiredTestCommandExecutionResultV0, error)
}

func (runner RequiredTestRunnerV0) RunRequiredTestsV0(
	ctx context.Context,
	request RequiredTestExecutionRequestV0,
) (RequiredTestExecutionResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeRequiredTestExecutionRequestV0(request)
	if issues := validateRequiredTestExecutionRequestV0(request, runner); len(issues) > 0 {
		return RequiredTestExecutionResultV0{Issues: issues}, nil
	}

	result := RequiredTestExecutionResultV0{}
	for _, command := range request.TestCommands {
		execution, err := runner.Executor.RunRequiredTestCommandV0(
			ctx,
			RequiredTestCommandExecutionRequestV0{
				RunRef:        request.RunRef,
				TaskRef:       request.TaskRef,
				TestCommand:   command,
				CorrelationID: request.CorrelationID,
				EvidenceRefs:  request.EvidenceRefs,
			},
		)
		if err != nil {
			return result, err
		}
		execution = normalizeRequiredTestCommandExecutionResultV0(execution)
		if issues := validateRequiredTestCommandExecutionResultV0(execution); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			continue
		}
		evidence := RequiredTestEvidenceV0{
			SchemaVersion:     RequiredTestEvidenceSchemaVersionV0,
			EvidenceRef:       requiredTestEvidenceRefForCommandV0(request, command),
			RunRef:            request.RunRef,
			TaskRef:           request.TaskRef,
			TestCommand:       command,
			Status:            execution.Status,
			DeliveryRef:       request.DeliveryRef,
			ReviewRequestID:   request.ReviewRequestID,
			ReviewResultRef:   request.ReviewResultRef,
			AcceptedReviewRef: request.AcceptedReviewRef,
			OccurredAt:        request.OccurredAt,
			EvidenceRefs: compactStringsV0(append(
				append([]string(nil), request.EvidenceRefs...),
				execution.EvidenceRefs...,
			)),
		}
		normalized, err := NewRequiredTestEvidenceV0(evidence)
		if err != nil {
			return result, err
		}
		if err := runner.EvidenceWriter.SaveRequiredTestEvidenceV0(ctx, normalized); err != nil {
			return result, err
		}
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, normalized.EvidenceRef))
		if normalized.Status == RequiredTestEvidenceStatusPassedV0 {
			result.PassedEvidenceRefs = compactStringsV0(append(result.PassedEvidenceRefs, normalized.EvidenceRef))
		}
		if normalized.Status == RequiredTestEvidenceStatusFailedV0 {
			result.FailedEvidenceRefs = compactStringsV0(append(result.FailedEvidenceRefs, normalized.EvidenceRef))
		}
	}
	return result, nil
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
	runner RequiredTestRunnerV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if runner.Executor == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test_executor", "required_test_executor requerido"))
	}
	if runner.EvidenceWriter == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "required_test_evidence_writer", "required_test_evidence_writer requerido"))
	}
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

func requiredTestEvidenceRefForCommandV0(
	request RequiredTestExecutionRequestV0,
	command string,
) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		request.RunRef,
		request.TaskRef,
		command,
		request.DeliveryRef,
		request.ReviewRequestID,
		request.ReviewResultRef,
		request.AcceptedReviewRef,
	}, "\x00")))
	return "test-evidence-ref-v0-" + hex.EncodeToString(hash[:])[:24]
}
