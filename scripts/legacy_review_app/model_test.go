// Este fichero acredita que la revisión solo carga inventarios semánticos,
// privados y acotados; nunca intenta representar el censo físico completo.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryRejectsPhysicalRecordsWithoutSemanticIdentity(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "inventario-fisico.jsonl")
	content := `{"record_kind":"tree_entry","commit_id":"abc","path":"internal/a.go"}` + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInventory(filePath); err == nil ||
		!strings.Contains(err.Error(), "identificador semántico estable") {
		t.Fatalf("el censo físico fue aceptado como inventario semántico: %v", err)
	}
}

func TestInventoryRejectsUnsafeOrOversizedFiles(t *testing.T) {
	directory := t.TempDir()
	public := filepath.Join(directory, "publico.jsonl")
	if err := os.WriteFile(public, []byte(`{"id":"conducta:uno"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInventory(public); err == nil {
		t.Fatal("se aceptó un inventario legible por terceros")
	}

	oversized := filepath.Join(directory, "enorme.jsonl")
	file, err := os.OpenFile(oversized, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxInventoryFileBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInventory(oversized); err == nil {
		t.Fatal("se aceptó un inventario físico sobredimensionado")
	}
}

func TestInventoryRejectsMoreSemanticItemsThanItsBudget(t *testing.T) {
	var content strings.Builder
	for item := 0; item <= maxInventoryItems; item++ {
		fmt.Fprintf(&content, "{\"id\":\"conducta:%05d\"}\n", item)
	}
	filePath := filepath.Join(t.TempDir(), "demasiados.jsonl")
	if err := os.WriteFile(filePath, []byte(content.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInventory(filePath); err == nil ||
		!strings.Contains(err.Error(), "elementos semánticos") {
		t.Fatalf("se aceptó un inventario semántico fuera de presupuesto: %v", err)
	}
}
