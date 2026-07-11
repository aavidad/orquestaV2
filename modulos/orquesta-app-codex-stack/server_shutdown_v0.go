package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func serverShutdownExecutorV0(
	config ConfigV0,
	stack *StackV0,
) orquestamcp.MCPServerShutdownToolExecutorV0 {
	return orquestamcp.NewMCPServerShutdownToolExecutorV0(
		orquestaservershutdown.ServerShutdownDepsV0{
			QueueReader:         stackShutdownQueueReaderFromConfigV0(config),
			RunControlReader:    config.Stores.RunControl,
			RunControlWriter:    stackShutdownRunControlWriterFromConfigV0(config),
			RunCheckpointWriter: config.Stores.RunControl,
			CheckpointPreparer:  stackShutdownCheckpointPreparerV0{Config: config},
			Supervisor:          stackShutdownSupervisorV0{Stack: stack},
			StatsReader:         stackShutdownStatsReaderV0{Config: config},
			ActiveWorkReader:    stackShutdownActiveWorkReaderV0{Config: config},
			ActiveWorkCleaner:   stackShutdownActiveWorkCleanerV0{Config: config},
		},
	)
}

func NewShutdownActiveWorkReaderV0(
	config ConfigV0,
) orquestaservershutdown.ActiveShutdownWorkReaderPortV0 {
	return stackShutdownActiveWorkReaderV0{Config: config}
}

func stackShutdownRunControlWriterFromConfigV0(
	config ConfigV0,
) orquestaruncontrol.RunControlWriterPortV0 {
	if config.Stores.RunControl == nil {
		return nil
	}
	return stackShutdownRunControlWriterV0{
		Inner:          goalFirstRunControlPortFromConfigV0(config),
		Reader:         config.Stores.RunControl,
		Terminal:       config.Stores.RunControl,
		GoalStateStore: config.Stores.AppGoalStateStore,
	}
}

func stackShutdownQueueReaderFromConfigV0(
	config ConfigV0,
) orquestarunqueue.RunQueueReaderPortV0 {
	if config.Stores.RunQueue == nil {
		return nil
	}
	return stackShutdownGoalFirstQueueReaderV0{
		Inner:  config.Stores.RunQueue,
		Config: config,
	}
}

type stackShutdownSupervisorV0 struct {
	Stack *StackV0
}

func (supervisor stackShutdownSupervisorV0) RunGlobalSupervisorV0(
	ctx context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	if supervisor.Stack == nil {
		return orquestarunsupervisor.RunSupervisorResultV0{}, nil
	}
	return supervisor.Stack.RunGlobalSupervisorV0(ctx, command)
}

type stackShutdownStatsReaderV0 struct {
	Config ConfigV0
}

type stackShutdownActiveWorkReaderV0 struct {
	Config ConfigV0
}

type stackShutdownActiveWorkCleanerV0 struct {
	Config ConfigV0
}

const (
	serverShutdownGoalTerminalRunControlPendingEvidenceV0    = "evidence-ref-server-shutdown-goal-terminal-run-control-pending"
	serverShutdownGoalTerminalRunControlReconciledEvidenceV0 = "evidence-ref-server-shutdown-goal-terminal-run-control-reconciled"
)

