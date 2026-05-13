package orquestaappcodexstack

import (
	"context"
	"strings"

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
	return orquestaservershutdown.RunShutdownStatsV0{
		RunRef:              stats.RunRef,
		AgentsInFlight:      stats.Counts.AgentsInFlight,
		AgentsStopRequested: stats.Counts.AgentsStopRequested,
		AgentsStopConfirmed: stats.Counts.AgentsStopConfirmed,
		EvidenceRefs:        compactStringsV0(request.EvidenceRefs),
	}, nil
}

type stackShutdownCheckpointPreparerV0 struct {
	Config ConfigV0
}

func (preparer stackShutdownCheckpointPreparerV0) PrepareAgentShutdownV0(
	ctx context.Context,
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
) (orquestaservershutdown.PrepareAgentShutdownResultV0, error) {
	stats, err := stackShutdownStatsReaderV0{Config: preparer.Config}.ReadRunShutdownStatsV0(
		ctx,
		orquestaservershutdown.RunShutdownStatsRequestV0{
			RunRef:        command.RunRef,
			CorrelationID: command.CorrelationID,
			EvidenceRefs:  command.EvidenceRefs,
		},
	)
	if err != nil {
		return orquestaservershutdown.PrepareAgentShutdownResultV0{}, err
	}
	if stats.AgentsInFlight > 0 {
		return orquestaservershutdown.PrepareAgentShutdownResultV0{
			RunRef:             strings.TrimSpace(command.RunRef),
			CheckpointRecorded: false,
			EvidenceRefs:       compactStringsV0(command.EvidenceRefs),
		}, nil
	}
	checkpointRef := "checkpoint-ref-shutdown-" + safeStackShutdownRefPartV0(command.RunRef)
	return orquestaservershutdown.PrepareAgentShutdownResultV0{
		RunRef:             strings.TrimSpace(command.RunRef),
		CheckpointRecorded: true,
		CheckpointRef:      checkpointRef,
		EvidenceRefs:       compactStringsV0(append(command.EvidenceRefs, checkpointRef)),
	}, nil
}

func safeStackShutdownRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
