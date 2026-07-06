package orquestaappcodexstack

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func (stack StackV0) reconcileQueuedRunningStaleRunsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) error {
	if stack.Stores.RunQueue == nil || stack.Stores.RunControl == nil || stack.Ports.RunStore == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             strings.TrimSpace(command.QueueRef),
			AppRefs:              append([]string(nil), command.AppRefs...),
			Limit:                queuedStoppedRecoveryReadLimitV0(command),
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.TrimSpace(candidate.Status) != orquestarunqueue.RunStatusRunningV0 {
			continue
		}
		if err := stack.reconcileQueuedRunningStaleCandidateV0(ctx, command, candidate); err != nil {
			if codexSupervisorRunEventsBudgetExceededV0(err) {
				if markErr := stack.markQueuedCandidateRunEventsOversizedV0(ctx, command, candidate); markErr != nil {
					return markErr
				}
				continue
			}
			return err
		}
	}
	return nil
}

func (stack StackV0) reconcileQueuedRunningStaleCandidateV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	state, err := stack.readQueuedRunControlStateOrDefaultV0(ctx, candidate.RunRef)
	if err != nil {
		return err
	}
	if orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) != orquestaruncontrol.RunControlStatusRunningV0 {
		return nil
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, candidate.RunRef)
	if err != nil {
		if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return nil
		}
		return err
	}
	if !stackRunIsActiveV0(run) || len(compactStringsV0(run.StartedAgents)) == 0 {
		return nil
	}
	if len(stackDrainPendingStartedAgentRefsV0(run)) == 0 {
		return nil
	}
	liveness, err := stack.queuedRunningStaleProcessLivenessV0(ctx, run.RunID)
	if err != nil || !liveness.Verifiable || liveness.Live {
		return err
	}
	publicDecision := queuedRunningStalePublicStatusDecisionV0(candidate, liveness)
	if publicDecision.PublicStatus == orquestaruncoordinator.ExternalWorkPublicStatusRunningV0 {
		return nil
	}
	cause, err := stack.queuedRunningStaleCauseV0(ctx, run)
	if err != nil {
		return err
	}
	cause.EvidenceRefs = compactStringsV0(append(
		cause.EvidenceRefs,
		queuedRunningStalePublicStatusDecisionEvidenceRefsV0(publicDecision)...,
	))
	if !queuedRunningStaleRunCanReconcileV0(run, liveness, cause) {
		return nil
	}
	if liveness.RecordCount == 0 {
		cause.EvidenceRefs = compactStringsV0(append(
			cause.EvidenceRefs,
			"evidence-ref-agent-process-registry-empty",
		))
	}
	if _, _, err := stack.reconcileStoppedPendingAgentsV0(
		ctx,
		DrainRunRequestV0{
			RunRef:               strings.TrimSpace(run.RunID),
			OccurredAt:           formatStackCoordinatorTimeV0(command.OccurredAt),
			CorrelationID:        strings.TrimSpace(command.CorrelationID),
			MaxBursts:            command.DrainLimits.MaxBursts,
			MaxStepsPerBurst:     command.DrainLimits.MaxStepsPerBurst,
			MaxCommands:          command.DrainLimits.MaxCommands,
			MaxOutboxPerCycle:    command.DrainLimits.MaxOutboxPerCycle,
			MaxDispatchesPerWait: command.DrainLimits.MaxDispatchesPerWait,
			MaxExternalWaits:     command.DrainLimits.MaxExternalWaits,
		},
		run,
	); err != nil {
		return err
	}
	latest, err := stack.Ports.RunStore.LoadRunV0(ctx, candidate.RunRef)
	if err != nil {
		if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return nil
		}
		return err
	}
	if stack.runningStaleCandidateStillHasLiveProcessV0(ctx, latest) {
		return nil
	}
	if !codexStackRunLooksExternalWorkV0(run.ProjectRef, run.AppSpecRef) &&
		stackRunHasRecoverableTerminalAssessmentV0(latest) {
		return nil
	}
	reason := firstNonEmptyQueuedSourceV0(
		cause.Reason,
		"running_stale_sin_proceso_vivo_verificable",
	)
	evidenceRefs := compactStringsV0(append(
		[]string{
			"evidence-ref-run-control-running-stale-no-live-process",
			"evidence-ref-live-agent-reconciliation-direct",
		},
		cause.EvidenceRefs...,
	))
	completed, err := stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         strings.TrimSpace(candidate.RunRef),
			TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
			Reason:         reason,
			IdempotencyKey: "idem-run-control-running-stale-" + codexStackOperationalClosureSafeRefV0(reason) + "-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
			EvidenceRefs: compactStringsV0(append(
				append([]string(nil), state.EvidenceRefs...),
				evidenceRefs...,
			)),
		},
	)
	if err != nil {
		return err
	}
	return stack.syncQueuedRunningStaleCandidateStoppedV0(ctx, command, candidate, completed, reason, evidenceRefs)
}