func (reader stackShutdownActiveWorkReaderV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	out := orquestaservershutdown.ActiveShutdownWorkResultV0{}
	for _, activeReader := range stackShutdownActiveWorkReadersFromGoalBackendV0(reader.Config) {
		active, err := activeReader.ReadActiveShutdownWorkV0(ctx, request)
		if err != nil {
			return orquestaservershutdown.ActiveShutdownWorkResultV0{}, err
		}
		out = stackShutdownMergeActiveWorkResultV0(out, active)
	}
	if reader.Config.Stores.AppGoalStateStore == nil {
		return out, nil
	}
	lister, ok := reader.Config.Stores.AppGoalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok {
		return out, nil
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		Statuses: []string{
			orquestagoal.GoalStatusRunningV0,
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		},
		MaxItems: request.MaxItems,
	})
	if err != nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, err
	}
	backendObserved := stackShutdownActiveWorkResultHasGoalBackendV0(out)
	for _, state := range states {
		if stackShutdownGoalBackendStillRunningV0(state) {
			backendObserved = true
			break
		}
	}
	for _, state := range states {
		if stackShutdownGoalBackendStillRunningV0(state) {
			evidenceRefs := stackShutdownGoalBackendEvidenceRefsV0(state)
			out.ActiveWorks = append(out.ActiveWorks, orquestaservershutdown.ActiveShutdownWorkV0{
				Kind:            "goal_backend",
				RunRef:          strings.TrimSpace(state.RunRef),
				WorkRef:         strings.TrimSpace(state.GoalRef),
				ExternalWorkRef: strings.TrimSpace(state.ExternalGoalRef),
				Status:          orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
				EvidenceRefs:    evidenceRefs,
			})
			out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, evidenceRefs...))
			continue
		}
		if !backendObserved {
			control, pending, err := stackShutdownTerminalGoalRunControlPendingV0(
				ctx,
				reader.Config.Stores.RunControl,
				state,
			)
			if err != nil {
				return orquestaservershutdown.ActiveShutdownWorkResultV0{}, err
			}
			if pending {
				evidenceRefs := compactStringsV0(append(
					append([]string(nil), state.EvidenceRefs...),
					serverShutdownGoalTerminalRunControlPendingEvidenceV0,
				))
				out.ActiveWorks = append(out.ActiveWorks, orquestaservershutdown.ActiveShutdownWorkV0{
					Kind:            "goal_first",
					RunRef:          strings.TrimSpace(state.RunRef),
					WorkRef:         strings.TrimSpace(state.GoalRef),
					ExternalWorkRef: strings.TrimSpace(state.ExternalGoalRef),
					Status:          string(orquestaruncontrol.NormalizeRunControlStatusV0(control.Status)),
					EvidenceRefs:    evidenceRefs,
				})
				out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, evidenceRefs...))
				continue
			}
		}
		if !orquestagoal.GoalWorkStatePendingObservationV0(state) {
			continue
		}
		out.ActiveWorks = append(out.ActiveWorks, orquestaservershutdown.ActiveShutdownWorkV0{
			Kind:            "goal_first",
			RunRef:          strings.TrimSpace(state.RunRef),
			WorkRef:         strings.TrimSpace(state.GoalRef),
			ExternalWorkRef: strings.TrimSpace(state.ExternalGoalRef),
			Status:          strings.TrimSpace(state.Status),
			EvidenceRefs:    compactStringsV0(state.EvidenceRefs),
		})
		out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, state.EvidenceRefs...))
	}
	return out, nil
}

type stackShutdownGoalFirstQueueReaderV0 struct {
	Inner  orquestarunqueue.RunQueueReaderPortV0
	Config ConfigV0
}

func (reader stackShutdownGoalFirstQueueReaderV0) ListRunSchedulingCandidatesV0(
	ctx context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	candidates, err := reader.Inner.ListRunSchedulingCandidatesV0(ctx, request)
	if err != nil {
		return nil, err
	}
	supplemental, err := reader.goalFirstPendingControlCandidatesV0(ctx, candidates, request)
	if err != nil {
		return nil, err
	}
	return stackShutdownCompactQueueCandidatesV0(append(candidates, supplemental...)), nil
}

