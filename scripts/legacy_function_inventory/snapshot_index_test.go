// Estos contratos impiden que el censo estructural acepte deriva, rutas excluidas o cuerpos.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSnapshotFunctionIndexIsCompleteDeterministicAndBodyFree(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	writeLookupCensusContract(t, repository,
		"modulos/orquesta-sample", "modulos/orquesta-copy")
	writeTestFile(t, repository, "modulos/orquesta-sample/worker.go",
		"package sample\ntype Worker struct{}\nfunc Stable() int { return 1 }\nfunc (Worker) Run() error { return nil }\n")
	writeTestFile(t, repository, "modulos/orquesta-copy/helper.go",
		"package copy\nfunc Copy() string { return \"secret-body\" }\n")
	writeTestFile(t, repository, "modulos/orquesta-copy/helper_test.go",
		"package copy\nfunc TestCopy() {}\n")
	gitTest(t, repository, "add", ".")

	contract, err := loadLegacyGoCensusContract(repository)
	if err != nil {
		t.Fatal(err)
	}
	contract.Baseline.ProductionFileCount = 2
	encoded, err := json.Marshal(contract)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repository, legacyGoCensusPath, string(encoded)+"\n")
	gitTest(t, repository, "add", legacyGoCensusPath)

	output := t.TempDir()
	firstJSONL := filepath.Join(output, "first.jsonl")
	firstManifest := filepath.Join(output, "first.json")
	if err := writeSnapshotFunctionIndex(repository, firstJSONL, firstManifest); err != nil {
		t.Fatal(err)
	}
	secondJSONL := filepath.Join(output, "second.jsonl")
	secondManifest := filepath.Join(output, "second.json")
	if err := writeSnapshotFunctionIndex(repository, secondJSONL, secondManifest); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(firstJSONL)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondJSONL)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("el índice no es determinista")
	}
	for _, forbidden := range []string{"canonical_source", "secret-body", "return nil"} {
		if strings.Contains(string(first), forbidden) {
			t.Fatalf("el índice publicó fuente o cuerpo: %s", forbidden)
		}
	}
	lines := bytes.Split(bytes.TrimSuffix(first, []byte{'\n'}), []byte{'\n'})
	if len(lines) != 3 {
		t.Fatalf("declaraciones=%d, want 3", len(lines))
	}
	var manifest snapshotIndexManifest
	rawManifest, err := os.ReadFile(firstManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeUniqueStrictJSON(rawManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ProductionFileCount != 2 || manifest.FunctionCount != 2 ||
		manifest.MethodCount != 1 || manifest.DeclarationCount != 3 ||
		manifest.UniqueOccurrenceCount != 3 || manifest.ContainsBodies ||
		manifest.ClaimsAccreditation {
		t.Fatalf("manifiesto inesperado: %#v", manifest)
	}
}

func TestSnapshotFunctionIndexRejectsSourceOverridesAndCountDrift(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	writeLookupCensusContract(t, repository, "modulos/orquesta-sample")
	path := "modulos/orquesta-sample/worker.go"
	writeTestFile(t, repository, path, "package sample\nfunc Stable() {}\n")
	gitTest(t, repository, "add", ".")
	output := t.TempDir()
	if err := writeSnapshotFunctionIndex(repository,
		filepath.Join(output, "count.jsonl"), filepath.Join(output, "count.json")); err == nil ||
		!strings.Contains(err.Error(), "baseline") {
		t.Fatalf("count drift aceptado: %v", err)
	}
	writeTestFile(t, repository, path, "package sample\nfunc Changed() {}\n")
	if err := writeSnapshotFunctionIndex(repository,
		filepath.Join(output, "override.jsonl"), filepath.Join(output, "override.json")); err == nil ||
		!strings.Contains(err.Error(), "overrides") {
		t.Fatalf("override aceptado: %v", err)
	}
}
