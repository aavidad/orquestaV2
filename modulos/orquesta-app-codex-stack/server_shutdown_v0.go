package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func serverShutdownExecutorV0(
	config ConfigV0,
	stack *StackV0,
) orquestamcp.MCPServerShutdownToolExecutorV0 {
	return orquestamcp.NewMCPServerShutdownToolExecutorV0(
		orquestaservershutdown.ServerShutdownDepsV0{
			QueueReader:         config.Stores.RunQueue,
			RunControlReader:    config.Stores.RunControl,
			RunControlWriter:    config.Stores.RunControl,
			RunCheckpointWriter: config.Stores.RunControl,
			CheckpointPreparer:  stackShutdownCheckpointPreparerV0{Config: config},
			Supervisor:          stackShutdownSupervisorV0{Stack: stack},
			StatsReader:         stackShutdownStatsReaderV0{Config: config},
			ActiveWorkReader:    stackShutdownActiveWorkReaderV0{Config: config},
		},
	)
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

func stackShutdownActiveWorkReadersFromGoalBackendV0(
	config ConfigV0,
) []orquestaservershutdown.ActiveShutdownWorkReaderPortV0 {
	out := []orquestaservershutdown.ActiveShutdownWorkReaderPortV0{}
	for _, candidate := range []interface{}{
		config.AppGoalObserver,
		config.AppGoalLauncher,
		config.AppGoalReworkLauncher,
	} {
		activeReader, ok := candidate.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0)
		if !ok || activeReader == nil {
			continue
		}
		out = append(out, activeReader)
	}
	return out
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
	if stackShutdownGoalBackendRunningStateV0(state) {
		return true
	}
	return stackShutdownGoalBackendActiveTimeoutV0(state)
}

func stackShutdownGoalBackendRunningStateV0(state orquestagoal.GoalWorkStateV0) bool {
	return strings.TrimSpace(state.Status) == orquestagoal.GoalStatusRunningV0
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
	if stackShutdownGoalBackendRunningStateV0(state) {
		refs = append(refs, "evidence-ref-shutdown-goal-backend-running-state")
	}
	if stackShutdownGoalBackendActiveTimeoutV0(state) {
		refs = append(refs, "evidence-ref-shutdown-goal-backend-active-timeout")
	}
	return compactStringsV0(refs)
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