func (reader stackShutdownGoalFirstQueueReaderV0) goalFirstPendingControlCandidatesV0(
	ctx context.Context,
	existing []orquestarunqueue.RunSchedulingCandidateV0,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	if reader.Config.Stores.AppGoalStateStore == nil ||
		reader.Config.Stores.RunControl == nil {
		return []orquestarunqueue.RunSchedulingCandidateV0{}, nil
	}
	lister, ok := reader.Config.Stores.AppGoalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok || lister == nil {
		return []orquestarunqueue.RunSchedulingCandidateV0{}, nil
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		Statuses: []string{
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		},
		MaxItems: request.Limit,
	})
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, candidate := range existing {
		if runRef := strings.TrimSpace(candidate.RunRef); runRef != "" {
			seen[runRef] = struct{}{}
		}
	}
	out := make([]orquestarunqueue.RunSchedulingCandidateV0, 0, len(states))
	for _, state := range states {
		state, err = orquestagoal.NewGoalWorkStateV0(state)
		if err != nil {
			return nil, err
		}
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" || request.RunRef != "" && runRef != strings.TrimSpace(request.RunRef) {
			continue
		}
		if _, exists := seen[runRef]; exists {
			continue
		}
		control, pending, err := stackShutdownTerminalGoalRunControlPendingV0(ctx, reader.Config.Stores.RunControl, state)
		if err != nil {
			return nil, err
		}
		if !pending {
			continue
		}
		appRef := stackShutdownGoalCandidateAppRefV0(state)
		if len(compactStringsV0(request.AppRefs)) > 0 && !stackShutdownAppRefAllowedV0(appRef, request.AppRefs) {
			continue
		}
		seen[runRef] = struct{}{}
		out = append(out, orquestarunqueue.RunSchedulingCandidateV0{
			RunRef:        runRef,
			AppRef:        appRef,
			Status:        orquestarunqueue.RunStatusRunningV0,
			PriorityScore: 1,
			EvidenceRefs: compactStringsV0(append(
				append([]string(nil), state.EvidenceRefs...),
				serverShutdownGoalTerminalRunControlPendingEvidenceV0,
				string(orquestaruncontrol.NormalizeRunControlStatusV0(control.Status)),
			)),
		})
	}
	if out == nil {
		return []orquestarunqueue.RunSchedulingCandidateV0{}, nil
	}
	return out, nil
}

func stackShutdownTerminalGoalRunControlPendingV0(
	ctx context.Context,
	runControl orquestaruncontrol.RunControlReaderPortV0,
	state orquestagoal.GoalWorkStateV0,
) (orquestaruncontrol.RunControlStateV0, bool, error) {
	if runControl == nil || !stackShutdownGoalReadyForRunControlTerminalV0(state) {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	runRef := strings.TrimSpace(state.RunRef)
	if runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	control, err := runControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			return orquestaruncontrol.RunControlStateV0{}, false, nil
		}
		return orquestaruncontrol.RunControlStateV0{}, false, err
	}
	_, pending := stackShutdownTerminalRunControlTargetV0(control.Status)
	return control, pending, nil
}

func stackShutdownGoalReadyForRunControlTerminalV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	return orquestagoal.GoalWorkResultTerminalV0(state.Status) &&
		!orquestagoal.GoalWorkStatePendingObservationV0(state)
}

func stackShutdownTerminalRunControlTargetV0(
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

func stackShutdownActiveWorkResultHasGoalBackendV0(
	result orquestaservershutdown.ActiveShutdownWorkResultV0,
) bool {
	for _, work := range result.ActiveWorks {
		if strings.TrimSpace(work.Kind) == "goal_backend" ||
			strings.TrimSpace(work.Status) == orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 {
			return true
		}
	}
	return false
}

func stackShutdownGoalCandidateAppRefV0(
	state orquestagoal.GoalWorkStateV0,
) string {
	return firstNonEmptyQueuedSourceV0(
		state.Spec.ProjectRef,
		state.Spec.DomainRef,
		state.RunRef,
	)
}

func stackShutdownAppRefAllowedV0(
	appRef string,
	allowed []string,
) bool {
	appRef = strings.TrimSpace(appRef)
	for _, candidate := range allowed {
		if appRef != "" && appRef == strings.TrimSpace(candidate) {
			return true
		}
	}
	return false
}

func stackShutdownCompactQueueCandidatesV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) []orquestarunqueue.RunSchedulingCandidateV0 {
	out := make([]orquestarunqueue.RunSchedulingCandidateV0, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate.RunRef = strings.TrimSpace(candidate.RunRef)
		candidate.AppRef = strings.TrimSpace(candidate.AppRef)
		candidate.Status = strings.TrimSpace(candidate.Status)
		candidate.EvidenceRefs = compactStringsV0(candidate.EvidenceRefs)
		if candidate.RunRef == "" {
			continue
		}
		if _, exists := seen[candidate.RunRef]; exists {
			continue
		}
		seen[candidate.RunRef] = struct{}{}
		out = append(out, candidate)
	}
	if out == nil {
		return []orquestarunqueue.RunSchedulingCandidateV0{}
	}
	return out
}

