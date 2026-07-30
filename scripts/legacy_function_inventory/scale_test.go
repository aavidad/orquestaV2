// Esta prueba mide que las funciones crecen por blobs y variantes únicos, no
// por el producto entre confirmaciones y declaraciones.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryGrowthUsesUniqueObjectsInsteadOfCommitFunctionProduct(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	gitTest(t, repository, "config", "user.name", "Inventario")
	gitTest(t, repository, "config", "user.email", "inventario@example.invalid")
	var source strings.Builder
	source.WriteString("package sample\n")
	const functionCount = 50
	for index := 0; index < functionCount; index++ {
		source.WriteString("func Function")
		source.WriteString(strconv.Itoa(index))
		source.WriteString("() {}\n")
	}
	writeTestFile(t, repository, "stable.go", source.String())
	gitTest(t, repository, "add", "stable.go")
	gitTest(t, repository, "commit", "-qm", "contenido único")
	const emptyCommits = 40
	for index := 0; index < emptyCommits; index++ {
		gitTest(t, repository, "commit", "--allow-empty", "-qm", "confirmación sin cambio")
	}

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "inventory.jsonl"),
		manifest:   manifestPath,
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var result manifest
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatal(err)
	}
	if result.Counts["commit"] != emptyCommits+1 ||
		result.Counts["tree_object"] != 1 ||
		result.Counts["tree_entry"] != 1 ||
		result.Counts["go_blob"] != 1 ||
		result.Counts["go_declaration"] != functionCount ||
		result.Counts["function_variant"] != functionCount {
		t.Fatalf("el grafo no quedó normalizado por objetos únicos: %#v", result.Counts)
	}
	expandedOccurrences := result.Counts["commit"] * functionCount
	functionGraphRecords := result.Counts["go_declaration"] + result.Counts["function_variant"]
	if functionGraphRecords*10 >= expandedOccurrences {
		t.Fatalf(
			"medición insuficiente: grafo=%d expansión confirmación×función=%d",
			functionGraphRecords,
			expandedOccurrences,
		)
	}
}
