// Este fichero prueba propiedades de publicación que comparten los escenarios.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishingPairRestoresPreviousManifestWhenSecondRenameFails(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "docs/estado.md", []byte("anterior\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "estado anterior")

	directory := t.TempDir()
	jsonl := filepath.Join(directory, "inventario.jsonl")
	manifestPath := filepath.Join(directory, "inventario.manifest.json")
	if err := run(options{repository: repository, jsonl: jsonl, manifest: manifestPath}); err != nil {
		t.Fatal(err)
	}
	previousJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	previousManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	writeTestBytes(t, repository, "docs/estado.md", []byte("nuevo\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "estado nuevo")

	renameCalls := 0
	syntheticFailure := errors.New("fallo sintético del segundo renombrado")
	err = run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   manifestPath,
		renameFile: func(source, destination string) error {
			renameCalls++
			if renameCalls == 2 {
				return syntheticFailure
			}
			return os.Rename(source, destination)
		},
	})
	if !errors.Is(err, syntheticFailure) {
		t.Fatalf("se esperaba el fallo sintético; obtenido: %v", err)
	}
	if renameCalls != 2 {
		t.Fatalf("renombrados de publicación=%d; esperados=2", renameCalls)
	}
	currentJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	currentManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(currentJSONL) != string(previousJSONL) ||
		string(currentManifest) != string(previousManifest) {
		t.Fatal("el fallo dejó una pareja distinta de la publicación anterior")
	}
	var restored manifest
	readJSONFile(t, manifestPath, &restored)
	if restored.InventorySHA256 != inventoryDigest(currentJSONL) {
		t.Fatalf("el manifiesto restaurado no valida el JSONL anterior: %#v", restored)
	}
}

func TestAbruptStopLeavesDetectableMismatchAndRetryRecovers(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "docs/estado.md", []byte("anterior\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "estado anterior")

	directory := t.TempDir()
	jsonl := filepath.Join(directory, "inventario.jsonl")
	manifestPath := filepath.Join(directory, "inventario.manifest.json")
	if err := run(options{repository: repository, jsonl: jsonl, manifest: manifestPath}); err != nil {
		t.Fatal(err)
	}
	previousJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}

	writeTestBytes(t, repository, "docs/estado.md", []byte("nuevo\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "estado nuevo")
	simulatedStop := errors.New("caída abrupta simulada")
	err = run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   manifestPath,
		interruptAfterManifest: func() error {
			return simulatedStop
		},
	})
	if !errors.Is(err, simulatedStop) {
		t.Fatalf("se esperaba la caída simulada; obtenido: %v", err)
	}
	currentJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	if string(currentJSONL) != string(previousJSONL) {
		t.Fatal("la interrupción sustituyó el JSONL antes del segundo renombrado")
	}
	var interrupted manifest
	readJSONFile(t, manifestPath, &interrupted)
	if interrupted.InventorySHA256 == inventoryDigest(currentJSONL) {
		t.Fatal("la pareja interrumpida parecía válida aunque mezclaba dos publicaciones")
	}

	if err := run(options{repository: repository, jsonl: jsonl, manifest: manifestPath}); err != nil {
		t.Fatalf("la repetición no recuperó la pareja: %v", err)
	}
	recoveredJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	var recovered manifest
	readJSONFile(t, manifestPath, &recovered)
	if recovered.InventorySHA256 != inventoryDigest(recoveredJSONL) {
		t.Fatalf("la pareja recuperada sigue siendo inválida: %#v", recovered)
	}
	if string(recoveredJSONL) == string(previousJSONL) {
		t.Fatal("la recuperación no publicó el estado nuevo")
	}
}

func assertMode(t *testing.T, filePath string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if actual := info.Mode().Perm(); actual != expected {
		t.Fatalf("%s tiene modo %o; esperado %o", filePath, actual, expected)
	}
}
