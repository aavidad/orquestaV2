package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func drainRunWithStartedAgentsKnownV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	run.Agents = compactStringsV0(append(append([]string{}, run.Agents...), run.StartedAgents...))
	return run
}

func (stack StackV0) reconcileStoppedPendingAgentsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	direct, applied, err := stack.reconcileStoppedSnapshotsV0(ctx, request, run)
	if err != nil || applied {
		return direct, applied, err
	}
	direct, applied, err = stack.reconcileStoppedProcessRegistrySnapshotsV0(ctx, request, run)
	if err != nil || applied {
		return direct, applied, err
	}
	direct, applied, err = stack.reconcileMissingProcessRegistryPendingAgentsV0(ctx, request, run)
	if err != nil || applied {
		return direct, applied, err
	}
	if stack.Ports.ProgressSource == nil {
		return run, false, nil
	}
	source := drainProgressSourceForWaitAgentRefsV0(stack.Ports.ProgressSource, request.WaitAgentRefs)
	observations, err := source.BuildAgentProgressObservationsV0(
		ctx,
		orquestacionnucleoapp.AgentProgressObservationRequestV0{
			Run:           run,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			EvidenceRefs:  []string{"evidence-ref-live-agent-reconciliation"},
		},
	)
	if err != nil {
		return run, false, err
	}
	applied = false
	for _, observation := range observations {
		if !shouldDirectlyReconcileStoppedAgentV0(observation) {
			continue
		}
		if err := stack.applyStoppedAgentReconciliationV0(ctx, request, run, observation); err != nil {
			return run, applied, err
		}
		applied = true
	}
	if !applied {
		return run, false, nil
	}
	next, err := stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
	return next, true, err
}

func (stack StackV0) reconcileMissingProcessRegistryPendingAgentsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if stack.Stores.ProcessRegistry == nil {
		return run, false, nil
	}
	lister, ok := stack.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok {
		return run, false, nil
	}
	pending := pendingAgentRefsForProcessRegistryReconciliationV0(run, request.WaitAgentRefs)
	if len(pending) == 0 {
		return run, false, nil
	}
	records, err := lister.ListAgentProcessesV0(
		ctx,
		orquestacionnucleoapp.AgentProcessRegistryListFilterV0{
			RunID: strings.TrimSpace(run.RunID),
		},
	)
	if err != nil {
		return run, false, err
	}
	registered := processRegistryAgentRefSetV0(records)
	applied := false
	for _, agentRef := range pending {
		if _, ok := registered[agentRef]; ok {
			continue
		}
		observation := missingProcessRegistryProgressObservationV0(run, agentRef)
		if err := stack.applyStoppedAgentReconciliationV0(ctx, request, run, observation); err != nil {
			return run, applied, err
		}
		applied = true
	}
	if !applied {
		return run, false, nil
	}
	next, err := stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
	return next, true, err
}

func (stack StackV0) reconcileStoppedProcessRegistrySnapshotsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if stack.Stores.ProcessRegistry == nil || stack.CodexSnapshotSource == nil {
		return run, false, nil
	}
	pending := pendingAgentRefsForProcessRegistryReconciliationV0(run, request.WaitAgentRefs)
	if len(pending) == 0 {
		return run, false, nil
	}
	agentsWithDescriptor, err := stack.processRegistrySnapshotDescriptorAgentsV0(ctx, request, run)
	if err != nil {
		return run, false, err
	}
	applied := false
	for _, agentRef := range pending {
		if _, ok := agentsWithDescriptor[strings.TrimSpace(agentRef)]; ok {
			continue
		}
		record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(
			ctx,
			strings.TrimSpace(run.RunID),
			strings.TrimSpace(agentRef),
		)
		if err != nil {
			continue
		}
		observation, ready, err := stack.stoppedProcessRegistrySnapshotObservationV0(run, record)
		if err != nil {
			return run, applied, err
		}
		if !ready {
			continue
		}
		if err := stack.applyStoppedAgentReconciliationV0(ctx, request, run, observation); err != nil {
			return run, applied, err
		}
		applied = true
	}
	if !applied {
		return run, false, nil
	}
	next, err := stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
	return next, true, err
}

func (stack StackV0) processRegistrySnapshotDescriptorAgentsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (map[string]struct{}, error) {
	result := map[string]struct{}{}
	if stack.Stores.ReceiptStore == nil {
		return result, nil
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         run.RunID,
			StartedAgents: compactStringsV0(run.StartedAgents),
			CorrelationID: strings.TrimSpace(request.CorrelationID),
			EvidenceRefs:  []string{"evidence-ref-live-agent-reconciliation-direct"},
		},
	)
	if err != nil {
		return nil, err
	}
	for _, descriptor := range descriptors {
		agentRef := strings.TrimSpace(descriptor.AgentRef)
		if agentRef == "" {
			continue
		}
		result[agentRef] = struct{}{}
	}
	return result, nil
}

