package orquestacli

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	OrquestaCLIDefaultProgramNameV0 = "orquesta-cli"
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
	if runner.Stderr == nil {
		runner.Stderr = io.Discard
	}
	if len(args) == 0 || isCLIHelpArgV0(args[0]) {
		if _, err := io.WriteString(runner.Stdout, cliHelpTextForArgsV0(args)+"\n"); err != nil {
			writeCLIStdioFailureV0(runner.Stderr, "help", "stdout", "help_text", err)
			return 1
		}
		return 0
	}
	env, code := dispatchOrquestaCLIV0(ctx, args, runner)
	if err := writeCLIEnvelopeV0(runner.Stdout, env); err != nil {
		writeCLIStdioFailureV0(runner.Stderr, cliStdioCommandNameV0(args), "stdout", "json_encode", err)
		return 1
	}
	return code
}

func dispatchOrquestaCLIV0(ctx context.Context, args []string, runner OrquestaCLIRunnerV0) (CliOutputEnvelopeV0, int) {
	if entry, rest := findCLICommandCatalogEntryV0(args); entry != nil {
		return entry.Handler(ctx, rest, runner)
	}
	{
		inv := NormalizeCliInvocationContextV0(CliInvocationContextV0{Command: strings.Join(args, " ")})
		env := NewCliOutputErrorEnvelopeV0(inv, "orquesta-cli", "v0", []CliPublicErrorV0{
			NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "command", cliUnsupportedCommandDetailV0(args)),
		}, CliOutputMetaV0{})
		return env, 2
	}
}

func writeCLIEnvelopeV0(out io.Writer, env CliOutputEnvelopeV0) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(env)
}

func writeCLIStdioFailureV0(stderr io.Writer, command, stream, stage string, err error) {
	if stderr == nil {
		return
	}
	failure := orquestaobservability.NewCommandStdioWriteFailureV0(command, stream, stage, err)
	_ = json.NewEncoder(stderr).Encode(failure)
}

func cliStdioCommandNameV0(args []string) string {
	if len(args) == 0 {
		return "orquesta-cli"
	}
	first := strings.TrimSpace(args[0])
	if first == "" || strings.HasPrefix(first, "-") {
		return "orquesta-cli"
	}
	return first
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
