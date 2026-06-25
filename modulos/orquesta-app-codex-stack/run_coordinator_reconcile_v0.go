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

func (stack StackV0) recoverQueuedStoppedActiveRunsV0(
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
	policy := buildQueuedStoppedRecoveryPolicyV0(candidates)
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !policy.shouldRecoverV0(candidate) {
			continue
		}
		if err := stack.recoverQueuedStoppedCandidateV0(ctx, command, candidate); err != nil {
			return err
		}
	}
	return nil
}

const defaultQueuedStoppedRecoveryReadLimitV0 = 70

func queuedStoppedRecoveryReadLimitV0(
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) int {
	if command.QueueLimit > 0 {
		return command.QueueLimit
	}
	return defaultQueuedStoppedRecoveryReadLimitV0
}

type queuedStoppedRecoveryPolicyV0 struct {
	blockedGroups map[string]bool
	latestStopped map[string]orquestarunqueue.RunSchedulingCandidateV0
}

func buildQueuedStoppedRecoveryPolicyV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) queuedStoppedRecoveryPolicyV0 {
	policy := queuedStoppedRecoveryPolicyV0{
		blockedGroups: map[string]bool{},
		latestStopped: map[string]orquestarunqueue.RunSchedulingCandidateV0{},
	}
	for _, candidate := range candidates {
		group := queuedStoppedCanonicalRunRefV0(candidate.RunRef)
		if group == "" {
			continue
		}
		switch strings.TrimSpace(candidate.Status) {
		case orquestarunqueue.RunStatusClosedV0,
			orquestarunqueue.RunStatusDeliveredV0,
			orquestarunqueue.RunStatusReadyV0,
			orquestarunqueue.RunStatusRunningV0:
			policy.blockedGroups[group] = true
		case orquestarunqueue.RunStatusStoppedV0:
			current, ok := policy.latestStopped[group]
			if !ok || queuedStoppedCandidateAfterV0(candidate, current) {
				policy.latestStopped[group] = candidate
			}
		}
	}
	return policy
}

func (policy queuedStoppedRecoveryPolicyV0) shouldRecoverV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) bool {
	if strings.TrimSpace(candidate.Status) != orquestarunqueue.RunStatusStoppedV0 {
		return true
	}
	group := queuedStoppedCanonicalRunRefV0(candidate.RunRef)
	if group == "" || policy.blockedGroups[group] {
		return false
	}
	latest, ok := policy.latestStopped[group]
	return ok && strings.TrimSpace(latest.RunRef) == strings.TrimSpace(candidate.RunRef)
}

func queuedStoppedCanonicalRunRefV0(runRef string) string {
	runRef = strings.TrimSpace(runRef)
	if index := strings.Index(runRef, "-retry-"); index > 0 {
		return runRef[:index]
	}
	return runRef
}

func queuedStoppedCandidateAfterV0(
	left orquestarunqueue.RunSchedulingCandidateV0,
	right orquestarunqueue.RunSchedulingCandidateV0,
) bool {
	if left.UpdatedAt.Equal(right.UpdatedAt) {
		return strings.TrimSpace(left.RunRef) > strings.TrimSpace(right.RunRef)
	}
	if left.UpdatedAt.IsZero() {
		return false
	}
	if right.UpdatedAt.IsZero() {
		return true
	}
	return left.UpdatedAt.After(right.UpdatedAt)
}

func (stack StackV0) recoverQueuedStoppedCandidateV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: candidate.RunRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			state = orquestaruncontrol.DefaultRunControlStateV0(candidate.RunRef)
		} else {
			return err
		}
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, candidate.RunRef)
	if err != nil {
		if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return stack.completeQueuedRunControlWithoutActiveRunV0(ctx, command, candidate, state)
		}
		return err
	}
	if !stackRunIsActiveV0(run) {
		recovered, ok, err := stack.recoverBlockedAppChangeAutoPlanRunV0(ctx, command, state, run)
		if err != nil {
			return err
		}
		if !ok {
			recovered, ok, err = stack.recoverBlockedDirectorDecisionSourceRunForQueueV0(ctx, command, state, run)
			if err != nil {
				return err
			}
		}
		if !ok {
			recovered, ok, err = stack.recoverBlockedAutoprogrammingOpenReviewRunForQueueV0(ctx, command, state, run)
			if err != nil {
				return err
			}
		}
		if !ok {
			recovered, ok, err = stack.recoverBlockedOpenPhaseProjectionRunForQueueV0(ctx, command, state, run)
			if err != nil {
				return err
			}
		}
		if !ok {
			return stack.completeQueuedRunControlWithoutActiveRunV0(ctx, command, candidate, state)
		}
		run = recovered
	}
	if err := stack.recoverQueuedStoppedAgentProgressV0(ctx, command, run); err != nil {
		if codexStackQueuedRecoveryAdvisoryErrorV0(err) {
			return stack.recoverQueuedRunControlV0(ctx, command, candidate, state)
		}
		return err
	}
	completed, err := stack.completeQueuedRunControlIfStopHasNoPendingAgentsV0(ctx, command, candidate, state, run)
	if err != nil || completed {
		return err
	}
	return stack.recoverQueuedRunControlV0(ctx, command, candidate, state)
}