func (stack StackV0) stoppedProcessRegistrySnapshotObservationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) (orquestacionnucleoapp.AgentProgressObservationV0, bool, error) {
	snapshot, err := stack.CodexSnapshotSource.SnapshotV0(strings.TrimSpace(record.ProcessRef))
	if err != nil {
		if codexStackProcessRuntimeMissingV0(err) {
			report := processRegistrySnapshotProgressReportV0(run, record, "missing")
			return processRegistrySnapshotProgressObservationV0(run, report), true, nil
		}
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeStoppedV0 {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	report := processRegistrySnapshotProgressReportV0(run, record, "stopped")
	return processRegistrySnapshotProgressObservationV0(run, report), true, nil
}

func processRegistrySnapshotProgressObservationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + report.ReportID,
		Report:           report,
		PhaseID:          firstNonEmptyQueuedSourceV0(strings.TrimSpace(string(run.CurrentPhase)), "phase-ref-live-agent-reconciliation"),
		DecisionRequired: true,
		EvidenceRefs:     append([]string(nil), report.EvidenceRefs...),
	}
}

func processRegistrySnapshotProgressReportV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	runtimeState string,
) orquestaruntime.AgentProgressReportV0 {
	agentRef := strings.TrimSpace(record.AgentRequestID)
	processRef := strings.TrimSpace(record.ProcessRef)
	safe := codexStackOperationalClosureSafeRefV0(
		strings.TrimSpace(run.RunID) + "-" + agentRef + "-" + processRef + "-" + strings.TrimSpace(runtimeState),
	)
	digest := codexStackDeterministicDigestV0(safe)
	if len(digest) > 32 {
		digest = digest[:32]
	}
	evidenceRefs := []string{
		"evidence-ref-no-ack",
		"evidence-ref-live-agent-reconciliation-direct",
		"evidence-ref-agent-process-registry",
	}
	summary := "Proceso runtime registrado parado sin ACK durable; se reconcilia como agente perdido para no bloquear el supervisor."
	reportPrefix := "agent-progress-report-ref-live-registry-stopped-"
	if strings.TrimSpace(runtimeState) == "missing" {
		evidenceRefs = append(evidenceRefs, "evidence-ref-process-runtime-missing")
		summary = "Proceso runtime registrado sin snapshot verificable tras reinicio; se reconcilia como agente perdido para no bloquear el supervisor."
		reportPrefix = "agent-progress-report-ref-live-registry-missing-"
	} else {
		evidenceRefs = append(evidenceRefs, "evidence-ref-process-runtime-stopped")
	}
	return orquestaruntime.AgentProgressReportV0{
		ReportID:         reportPrefix + digest,
		RunID:            strings.TrimSpace(run.RunID),
		AgentRequestID:   agentRef,
		Status:           orquestaruntime.AgentStoppedV0,
		DecisionRequired: true,
		Summary:          summary,
		EvidenceRefs:     compactStringsV0(evidenceRefs),
	}
}

func pendingAgentRefsForProcessRegistryReconciliationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	waitAgentRefs []string,
) []string {
	started := compactStringsV0(run.StartedAgents)
	if len(started) == 0 {
		return nil
	}
	scoped := compactStringsV0(waitAgentRefs)
	pending := make([]string, 0, len(started))
	for _, agentRef := range started {
		if len(scoped) > 0 && !drainAgentRefMatchesWaitAgentRefsV0(agentRef, scoped) {
			continue
		}
		if !drainRunHasPendingExternalAgentRefsV0(run, []string{agentRef}) {
			continue
		}
		pending = append(pending, agentRef)
	}
	return compactStringsV0(pending)
}

func processRegistryAgentRefSetV0(
	records []orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) map[string]struct{} {
	result := make(map[string]struct{}, len(records))
	for _, record := range records {
		agentRef := strings.TrimSpace(record.AgentRequestID)
		if agentRef == "" {
			continue
		}
		result[agentRef] = struct{}{}
	}
	return result
}

func missingProcessRegistryProgressObservationV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	report := missingProcessRegistryProgressReportV0(run, agentRef)
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + report.ReportID,
		Report:           report,
		PhaseID:          firstNonEmptyQueuedSourceV0(strings.TrimSpace(string(run.CurrentPhase)), "phase-ref-live-agent-reconciliation"),
		DecisionRequired: true,
		EvidenceRefs:     append([]string(nil), report.EvidenceRefs...),
	}
}

