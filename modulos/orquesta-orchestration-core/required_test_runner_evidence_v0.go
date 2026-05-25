package orquestacionnucleoapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

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