func codexStackQueuedRecoveryAdvisoryErrorV0(err error) bool {
	var workflowErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &workflowErr) {
		return workflowErr.Code == orquestacoreworkflow.ErrTransicionInvalidaV0 &&
			strings.HasPrefix(strings.TrimSpace(workflowErr.Field), "payload")
	}
	var coreErr orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreErr) {
		return coreErr.Code == orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0 &&
			strings.HasPrefix(strings.TrimSpace(coreErr.Field), "payload")
	}
	return false
}

func (stack StackV0) completeQueuedRunControlIfStopHasNoPendingAgentsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if !codexStackQueueStatusCanCompleteRunControlStopV0(candidate.Status, state.Status) {
		return false, nil
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if !evaluation.StopAgentsAllowed || len(stackRunControlUnconfirmedStopAgentRefsV0(run)) > 0 {
		return false, nil
	}
	if len(stackDrainOpenTaskRefsV0(run)) > 0 {
		return false, nil
	}
	if pending, err := stack.domainWorkRunHasPendingSubmissionWithoutAcceptedReceiptV0(ctx, run); err != nil || pending {
		return false, err
	}
	if live, err := stack.runControlHasLiveRegisteredProcessesV0(ctx, run.RunID); err != nil || live {
		return false, err
	}
	target, ok := codexStackTerminalStatusForRunControlRequestV0(state.Status)
	if !ok {
		return false, nil
	}
	completed, err := stack.Stores.RunControl.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         candidate.RunRef,
		TargetStatus:   target,
		RequestedBy:    "orquesta-run-control",
		Reason:         "run sin agentes pendientes; control terminalizado por reconciliacion",
		IdempotencyKey: "idem-run-control-complete-no-pending-agents-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		EvidenceRefs: append(
			append([]string(nil), state.EvidenceRefs...),
			"evidence-ref-run-control-complete-no-pending-agents",
		),
	})
	if err != nil {
		return false, err
	}
	if err := stack.syncQueuedRunControlBlockedCandidateV0(ctx, command, candidate, completed); err != nil {
		return true, err
	}
	return true, nil
}

func stackRunControlUnconfirmedStopAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	confirmed := map[string]bool{}
	for _, values := range [][]string{
		run.DeliveredAgents,
		run.FailedAgents,
		run.LostAgents,
		run.ConfirmedStoppedAgents,
		stackDrainPhaseArtifactAgentRefsV0(run),
	} {
		for _, agentRef := range compactStringsV0(values) {
			confirmed[agentRef] = true
		}
	}
	refs := make([]string, 0, len(run.StartedAgents))
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if confirmed[agentRef] {
			continue
		}
		refs = append(refs, agentRef)
	}
	return refs
}

func (stack StackV0) runControlHasLiveRegisteredProcessesV0(
	ctx context.Context,
	runRef string,
) (bool, error) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || stack.Stores.ProcessRegistry == nil || stack.CodexSnapshotSource == nil {
		return false, nil
	}
	lister, ok := stack.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0)
	if !ok {
		return false, nil
	}
	records, err := lister.ListAgentProcessesV0(
		ctx,
		orquestacionnucleoapp.AgentProcessRegistryListFilterV0{RunID: runRef},
	)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		snapshot, err := stack.CodexSnapshotSource.SnapshotV0(strings.TrimSpace(record.ProcessRef))
		if err != nil {
			if codexStackProcessRuntimeMissingV0(err) {
				continue
			}
			return false, err
		}
		if !runControlRegisteredProcessMatchesSnapshotV0(record, snapshot) ||
			snapshot.Status == orquestaruntime.ProcessRuntimeRunningV0 ||
			snapshot.Status == orquestaruntime.ProcessRuntimeStoppingV0 {
			return true, nil
		}
	}
	return false, nil
}

