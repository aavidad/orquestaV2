package orquestaappdirectorservice

import (
	"context"
	"errors"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorPlanRequiredTestEvidenceForMatchesV0(
	ctx context.Context,
	runRef string,
	store orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	candidateRefs := append([]string(nil), activeStep.RequiredTestEvidenceRefs...)
	if len(candidateRefs) == 0 {
		for _, match := range matches {
			candidateRefs = append(candidateRefs, match.EvidenceRefs...)
		}
	}
	evidence := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(candidateRefs))
	for _, evidenceRef := range compactServiceRefsV0(candidateRefs) {
		items, err := store.LoadRequiredTestEvidenceV0(ctx, runRef, []string{evidenceRef})
		if err != nil {
			if operationalDirectorPlanMissingRequiredTestEvidenceV0(err) {
				continue
			}
			return nil, err
		}
		evidence = append(evidence, items...)
	}
	return evidence, nil
}

func operationalDirectorPlanMissingRequiredTestEvidenceV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "required_test_evidence"
}

func operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
	runRef string,
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
	requiredTestsByTask map[string][]string,
) operationalDirectorPlanRequiredTestEvidenceEvaluationV0 {
	passedRefs := []string(nil)
	failedRefs := []string(nil)
	expected := 0
	passedCount := 0
	for _, match := range matches {
		for _, required := range operationalDirectorPlanRequiredTestsForMatchV0(requiredTestsByTask, requiredTests, match) {
			expected++
			passedRef, failedRef := operationalDirectorPlanRequiredTestEvidenceRefV0(runRef, required, match, evidence)
			if passedRef != "" {
				passedRefs = append(passedRefs, passedRef)
				passedCount++
				continue
			}
			if failedRef != "" {
				failedRefs = append(failedRefs, failedRef)
			}
		}
	}
	passedRefs = compactServiceRefsV0(passedRefs)
	return operationalDirectorPlanRequiredTestEvidenceEvaluationV0{
		PassedRefs: passedRefs,
		FailedRefs: compactServiceRefsV0(failedRefs),
		Complete:   expected > 0 && passedCount == expected,
	}
}

func operationalDirectorPlanRequiredTestEvidenceRefV0(
	runRef string,
	required string,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) (string, string) {
	passedRef := ""
	failedRef := ""
	for _, item := range evidence {
		if strings.TrimSpace(item.RunRef) != strings.TrimSpace(runRef) ||
			strings.TrimSpace(item.TaskRef) != strings.TrimSpace(match.TaskRef) ||
			strings.TrimSpace(item.TestCommand) != strings.TrimSpace(required) ||
			strings.TrimSpace(item.DeliveryRef) != strings.TrimSpace(match.DeliveryRef) ||
			strings.TrimSpace(item.ReviewRequestID) != strings.TrimSpace(match.ReviewRequestID) ||
			strings.TrimSpace(item.ReviewResultRef) != strings.TrimSpace(match.ReviewResultRef) ||
			strings.TrimSpace(item.AcceptedReviewRef) != strings.TrimSpace(match.AcceptedReviewRef) {
			continue
		}
		switch item.Status {
		case orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0:
			passedRef = strings.TrimSpace(item.EvidenceRef)
		case orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0:
			failedRef = strings.TrimSpace(item.EvidenceRef)
		}
	}
	if passedRef == "" && failedRef == "" && operationalDirectorPlanRunScopedRequiredTestV0(required) {
		return operationalDirectorPlanRunScopedRequiredTestEvidenceRefV0(runRef, required, evidence)
	}
	return passedRef, failedRef
}

func operationalDirectorPlanRunScopedRequiredTestEvidenceRefV0(
	runRef string,
	required string,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) (string, string) {
	passedRef := ""
	failedRef := ""
	for _, item := range evidence {
		if strings.TrimSpace(item.RunRef) != strings.TrimSpace(runRef) ||
			strings.TrimSpace(item.TestCommand) != strings.TrimSpace(required) {
			continue
		}
		switch item.Status {
		case orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0:
			passedRef = strings.TrimSpace(item.EvidenceRef)
		case orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0:
			failedRef = strings.TrimSpace(item.EvidenceRef)
		}
	}
	return passedRef, failedRef
}

func operationalDirectorPlanRunScopedRequiredTestV0(required string) bool {
	required = strings.ToLower(strings.TrimSpace(required))
	return strings.Contains(required, "smoke") ||
		strings.Contains(required, "arquitectura") ||
		strings.Contains(required, "architecture") ||
		strings.Contains(required, "si se toca") ||
		strings.Contains(required, "if touched") ||
		strings.Contains(required, "global")
}