func stackShutdownActiveWorkReadersFromGoalBackendV0(
	config ConfigV0,
) []orquestaservershutdown.ActiveShutdownWorkReaderPortV0 {
	out := []orquestaservershutdown.ActiveShutdownWorkReaderPortV0{}
	seen := map[string]struct{}{}
	for _, candidate := range []interface{}{
		config.AppGoalObserver,
		config.AppGoalLauncher,
		config.AppGoalReworkLauncher,
	} {
		activeReader, ok := candidate.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0)
		if !ok || activeReader == nil {
			continue
		}
		if stackShutdownActiveWorkPortSeenV0(seen, activeReader) {
			continue
		}
		out = append(out, activeReader)
	}
	return out
}

func (cleaner stackShutdownActiveWorkCleanerV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	out := orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}
	for _, activeCleaner := range stackShutdownActiveWorkCleanersFromGoalBackendV0(cleaner.Config) {
		cleaned, err := activeCleaner.CleanupActiveShutdownWorkV0(ctx, command)
		if err != nil {
			return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, err
		}
		out.CleanedWorkCount += cleaned.CleanedWorkCount
		out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, cleaned.EvidenceRefs...))
	}
	return out, nil
}

func stackShutdownActiveWorkCleanersFromGoalBackendV0(
	config ConfigV0,
) []orquestaservershutdown.ActiveShutdownWorkCleanerPortV0 {
	out := []orquestaservershutdown.ActiveShutdownWorkCleanerPortV0{}
	seen := map[string]struct{}{}
	for _, candidate := range []interface{}{
		config.AppGoalObserver,
		config.AppGoalLauncher,
		config.AppGoalReworkLauncher,
	} {
		activeCleaner, ok := candidate.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0)
		if !ok || activeCleaner == nil {
			continue
		}
		if stackShutdownActiveWorkPortSeenV0(seen, activeCleaner) {
			continue
		}
		out = append(out, activeCleaner)
	}
	return out
}

func stackShutdownActiveWorkPortSeenV0(
	seen map[string]struct{},
	port interface{},
) bool {
	identity, ok := port.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
	if !ok || identity == nil {
		return false
	}
	key := strings.TrimSpace(identity.ActiveShutdownWorkIdentityV0())
	if key == "" {
		return false
	}
	if _, exists := seen[key]; exists {
		return true
	}
	seen[key] = struct{}{}
	return false
}

func stackShutdownMergeActiveWorkResultV0(
	left orquestaservershutdown.ActiveShutdownWorkResultV0,
	right orquestaservershutdown.ActiveShutdownWorkResultV0,
) orquestaservershutdown.ActiveShutdownWorkResultV0 {
	left.ActiveWorks = stackShutdownCompactActiveWorksV0(append(left.ActiveWorks, right.ActiveWorks...))
	left.EvidenceRefs = compactStringsV0(append(left.EvidenceRefs, right.EvidenceRefs...))
	return left
}

