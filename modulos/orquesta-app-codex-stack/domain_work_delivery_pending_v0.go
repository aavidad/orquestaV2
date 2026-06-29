package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const domainWorkAcceptedJobPendingReceiptEvidenceRefV0 = "evidence-ref-domain-work-accepted-job-without-accepted-receipt"

func (stack StackV0) domainWorkRunHasPendingCompletionV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, []string, error) {
	if pending, err := stack.domainWorkRunHasPendingSubmissionWithoutAcceptedReceiptV0(ctx, run); err != nil || pending {
		return pending, nil, err
	}
	return stack.domainWorkRunHasAcceptedJobWithoutAcceptedReceiptV0(ctx, run)
}

func (stack StackV0) domainWorkRunHasPendingSubmissionWithoutAcceptedReceiptV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if !stack.domainWorkDeliveryBridgeReadyV0() {
		return false, nil
	}
	records, err := stack.domainWorkSubmissionRecordsForRunV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
			strings.TrimSpace(record.ReceiptRef) == "" {
			return true, nil
		}
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return false, err
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: run.RunID},
	)
	if err != nil {
		return false, err
	}
	for _, descriptor := range descriptors {
		ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
		if ackRef == "" || !stringInDrainSetV0(run.Deliveries, ackRef) {
			continue
		}
		task, ok := domainWorkTaskForDescriptorV0(tasks, descriptor)
		if !ok {
			continue
		}
		if !domainWorkSubmissionRecordsContainAcceptedReceiptV0(records, run.RunID, task.TaskID, ackRef) {
			return true, nil
		}
	}
	return false, nil
}

func (stack StackV0) domainWorkRunHasAcceptedJobWithoutAcceptedReceiptV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, []string, error) {
	runRef := strings.TrimSpace(run.RunID)
	if runRef == "" || stack.DomainDelivery.JobRecords == nil {
		return false, nil, nil
	}
	records, err := stack.DomainDelivery.JobRecords.ListDomainWorkJobRecordsV0(
		ctx,
		orquestadomainwork.DomainWorkJobRecordFilterV0{
			Status: orquestadomainwork.DomainWorkStatusAcceptedV0,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
				Kind: "run_ref",
				Ref:  runRef,
			}},
		},
	)
	if err != nil || len(records) == 0 {
		return false, nil, err
	}
	submissions, err := stack.domainWorkSubmissionRecordsForRunV0(ctx, runRef)
	if err != nil {
		return false, nil, err
	}
	for _, record := range records {
		jobRef := strings.TrimSpace(record.Job.JobRef)
		status := strings.TrimSpace(record.Job.Status)
		if status != orquestadomainwork.DomainWorkStatusAcceptedV0 || jobRef == "" {
			continue
		}
		if !domainWorkSubmissionRecordsContainAcceptedJobReceiptV0(submissions, runRef, jobRef) {
			return true, []string{domainWorkAcceptedJobPendingReceiptEvidenceRefV0}, nil
		}
	}
	return false, nil, nil
}

func (stack StackV0) domainWorkSubmissionRecordsForRunV0(
	ctx context.Context,
	runRef string,
) ([]DomainWorkArtifactSubmissionRecordV0, error) {
	reader, ok := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0)
	if !ok || reader == nil {
		return nil, nil
	}
	return reader.ListDomainWorkArtifactSubmissionsV0(ctx, DomainWorkArtifactSubmissionRecordFilterV0{
		RunRef: runRef,
	})
}

func domainWorkSubmissionRecordsContainAcceptedReceiptV0(
	records []DomainWorkArtifactSubmissionRecordV0,
	runRef string,
	taskRef string,
	deliveryRef string,
) bool {
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.RunRef == strings.TrimSpace(runRef) &&
			record.TaskRef == strings.TrimSpace(taskRef) &&
			record.DeliveryRef == strings.TrimSpace(deliveryRef) &&
			record.Status == DomainWorkArtifactSubmissionStatusAcceptedV0 &&
			strings.TrimSpace(record.ReceiptRef) != "" {
			return true
		}
	}
	return false
}

func domainWorkSubmissionRecordsContainAcceptedJobReceiptV0(
	records []DomainWorkArtifactSubmissionRecordV0,
	runRef string,
	jobRef string,
) bool {
	runRef = strings.TrimSpace(runRef)
	jobRef = strings.TrimSpace(jobRef)
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.RunRef == runRef &&
			record.JobRef == jobRef &&
			record.Status == DomainWorkArtifactSubmissionStatusAcceptedV0 &&
			strings.TrimSpace(record.ReceiptRef) != "" {
			return true
		}
	}
	return false
}
