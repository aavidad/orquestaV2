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

func (source codexStackOperationalClosureSourceV0) codexStackOperationalClosureOPESSubrolesMaterializedV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (bool, error) {
	record, ok, err := source.codexStackOperationalClosureAppChangeRecordForTaskV0(ctx, run.RunID, task.TaskID)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	required := codexStackOperationalClosureRequiredOPESSubrolesV0(record)
	if required <= 0 {
		return true, nil
	}
	childRefs := codexStackOperationalClosureChildTaskRefsV0(run, task)
	if len(childRefs) < required {
		return false, nil
	}
	knownTasks := map[string]bool{}
	for _, item := range tasks {
		knownTasks[strings.TrimSpace(item.TaskID)] = true
	}
	for _, childRef := range childRefs {
		if !knownTasks[childRef] || !codexStackOperationalClosureContainsV0(run.Tasks, childRef) {
			return false, nil
		}
	}
	return true, nil
}

func codexStackOperationalClosureRequiredOPESSubrolesV0(
	record orquestaappchange.AppChangeRecordV0,
) int {
	work := record.Request.ExternalWork
	if work == nil || !externalJobIntegrationExternalWorkLooksOPESV0(record.Request.AppRef, work) {
		return 0
	}
	required := externalJobIntegrationSubrolesRequiredCountV0(work.InputFields)
	for _, ref := range work.InterfaceRefs {
		if strings.TrimSpace(ref) == "opes.padre-tema-6-subroles.v1" && required < 6 {
			required = 6
		}
	}
	return required
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
	if codexStackOperationalClosureRecordIsOPESV0(record) &&
		!codexStackOperationalClosureOPESFinalPackageReadyV0(record, accepted) {
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
	return codexStackOperationalClosureRefIsOPESV0(record.Request.ExternalWork.ProjectRef) ||
		codexStackOperationalClosureRefIsOPESV0(record.Request.AppRef)
}

func codexStackOperationalClosureRefIsOPESV0(ref string) bool {
	ref = strings.TrimSpace(ref)
	return ref == "opes" || strings.HasPrefix(ref, "opes-")
}

func codexStackOperationalClosureOPESFinalPackageReadyV0(
	record orquestaappchange.AppChangeRecordV0,
	submission DomainWorkArtifactSubmissionRecordV0,
) bool {
	if !codexStackOperationalClosureOPESFinalPackageV0(record, submission) {
		return true
	}
	tokens := codexStackOperationalClosureDomainWorkEvidenceTokensV0(submission)
	return codexStackOperationalClosureEvidenceContainsAnyV0(tokens,
		"opes-editorial-minimums-passed",
		"opes-extension-minima-passed",
		"opes-extension-minima-nivel-passed",
		"informe_extension_temario",
		"extension_report_ref",
		"minimum_extension_status-passed",
		"minimos-extension-passed",
	) && codexStackOperationalClosureEvidenceContainsAnyV0(tokens,
		"opes-common-canonical-reuse-passed",
		"opes-common-master-not-applicable",
		"opes-derivacion-comunes-maestro-passed",
		"matriz_reutilizacion_comunes",
		"common_reuse_matrix_ref",
		"master_derivation_matrix_ref",
		"common_topics_not_applicable",
	)
}

func codexStackOperationalClosureOPESFinalPackageV0(
	record orquestaappchange.AppChangeRecordV0,
	submission DomainWorkArtifactSubmissionRecordV0,
) bool {
	work := record.Request.ExternalWork
	workKind := ""
	if work != nil {
		workKind = strings.TrimSpace(work.WorkKind)
	}
	switch workKind {
	case "finalize_topic_package",
		"finalize_temario_package",
		"close_temario_package",
		"finalize_syllabus_package",
		"close_syllabus_package":
		return true
	}
	switch strings.TrimSpace(submission.ArtifactType) {
	case "completed_syllabus_package":
		return true
	default:
		return false
	}
}

func codexStackOperationalClosureDomainWorkEvidenceTokensV0(
	submission DomainWorkArtifactSubmissionRecordV0,
) []string {
	tokens := append([]string(nil), submission.EvidenceRefs...)
	tokens = append(tokens, submission.PayloadRefs...)
	for _, field := range submission.PayloadFields {
		tokens = append(tokens, field.Name, field.Value)
		tokens = append(tokens, field.Values...)
		if len(field.ValueJSON) > 0 {
			tokens = append(tokens, string(field.ValueJSON))
		}
	}
	for _, ref := range submission.ExternalRefs {
		tokens = append(tokens, ref.Kind, ref.Ref)
	}
	return codexStackOperationalClosureCompactRefsV0(tokens)
}

func codexStackOperationalClosureEvidenceContainsAnyV0(
	values []string,
	needles ...string,
) bool {
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(value, strings.ToLower(strings.TrimSpace(needle))) {
				return true
			}
		}
	}
	return false
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
	expectedArtifactType := domainWorkExpectedArtifactTypeV0(work.InputFields, work.WorkKind)
	return strings.TrimSpace(submission.JobRef) == strings.TrimSpace(work.JobRef) &&
		strings.TrimSpace(submission.ArtifactType) == strings.TrimSpace(expectedArtifactType)
}