func stackShutdownCompactActiveWorksV0(
	works []orquestaservershutdown.ActiveShutdownWorkV0,
) []orquestaservershutdown.ActiveShutdownWorkV0 {
	out := make([]orquestaservershutdown.ActiveShutdownWorkV0, 0, len(works))
	seen := map[string]struct{}{}
	for _, work := range works {
		work.Kind = strings.TrimSpace(work.Kind)
		work.RunRef = strings.TrimSpace(work.RunRef)
		work.WorkRef = strings.TrimSpace(work.WorkRef)
		work.ExternalWorkRef = strings.TrimSpace(work.ExternalWorkRef)
		work.Status = strings.TrimSpace(work.Status)
		work.EvidenceRefs = compactStringsV0(work.EvidenceRefs)
		if work.Kind == "" && work.RunRef == "" && work.WorkRef == "" && work.ExternalWorkRef == "" {
			continue
		}
		key := strings.Join([]string{work.Kind, work.RunRef, work.WorkRef, work.ExternalWorkRef, work.Status}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, work)
	}
	if out == nil {
		return []orquestaservershutdown.ActiveShutdownWorkV0{}
	}
	return out
}

func stackShutdownGoalBackendStillRunningV0(state orquestagoal.GoalWorkStateV0) bool {
	return stackShutdownGoalBackendActiveTimeoutV0(state)
}

func stackShutdownGoalBackendActiveTimeoutV0(state orquestagoal.GoalWorkStateV0) bool {
	status := strings.TrimSpace(state.Status)
	if status != orquestagoal.GoalStatusBlockedV0 && status != orquestagoal.GoalStatusInvalidV0 {
		return false
	}
	if state.LastResult != nil {
		if strings.Contains(strings.TrimSpace(state.LastResult.Summary), "codex_app_server_goal_active_timeout") {
			return true
		}
		for _, issue := range state.LastResult.Issues {
			if strings.Contains(strings.TrimSpace(issue.Code), "codex_app_server_goal_active_timeout") {
				return true
			}
		}
		for _, ref := range state.LastResult.EvidenceRefs {
			if stackShutdownEvidenceLooksGoalActiveTimeoutV0(ref) {
				return true
			}
		}
	}
	for _, ref := range state.EvidenceRefs {
		if stackShutdownEvidenceLooksGoalActiveTimeoutV0(ref) {
			return true
		}
	}
	return false
}

func stackShutdownGoalBackendEvidenceRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := append([]string(nil), state.EvidenceRefs...)
	if stackShutdownGoalBackendActiveTimeoutV0(state) {
		refs = append(refs, "evidence-ref-shutdown-goal-backend-active-timeout")
	}
	return compactStringsV0(refs)
}

type stackShutdownRunControlWriterV0 struct {
	Inner          orquestaruncontrol.RunControlWriterPortV0
	Reader         orquestaruncontrol.RunControlReaderPortV0
	Terminal       orquestaruncontrol.RunControlTerminalWriterPortV0
	GoalStateStore orquestagoal.GoalWorkStateStorePortV0
}

func (writer stackShutdownRunControlWriterV0) PauseRunV0(
	ctx context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if writer.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
	return writer.Inner.PauseRunV0(ctx, command)
}

func (writer stackShutdownRunControlWriterV0) ResumeRunV0(
	ctx context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if writer.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
	return writer.Inner.ResumeRunV0(ctx, command)
}

func (writer stackShutdownRunControlWriterV0) StopRunV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if writer.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
	if command.Forced {
		if completed, ok, err := writer.completeTerminalGoalRunControlV0(
			ctx,
			command.RunRef,
			orquestaruncontrol.RunControlStatusStoppedV0,
			command.RequestedBy,
			command.Reason,
			command.IdempotencyKey,
			command.EvidenceRefs,
		); ok || err != nil {
			return completed, err
		}
	}
	state, err := writer.Inner.StopRunV0(ctx, command)
	if err == nil && command.Forced {
		if completed, ok, completeErr := writer.completeTerminalGoalRunControlV0(
			ctx,
			command.RunRef,
			orquestaruncontrol.RunControlStatusStoppedV0,
			command.RequestedBy,
			command.Reason,
			command.IdempotencyKey,
			command.EvidenceRefs,
		); ok || completeErr != nil {
			return completed, completeErr
		}
		writer.reconcileForcedGoalStateV0(ctx, command)
	}
	return state, err
}

