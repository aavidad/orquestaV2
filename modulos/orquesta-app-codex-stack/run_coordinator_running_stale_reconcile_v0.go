package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
	if !codexStackRunLooksOPESDirectWorkV0(run.ProjectRef, run.AppSpecRef) {
		return nil
	}
	liveness, err := stack.queuedRunningStaleProcessLivenessV0(ctx, run.RunID)
	if err != nil || !liveness.Verifiable || liveness.Live || liveness.RecordCount == 0 {
		return err
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
	completed, err := stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         strings.TrimSpace(candidate.RunRef),
			TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
			Reason:         "running_stale_sin_proceso_vivo_verificable",
			IdempotencyKey: "idem-run-control-running-stale-no-live-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
			EvidenceRefs: compactStringsV0(append(
				append([]string(nil), state.EvidenceRefs...),
				"evidence-ref-run-control-running-stale-no-live-process",
				"evidence-ref-live-agent-reconciliation-direct",
			)),
		},
	)
	if err != nil {
		return err
	}
	return stack.syncQueuedRunningStaleCandidateStoppedV0(ctx, command, candidate, completed)
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
	for _, record := range records {
		snapshot, err := stack.CodexSnapshotSource.SnapshotV0(strings.TrimSpace(record.ProcessRef))
		if err != nil {
			if codexStackProcessRuntimeMissingV0(err) {
				continue
			}
			return queuedRunningStaleProcessLivenessV0{}, err
		}
		if !runControlRegisteredProcessMatchesSnapshotV0(record, snapshot) ||
			snapshot.Status == orquestaruntime.ProcessRuntimeRunningV0 ||
			snapshot.Status == orquestaruntime.ProcessRuntimeStoppingV0 {
			result.Live = true
			return result, nil
		}
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
) error {
	refs := compactStringsV0(append(
		append([]string(nil), candidate.EvidenceRefs...),
		"evidence-ref-run-queue-running-stale-no-live-process-reconciled",
		"evidence-ref-run-control-running-stale-no-live-process",
	))
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, stackRunQueueCommandFromCandidateV0(
		command,
		candidate,
		orquestarunqueue.RunStatusStoppedV0,
		"orquesta-app-codex-stack-run-control-reconciler",
		"queued_running_stale_no_live_process_reconciled",
		"idem-run-queue-running-stale-no-live-"+codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		compactStringsV0(append(refs, state.EvidenceRefs...)),
	))
	return err
}