func (stack StackV0) readQueuedRunControlStateOrDefaultV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}

type queuedRunningStaleProcessLivenessV0 struct {
	Verifiable  bool
	RecordCount int
	Live        bool
}

type queuedRunningStaleCauseV0 struct {
	Reason                  string
	RuntimeEvidenceObserved bool
	EvidenceRefs            []string
}

func queuedRunningStalePublicStatusDecisionV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	liveness queuedRunningStaleProcessLivenessV0,
) orquestaruncoordinator.ExternalWorkReconciliationDecisionV0 {
	classification := orquestaruncoordinator.RunLivenessClassificationV0{
		Class:                  orquestaruncoordinator.RunLivenessClassRunningStaleNoProcessV0,
		Reason:                 "running_stale_no_live_process_detected",
		RecommendedAction:      "reconcile_if_no_live_process_or_wait_for_late_ack",
		QueueStatusSuggestion:  orquestarunqueue.RunStatusStoppedV0,
		ConfirmedNoLiveProcess: liveness.Verifiable && !liveness.Live,
		Running:                true,
		Stale:                  true,
		Verifiable:             liveness.Verifiable,
		SafeToReconcile:        liveness.Verifiable && !liveness.Live,
	}
	return orquestaruncoordinator.ReconcileExternalWorkPublicStatusV0(
		orquestaruncoordinator.ExternalWorkReconciliationInputV0{
			RunRef:                 strings.TrimSpace(candidate.RunRef),
			ProjectionStatus:       strings.TrimSpace(candidate.Status),
			ProcessRegistryChecked: liveness.Verifiable,
			ProcessAlive:           liveness.Live,
			Liveness:               classification,
			EvidenceRefs: []string{
				"evidence-ref-external-work-public-status-reconciler",
			},
		},
	)
}

func queuedRunningStalePublicStatusDecisionEvidenceRefsV0(
	decision orquestaruncoordinator.ExternalWorkReconciliationDecisionV0,
) []string {
	refs := append([]string{}, decision.EvidenceRefs...)
	if decision.PublicStatus != orquestaruncoordinator.ExternalWorkPublicStatusRunningV0 {
		refs = append(refs, "evidence-ref-external-work-public-status-not-running-stale-no-process")
	}
	return compactStringsV0(refs)
}

func (stack StackV0) queuedRunningStaleCauseV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (queuedRunningStaleCauseV0, error) {
	result := queuedRunningStaleCauseV0{}
	if stack.Stores.ReceiptStore == nil {
		return result, nil
	}
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         strings.TrimSpace(run.RunID),
			StartedAgents: compactStringsV0(run.StartedAgents),
			EvidenceRefs:  []string{"evidence-ref-run-queue-running-stale-cause"},
		},
	)
	if err != nil {
		return queuedRunningStaleCauseV0{}, err
	}
	samples := make([]string, 0, len(descriptors)*4)
	for _, descriptor := range descriptors {
		dir := strings.TrimSpace(filepath.Dir(strings.TrimSpace(descriptor.AckPath)))
		if dir == "" || dir == "." {
			continue
		}
		for _, name := range []string{
			orquestaruntimecodex.CodexUsageAccountingFileNameV0,
			orquestaruntimecodex.CodexStderrFileNameV0,
			orquestaruntimecodex.CodexStdoutFileNameV0,
			orquestaruntimecodex.CodexLastMessageFileNameV0,
		} {
			data, ok := codexStackReadTailFileV0(filepath.Join(dir, name), codexStackRuntimeLogTailMaxBytesV0)
			if !ok {
				continue
			}
			result.RuntimeEvidenceObserved = true
			samples = append(samples, string(data))
		}
	}
	usage := orquestaruntimecodex.BuildCodexUsageAccountingSnapshotV0(samples)
	if usage.Observed && (usage.QuotaStatus == orquestaruntimecodex.CodexUsageQuotaExhaustedV0 ||
		usage.QuotaStatus == orquestaruntimecodex.CodexUsageQuotaLimitedV0) {
		result.Reason = "provider_usage_limit_retry_after"
		result.EvidenceRefs = append(
			result.EvidenceRefs,
			"evidence-ref-provider-usage-limit-retry-after",
			"evidence-ref-codex-usage-quota-"+usage.QuotaStatus,
		)
	}
	return result, nil
}

