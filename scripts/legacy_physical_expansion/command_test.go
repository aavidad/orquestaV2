// Estas pruebas comprueban la orden finita y la ausencia de capacidades laterales.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type prefixFailWriter struct {
	bytes.Buffer
	limit int
}

func (writer *prefixFailWriter) Write(value []byte) (int, error) {
	if len(value) > writer.limit {
		value = value[:writer.limit]
	}
	count, _ := writer.Buffer.Write(value)
	return count, errors.New("fallo de salida simulado")
}

func TestCommandReservesStdoutForSuccessfulJSON(t *testing.T) {
	var output, diagnostics bytes.Buffer
	if code := runCommand([]string{"--v3", sourceFixture}, &output, &diagnostics); code != 0 {
		t.Fatalf("código=%d diagnóstico=%s", code, diagnostics.String())
	}
	if output.Len() == 0 || diagnostics.Len() != 0 {
		t.Fatal("el éxito mezcló JSON y diagnóstico")
	}
	for _, arguments := range [][]string{nil, {"--v3"}, {"--otro", sourceFixture}, {"--v3", sourceFixture, "extra"}} {
		output.Reset()
		diagnostics.Reset()
		if code := runCommand(arguments, &output, &diagnostics); code == 0 ||
			output.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("error mezclado con stdout: argumentos=%v código=%d", arguments, code)
		}
	}
	output.Reset()
	diagnostics.Reset()
	if code := runCommand([]string{"--help"}, &output, &diagnostics); code != 0 ||
		output.Len() != 0 || !strings.Contains(diagnostics.String(), "stderr") {
		t.Fatal("la ayuda no quedó separada del canal JSON")
	}
}

func TestPartialOutputNeverCountsAsSuccess(t *testing.T) {
	output := &prefixFailWriter{limit: 37}
	var diagnostics bytes.Buffer
	code := runCommand([]string{"--v3", sourceFixture}, output, &diagnostics)
	if code == 0 || output.Len() != output.limit || diagnostics.Len() == 0 {
		t.Fatalf("prefijo aceptado: código=%d bytes=%d diagnóstico=%q", code, output.Len(), diagnostics.String())
	}
	if json.Valid(output.Bytes()) {
		t.Fatal("el prefijo parcial simulado parece un resultado JSON aceptable")
	}
}

func TestFIFOIsRejectedWithoutBlocking(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "entrada.fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := readRegularInput(fifo, maxSourceBytes)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("un FIFO fue aceptado como entrada")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("la apertura del FIFO quedó bloqueada")
	}
}

func TestFinalSymlinkIsRejectedBeforeReading(t *testing.T) {
	target, err := filepath.Abs(sourceFixture)
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "v3-enlace.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularInput(link, maxSourceBytes); err == nil {
		t.Fatal("el enlace simbólico final fue aceptado")
	}
}

func TestImplementationHasNoEnvironmentNetworkProcessesOrWrites(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, forbidden := range []string{
			"os.Getenv", "os.LookupEnv", "os.Environ", `"os/exec"`, `"net`,
			"exec.Command", "os.Create", "os.WriteFile", "os.Mkdir", "os.Remove",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contiene la capacidad lateral %q", file, forbidden)
			}
		}
	}
}
