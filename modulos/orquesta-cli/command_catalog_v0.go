package orquestacli

import (
	"context"
	"strings"
)

const (
	CliCommandStateActiveV0 = "active"
	CliCommandStateLegacyV0 = "legacy"
	CliCommandStateHiddenV0 = "hidden"

	CliCommandEffectReadV0       = "read"
	CliCommandEffectWriteV0      = "write"
	CliCommandEffectQuarantineV0 = "quarantine"
)

type cliCommandHandlerV0 func(context.Context, []string, OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int)

type cliCommandCatalogEntryV0 struct {
	Path          []string
	FlagsES       []string
	FlagsEN       []string
	ClientTarget  string
	EffectProfile string
	HelpES        string
	HelpEN        string
	State         string
	Handler       cliCommandHandlerV0
}

var cliCommandCatalogV0 = []cliCommandCatalogEntryV0{
	{[]string{"app", "spec", "solicitar"}, []string{"--server-url URL", "--input request.json", "--json"}, []string{"--server-url URL", "--input request.json", "--json"}, "SolicitarNuevaApp REST", CliCommandEffectWriteV0, "solicita AppSpec v0 al servidor", "requests AppSpec v0 from server", CliCommandStateActiveV0, runCLIAppSpecSolicitarV0},
	{[]string{"app", "spec", "bootstrap"}, []string{"--input command.json", "--json"}, []string{"--input command.json", "--json"}, "AppDirector legacy quarantine", CliCommandEffectQuarantineV0, "legacy en cuarentena; use /api/v0/apps/director", "legacy quarantined; use /api/v0/apps/director", CliCommandStateLegacyV0, runCLIAppSpecBootstrapV0},
	{[]string{"servidor", "estado"}, []string{"--server-url URL", "--json"}, []string{"--server-url URL", "--json"}, "ServerStatus REST", CliCommandEffectReadV0, "consulta estado publico del servidor", "queries public server status", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIServerStatusV0)},
	{[]string{"autoprogramacion", "preparar"}, []string{"--server-url URL", "--input request.json", "--json"}, []string{"--server-url URL", "--input request.json", "--json"}, "Autoprogramming prepare-run REST", CliCommandEffectWriteV0, "prepara run de autoprogramacion", "prepares autoprogramming run", CliCommandStateActiveV0, runCLIAutoprogrammingPrepareV0},
	{[]string{"autoprogramacion", "estado", "ver"}, []string{"--server-url URL", "--run-ref RUN_REF", "--json"}, []string{"--server-url URL", "--run-ref RUN_REF", "--json"}, "Autoprogramming status REST", CliCommandEffectReadV0, "consulta estado de autoprogramacion", "queries autoprogramming status", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIAutoprogrammingStatusV0)},
	{[]string{"autoprogramacion", "supervisar"}, []string{"--server-url URL", "--run-ref RUN_REF", "--max-ticks 1", "--json"}, []string{"--server-url URL", "--run-ref RUN_REF", "--max-ticks 1", "--json"}, "Run supervisor REST", CliCommandEffectWriteV0, "supervisa ticks acotados", "supervises bounded ticks", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIAutoprogrammingSuperviseV0)},
	{[]string{"autoprogramacion", "cola", "listar"}, []string{"--server-url URL", "--limit 20", "--json"}, []string{"--server-url URL", "--limit 20", "--json"}, "Run queue priority REST", CliCommandEffectReadV0, "lista cola priorizada", "lists ranked queue", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIAutoprogrammingQueueV0)},
	{[]string{"autoprogramacion", "run", "ver"}, []string{"--server-url URL", "--run-ref RUN_REF", "--json"}, []string{"--server-url URL", "--run-ref RUN_REF", "--json"}, "Director stats REST", CliCommandEffectReadV0, "consulta run", "queries run", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIAutoprogrammingRunV0)},
	{[]string{"autoprogramacion", "run", "controlar"}, []string{"--server-url URL", "--run-ref RUN_REF", "--action pause", "--json"}, []string{"--server-url URL", "--run-ref RUN_REF", "--action pause", "--json"}, "Run control REST", CliCommandEffectWriteV0, "controla run por puerto publico", "controls run through public port", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIAutoprogrammingRunControlV0)},
	{[]string{"doctor", "contratos"}, []string{"--server-url URL", "--scope proyecto", "--json"}, []string{"--server-url URL", "--scope project", "--json"}, "OperationalStatusQuery REST", CliCommandEffectReadV0, "consulta diagnostico contractual", "queries contract diagnostics", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIDoctorContratosV0)},
	{[]string{"contratos", "funcion", "listar"}, []string{"--server-url URL", "--module MODULO", "--json"}, []string{"--server-url URL", "--module MODULE", "--json"}, "FunctionContract REST", CliCommandEffectReadV0, "lista contratos de funcion", "lists function contracts", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIFunctionContractListarV0)},
	{[]string{"contratos", "funcion", "ver"}, []string{"--server-url URL", "--ref FUNCTION_CONTRACT_REF", "--json"}, []string{"--server-url URL", "--ref FUNCTION_CONTRACT_REF", "--json"}, "FunctionContract REST", CliCommandEffectReadV0, "muestra contrato de funcion", "shows function contract", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIFunctionContractVerV0)},
	{[]string{"contratos", "funcion", "registrar"}, []string{"--json"}, []string{"--json"}, "FunctionContract blocked mutation", CliCommandEffectQuarantineV0, "mutacion bloqueada hasta contrato durable", "mutation blocked until durable contract", CliCommandStateLegacyV0, wrapCLIHandlerNoRunnerV0(runCLIFunctionContractRegistrarV0)},
	{[]string{"gobernanza", "catalogo", "listar"}, []string{"--server-url URL", "--module MODULO", "--json"}, []string{"--server-url URL", "--module MODULE", "--json"}, "GovernanceCatalog REST", CliCommandEffectReadV0, "lista catalogo de gobernanza", "lists governance catalog", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIGovernanceCatalogoListarV0)},
	{[]string{"gobernanza", "catalogo", "ver"}, []string{"--server-url URL", "--module MODULO", "--json"}, []string{"--server-url URL", "--module MODULE", "--json"}, "GovernanceCatalog REST", CliCommandEffectReadV0, "muestra catalogo de gobernanza", "shows governance catalog", CliCommandStateActiveV0, wrapCLIHandlerNoRunnerV0(runCLIGovernanceCatalogoVerV0)},
}