func missingProcessRegistryProgressReportV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) orquestaruntime.AgentProgressReportV0 {
	safe := codexStackOperationalClosureSafeRefV0(
		strings.TrimSpace(run.RunID) + "-" + strings.TrimSpace(agentRef),
	)
	digest := codexStackDeterministicDigestV0(safe)
	if len(digest) > 32 {
		digest = digest[:32]
	}
	return orquestaruntime.AgentProgressReportV0{
		ReportID:         "agent-progress-report-ref-missing-process-registry-" + digest,
		RunID:            strings.TrimSpace(run.RunID),
		AgentRequestID:   strings.TrimSpace(agentRef),
		Status:           orquestaruntime.AgentStoppedV0,
		DecisionRequired: true,
		Summary:          "Agente pendiente sin registro de proceso verificable; se reconcilia como perdido para no bloquear el supervisor.",
		EvidenceRefs: compactStringsV0([]string{
			"evidence-ref-no-ack",
			"evidence-ref-live-agent-reconciliation-direct",
			"evidence-ref-agent-process-registry-missing",
		}),
	}
}

func (stack StackV0) reconcileStoppedSnapshotsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if stack.Stores.ReceiptStore == nil || stack.Stores.ProcessRegistry == nil || stack.CodexSnapshotSource == nil {
		return run, false, nil
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:          run.RunID,
			StartedAgents:  compactStringsV0(run.StartedAgents),
			Deliveries:     compactStringsV0(run.Deliveries),
			PhaseArtifacts: compactStringsV0(run.PhaseArtifacts),
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			EvidenceRefs:   []string{"evidence-ref-live-agent-reconciliation-direct"},
		},
	)
	if err != nil {
		return run, false, err
	}
	applied := false
	for _, descriptor := range descriptors {
		if !stoppedSnapshotDescriptorInScopeV0(run, request.WaitAgentRefs, descriptor.AgentRef) {
			continue
		}
		observation, ready, err := stack.stoppedSnapshotObservationV0(ctx, request, run, descriptor)
		if err != nil {
			return run, applied, err
		}
		if !ready {
			continue
		}
		if err := stack.applyStoppedAgentReconciliationV0(ctx, request, run, observation); err != nil {
			return run, applied, err
		}
		applied = true
	}
	if !applied {
		return run, false, nil
	}
	next, err := stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
	return next, true, err
}

func stoppedSnapshotDescriptorInScopeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	waitAgentRefs []string,
	agentRef string,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	if agentRef == "" {
		return false
	}
	if len(compactStringsV0(waitAgentRefs)) > 0 && !drainAgentRefMatchesWaitAgentRefsV0(agentRef, waitAgentRefs) {
		return false
	}
	return drainRunHasPendingExternalAgentRefsV0(run, []string{agentRef})
}

func (stack StackV0) stoppedSnapshotObservationV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentProgressObservationV0, bool, error) {
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(
		ctx,
		strings.TrimSpace(run.RunID),
		strings.TrimSpace(descriptor.AgentRef),
	)
	if err != nil {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	snapshot, err := stack.CodexSnapshotSource.SnapshotV0(record.ProcessRef)
	if err != nil {
		if codexStackProcessRuntimeMissingV0(err) {
			report := missingSnapshotProgressReportV0(run, descriptor)
			return orquestacionnucleoapp.AgentProgressObservationV0{
				CandidateRef:     "progress-candidate-ref-" + report.ReportID,
				Report:           report,
				PhaseID:          firstNonEmptyQueuedSourceV0(string(run.CurrentPhase), descriptor.Spec.AgentPacket.Phase),
				TaskRef:          strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef),
				DecisionRequired: true,
				EvidenceRefs:     append([]string(nil), report.EvidenceRefs...),
			}, true, nil
		}
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeStoppedV0 {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	report := stoppedSnapshotProgressReportV0(run, descriptor)
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + report.ReportID,
		Report:           report,
		PhaseID:          firstNonEmptyQueuedSourceV0(string(run.CurrentPhase), descriptor.Spec.AgentPacket.Phase),
		TaskRef:          strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef),
		DecisionRequired: true,
		EvidenceRefs:     append([]string(nil), report.EvidenceRefs...),
	}, true, nil
}

func codexStackProcessRuntimeMissingV0(err error) bool {
	var runtimeErr orquestaruntime.ProcessRuntimeErrorV0
	return errors.As(err, &runtimeErr) &&
		runtimeErr.Code == orquestaruntime.ProcessRuntimeNoEncontradoV0
}

