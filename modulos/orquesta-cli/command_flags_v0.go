package orquestacli

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CliDefaultJSONInputMaxBytesV0 = int64(1 << 20)
	CliMaxJSONInputMaxBytesV0     = int64(8 << 20)
)

type cliCommonFlagsV0 struct {
	ServerURL      string
	InputPath      string
	InputMaxBytes  int64
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
	fs.Int64Var(&common.InputMaxBytes, "input-max-bytes", CliDefaultJSONInputMaxBytesV0, "limite bytes JSON de entrada")
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

func readCLIJSONInputWithLimitV0(runner OrquestaCLIRunnerV0, inputPath string, maxBytes int64, out any) error {
	raw, err := readCLIInputBytesV0(runner, inputPath, maxBytes)
	if err != nil {
		return err
	}
	return decodeCLIJSONInputBytesV0(raw, out)
}

func decodeCLIJSONInputBytesV0(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "json_entrada_invalido", 0, false)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "json_entrada_multiple", 0, false)
	}
	return nil
}

func readCLIInputBytesV0(runner OrquestaCLIRunnerV0, inputPath string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 || maxBytes > CliMaxJSONInputMaxBytesV0 {
		return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "input_limit_invalido", 0, false)
	}
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" || inputPath == "-" {
		if runner.Stdin == nil {
			return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "stdin_requerido", 0, false)
		}
		return readLimitedCLIInputV0(runner.Stdin, maxBytes, "stdin_no_legible")
	}
	if err := validateCLIInputFilePathV0(inputPath); err != nil {
		return nil, err
	}
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "archivo_no_legible", 0, false)
	}
	defer file.Close()
	return readLimitedCLIInputV0(file, maxBytes, "archivo_no_legible")
}

func readLimitedCLIInputV0(reader io.Reader, maxBytes int64, readErrorDetail string) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", readErrorDetail, 0, false)
	}
	if int64(len(raw)) > maxBytes {
		return nil, NewCliClientErrorV0(CliErrInputTooLargeV0, "input", "input_too_large", 0, false)
	}
	return raw, nil
}

func validateCLIInputFilePathV0(inputPath string) error {
	cleaned := filepath.Clean(strings.TrimSpace(inputPath))
	slashed := strings.ToLower(filepath.ToSlash(cleaned))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "input_file_not_allowed", 0, false)
	}
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "~") {
		return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "input_file_requires_relative_path", 0, false)
	}
	for _, part := range strings.Split(slashed, "/") {
		if isSensitiveCLIInputPathPartV0(part) {
			return NewCliClientErrorV0(CliErrOpcionInvalidaV0, "input", "input_file_not_allowed", 0, false)
		}
	}
	return nil
}

func isSensitiveCLIInputPathPartV0(part string) bool {
	switch part {
	case ".orquesta-runtime", "prompt", "prompts", "transcript", "transcripts", "log", "logs",
		"agent_packet.json", "agent_ack.json", "director_decisions.json",
		"orquesta_shutdown_request.json", "agent_shutdown_checkpoint_ack.json",
		"codex_last_message.txt":
		return true
	default:
		return strings.HasSuffix(part, ".log")
	}
}

func cliInputSourceFromPathV0(inputPath string) string {
	if strings.TrimSpace(inputPath) == "" || strings.TrimSpace(inputPath) == "-" {
		return CliInputSourceStdinV0
	}
	return CliInputSourceFileExplicitV0
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
