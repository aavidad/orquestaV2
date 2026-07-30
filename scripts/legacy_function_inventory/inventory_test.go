// Estas pruebas acreditan el contrato observable del censo completo:
// determinismo, cobertura, rechazo de entradas vacías y límite de procesos Git.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryIsDeterministicAndCoversHistoricalVariantsAndFailures(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")

	writeTestFile(t, repository, "main.go", "package sample\n\nfunc Stable() int { return 1 }\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "primera")
	first := strings.TrimSpace(gitTest(t, repository, "rev-parse", "HEAD"))
	gitTest(t, repository, "tag", "-a", "v1", "-m", "v1")

	writeTestFile(t, repository, "main.go", "package sample\n\ntype T struct{}\nfunc Stable() int { return 2 }\nfunc (T) Method() {}\n")
	gitTest(t, repository, "add", "main.go")
	gitTest(t, repository, "commit", "-qm", "segunda")
	gitTest(t, repository, "branch", "feature")
	gitTest(t, repository, "checkout", "-q", "--detach", first)
	writeTestFile(t, repository, "broken.go", "package sample\nfunc Broken(\n")
	gitTest(t, repository, "add", "broken.go")
	gitTest(t, repository, "commit", "-qm", "rota")
	gitTest(t, repository, "branch", "broken")
	gitTest(t, repository, "checkout", "-q", "feature")

	firstJSONL := filepath.Join(t.TempDir(), "first.jsonl")
	firstManifest := filepath.Join(t.TempDir(), "first.json")
	secondJSONL := filepath.Join(t.TempDir(), "second.jsonl")
	secondManifest := filepath.Join(t.TempDir(), "second.json")
	if err := run(options{repository: repository, jsonl: firstJSONL, manifest: firstManifest}); err != nil {
		t.Fatal(err)
	}
	if err := run(options{repository: repository, jsonl: secondJSONL, manifest: secondManifest}); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, firstJSONL, secondJSONL)
	assertSameFile(t, firstManifest, secondManifest)

	records := readTestRecords(t, firstJSONL)
	counts := map[string]int{}
	var methodSeen, failureSeen bool
	stableVariants := map[string]struct{}{}
	for _, item := range records {
		counts[item.RecordKind]++
		if item.RecordKind == "function_occurrence" && item.SymbolKind == "method" && item.Name == "Method" {
			methodSeen = true
		}
		if item.RecordKind == "parse_failure" && item.ErrorCode == "go_parse_failed" {
			failureSeen = true
		}
		if item.RecordKind == "function_variant" && item.Name == "Stable" {
			stableVariants[item.VariantRef] = struct{}{}
		}
	}
	if !methodSeen || !failureSeen || len(stableVariants) != 2 {
		t.Fatalf("cobertura incompleta: método=%v fallo=%v variantes Stable=%d", methodSeen, failureSeen, len(stableVariants))
	}
	if counts["ref"] < 3 || counts["commit_ref_reachability"] == 0 ||
		counts["commit"] != 3 || counts["tree"] == 0 || counts["go_blob"] < 3 {
		t.Fatalf("conteos inesperados: %#v", counts)
	}

	var manifestValue manifest
	content, err := os.ReadFile(firstManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &manifestValue); err != nil {
		t.Fatal(err)
	}
	if manifestValue.InventorySHA256 == "" || manifestValue.RefSnapshotSHA256 == "" ||
		manifestValue.Counts["function_variant"] < 3 {
		t.Fatalf("manifiesto incompleto: %#v", manifestValue)
	}
}

func TestInventoryRejectsRepositoryWithoutRefs(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "out.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err == nil {
		t.Fatal("se esperaba rechazo del repositorio sin confirmaciones")
	}
}

func TestInventoryUsesBoundedGitProcesses(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	for index := 0; index < 12; index++ {
		writeTestFile(t, repository, "main.go", "package sample\nfunc Value() int { return "+strconv.Itoa(index)+" }\n")
		gitTest(t, repository, "add", "main.go")
		gitTest(t, repository, "commit", "-qm", "cambio")
		if index%2 == 0 {
			gitTest(t, repository, "branch", "rama-"+strconv.Itoa(index))
		}
	}

	processes := 0
	err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "out.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifest.json"),
		gitProcessStarted: func() {
			processes++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if processes > 7 {
		t.Fatalf("el censo abrió %d procesos Git; se esperaban como máximo 7", processes)
	}
}
