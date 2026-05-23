package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureLoadRequiredTestEvidenceForResultV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	result orquestacoreworkflow.ReviewResultV0,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	candidateRefs := append([]string(nil), request.RequiredTestEvidenceRefs...)
	candidateRefs = append(candidateRefs, result.EvidenceRefs...)
	evidence := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(candidateRefs))
	for _, evidenceRef := range codexStackOperationalClosureCompactRefsV0(candidateRefs) {
		items, err := source.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, request.Run.RunID, []string{evidenceRef})
		if err != nil {
			if codexStackOperationalClosureMissingTestEvidenceV0(err) {
				continue
			}
			return nil, err
		}
		evidence = append(evidence, items...)
	}
	return evidence, nil
}

func codexStackOperationalClosureMissingTestEvidenceV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "required_test_evidence"
}

func codexStackOperationalClosurePassedTestEvidenceRefsV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) []string {
	refs := make([]string, 0, len(task.RequiredTests))
	for _, required := range codexStackOperationalClosureCompactRefsV0(task.RequiredTests) {
		ref, ok := codexStackOperationalClosurePassedTestEvidenceRefV0(task, required, evidence, accepted, result)
		if !ok {
			return nil
		}
		refs = append(refs, ref)
	}
	return codexStackOperationalClosureCompactRefsV0(refs)
}

func codexStackOperationalClosurePassedTestEvidenceRefV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	required string,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
	result orquestacoreworkflow.ReviewResultV0,
) (string, bool) {
	for _, item := range evidence {
		if item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
			strings.TrimSpace(item.TaskRef) == strings.TrimSpace(task.TaskID) &&
			strings.TrimSpace(item.TestCommand) == strings.TrimSpace(required) &&
			strings.TrimSpace(item.DeliveryRef) == strings.TrimSpace(result.DeliveryRef) &&
			strings.TrimSpace(item.ReviewRequestID) == strings.TrimSpace(result.ReviewRequestID) &&
			strings.TrimSpace(item.ReviewResultRef) == strings.TrimSpace(result.ReviewResultRef) &&
			strings.TrimSpace(item.AcceptedReviewRef) == strings.TrimSpace(accepted.AcceptedReviewRef) {
			return item.EvidenceRef, true
		}
	}
	return "", false
}
