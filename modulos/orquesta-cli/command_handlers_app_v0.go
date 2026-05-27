package orquestacli

import (
	"context"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func runCLIAppSpecSolicitarV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandSolicitarAppV0)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandSolicitarAppV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandSolicitarAppV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandSolicitarAppV0, common)

	var req orquestafactory.AppSpecRequestV0
	if err := readCLIJSONInputWithLimitV0(runner, common.InputPath, common.InputMaxBytes, &req); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandSolicitarAppV0, err)
	}
	client, err := NewSolicitarNuevaAppCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorEnvelopeV0(inv, err, runnerNowV0(runner))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.SolicitarNuevaApp(ctx, inv, req)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIAppSpecBootstrapV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(BootstrapAppSpecCliDefaultCommandV0)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(BootstrapAppSpecCliDefaultCommandV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(BootstrapAppSpecCliDefaultCommandV0, common)
	}
	inv := invocationFromCLIFlagsV0(BootstrapAppSpecCliDefaultCommandV0, common)

	client, err := NewBootstrapAppSpecCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorBootstrapAppSpecEnvelopeV0(inv, err, runnerNowV0(runner))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.BootstrapProyectoDesdeAppSpec(ctx, inv, bootstrapAppSpecQuarantinedCommandV0())
	return env, exitCodeForEnvelopeV0(env)
}
