// Estas pruebas fijan canales, aperturas privadas y fallos no bloqueantes.
package main

import (
	"bytes"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type failingWriter struct {
	bytes.Buffer
	limit int
}

type shortNilWriter struct {
	bytes.Buffer
	limit int
}

func (writer *shortNilWriter) Write(value []byte) (int, error) {
	count, _ := writer.Buffer.Write(value[:writer.limit])
	return count, nil
}

func (writer *failingWriter) Write(value []byte) (int, error) {
	if len(value) > writer.limit {
		value = value[:writer.limit]
	}
	count, _ := writer.Buffer.Write(value)
	return count, errors.New("fallo simulado")
}

func TestCommandKeepsStdoutForCompleteVerdict(t *testing.T) {
	candidatePath := writePrivateCandidate(t, encodeCandidate(t, validCandidate(t)))
	arguments := []string{
		"--v3", v3Fixture, "--universe", universeFixture, "--candidate", candidatePath,
	}
	var output, diagnostics bytes.Buffer
	if code := runCommand(arguments, &output, &diagnostics); code != 0 ||
		output.Len() == 0 || diagnostics.Len() != 0 {
		t.Fatalf("código=%d salida=%d diagnóstico=%q", code, output.Len(), diagnostics.String())
	}
	output.Reset()
	diagnostics.Reset()
	if code := runCommand([]string{"--candidate", candidatePath}, &output, &diagnostics); code == 0 || output.Len() != 0 || diagnostics.Len() == 0 ||
		strings.Contains(diagnostics.String(), candidatePath) {
		t.Fatal("el error mezcló resultado o filtró la entrada")
	}
	output.Reset()
	diagnostics.Reset()
	if code := runCommand([]string{"--help"}, &output, &diagnostics); code != 0 ||
		output.Len() != 0 || !strings.Contains(diagnostics.String(), "stderr") {
		t.Fatal("la ayuda no usa únicamente stderr")
	}
}

func TestPartialStdoutIsNeverSuccess(t *testing.T) {
	candidatePath := writePrivateCandidate(t, encodeCandidate(t, validCandidate(t)))
	output := &failingWriter{limit: 31}
	var diagnostics bytes.Buffer
	code := runCommand([]string{
		"--v3", v3Fixture, "--universe", universeFixture, "--candidate", candidatePath,
	}, output, &diagnostics)
	if code == 0 || output.Len() != output.limit || diagnostics.Len() == 0 {
		t.Fatalf("prefijo aceptado: código=%d bytes=%d", code, output.Len())
	}
}

func TestShortWriteWithoutErrorIsNeverSuccess(t *testing.T) {
	candidatePath := writePrivateCandidate(t, encodeCandidate(t, validCandidate(t)))
	output := &shortNilWriter{limit: 29}
	var diagnostics bytes.Buffer
	code := runCommand([]string{
		"--v3", v3Fixture, "--universe", universeFixture, "--candidate", candidatePath,
	}, output, &diagnostics)
	if code == 0 || output.Len() != output.limit || diagnostics.Len() == 0 {
		t.Fatalf("escritura corta aceptada: código=%d bytes=%d", code, output.Len())
	}
}

func TestSpanishCatalogHasFallback(t *testing.T) {
	if catalogText("help") == "" || catalogText("clave_desconocida") != spanishCatalog["fallback"] {
		t.Fatal("el catálogo no conserva ayuda y fallback castellano")
	}
}

func TestFIFOAndDirectoryAreRejectedWithoutBlocking(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "entrada.fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := readRegularInput(fifo, maxCandidateBytes, true)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("se aceptó FIFO")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("la apertura FIFO se bloqueó")
	}
	if _, err := readRegularInput(t.TempDir(), maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó directorio")
	}
	socket := filepath.Join(t.TempDir(), "entrada.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if _, err := readRegularInput(socket, maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó socket")
	}
}

func TestSymlinkComponentsAndPublicCandidateAreRejected(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "candidato.json")
	if err := os.WriteFile(target, encodeCandidate(t, validCandidate(t)), 0o600); err != nil {
		t.Fatal(err)
	}
	finalLink := filepath.Join(directory, "final.json")
	if err := os.Symlink(target, finalLink); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularInput(finalLink, maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó enlace final")
	}
	parentLink := filepath.Join(t.TempDir(), "padre")
	if err := os.Symlink(directory, parentLink); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularInput(filepath.Join(parentLink, "candidato.json"), maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó componente simbólico")
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularInput(target, maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó candidato no privado")
	}
}

func TestInputSizeNPlusOneIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grande.json")
	if err := os.WriteFile(path, bytes.Repeat([]byte{'x'}, maxCandidateBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularInput(path, maxCandidateBytes, true); err == nil {
		t.Fatal("se aceptó entrada N+1")
	}
}
