package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func (stack StackV0) recoverableDomainWorkDeliveryObservationsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	if !stack.domainWorkDeliveryBridgeReadyV0() {
		return nil, nil
	}
	tasks, records, descriptors, err := stack.loadDomainWorkDeliveryContextV0(ctx, run)
	if err != nil {
		return nil, err
	}
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0)
	for _, descriptor := range descriptors {
		if !domainWorkDescriptorMatchesWaitAgentRefsV0(descriptor, request.WaitAgentRefs) {
			continue
		}
		if !domainWorkRecoveryCoreDeliveryEligibleV0(run, descriptor) {
			continue
		}
		observation, ok, err := stack.recoverableDomainWorkObservationForDescriptorV0(
			ctx,
			request,
			run,
			tasks,
			records,
			descriptor,
		)
		if err != nil {
			return nil, err
		}
		if ok && drainObservationMatchesWaitAgentRefsV0(observation, request.WaitAgentRefs) {
			observations = append(observations, observation)
		}
	}
	return observations, nil
}

func (stack StackV0) submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if !stack.domainWorkDeliveryBridgeReadyV0() {
		return nil
	}
	tasks, records, descriptors, err := stack.loadDomainWorkDeliveryContextV0(ctx, run)
	if err != nil {
		return err
	}
	for _, descriptor := range descriptors {
		if !domainWorkDescriptorMatchesWaitAgentRefsV0(descriptor, request.WaitAgentRefs) {
			continue
		}
		if !domainWorkRecoveryDirectSubmitEligibleV0(run, descriptor) {
			continue
		}
		observation, ok, err := stack.recoverableDomainWorkObservationForDescriptorV0(
			ctx,
			request,
			run,
			tasks,
			records,
			descriptor,
		)
		if err != nil || !ok {
			return err
		}
		if !drainObservationMatchesWaitAgentRefsV0(observation, request.WaitAgentRefs) {
			continue
		}
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

func (stack StackV0) loadDomainWorkDeliveryContextV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (
	[]orquestacoreworkflow.WorkflowTaskV0,
	[]orquestaappchange.AppChangeRecordV0,
	[]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	error,
) {
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return nil, nil, nil, err
	}
	records, err := stack.Stores.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: run.RunID},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: run.RunID},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return tasks, records, descriptors, nil
}

func (stack StackV0) recoverableDomainWorkObservationForDescriptorV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	records []orquestaappchange.AppChangeRecordV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentDeliveryObservationV0, bool, error) {
	if !domainWorkDescriptorMatchesWaitAgentRefsV0(descriptor, request.WaitAgentRefs) {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	if ackRef == "" || stringInDrainSetV0(run.Deliveries, ackRef) {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	if _, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	); len(issues) == 0 {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	task, ok := domainWorkTaskForDescriptorV0(tasks, descriptor)
	if !ok {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	record, ok := domainWorkRecordForTaskV0(records, task.TaskID)
	if !ok {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	ack, ok := recoverDomainWorkAckV0(descriptor, task, record)
	if !ok {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	codexObservation, issues := orquestaruntimecodex.BuildCodexDeliveryObservationV0(
		ack,
		descriptor.Spec,
	)
	if len(issues) > 0 {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	observation := orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + codexObservation.DeliveryRef,
		ArtifactRef:  codexObservation.DeliveryRef,
		DeliveryRef:  codexObservation.DeliveryRef,
		PhaseID:      codexObservation.PhaseID,
		TaskID:       codexObservation.TaskID,
		AgentRef:     codexObservation.AgentRef,
		Summary:      codexObservation.Summary,
		EvidenceRefs: compactCodexStackStringsV0(
			append(codexObservation.EvidenceRefs, "delivery-recovery-ref-valid-artifact"),
		),
	}
	if ok, err := stack.domainWorkRecoveryBuildsValidSubmissionV0(
		ctx,
		request,
		run,
		task,
		record,
		descriptor,
		ack,
		observation,
	); err != nil || !ok {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, err
	}
	return observation, true, nil
}

func (stack StackV0) domainWorkRecoveryBuildsValidSubmissionV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	record orquestaappchange.AppChangeRecordV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) (bool, error) {
	submission, ok, err := stack.DomainDelivery.Builder.BuildDomainWorkArtifactSubmissionV0(
		ctx,
		DomainWorkArtifactSubmissionBuildInputV0{
			Run:         run,
			Task:        task,
			Record:      record,
			Descriptor:  descriptor,
			Ack:         ack,
			Observation: observation,
			OccurredAt:  request.OccurredAt,
		},
	)
	if err != nil || !ok {
		return false, err
	}
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	submitted, err := stack.domainWorkSubmissionAlreadyRecordedV0(ctx, run, task, observation, submission)
	if err != nil || submitted {
		return false, err
	}
	return true, nil
}
