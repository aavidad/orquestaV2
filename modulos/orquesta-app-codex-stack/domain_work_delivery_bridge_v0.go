package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func (stack StackV0) submitDomainWorkArtifactsAfterDrainObservationsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
) error {
	if !stack.domainWorkDeliveryBridgeReadyV0() || len(observations) == 0 {
		return nil
	}
	observations = drainObservationsForWaitAgentRefsV0(observations, request.WaitAgentRefs)
	if len(observations) == 0 {
		return nil
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return err
	}
	records, err := stack.Stores.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: run.RunID},
	)
	if err != nil {
		return err
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: run.RunID},
	)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if err := stack.submitDomainWorkArtifactForObservationV0(
			ctx,
			request,
			run,
			observation,
			tasks,
			records,
			descriptors,
		); err != nil {
			return err
		}
	}
	return nil
}

func (stack StackV0) submitPendingDomainWorkArtifactsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if !stack.domainWorkDeliveryBridgeReadyV0() || len(run.Deliveries) == 0 {
		return nil
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: run.RunID},
	)
	if err != nil {
		return err
	}
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0, len(run.Deliveries))
	for _, deliveryRef := range run.Deliveries {
		descriptor, ok := domainWorkDescriptorForDeliveryRefV0(descriptors, deliveryRef)
		if !ok {
			continue
		}
		if !domainWorkDescriptorMatchesWaitAgentRefsV0(descriptor, request.WaitAgentRefs) {
			continue
		}
		observations = append(observations, orquestacionnucleoapp.AgentDeliveryObservationV0{
			DeliveryRef: deliveryRef,
			ArtifactRef: deliveryRef,
			PhaseID:     string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:      descriptor.Spec.AgentPacket.Task.TaskRef,
			AgentRef:    descriptor.AgentRef,
			Summary:     "Entrega compacta validada por recibo de agente.",
			EvidenceRefs: []string{
				deliveryRef,
				descriptor.Spec.AgentPacket.DeliveryRefs.MailboxRef,
				descriptor.Spec.AgentPacket.DeliveryRefs.ReadinessRef,
			},
		})
	}
	return stack.submitDomainWorkArtifactsAfterDrainObservationsV0(ctx, request, run, observations)
}

func (stack StackV0) domainWorkDeliveryBridgeReadyV0() bool {
	return stack.DomainWork != nil &&
		stack.DomainDelivery.Enabled &&
		stack.DomainDelivery.Builder != nil &&
		stack.DomainDelivery.Ledger != nil &&
		stack.Stores.TaskStore != nil &&
		stack.Stores.AppChangeStore != nil &&
		stack.Stores.ReceiptStore != nil
}

func domainWorkDescriptorForDeliveryRefV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	deliveryRef string,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) == deliveryRef {
			return descriptor, true
		}
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
}

func domainWorkDescriptorMatchesWaitAgentRefsV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	waitAgentRefs []string,
) bool {
	waitAgentRefs = compactStringsV0(waitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return true
	}
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	if agentRef == "" {
		agentRef = strings.TrimSpace(descriptor.Spec.RequestID)
	}
	if agentRef == "" {
		agentRef = strings.TrimSpace(descriptor.Spec.AgentPacket.RequestID)
	}
	return codexStackStringInSetV0(waitAgentRefs, agentRef)
}

