package orquestacli

import (
	"context"
	"flag"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func runCLIDoctorContratosV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(CliDefaultCommandOperationalV0)
	scope := fs.String("scope", orquestaobservability.OperationalStatusScopeProyectoV0, "scope")
	subjectRef := fs.String("subject-ref", "", "ref opaca")
	sections := fs.String("sections", orquestaobservability.OperationalStatusSectionEstadoV0, "secciones CSV")
	limit := fs.Int("limit", 20, "limite")
	addDiscardedCommonFlagsV0(fs)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(CliDefaultCommandOperationalV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(CliDefaultCommandOperationalV0, common)
	}
	inv := invocationFromCLIFlagsV0(CliDefaultCommandOperationalV0, common)
	query := orquestaobservability.OperationalStatusQueryV0{
		Scope:           *scope,
		SubjectRef:      *subjectRef,
		IncludeSections: splitCSVFlagV0(*sections),
		Limit:           *limit,
	}
	client, err := NewOperationalStatusCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorOperationalEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.ConsultarEstadoOperativo(ctx, inv, query)
	return env, exitCodeForEnvelopeV0(env)
}

func addDiscardedCommonFlagsV0(fs *flag.FlagSet) {
	fs.Bool("dry-run", false, "reservado por contrato")
}
