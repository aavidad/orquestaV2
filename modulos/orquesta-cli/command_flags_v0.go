package orquestacli

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"strings"
	"time"
)

type cliCommonFlagsV0 struct {
	ServerURL      string
	InputPath      string
	RequestID      string
	CorrelationID  string
	IdempotencyKey string
	Timeout        time.Duration
	Locale         string
	JSON           bool
}

func newCLIFlagSetV0(name string) (*flag.FlagSet, *cliCommonFlagsV0) {
	common := &cliCommonFlagsV0{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&common.ServerURL, "server-url", "", "endpoint publico")
	fs.StringVar(&common.InputPath, "input", "-", "JSON de entrada o -")
	fs.StringVar(&common.RequestID, "request-id", "", "request id")
	fs.StringVar(&common.CorrelationID, "correlation-id", "", "correlation id")
	fs.StringVar(&common.IdempotencyKey, "idempotency-key", "", "clave idempotente")
	fs.DurationVar(&common.Timeout, "timeout", CliDefaultTimeoutRESTV0, "timeout")
	fs.StringVar(&common.Locale, "locale", "es", "locale")
	fs.BoolVar(&common.JSON, "json", false, "salida JSON")
	return fs, common
}

func invocationFromCLIFlagsV0(command string, common *cliCommonFlagsV0) CliInvocationContextV0 {
	if common == nil {
		common = &cliCommonFlagsV0{}
	}
	return NormalizeCliInvocationContextV0(CliInvocationContextV0{
		Command:        command,
		RequestID:      common.RequestID,
		CorrelationID:  common.CorrelationID,
		IdempotencyKey: common.IdempotencyKey,
		ServerURL:      common.ServerURL,
		Timeout:        common.Timeout,
		OutputFormat:   CliOutputFormatJSONV0,
		Locale:         common.Locale,
		InputSource:    cliInputSourceFromPathV0(common.InputPath),
	})
}

func readCLIJSONInputV0(runner OrquestaCLIRunnerV0, inputPath string, out any) error {
	raw, err := readCLIInputBytesV0(runner, inputPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "json_entrada_invalido", 0, false)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "json_entrada_multiple", 0, false)
	}
	return nil
}

func readCLIInputBytesV0(runner OrquestaCLIRunnerV0, inputPath string) ([]byte, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" || inputPath == "-" {
		if runner.Stdin == nil {
			return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "stdin_requerido", 0, false)
		}
		raw, err := io.ReadAll(runner.Stdin)
		if err != nil {
			return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "stdin_no_legible", 0, false)
		}
		return raw, nil
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "archivo_no_legible", 0, false)
	}
	return raw, nil
}

func cliInputSourceFromPathV0(inputPath string) string {
	if strings.TrimSpace(inputPath) == "" || strings.TrimSpace(inputPath) == "-" {
		return CliInputSourceStdinV0
	}
	return CliInputSourceFileV0
}

func cliParseErrorEnvelopeV0(command string, err error) (CliOutputEnvelopeV0, int) {
	inv := NormalizeCliInvocationContextV0(CliInvocationContextV0{Command: command, OutputFormat: CliOutputFormatJSONV0})
	cliErr, ok := err.(CliClientErrorV0)
	if !ok {
		cliErr = NewCliClientErrorV0(CliErrOpcionInvalidaV0, "args", "opciones_invalidas", 0, false)
	}
	return NewCliOutputErrorEnvelopeV0(inv, "orquesta-cli", "v0", []CliPublicErrorV0{
		NewCliPublicErrorV0(cliErr.Code, cliErr.Field, cliErr.Detail),
	}, CliOutputMetaV0{}), 2
}

func cliUnexpectedArgsEnvelopeV0(command string, common *cliCommonFlagsV0) (CliOutputEnvelopeV0, int) {
	inv := invocationFromCLIFlagsV0(command, common)
	return NewCliOutputErrorEnvelopeV0(inv, "orquesta-cli", "v0", []CliPublicErrorV0{
		NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "args", "argumentos_posicionales_no_soportados"),
	}, CliOutputMetaV0{}), 2
}

func splitCSVFlagV0(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func runnerNowV0(runner OrquestaCLIRunnerV0) time.Time {
	if runner.Now != nil {
		return runner.Now()
	}
	return time.Now()
}
