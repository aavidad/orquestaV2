package orquestacionnucleoapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type RequiredTestRunnerV0 struct {
	Executor       RequiredTestCommandExecutorPortV0
	EvidenceReader RequiredTestEvidenceReaderPortV0
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
	if issues := validateRequiredTestExecutionRequestV0(request); len(issues) > 0 {
		return RequiredTestExecutionResultV0{Issues: issues}, nil
	}

	result := RequiredTestExecutionResultV0{}
	evidenceReader := runner.requiredTestEvidenceReaderV0()
	for _, command := range request.TestCommands {
		if evidenceReader != nil {
			existing, ok, err := requiredTestRunnerExistingEvidenceForCommandV0(ctx, evidenceReader, request, command)
			if err != nil {
				return result, err
			}
			if ok {
				if issues := validateRequiredTestExistingEvidenceForRequestV0(existing, request, command); len(issues) > 0 {
					result.Issues = append(result.Issues, issues...)
					continue
				}
				result = appendRequiredTestEvidenceResultV0(result, existing)
				continue
			}
		}
		if issues := validateRequiredTestExecutionPortsV0(runner); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			return result, nil
		}
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
		result = appendRequiredTestEvidenceResultV0(result, normalized)
	}
	return result, nil
}

func (runner RequiredTestRunnerV0) requiredTestEvidenceReaderV0() RequiredTestEvidenceReaderPortV0 {
	if runner.EvidenceReader != nil {
		return runner.EvidenceReader
	}
	if reader, ok := runner.EvidenceWriter.(RequiredTestEvidenceReaderPortV0); ok {
		return reader
	}
	return nil
}

func requiredTestRunnerExistingEvidenceForCommandV0(
	ctx context.Context,
	reader RequiredTestEvidenceReaderPortV0,
	request RequiredTestExecutionRequestV0,
	command string,
) (RequiredTestEvidenceV0, bool, error) {
	ref := requiredTestEvidenceRefForCommandV0(request, command)
	evidence, err := reader.LoadRequiredTestEvidenceV0(ctx, request.RunRef, []string{ref})
	if err != nil {
		if issue, ok := err.(ErrorV0); ok &&
			issue.Code == ErrNucleoOrquestacionStoreV0 &&
			issue.Field == "required_test_evidence" {
			return RequiredTestEvidenceV0{}, false, nil
		}
		return RequiredTestEvidenceV0{}, false, err
	}
	if len(evidence) == 0 {
		return RequiredTestEvidenceV0{}, false, nil
	}
	return NormalizeRequiredTestEvidenceV0(evidence[0]), true, nil
}

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

func appendRequiredTestEvidenceResultV0(
	result RequiredTestExecutionResultV0,
	evidence RequiredTestEvidenceV0,
) RequiredTestExecutionResultV0 {
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, evidence.EvidenceRef))
	if evidence.Status == RequiredTestEvidenceStatusPassedV0 {
		result.PassedEvidenceRefs = compactStringsV0(append(result.PassedEvidenceRefs, evidence.EvidenceRef))
	}
	if evidence.Status == RequiredTestEvidenceStatusFailedV0 {
		result.FailedEvidenceRefs = compactStringsV0(append(result.FailedEvidenceRefs, evidence.EvidenceRef))
	}
	return result
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
