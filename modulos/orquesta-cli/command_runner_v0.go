package orquestacli

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"
)

const (
	OrquestaCLIDefaultProgramNameV0 = "orquesta-cli"
	OrquestaCLIHelpTextESV0         = `Uso: orquesta-cli <comando> [opciones]

Comandos:
  app spec solicitar --server-url URL --input request.json --json
  app spec bootstrap --input command.json --json  (legacy en cuarentena; use /api/v0/apps/director)
  servidor estado --server-url URL --json
  autoprogramacion preparar --server-url URL --input request.json --json
  autoprogramacion estado ver --server-url URL --run-ref RUN_REF --json
  autoprogramacion supervisar --server-url URL --run-ref RUN_REF --max-ticks 1 --json
  autoprogramacion cola listar --server-url URL --limit 20 --json
  autoprogramacion run ver --server-url URL --run-ref RUN_REF --json
  autoprogramacion run controlar --server-url URL --run-ref RUN_REF --action pause --json
  doctor contratos --server-url URL --scope proyecto --json
  contratos funcion listar --server-url URL --module MODULO --json
  contratos funcion ver --server-url URL --ref FUNCTION_CONTRACT_REF --json
  contratos funcion registrar --json
  gobernanza catalogo listar --server-url URL --module MODULO --json
  gobernanza catalogo ver --server-url URL --module MODULO --json

Opciones comunes:
  --server-url URL, --request-id ID, --correlation-id ID, --timeout 10s, --locale es, --json`
	OrquestaCLIHelpTextENV0 = `Usage: orquesta-cli <command> [options]

Commands:
  app spec solicitar --server-url URL --input request.json --json
  app spec bootstrap --input command.json --json  (legacy quarantined; use /api/v0/apps/director)
  servidor estado --server-url URL --json
  autoprogramacion preparar --server-url URL --input request.json --json
  autoprogramacion estado ver --server-url URL --run-ref RUN_REF --json
  autoprogramacion supervisar --server-url URL --run-ref RUN_REF --max-ticks 1 --json
  autoprogramacion cola listar --server-url URL --limit 20 --json
  autoprogramacion run ver --server-url URL --run-ref RUN_REF --json
  autoprogramacion run controlar --server-url URL --run-ref RUN_REF --action pause --json
  doctor contratos --server-url URL --scope proyecto --json
  contratos funcion listar --server-url URL --module MODULE --json
  contratos funcion ver --server-url URL --ref FUNCTION_CONTRACT_REF --json
  contratos funcion registrar --json
  gobernanza catalogo listar --server-url URL --module MODULE --json
  gobernanza catalogo ver --server-url URL --module MODULE --json

Common options:
  --server-url URL, --request-id ID, --correlation-id ID, --timeout 10s, --locale es|en, --json`
)

type OrquestaCLIRunnerV0 struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Now    func() time.Time
}

func RunOrquestaCLIV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) int {
	if runner.Stdout == nil {
		runner.Stdout = io.Discard
	}
	if len(args) == 0 || isCLIHelpArgV0(args[0]) {
		_, _ = io.WriteString(runner.Stdout, cliHelpTextForArgsV0(args)+"\n")
		return 0
	}
	env, code := dispatchOrquestaCLIV0(ctx, args, runner)
	writeCLIEnvelopeV0(runner.Stdout, env)
	return code
}

func dispatchOrquestaCLIV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
	switch {
	case hasCLIPathV0(args, "app", "spec", "solicitar"):
		return runCLIAppSpecSolicitarV0(ctx, args[3:], runner)
	case hasCLIPathV0(args, "app", "spec", "bootstrap"):
		return runCLIAppSpecBootstrapV0(ctx, args[3:], runner)
	case hasCLIPathV0(args, "servidor", "estado"):
		return runCLIServerStatusV0(ctx, args[2:])
	case hasCLIPathV0(args, "autoprogramacion", "preparar"):
		return runCLIAutoprogrammingPrepareV0(ctx, args[2:], runner)
	case hasCLIPathV0(args, "autoprogramacion", "estado", "ver"):
		return runCLIAutoprogrammingStatusV0(ctx, args[3:])
	case hasCLIPathV0(args, "autoprogramacion", "supervisar"):
		return runCLIAutoprogrammingSuperviseV0(ctx, args[2:])
	case hasCLIPathV0(args, "autoprogramacion", "cola", "listar"):
		return runCLIAutoprogrammingQueueV0(ctx, args[3:])
	case hasCLIPathV0(args, "autoprogramacion", "run", "ver"):
		return runCLIAutoprogrammingRunV0(ctx, args[3:])
	case hasCLIPathV0(args, "autoprogramacion", "run", "controlar"):
		return runCLIAutoprogrammingRunControlV0(ctx, args[3:])
	case hasCLIPathV0(args, "doctor", "contratos"):
		return runCLIDoctorContratosV0(ctx, args[2:])
	case hasCLIPathV0(args, "contratos", "funcion", "listar"):
		return runCLIFunctionContractListarV0(ctx, args[3:])
	case hasCLIPathV0(args, "contratos", "funcion", "ver"):
		return runCLIFunctionContractVerV0(ctx, args[3:])
	case hasCLIPathV0(args, "contratos", "funcion", "registrar"):
		return runCLIFunctionContractRegistrarV0(ctx, args[3:])
	case hasCLIPathV0(args, "gobernanza", "catalogo", "listar"):
		return runCLIGovernanceCatalogoListarV0(ctx, args[3:])
	case hasCLIPathV0(args, "gobernanza", "catalogo", "ver"):
		return runCLIGovernanceCatalogoVerV0(ctx, args[3:])
	default:
		inv := NormalizeCliInvocationContextV0(CliInvocationContextV0{Command: strings.Join(args, " ")})
		env := NewCliOutputErrorEnvelopeV0(inv, "orquesta-cli", "v0", []CliPublicErrorV0{
			NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "command", "comando_no_soportado"),
		}, CliOutputMetaV0{})
		return env, 2
	}
}

func cliHelpTextForArgsV0(args []string) string {
	for index, arg := range args {
		trimmed := strings.TrimSpace(arg)
		switch trimmed {
		case "--locale":
			if index+1 < len(args) && strings.HasPrefix(strings.ToLower(strings.TrimSpace(args[index+1])), "en") {
				return OrquestaCLIHelpTextENV0
			}
		default:
			if strings.HasPrefix(trimmed, "--locale=") &&
				strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "--locale="))), "en") {
				return OrquestaCLIHelpTextENV0
			}
		}
	}
	return OrquestaCLIHelpTextESV0
}

func writeCLIEnvelopeV0(out io.Writer, env CliOutputEnvelopeV0) {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(env)
}

func exitCodeForEnvelopeV0(env CliOutputEnvelopeV0) int {
	if env.OK {
		return 0
	}
	return 1
}

func hasCLIPathV0(args []string, path ...string) bool {
	if len(args) < len(path) {
		return false
	}
	for idx, want := range path {
		if strings.TrimSpace(args[idx]) != want {
			return false
		}
	}
	return true
}

func isCLIHelpArgV0(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "-h", "--help", "help", "ayuda":
		return true
	default:
		return false
	}
}
