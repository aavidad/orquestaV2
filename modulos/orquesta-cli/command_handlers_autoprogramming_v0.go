package orquestacli

import (
	"context"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func runCLIServerStatusV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandServerStatusV0)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandServerStatusV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandServerStatusV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandServerStatusV0, common)
	client, err := NewServerStatusCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := serverStatusClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarEstadoServidor(ctx, inv)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIAutoprogrammingPrepareV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogPrepareV0)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogPrepareV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogPrepareV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogPrepareV0, common)
	var input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0
	if err := readCLIJSONInputV0(runner, common.InputPath, &input); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogPrepareV0, err)
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(runner))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.PrepararRun(ctx, inv, input)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIAutoprogrammingQueueV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogQueueV0)
	queueRef := fs.String("queue-ref", "", "ref opaca de cola")
	appRefs := fs.String("app-refs", "", "refs de app CSV")
	limit := fs.Int("limit", 20, "limite")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogQueueV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogQueueV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogQueueV0, common)
	input := orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: strings.TrimSpace(*queueRef),
		AppRefs:  splitCSVFlagV0(*appRefs),
		Limit:    *limit,
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarCola(ctx, inv, input)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIAutoprogrammingRunV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogRunV0)
	runRef := fs.String("run-ref", "", "ref opaca del run")
	progress := fs.Bool("progress", false, "incluir progreso de agentes")
	processRefs := fs.Bool("process-refs", true, "incluir refs de proceso")
	usage := fs.Bool("usage", true, "incluir uso de agentes")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogRunV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogRunV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogRunV0, common)
	if strings.TrimSpace(*runRef) == "" {
		env := autoprogSingleErrorEnvelopeV0(inv, CliErrOpcionInvalidaV0, "run_ref", "run_ref_requerido", 0, false, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, 2
	}
	input := orquestamcp.MCPDirectorStatsToolInputV0{
		RunRef:               strings.TrimSpace(*runRef),
		IncludeProcessRefs:   *processRefs,
		IncludeAgentProgress: *progress,
		IncludeAgentUsage:    *usage,
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarRun(ctx, inv, input)
	return env, exitCodeForEnvelopeV0(env)
}
