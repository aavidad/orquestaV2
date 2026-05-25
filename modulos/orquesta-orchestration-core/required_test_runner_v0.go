package orquestacionnucleoapp

import (
	"context"
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
