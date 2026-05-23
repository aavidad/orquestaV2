package orquestacli

import (
	"context"

	orquestacore "orquesta/modulos/orquesta-core"
)

func runCLIFunctionContractListarV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(FunctionContractCliDefaultListCommandV0)
	module := fs.String("module", "", "modulo")
	estado := fs.String("estado", "", "estado")
	archivo := fs.String("archivo-objetivo", "", "archivo objetivo")
	simbolo := fs.String("simbolo-objetivo", "", "simbolo objetivo")
	limit := fs.Int("limit", 50, "limite")
	cursor := fs.String("cursor", "", "cursor")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(FunctionContractCliDefaultListCommandV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(FunctionContractCliDefaultListCommandV0, common)
	}
	inv := invocationFromCLIFlagsV0(FunctionContractCliDefaultListCommandV0, common)
	client, err := NewFunctionContractCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorFunctionContractEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	req := ListarFunctionContractsRequestV0{
		Filtros: FunctionContractFiltrosV0{
			Modulo:          *module,
			Estado:          *estado,
			ArchivoObjetivo: *archivo,
			SimboloObjetivo: *simbolo,
		},
		Page: FunctionContractPageRequestV0{Limit: *limit, Cursor: *cursor},
	}
	env := client.ListarFunctionContracts(ctx, inv, req)
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIFunctionContractVerV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(FunctionContractCliDefaultViewCommandV0)
	ref := fs.String("ref", "", "function_contract_ref")
	version := fs.Int("version", 0, "version")
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(FunctionContractCliDefaultViewCommandV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(FunctionContractCliDefaultViewCommandV0, common)
	}
	inv := invocationFromCLIFlagsV0(FunctionContractCliDefaultViewCommandV0, common)
	if *ref == "" {
		env := NewCliOutputErrorEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, []CliPublicErrorV0{
			NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "ref", "ref_requerida"),
		}, CliOutputMetaV0{})
		return env, 2
	}
	client, err := NewFunctionContractCliClientV0(common.ServerURL, common.Timeout)
	if err != nil {
		env := clientErrorFunctionContractEnvelopeV0(inv, err, runnerNowV0(OrquestaCLIRunnerV0{}))
		return env, exitCodeForEnvelopeV0(env)
	}
	env := client.VerFunctionContract(ctx, inv, VerFunctionContractRequestV0{
		FunctionContractRef: *ref,
		Version:             *version,
	})
	return env, exitCodeForEnvelopeV0(env)
}

func runCLIFunctionContractRegistrarV0(ctx context.Context, args []string) (CliOutputEnvelopeV0, int) {
	fs, common := newCLIFlagSetV0(FunctionContractCliDefaultRegCommandV0)
	if err := fs.Parse(args); err != nil {
		return cliParseErrorEnvelopeV0(FunctionContractCliDefaultRegCommandV0, err)
	}
	if fs.NArg() != 0 {
		return cliUnexpectedArgsEnvelopeV0(FunctionContractCliDefaultRegCommandV0, common)
	}
	inv := invocationFromCLIFlagsV0(FunctionContractCliDefaultRegCommandV0, common)
	var client *FunctionContractCliClientV0
	env := client.RegistrarFunctionContract(ctx, inv, orquestacore.FunctionContractV0{})
	return env, exitCodeForEnvelopeV0(env)
}
