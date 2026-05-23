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

func runCLIAutoprogrammingStatusV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogStatusV0)
	runRef := fs.String("run-ref", "", "ref opaca del run")
	queueRef := fs.String("queue-ref", "", "ref opaca de cola")
	appRefs := fs.String("app-refs", "", "refs de app CSV")
	limit := fs.Int("limit", 20, "limite")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogStatusV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogStatusV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogStatusV0, common)
	input := orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RunRef:               strings.TrimSpace(*runRef),
		QueueRef:             strings.TrimSpace(*queueRef),
		AppRefs:              splitCSVFlagV0(*appRefs),
		QueueLimit:           *limit,
		IncludeProcessRefs:   true,
		IncludeAgentProgress: true,
		IncludeAgentUsage:    true,
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarEstado(ctx, inv, input)
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

func runCLIAutoprogrammingRunControlV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogRunControlV0)
	action := fs.String("action", "", "pause|resume|stop|cancel")
	runRef := fs.String("run-ref", "", "ref opaca del run")
	appRef := fs.String("app-ref", "", "ref opaca de app")
	externalJobRef := fs.String("external-job-ref", "", "ref opaca de job externo")
	requestedBy := fs.String("requested-by", "operator", "actor solicitante")
	reason := fs.String("reason", "", "motivo publico")
	forced := fs.Bool("forced", false, "forzar si contrato lo permite")
	evidenceRefs := fs.String("evidence-refs", "", "refs de evidencia CSV")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogRunControlV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogRunControlV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogRunControlV0, common)
	input := orquestamcp.MCPRunControlToolInputV0{
		Action:         strings.TrimSpace(*action),
		RunRef:         strings.TrimSpace(*runRef),
		AppRef:         strings.TrimSpace(*appRef),
		ExternalJobRef: strings.TrimSpace(*externalJobRef),
		RequestedBy:    strings.TrimSpace(*requestedBy),
		Reason:         strings.TrimSpace(*reason),
		Forced:         *forced,
		IdempotencyKey: strings.TrimSpace(common.IdempotencyKey),
		EvidenceRefs:   splitCSVFlagV0(*evidenceRefs),
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ControlarRun(ctx, inv, input)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIAutoprogrammingSuperviseV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandAutoprogSuperviseV0)
	runRef := fs.String("run-ref", "", "ref opaca del run")
	queueRef := fs.String("queue-ref", "", "ref opaca de cola")
	maxTicks := fs.Int("max-ticks", 1, "ticks maximos")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandAutoprogSuperviseV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandAutoprogSuperviseV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandAutoprogSuperviseV0, common)
	input := orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:   strings.TrimSpace(*runRef),
		QueueRef: strings.TrimSpace(*queueRef),
		MaxTicks: *maxTicks,
	}
	client, err := NewAutoprogrammingCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := autoprogClientErrorEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.Supervisar(ctx, inv, input)
	return env, exitCodeForEnvelopeV0(env)
}
