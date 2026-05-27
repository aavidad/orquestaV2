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
	if stack.Ports.ProgressSource == nil {
		return run, false, nil
	}
	direct, applied, err := stack.reconcileStoppedSnapshotsV0(ctx, request, run)
	if err != nil || applied {
		return direct, applied, err
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
	return orquestaruntime.AgentProgressReportV0{
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
}

func shouldDirectlyReconcileStoppedAgentV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) bool {
	report := observation.Report
	return report.Status == orquestaruntime.AgentStoppedV0 &&
		(reportEvidenceContainsV0(report, "evidence-ref-no-ack") ||
			reportEvidenceContainsV0(report, "evidence-ref-auth-config-blocker"))
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
