package orquestacli

import (
	"context"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func runCLIGovernanceCatalogoListarV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandGovernanceV0)
	module := fs.String("module", "", "modulo")
	role := fs.String("role", "", "rol")
	phase := fs.String("phase", "", "fase")
	tags := fs.String("tags", "", "tags CSV")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandGovernanceV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandGovernanceV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandGovernanceV0, common)
	client, err := NewGovernanceCatalogCliReaderV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorGovernanceEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarCatalogo(ctx, inv, orquestagovernance.GovernanceCatalogQueryV0{
		Module: *module,
		Role:   *role,
		Phase:  *phase,
		Tags:   splitCSVFlagV0(*tags),
	})
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIGovernanceCatalogoVerV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandGovernanceViewV0)
	module := fs.String("module", "", "modulo")
	role := fs.String("role", "", "rol")
	phase := fs.String("phase", "", "fase")
	tags := fs.String("tags", "", "tags CSV")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandGovernanceViewV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandGovernanceViewV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandGovernanceViewV0, common)
	client, err := NewGovernanceCatalogCliReaderV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorGovernanceEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarCatalogo(ctx, inv, orquestagovernance.GovernanceCatalogQueryV0{
		Module: *module,
		Role:   *role,
		Phase:  *phase,
		Tags:   splitCSVFlagV0(*tags),
	})
	return env, exitCodeForEnvelopeV0(env)
}