func missingSnapshotProgressReportV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaruntime.AgentProgressReportV0 {
	report := stoppedSnapshotProgressReportV0(run, descriptor)
	report.Summary = "Proceso sin snapshot runtime tras reinicio; se reconcilia como agente perdido sin bloquear el supervisor."
	report.EvidenceRefs = compactStringsV0(append(report.EvidenceRefs,
		"evidence-ref-process-runtime-missing",
	))
	return report
}

func stoppedSnapshotProgressReportV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaruntime.AgentProgressReportV0 {
	safe := codexStackOperationalClosureSafeRefV0(
		strings.TrimSpace(run.RunID) + "-" + strings.TrimSpace(descriptor.AgentRef),
	)
	digest := codexStackDeterministicDigestV0(safe)
	if len(digest) > 32 {
		digest = digest[:32]
	}
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:         "agent-progress-report-ref-live-reconciliation-" + digest,
		RunID:            strings.TrimSpace(run.RunID),
		AgentRequestID:   strings.TrimSpace(descriptor.AgentRef),
		Status:           orquestaruntime.AgentStoppedV0,
		DecisionRequired: true,
		Summary:          "Proceso parado sin ACK durable observado durante reconciliacion.",
		EvidenceRefs: compactStringsV0([]string{
			"evidence-ref-no-ack",
			"evidence-ref-live-agent-reconciliation-direct",
		}),
	}
	return orquestaruntimecodexdelivery.CodexProgressReportWithProcessFailureContextV0(
		descriptor,
		report,
	)
}

func shouldDirectlyReconcileStoppedAgentV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) bool {
	report := observation.Report
	return report.Status == orquestaruntime.AgentStoppedV0 &&
		(reportEvidenceContainsV0(report, "evidence-ref-no-ack") ||
			reportEvidenceContainsV0(report, "evidence-ref-auth-config-blocker") ||
			report.BudgetStatus == orquestaruntime.AgentProgressBudgetCapacityLimitedV0)
}

func (stack StackV0) applyStoppedAgentReconciliationV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) error {
	input := stoppedAgentSupervisionInputV0(request, run, observation)
	result, err := orquestadirector.BuildAgentProgressSupervisionV0(input)
	if err != nil {
		return err
	}
	commands := []orquestacoreworkflow.OrchestrationCommandV0{result.AssessCommand}
	if result.AskDirectorCommand != nil {
		commands = append(commands, *result.AskDirectorCommand)
	}
	if result.RegisterLostCommand != nil {
		commands = append(commands, *result.RegisterLostCommand)
	}
	for _, command := range commands {
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
			ctx,
			stack.Stores.RunStore,
			stack.Stores.EventSink,
			command,
		); err != nil {
			return err
		}
	}
	return nil
}

func stoppedAgentSupervisionInputV0(
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	report := observation.Report
	reportID := strings.TrimSpace(report.ReportID)
	return orquestadirector.AgentProgressSupervisionInputV0{
		CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-live-agent-reconciliation-" + codexStackOperationalClosureSafeRefV0(reportID),
			RunID:          strings.TrimSpace(run.RunID),
			IdempotencyKey: "idem-live-agent-reconciliation-" + codexStackOperationalClosureSafeRefV0(reportID),
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			RequestedBy:    "orquesta-app-codex-stack-live-agent-reconciliation",
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(request.OccurredAt), "1970-01-01T00:00:00Z"),
		},
		Report:        report,
		PhaseID:       stoppedAgentObservationPhaseV0(run, observation),
		TaskRef:       strings.TrimSpace(observation.TaskRef),
		DeliveryRef:   strings.TrimSpace(observation.DeliveryRef),
		AssessmentRef: firstNonEmptyQueuedSourceV0(strings.TrimSpace(observation.AssessmentRef), "assessment-ref-"+reportID),
		QuestionID:    firstNonEmptyQueuedSourceV0(strings.TrimSpace(observation.QuestionID), "question-ref-"+reportID),
	}
}

func stoppedAgentObservationPhaseV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) string {
	if phase := strings.TrimSpace(string(run.CurrentPhase)); phase != "" {
		return phase
	}
	return strings.TrimSpace(observation.PhaseID)
}

func reportEvidenceContainsV0(report orquestaruntime.AgentProgressReportV0, want string) bool {
	for _, ref := range report.EvidenceRefs {
		if strings.TrimSpace(ref) == want {
			return true
		}
	}
	return false
}
