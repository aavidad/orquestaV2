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
	if reader.Config.Stores.AppGoalStateStore == nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	lister, ok := reader.Config.Stores.AppGoalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		ActiveOnly: true,
		MaxItems:   request.MaxItems,
	})
	if err != nil {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, err
	}
	out := orquestaservershutdown.ActiveShutdownWorkResultV0{}
	for _, state := range states {
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
