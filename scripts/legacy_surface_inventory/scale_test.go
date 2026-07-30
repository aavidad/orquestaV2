// Esta prueba mide que los hechos crecen por objetos únicos y no por el
// producto entre confirmaciones y superficies.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestInventoryGrowthUsesUniqueTreeAndBlobObjects(t *testing.T) {
	repository := newTestRepository(t)
	const surfaceCount = 30
	for index := 0; index < surfaceCount; index++ {
		writeTestBytes(
			t,
			repository,
			"surface-"+strconv.Itoa(index)+".txt",
			[]byte("contenido "+strconv.Itoa(index)+"\n"),
		)
	}
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "árbol compartido")
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
	if result.Counts["confirmacion"] != emptyCommits+1 ||
		result.Counts["objeto_arbol"] != 1 ||
		result.Counts["entrada_arbol"] != surfaceCount ||
		result.Counts["hechos_blob"] != surfaceCount ||
		result.Counts["aparicion_superficie"] != 0 {
		t.Fatalf("el inventario no creció por objetos únicos: %#v", result.Counts)
	}
	expanded := result.Counts["confirmacion"] * surfaceCount
	normalized := result.Counts["entrada_arbol"] + result.Counts["hechos_blob"]
	if normalized*10 >= expanded {
		t.Fatalf("medición insuficiente: grafo=%d confirmación×superficie=%d", normalized, expanded)
	}
}

func TestClassificationMemoizesSharedTreeAtSamePath(t *testing.T) {
	sharedBlob := "blob-compartido"
	cache := map[string][]treeEntry{
		"raiz-uno": {{
			mode: "40000", oid: "arbol-compartido", name: []byte("compartido"),
		}},
		"raiz-dos": {{
			mode: "40000", oid: "arbol-compartido", name: []byte("compartido"),
		}},
		"arbol-compartido": {{
			mode: "100644", oid: sharedBlob, name: []byte("api.go"),
		}},
	}
	aggregates := map[string]*blobAggregate{
		sharedBlob: newBlobAggregate(blobFacts{
			size: 12, encoding: "utf-8", prefix: []byte("package api\n"),
		}),
	}
	contexts := classifyObjectGraph([]string{"raiz-uno", "raiz-dos"}, cache, aggregates)
	if contexts != 3 {
		t.Fatalf("contextos clasificados=%d; esperados=3 por memoización", contexts)
	}
	if actual := familyNames(record{Families: sortedFamilies(aggregates[sharedBlob].families)}); !containsString(actual, "http_mcp_web") {
		t.Fatalf("el subárbol compartido no se clasificó: %q", actual)
	}
}