func (writer stackShutdownRunControlWriterV0) CancelRunV0(
	ctx context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if writer.Inner == nil {
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
	if command.Forced {
		if completed, ok, err := writer.completeTerminalGoalRunControlV0(
			ctx,
			command.RunRef,
			orquestaruncontrol.RunControlStatusCanceledV0,
			command.RequestedBy,
			command.Reason,
			command.IdempotencyKey,
			command.EvidenceRefs,
		); ok || err != nil {
			return completed, err
		}
	}
	state, err := writer.Inner.CancelRunV0(ctx, command)
	if err == nil && command.Forced {
		if completed, ok, completeErr := writer.completeTerminalGoalRunControlV0(
			ctx,
			command.RunRef,
			orquestaruncontrol.RunControlStatusCanceledV0,
			command.RequestedBy,
			command.Reason,
			command.IdempotencyKey,
			command.EvidenceRefs,
		); ok || completeErr != nil {
			return completed, completeErr
		}
		writer.reconcileForcedGoalStateV0(ctx, orquestaruncontrol.StopRunCommandV0{
			RunRef:       command.RunRef,
			RequestedBy:  command.RequestedBy,
			Reason:       command.Reason,
			Forced:       command.Forced,
			EvidenceRefs: command.EvidenceRefs,
		})
	}
	return state, err
}

func (writer stackShutdownRunControlWriterV0) completeTerminalGoalRunControlV0(
	ctx context.Context,
	runRef string,
	target orquestaruncontrol.RunControlStatusV0,
	requestedBy string,
	reason string,
	idempotencyKey string,
	evidenceRefs []string,
) (orquestaruncontrol.RunControlStateV0, bool, error) {
	if writer.Terminal == nil || writer.GoalStateStore == nil {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	state, err := writer.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestaruncontrol.RunControlStateV0{}, false, err
	}
	if !stackShutdownGoalReadyForRunControlTerminalV0(state) {
		return orquestaruncontrol.RunControlStateV0{}, false, nil
	}
	target, err = writer.targetForCurrentRunControlV0(ctx, runRef, target)
	if err != nil {
		return orquestaruncontrol.RunControlStateV0{}, true, err
	}
	completed, err := writer.Terminal.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       runRef,
		TargetStatus: target,
		RequestedBy:  firstNonEmptyQueuedSourceV0(requestedBy, "orquesta-server-shutdown"),
		Reason: firstNonEmptyQueuedSourceV0(
			reason,
			"server shutdown reconciled terminal goal-first run control",
		),
		IdempotencyKey: firstNonEmptyQueuedSourceV0(
			idempotencyKey,
			"idem-server-shutdown-goal-terminal-run-control-"+
				string(target)+"-"+safeStackShutdownRefPartV0(runRef),
		),
		EvidenceRefs: compactStringsV0(append(
			append(append([]string(nil), evidenceRefs...), state.EvidenceRefs...),
			serverShutdownGoalTerminalRunControlReconciledEvidenceV0,
		)),
	})
	return completed, true, err
}

func (writer stackShutdownRunControlWriterV0) targetForCurrentRunControlV0(
	ctx context.Context,
	runRef string,
	fallback orquestaruncontrol.RunControlStatusV0,
) (orquestaruncontrol.RunControlStatusV0, error) {
	if writer.Reader == nil {
		return fallback, nil
	}
	state, err := writer.Reader.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			return fallback, nil
		}
		return "", err
	}
	if target, ok := stackShutdownTerminalRunControlTargetV0(state.Status); ok {
		return target, nil
	}
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestaruncontrol.RunControlStatusCanceledV0, nil
	case orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestaruncontrol.RunControlStatusStoppedV0, nil
	default:
		return fallback, nil
	}
}

