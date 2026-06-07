package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

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
