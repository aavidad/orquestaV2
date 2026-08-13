// Estos contratos prueban que la consulta AST conserva identidad sin publicar cuerpos legacy.
package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLookupIndexSymbolsReusesV4IdentityWithoutSourceBodies(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	source := "package sample\n\ntype Worker struct{}\nfunc Stable() int { return 1 }\nfunc (Worker) Run(value int) error { return nil }\n"
	writeTestFile(t, repository, "modulos/orquesta-sample/worker.go", source)
	writeTestFile(t, repository, "modulos/orquesta-copy/worker.go", source)
	writeLookupCensusContract(t, repository,
		"modulos/orquesta-sample", "modulos/orquesta-copy")
	gitTest(t, repository, "add", ".")

	first, err := lookupIndexSymbols(repository, "modulos/orquesta-sample/worker.go", "Run")
	if err != nil {
		t.Fatal(err)
	}
	second, err := lookupIndexSymbols(repository, "modulos/orquesta-copy/worker.go", "Run")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("consultas inesperadas: first=%#v second=%#v", first, second)
	}
	if first[0].DeclarationRef != second[0].DeclarationRef ||
		first[0].VariantRef != second[0].VariantRef ||
		first[0].OccurrenceRef == second[0].OccurrenceRef {
		t.Fatalf("identidades de declaración/aparición incorrectas: first=%#v second=%#v", first[0], second[0])
	}
	if first[0].SymbolKind != "method" || first[0].Receiver != "Worker" ||
		first[0].Signature != "func (Worker) Run(value int) error" ||
		first[0].BodySHA == "" {
		t.Fatalf("metadata incompleta: %#v", first[0])
	}
	encoded, err := json.Marshal(first[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"canonical_source", "return nil", source} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("la consulta publicó cuerpo o fuente: %s", encoded)
		}
	}
}

func TestLookupIndexSymbolsRejectsOverridesTestsAndMissingSymbols(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	path := "modulos/orquesta-sample/worker.go"
	writeTestFile(t, repository, path, "package sample\nfunc Stable() {}\n")
	writeTestFile(t, repository, "modulos/orquesta-sample/worker_test.go", "package sample\nfunc Helper() {}\n")
	writeLookupCensusContract(t, repository, "modulos/orquesta-sample")
	gitTest(t, repository, "add", ".")

	if _, err := lookupIndexSymbols(repository, "modulos/orquesta-sample/worker_test.go", "Helper"); err == nil {
		t.Fatal("se aceptó un fichero test")
	}
	if _, err := lookupIndexSymbols(repository, path, "Missing"); err == nil {
		t.Fatal("se aceptó un símbolo ausente")
	}
	writeTestFile(t, repository, path, "package sample\nfunc Changed() {}\n")
	if _, err := lookupIndexSymbols(repository, path, "Stable"); err == nil ||
		!strings.Contains(err.Error(), "overrides") {
		t.Fatalf("override no rechazado: %v", err)
	}
}

func TestLookupIndexSymbolsRejectsPartialGoParseAndDuplicateCensusKey(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	path := "modulos/orquesta-sample/worker.go"
	writeTestFile(t, repository, path, "package sample\nfunc Stable() {}\nfunc Broken( {\n")
	writeLookupCensusContract(t, repository, "modulos/orquesta-sample")
	gitTest(t, repository, "add", ".")
	if _, err := lookupIndexSymbols(repository, path, "Stable"); err == nil ||
		!strings.Contains(err.Error(), "no parseable") {
		t.Fatalf("parseo parcial aceptado: %v", err)
	}

	var decoded map[string]any
	if err := decodeUniqueStrictJSON([]byte(`{"schema_version":1,"schema_version":2}`), &decoded); err == nil ||
		!strings.Contains(err.Error(), "duplicada") {
		t.Fatalf("clave duplicada aceptada: %v", err)
	}
}

func TestExactIndexBlobOIDRejectsAmbiguousStageAndWrongPath(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("100644 0123456789012345678901234567890123456789 1\tmodulos/orquesta-x/a.go\x00"),
		[]byte("100644 0123456789012345678901234567890123456789 0\tmodulos/orquesta-x/b.go\x00"),
	} {
		if _, err := exactIndexBlobOID(raw, "modulos/orquesta-x/a.go"); err == nil {
			t.Fatalf("entrada ambigua aceptada: %q", raw)
		}
	}
}

func writeLookupCensusContract(t *testing.T, repository string, modulePaths ...string) {
	t.Helper()
	contract := legacyGoCensusContract{
		DocumentKind:  "legacy_go_census",
		SchemaVersion: 1,
		SourceRoot:    "modulos",
		ModuleRules: []legacyGoModuleRule{{
			ID: "test", Paths: modulePaths, Disposition: "characterize",
			CapabilityIDs: []string{"TEST-01"}, Reason: "fixture",
			CharacterizationRequired: true,
		}},
	}
	contract.SourcePolicy.Include = "all_descendant_go_files_of_exact_module_paths"
	contract.SourcePolicy.ExcludeTestSuffix = "_test.go"
	contract.SourcePolicy.SymbolKinds = []string{"const", "var", "type", "func", "method"}
	contract.Baseline.CensusSHA256 = "sha256:test"
	encoded, err := json.Marshal(contract)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repository, legacyGoCensusPath, string(encoded)+"\n")
}