func queuedRunningStaleCauseAllowsNoProcessReconcileV0(cause queuedRunningStaleCauseV0) bool {
	return strings.TrimSpace(cause.Reason) == "provider_usage_limit_retry_after"
}

func queuedRunningStaleRunCanReconcileV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	liveness queuedRunningStaleProcessLivenessV0,
	cause queuedRunningStaleCauseV0,
) bool {
	if !liveness.Verifiable || liveness.Live {
		return false
	}
	if liveness.RecordCount > 0 {
		return true
	}
	return codexStackRunLooksOPESDirectWorkV0(run.ProjectRef, run.AppSpecRef) &&
		queuedRunningStaleCauseAllowsNoProcessReconcileV0(cause)
}

func (stack StackV0) queuedRunningStaleProcessLivenessV0(
	ctx context.Context,
	runRef string,
) (queuedRunningStaleProcessLivenessV0, error) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || stack.Stores.ProcessRegistry == nil || stack.CodexSnapshotSource == nil {
		return queuedRunningStaleProcessLivenessV0{}, nil
	}
	lister, ok := stack.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok {
		return queuedRunningStaleProcessLivenessV0{}, nil
	}
	records, err := lister.ListAgentProcessesV0(
		ctx,
		orquestacionnucleoapp.AgentProcessRegistryListFilterV0{RunID: runRef},
	)
	if err != nil {
		return queuedRunningStaleProcessLivenessV0{}, err
	}
	result := queuedRunningStaleProcessLivenessV0{
		Verifiable:  true,
		RecordCount: len(records),
	}
	unknownSnapshot := false
	for _, record := range records {
		snapshot, err := stack.CodexSnapshotSource.SnapshotV0(strings.TrimSpace(record.ProcessRef))
		if err != nil {
			if codexStackProcessRuntimeMissingV0(err) {
				unknownSnapshot = true
				continue
			}
			return queuedRunningStaleProcessLivenessV0{}, err
		}
		if !runControlRegisteredProcessMatchesSnapshotV0(record, snapshot) {
			result.Live = true
			return result, nil
		}
		switch snapshot.Status {
		case orquestaruntime.ProcessRuntimeRunningV0, orquestaruntime.ProcessRuntimeStoppingV0:
			result.Live = true
			return result, nil
		case orquestaruntime.ProcessRuntimeStoppedV0:
		default:
			unknownSnapshot = true
		}
	}
	if unknownSnapshot {
		result.Verifiable = false
	}
	return result, nil
}

func (stack StackV0) runningStaleCandidateStillHasLiveProcessV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	liveness, err := stack.queuedRunningStaleProcessLivenessV0(ctx, run.RunID)
	return err != nil || !liveness.Verifiable || liveness.Live
}

func (stack StackV0) syncQueuedRunningStaleCandidateStoppedV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
	reason string,
	evidenceRefs []string,
) error {
	refs := compactStringsV0(append(
		append(append([]string(nil), candidate.EvidenceRefs...), evidenceRefs...),
		"evidence-ref-run-queue-running-stale-no-live-process-reconciled",
	))
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, stackRunQueueCommandFromCandidateV0(
		command,
		candidate,
		orquestarunqueue.RunStatusStoppedV0,
		"orquesta-app-codex-stack-run-control-reconciler",
		firstNonEmptyQueuedSourceV0(reason, "queued_running_stale_no_live_process_reconciled"),
		"idem-run-queue-running-stale-"+codexStackOperationalClosureSafeRefV0(firstNonEmptyQueuedSourceV0(reason, "no-live"))+"-"+codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		compactStringsV0(append(refs, state.EvidenceRefs...)),
	))
	return err
}
