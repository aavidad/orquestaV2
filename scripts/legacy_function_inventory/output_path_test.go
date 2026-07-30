// Estas pruebas impiden que una salida aparentemente externa termine
// físicamente dentro del repositorio por medio de enlaces simbólicos.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputRejectsSymlinkedParentEnteringRepository(t *testing.T) {
	repository := t.TempDir()
	outside := t.TempDir()
	alias := filepath.Join(outside, "entrada")
	if err := os.Symlink(repository, alias); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(alias, "generado", "inventario.jsonl")
	_, err := normalizedOutput(repository, output)
	if err == nil || !strings.Contains(err.Error(), "dentro del repositorio") {
		t.Fatalf("se esperaba rechazo de la ruta física interna; obtenido: %v", err)
	}
}

func TestOutputRejectsFinalSymlinkEvenWhenItPointsOutside(t *testing.T) {
	repository := t.TempDir()
	directory := t.TempDir()
	target := filepath.Join(directory, "real.jsonl")
	if err := os.WriteFile(target, []byte("anterior"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "alias.jsonl")
	if err := os.Symlink(target, output); err != nil {
		t.Fatal(err)
	}
	_, err := normalizedOutput(repository, output)
	if err == nil || !strings.Contains(err.Error(), "enlace simbólico") {
		t.Fatalf("se esperaba rechazo del enlace final; obtenido: %v", err)
	}
}

func TestOutputAliasesResolveToOnePhysicalDestination(t *testing.T) {
	repository := t.TempDir()
	destination := t.TempDir()
	alias := filepath.Join(t.TempDir(), "salida")
	if err := os.Symlink(destination, alias); err != nil {
		t.Fatal(err)
	}
	direct, err := normalizedOutput(repository, filepath.Join(destination, "inventario.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	indirect, err := normalizedOutput(repository, filepath.Join(alias, "inventario.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if direct != indirect {
		t.Fatalf("los alias físicos no convergen: directo=%s alias=%s", direct, indirect)
	}
}
