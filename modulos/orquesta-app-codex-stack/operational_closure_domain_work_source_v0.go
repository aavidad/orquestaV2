package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureTaskIsAppChangeExternalWorkV0(
	ctx context.Context,
	runRef string,
	task orquestacoreworkflow.WorkflowTaskV0,
) (bool, error) {
	if !workflowTaskHasDomainWorkContractV0(task) || source.AppChangeStore == nil {
		return false, nil
	}
	_, ok, err := source.codexStackOperationalClosureAppChangeRecordForTaskV0(ctx, runRef, task.TaskID)
	return ok, err
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureDomainWorkEvidenceRefsForDeliveryV0(
	ctx context.Context,
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	deliveryRef string,
) ([]string, bool, bool, error) {
	if !workflowTaskHasDomainWorkContractV0(task) {
		return nil, false, false, nil
	}
	record, ok, err := source.codexStackOperationalClosureAppChangeRecordForTaskV0(ctx, request.Run.RunID, task.TaskID)
	if err != nil || !ok || record.Request.ExternalWork == nil {
		return nil, false, false, err
	}
	if source.DomainSubmissionLedger == nil {
		return nil, false, false, nil
	}
	records, err := source.DomainSubmissionLedger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef:      request.Run.RunID,
			TaskRef:     task.TaskID,
			DeliveryRef: deliveryRef,
			Status:      DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) == 0 {
		return nil, true, false, err
	}
	accepted := normalizeDomainWorkArtifactSubmissionRecordV0(records[0])
	if !codexStackOperationalClosureDomainWorkSubmissionIsCausalV0(accepted, record, deliveryRef) {
		return nil, true, false, nil
	}
	refs := append([]string(nil), accepted.EvidenceRefs...)
	refs = append(refs, accepted.ReceiptRef, "evidence-ref-domain-work-accepted")
	if codexStackOperationalClosureRecordIsOPESV0(record) {
		refs = append(refs, "evidence-ref-opes-domain-work-accepted")
	}
	return codexStackOperationalClosureCompactRefsV0(refs), true, true, nil
}

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureAppChangeRecordForTaskV0(
	ctx context.Context,
	runRef string,
	taskRef string,
) (orquestaappchange.AppChangeRecordV0, bool, error) {
	if source.AppChangeStore == nil {
		return orquestaappchange.AppChangeRecordV0{}, false, nil
	}
	records, err := source.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: runRef},
	)
	if err != nil {
		return orquestaappchange.AppChangeRecordV0{}, false, err
	}
	record, ok := domainWorkRecordForTaskV0(records, taskRef)
	return record, ok, nil
}

func codexStackOperationalClosureRecordIsOPESV0(
	record orquestaappchange.AppChangeRecordV0,
) bool {
	if record.Request.ExternalWork == nil {
		return false
	}
	return strings.TrimSpace(record.Request.ExternalWork.ProjectRef) == "opes" ||
		strings.TrimSpace(record.Request.AppRef) == "opes"
}

func codexStackOperationalClosureDomainWorkSubmissionIsCausalV0(
	submission DomainWorkArtifactSubmissionRecordV0,
	record orquestaappchange.AppChangeRecordV0,
	deliveryRef string,
) bool {
	work := record.Request.ExternalWork
	if work == nil {
		return false
	}
	if submission.Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
		strings.TrimSpace(submission.ReceiptRef) == "" ||
		!submission.CompleteJob {
		return false
	}
	if strings.TrimSpace(submission.DeliveryRef) != strings.TrimSpace(deliveryRef) ||
		strings.TrimSpace(submission.ArtifactRef) != strings.TrimSpace(deliveryRef) {
		return false
	}
	return strings.TrimSpace(submission.JobRef) == strings.TrimSpace(work.JobRef) &&
		strings.TrimSpace(submission.ArtifactType) == domainWorkArtifactTypeForWorkKindV0(work.WorkKind)
}
