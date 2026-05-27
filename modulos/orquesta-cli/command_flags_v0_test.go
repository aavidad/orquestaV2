package orquestacli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCLIInputBytesV0AcotaStdinSinVolcarContenido(t *testing.T) {
	secret := "valor-local-no-debe-aparecer"
	_, err := readCLIInputBytesV0(OrquestaCLIRunnerV0{
		Stdin: strings.NewReader(secret),
	}, "-", 8)
	if err == nil {
		t.Fatal("esperaba input_too_large")
	}
	var cliErr CliClientErrorV0
	if !errors.As(err, &cliErr) {
		t.Fatalf("error no CLI: %T", err)
	}
	if cliErr.Code != CliErrInputTooLargeV0 || strings.Contains(cliErr.Detail, secret) {
		t.Fatalf("error inesperado: %+v", cliErr)
	}
}

func TestReadCLIInputBytesV0BloqueaRutasSensiblesAntesDeLeer(t *testing.T) {
	cases := []string{
		filepath.Join(string(filepath.Separator), "tmp", "request.json"),
		filepath.Join(".orquesta-runtime", "agent_ack.json"),
		filepath.Join("logs", "request.json"),
		"codex_last_message.txt",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			_, err := readCLIInputBytesV0(OrquestaCLIRunnerV0{}, path, CliDefaultJSONInputMaxBytesV0)
			if err == nil {
				t.Fatal("esperaba bloqueo de ruta")
			}
			var cliErr CliClientErrorV0
			if !errors.As(err, &cliErr) {
				t.Fatalf("error no CLI: %T", err)
			}
			if cliErr.Code != CliErrOpcionInvalidaV0 || strings.Contains(cliErr.Detail, path) {
				t.Fatalf("error publico filtra ruta o codigo inesperado: %+v", cliErr)
			}
		})
	}
}

func TestCLIInputSourceFromPathV0ClasificaFuenteExplicita(t *testing.T) {
	if got := cliInputSourceFromPathV0("-"); got != CliInputSourceStdinV0 {
		t.Fatalf("stdin source=%q", got)
	}
	if got := cliInputSourceFromPathV0("request.json"); got != CliInputSourceFileExplicitV0 {
		t.Fatalf("file source=%q", got)
	}
	errs := ValidateCliInvocationContextV0(CliInvocationContextV0{InputSource: CliInputSourceInlineV0, OutputFormat: CliOutputFormatJSONV0})
	if len(errs) != 0 {
		t.Fatalf("inline source rechazado: %+v", errs)
	}
}