func wrapCLIHandlerNoRunnerV0(fn func(context.Context, []string) (CliOutputEnvelopeV0, int)) cliCommandHandlerV0 {
	return func(ctx context.Context, args []string, _ OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
		return fn(ctx, args)
	}
}

func findCLICommandCatalogEntryV0(args []string) (*cliCommandCatalogEntryV0, []string) {
	for idx := range cliCommandCatalogV0 {
		entry := &cliCommandCatalogV0[idx]
		if entry.State == CliCommandStateHiddenV0 {
			continue
		}
		if hasCLIPathV0(args, entry.Path...) {
			return entry, args[len(entry.Path):]
		}
	}
	return nil, nil
}

func cliHelpTextForArgsV0(args []string) string {
	if cliHelpLocaleFromArgsV0(args) == "en" {
		return cliHelpTextFromCatalogV0("en")
	}
	return cliHelpTextFromCatalogV0("es")
}

func cliHelpTextFromCatalogV0(locale string) string {
	var b strings.Builder
	if locale == "en" {
		b.WriteString("Usage: orquesta-cli <command> [options]\n\nCommands:\n")
	} else {
		b.WriteString("Uso: orquesta-cli <comando> [opciones]\n\nComandos:\n")
	}
	for _, entry := range cliCommandCatalogV0 {
		if entry.State == CliCommandStateHiddenV0 {
			continue
		}
		b.WriteString("  ")
		b.WriteString(cliCommandUsageV0(entry, locale))
		b.WriteString("\n")
	}
	if locale == "en" {
		b.WriteString("\nCommon options:\n")
		b.WriteString("  --server-url URL, --request-id ID, --correlation-id ID, --timeout 10s, --locale es|en, --json")
	} else {
		b.WriteString("\nOpciones comunes:\n")
		b.WriteString("  --server-url URL, --request-id ID, --correlation-id ID, --timeout 10s, --locale es, --json")
	}
	return b.String()
}

func cliCommandUsageV0(entry cliCommandCatalogEntryV0, locale string) string {
	flags := entry.FlagsES
	help := entry.HelpES
	if locale == "en" {
		flags = entry.FlagsEN
		help = entry.HelpEN
	}
	parts := append([]string{}, entry.Path...)
	parts = append(parts, flags...)
	usage := strings.Join(parts, " ")
	if entry.State == CliCommandStateLegacyV0 || entry.EffectProfile == CliCommandEffectQuarantineV0 {
		return usage + "  (" + help + ")"
	}
	return usage
}

func cliHelpLocaleFromArgsV0(args []string) string {
	for index, arg := range args {
		trimmed := strings.TrimSpace(arg)
		switch trimmed {
		case "--locale":
			if index+1 < len(args) && strings.HasPrefix(strings.ToLower(strings.TrimSpace(args[index+1])), "en") {
				return "en"
			}
		default:
			if strings.HasPrefix(trimmed, "--locale=") &&
				strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "--locale="))), "en") {
				return "en"
			}
		}
	}
	return "es"
}

func cliUnsupportedCommandDetailV0(args []string) string {
	path := cliNormalizeUnknownPathV0(args)
	suggestions := cliCommandSuggestionsV0(path)
	if len(suggestions) == 0 {
		return "path=" + path + ";sugerencias="
	}
	return "path=" + path + ";sugerencias=" + strings.Join(suggestions, "|")
}

func cliNormalizeUnknownPathV0(args []string) string {
	var parts []string
	for _, arg := range args {
		token := strings.ToLower(strings.TrimSpace(arg))
		if token == "" || strings.HasPrefix(token, "-") {
			continue
		}
		parts = append(parts, cliSafeCommandTokenV0(token))
		if len(parts) >= 4 {
			break
		}
	}
	if len(parts) == 0 {
		return "<empty>"
	}
	return strings.Join(parts, " ")
}

func cliSafeCommandTokenV0(token string) string {
	for _, marker := range []string{"://", "/", "\\", "=", "@", "token", "secret", "password", "passwd", "key"} {
		if strings.Contains(token, marker) {
			return "<redacted>"
		}
	}
	if len(token) > 48 {
		return token[:48]
	}
	return token
}

func cliCommandSuggestionsV0(path string) []string {
	if path == "" || path == "<empty>" || strings.Contains(path, "<redacted>") {
		return nil
	}
	first := strings.Fields(path)
	if len(first) == 0 {
		return nil
	}
	var out []string
	for _, entry := range cliCommandCatalogV0 {
		if entry.State == CliCommandStateHiddenV0 || len(entry.Path) == 0 {
			continue
		}
		if entry.Path[0] == first[0] || strings.HasPrefix(entry.Path[0], first[0]) || strings.HasPrefix(first[0], entry.Path[0]) {
			out = append(out, strings.Join(entry.Path, " "))
			if len(out) == 3 {
				return out
			}
		}
	}
	return out
}