func (stack StackV0) submitDomainWorkArtifactForObservationV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	records []orquestaappchange.AppChangeRecordV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) error {
	if !drainObservationMatchesWaitAgentRefsV0(observation, request.WaitAgentRefs) {
		return nil
	}
	task, ok := domainWorkTaskForObservationV0(tasks, observation)
	if !ok {
		return nil
	}
	record, ok := domainWorkRecordForTaskV0(records, task.TaskID)
	if !ok {
		return nil
	}
	descriptor, ok := domainWorkDescriptorForObservationV0(descriptors, observation)
	if !ok {
		return nil
	}
	ack, ok := domainWorkSubmissionAckV0(descriptor, task, record)
	if !ok {
		return nil
	}
	input := DomainWorkArtifactSubmissionBuildInputV0{
		Run:         run,
		Task:        task,
		Record:      record,
		Descriptor:  descriptor,
		Ack:         ack,
		Observation: observation,
		OccurredAt:  request.OccurredAt,
	}
	submission, ok, err := stack.DomainDelivery.Builder.BuildDomainWorkArtifactSubmissionV0(ctx, input)
	if err != nil || !ok {
		return err
	}
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	if submitted, err := stack.DomainDelivery.Ledger.HasDomainWorkArtifactSubmissionV0(
		ctx,
		submission.IdempotencyKey,
	); err != nil || submitted {
		return err
	}
	result, err := stack.DomainWork.Execute(ctx, orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:          submission.RequestID,
		CorrelationID:      submission.CorrelationID,
		Action:             orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: submission,
	})
	if err != nil {
		return err
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 || result.Receipt == nil {
		if recordErr := stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
			ctx,
			DomainWorkArtifactSubmissionRecordV0{
				IdempotencyKey: submission.IdempotencyKey,
				Status:         DomainWorkArtifactSubmissionStatusRejectedV0,
				RunRef:         run.RunID,
				TaskRef:        task.TaskID,
				DeliveryRef:    observation.DeliveryRef,
				EvidenceRefs:   submission.EvidenceRefs,
				IssueRefs:      domainWorkSubmitIssueRefsV0(result),
				RecordedAt:     request.OccurredAt,
			},
		); recordErr != nil {
			return recordErr
		}
		return fmt.Errorf("domain_work_submit_artifact_failed")
	}
	return stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(
		ctx,
		DomainWorkArtifactSubmissionRecordV0{
			IdempotencyKey: submission.IdempotencyKey,
			Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
			RunRef:         run.RunID,
			TaskRef:        task.TaskID,
			DeliveryRef:    observation.DeliveryRef,
			ReceiptRef:     result.Receipt.ReceiptRef,
			EvidenceRefs: compactStringsV0(append(
				append([]string(nil), submission.EvidenceRefs...),
				result.Receipt.EvidenceRefs...,
			)),
			RecordedAt: request.OccurredAt,
		},
	)
}

func domainWorkSubmitIssueRefsV0(
	result orquestamcp.MCPDomainWorkToolResultV0,
) []string {
	refs := make([]string, 0, len(result.Errores)+1)
	refs = append(refs, "domain-work-submit-artifact-rejected")
	for _, issue := range result.Errores {
		refs = append(refs, issue.Code)
	}
	return compactStringsV0(refs)
}

func domainWorkTaskForObservationV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool) {
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) != strings.TrimSpace(observation.TaskID) ||
			!workflowTaskHasDomainWorkContractV0(task) {
			continue
		}
		return task, true
	}
	return orquestacoreworkflow.WorkflowTaskV0{}, false
}

func workflowTaskHasDomainWorkContractV0(task orquestacoreworkflow.WorkflowTaskV0) bool {
	for _, ref := range task.FunctionContractRefs {
		if strings.TrimSpace(ref.FunctionName) == "ApplyExternalDomainWorkV0" {
			return true
		}
	}
	return false
}

func domainWorkRecordForTaskV0(
	records []orquestaappchange.AppChangeRecordV0,
	taskRef string,
) (orquestaappchange.AppChangeRecordV0, bool) {
	for _, record := range records {
		if orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef) == strings.TrimSpace(taskRef) &&
			record.Request.ExternalWork != nil {
			return record, true
		}
	}
	return orquestaappchange.AppChangeRecordV0{}, false
}

func domainWorkDescriptorForObservationV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.AgentRef) == strings.TrimSpace(observation.AgentRef) &&
			strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) == strings.TrimSpace(observation.TaskID) &&
			strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) == strings.TrimSpace(observation.DeliveryRef) {
			return descriptor, true
		}
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
}