func runControlRegisteredProcessMatchesSnapshotV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) bool {
	if strings.TrimSpace(record.ProcessRef) == "" ||
		strings.TrimSpace(record.ProcessRef) != strings.TrimSpace(snapshot.ProcessRef) {
		return false
	}
	if strings.TrimSpace(record.SessionRef) != "" &&
		strings.TrimSpace(record.SessionRef) != strings.TrimSpace(snapshot.SessionRef) {
		return false
	}
	if strings.TrimSpace(record.LaunchRef) != "" &&
		strings.TrimSpace(snapshot.LaunchRef) != "" &&
		strings.TrimSpace(record.LaunchRef) != strings.TrimSpace(snapshot.LaunchRef) {
		return false
	}
	return true
}

func codexStackQueueStatusCanCompleteRunControlStopV0(
	queueStatus string,
	controlStatus orquestaruncontrol.RunControlStatusV0,
) bool {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(controlStatus) {
	case orquestaruncontrol.RunControlStatusStopRequestedV0:
		switch strings.TrimSpace(queueStatus) {
		case orquestarunqueue.RunStatusRunningV0, orquestarunqueue.RunStatusStoppedV0:
			return true
		default:
			return false
		}
	case orquestaruncontrol.RunControlStatusCancelRequestedV0:
		switch strings.TrimSpace(queueStatus) {
		case orquestarunqueue.RunStatusRunningV0, orquestarunqueue.RunStatusCanceledV0:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func (stack StackV0) completeQueuedRunControlWithoutActiveRunV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) error {
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if !evaluation.StopAgentsAllowed {
		return nil
	}
	target, ok := codexStackTerminalStatusForRunControlRequestV0(state.Status)
	if !ok {
		return nil
	}
	completed, err := stack.Stores.RunControl.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         candidate.RunRef,
		TargetStatus:   target,
		RequestedBy:    "orquesta-run-control",
		Reason:         "run sin trabajo activo; control terminalizado por reconciliacion",
		IdempotencyKey: "idem-run-control-complete-without-active-run-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		EvidenceRefs: append(
			append([]string(nil), state.EvidenceRefs...),
			"evidence-ref-run-control-complete-without-active-run",
		),
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(candidate.Status) == orquestarunqueue.RunStatusClosedV0 {
		return nil
	}
	return stack.syncQueuedRunControlBlockedCandidateV0(ctx, command, candidate, completed)
}

func (stack StackV0) recoverQueuedRunControlV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) error {
	if state.Status == orquestaruncontrol.RunControlStatusStoppedV0 &&
		stackRunControlCanAutoResumeQueuedCandidateV0(state) {
		var err error
		state, err = stack.Stores.RunControl.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
			RunRef:         candidate.RunRef,
			RequestedBy:    "orquesta-app-codex-stack-run-control-reconciler",
			Reason:         "queued_ready_active_run_reconciled",
			IdempotencyKey: "idem-run-control-reconcile-ready-" + codexStackOperationalClosureSafeRefV0(candidate.RunRef),
			EvidenceRefs: []string{
				"evidence-ref-run-control-queued-ready-active-reconciled",
				"evidence-ref-run-control-stopped-state-preserved-in-history",
			},
		})
		if err != nil {
			return err
		}
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if evaluation.StopAgentsAllowed && !orquestarunqueue.IsExecutableRunStatusV0(candidate.Status) {
		_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, stackRunQueueCommandFromCandidateV0(
			command,
			candidate,
			orquestarunqueue.RunStatusReadyV0,
			"orquesta-app-codex-stack-run-control-reconciler",
			"queued_non_executable_stop_requested_reconciled",
			"idem-run-queue-reconcile-stop-ready-"+codexStackOperationalClosureSafeRefV0(candidate.RunRef),
			compactStringsV0(append(
				append([]string(nil), candidate.EvidenceRefs...),
				"evidence-ref-run-queue-stop-requested-ready-for-drain",
			)),
		))
		return err
	}
	if !evaluation.DispatchAllowed {
		return stack.syncQueuedRunControlBlockedCandidateV0(ctx, command, candidate, state)
	}
	if orquestarunqueue.IsExecutableRunStatusV0(candidate.Status) {
		return nil
	}
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, stackRunQueueCommandFromCandidateV0(
		command,
		candidate,
		orquestarunqueue.RunStatusReadyV0,
		"orquesta-app-codex-stack-run-control-reconciler",
		"queued_non_executable_active_run_reconciled",
		"idem-run-queue-reconcile-ready-"+codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		compactStringsV0(append(append([]string(nil), candidate.EvidenceRefs...),
			"evidence-ref-run-queue-non-executable-active-reconciled",
			"evidence-ref-run-control-dispatch-allowed",
		)),
	))
	return err
}