func (writer stackShutdownRunControlWriterV0) reconcileForcedGoalStateV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) {
	if writer.GoalStateStore == nil {
		return
	}
	runRef := strings.TrimSpace(command.RunRef)
	if runRef == "" {
		return
	}
	state, err := writer.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil || orquestagoal.GoalWorkResultTerminalV0(state.Status) {
		return
	}
	issueCode := stackShutdownForcedGoalIssueCodeV0(state)
	evidenceRefs := compactStringsV0(append(
		append([]string(nil), command.EvidenceRefs...),
		"evidence-ref-server-shutdown-goal-forced-terminal-reconciled",
	))
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         strings.TrimSpace(state.GoalRef),
		ExternalGoalRef: strings.TrimSpace(state.ExternalGoalRef),
		Summary:         "server shutdown forced stop reconciled goal-first state",
		ArtifactRefs:    stackShutdownForcedGoalArtifactRefsV0(state),
		EvidenceRefs:    evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  issueCode,
			Field: "server_shutdown",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  issueCode,
			Field: "server_shutdown",
		}},
	}
	state.EvidenceRefs = compactStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	_ = writer.GoalStateStore.SaveGoalWorkStateV0(ctx, state)
}

func stackShutdownForcedGoalIssueCodeV0(state orquestagoal.GoalWorkStateV0) string {
	artifactRefs := stackShutdownForcedGoalArtifactRefsV0(state)
	if len(artifactRefs) == 0 && len(stackShutdownForcedGoalDomainReceiptRefsV0(state)) == 0 {
		return "operator_forced_stop_no_artifacts"
	}
	return "operator_forced_stop_goal_first"
}

func stackShutdownForcedGoalArtifactRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastResult == nil {
		return []string{}
	}
	return compactStringsV0(state.LastResult.ArtifactRefs)
}

func stackShutdownForcedGoalDomainReceiptRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastResult == nil {
		return []string{}
	}
	return compactStringsV0(state.LastResult.DomainReceiptRefs)
}

func stackShutdownEvidenceLooksGoalActiveTimeoutV0(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.Contains(ref, "codex-app-server-goal-active-timeout") ||
		strings.Contains(ref, "codex_app_server_goal_active_timeout")
}

func (reader stackShutdownStatsReaderV0) ReadRunShutdownStatsV0(
	ctx context.Context,
	request orquestaservershutdown.RunShutdownStatsRequestV0,
) (orquestaservershutdown.RunShutdownStatsV0, error) {
	run, err := reader.Config.Stores.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return orquestaservershutdown.RunShutdownStatsV0{}, err
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		reader.Config.Stores.ProcessRegistry,
		statsProgressSourceV0(reader.Config),
		agentUsageSourceV0(reader.Config),
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{
			CorrelationID: request.CorrelationID,
			EvidenceRefs:  request.EvidenceRefs,
		},
	)
	liveness := buildStackShutdownAgentLivenessV0(ctx, reader.Config, stats)
	return orquestaservershutdown.RunShutdownStatsV0{
		RunRef:                  stats.RunRef,
		AgentsInFlight:          stats.Counts.AgentsInFlight,
		ProcessLivenessObserved: liveness.Observed,
		AgentsRunningLive:       len(liveness.LiveRefs),
		AgentsRunningStale:      len(liveness.StaleRefs),
		AgentsLost:              len(liveness.LostRefs),
		AgentsStopRequested:     stats.Counts.AgentsStopRequested,
		AgentsStopConfirmed:     stats.Counts.AgentsStopConfirmed,
		EvidenceRefs: compactStringsV0(
			append(request.EvidenceRefs, liveness.EvidenceRefs...),
		),
	}, nil
}