func (stack StackV0) syncQueuedRunControlBlockedCandidateV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) error {
	status, ok := codexStackQueueStatusForRunControlV0(state)
	if !ok || strings.TrimSpace(candidate.Status) == status {
		return nil
	}
	refs := append([]string(nil), candidate.EvidenceRefs...)
	refs = append(refs, "evidence-ref-run-queue-run-control-blocked-reconciled")
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, stackRunQueueCommandFromCandidateV0(
		command,
		candidate,
		status,
		"orquesta-app-codex-stack-run-control-reconciler",
		"queued_run_control_blocked_reconciled",
		"idem-run-queue-reconcile-blocked-"+codexStackOperationalClosureSafeRefV0(candidate.RunRef),
		refs,
	))
	return err
}

func stackRunQueueCommandFromCandidateV0(
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	status string,
	requestedBy string,
	reason string,
	idempotencyKey string,
	evidenceRefs []string,
) orquestarunqueue.RunQueuePriorityCommandV0 {
	return orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           candidate.RunRef,
		QueueRef:         command.QueueRef,
		AppRef:           candidate.AppRef,
		Status:           status,
		PriorityScore:    candidate.PriorityScore,
		UpdatedAt:        command.OccurredAt,
		FairnessGroupRef: candidate.FairnessGroupRef,
		AttemptGroup:     candidate.AttemptGroup,
		ParentRunRef:     candidate.ParentRunRef,
		SupersedesRunRef: candidate.SupersedesRunRef,
		RescueReason:     candidate.RescueReason,
		RequestedBy:      requestedBy,
		Reason:           reason,
		IdempotencyKey:   idempotencyKey,
		EvidenceRefs:     evidenceRefs,
		WorksetClaims:    candidate.WorksetClaims,
	}
}

func codexStackTerminalStatusForRunControlRequestV0(
	status orquestaruncontrol.RunControlStatusV0,
) (orquestaruncontrol.RunControlStatusV0, bool) {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(status) {
	case orquestaruncontrol.RunControlStatusStopRequestedV0:
		return orquestaruncontrol.RunControlStatusStoppedV0, true
	case orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return orquestaruncontrol.RunControlStatusCanceledV0, true
	default:
		return "", false
	}
}

func codexStackQueueStatusForRunControlV0(
	state orquestaruncontrol.RunControlStateV0,
) (string, bool) {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestarunqueue.RunStatusStoppedV0, true
	case orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestarunqueue.RunStatusCanceledV0, true
	default:
		return "", false
	}
}

func stackRunIsActiveV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return run.Status == orquestacoreworkflow.OrchestrationRunStatusActiveV0
}

func stackRunControlCanAutoResumeQueuedCandidateV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	if !stackRunControlAutoResumeExplicitlyAllowedV0(state) {
		return false
	}
	if stackRunControlLooksServerShutdownV0(state) {
		return true
	}
	idempotency := strings.TrimSpace(state.Meta.IdempotencyKey)
	reason := strings.TrimSpace(state.Meta.Reason)
	requestedBy := strings.TrimSpace(state.Meta.RequestedBy)
	if requestedBy == "orquesta-run-control" &&
		strings.HasPrefix(idempotency, "idem-run-control-complete-") &&
		strings.HasSuffix(idempotency, "-stopped") &&
		strings.Contains(reason, "agentes drenados por control de run") {
		return true
	}
	return false
}

func stackRunControlAutoResumeExplicitlyAllowedV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	for _, ref := range state.EvidenceRefs {
		if strings.TrimSpace(ref) == orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0 {
			return true
		}
	}
	return false
}

func stackRunControlLooksServerShutdownV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	idempotency := strings.TrimSpace(state.Meta.IdempotencyKey)
	reason := strings.TrimSpace(state.Meta.Reason)
	requestedBy := strings.TrimSpace(state.Meta.RequestedBy)
	if strings.HasPrefix(idempotency, "idem-orquesta-server-stop") ||
		strings.HasPrefix(idempotency, "idem-run-control-complete-idem-orquesta-server-stop") {
		return true
	}
	return requestedBy == "orquesta-director" &&
		(strings.Contains(reason, "apagado controlado solicitado por CLI") ||
			strings.Contains(strings.ToLower(reason), "shutdown"))
}
