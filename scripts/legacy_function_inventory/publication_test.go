// Estas pruebas fuerzan cada renombrado de la publicación y acreditan que un
// error no deja un inventario nuevo acompañado por un manifiesto viejo.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationRollsBackWhenEitherFinalRenameFails(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			directory := t.TempDir()
			inventoryPath := filepath.Join(directory, "inventory.jsonl")
			manifestPath := filepath.Join(directory, "manifest.json")
			inventoryTemporary := filepath.Join(directory, "inventory.tmp")
			manifestTemporary := filepath.Join(directory, "manifest.tmp")
			writePublicationFile(t, inventoryPath, "inventario anterior")
			writePublicationFile(t, manifestPath, "manifiesto anterior")
			writePublicationFile(t, inventoryTemporary, "inventario nuevo")
			writePublicationFile(t, manifestTemporary, "manifiesto nuevo")

			renames := 0
			injected := errors.New("renombrado inyectado")
			operations := operatingSystemPublication
			operations.rename = func(oldPath, newPath string) error {
				renames++
				if renames == failAt {
					return injected
				}
				return os.Rename(oldPath, newPath)
			}
			err := publishPairWithOps(
				inventoryTemporary, inventoryPath,
				manifestTemporary, manifestPath,
				operations,
			)
			if !errors.Is(err, injected) {
				t.Fatalf("se esperaba fallo del renombrado %d; obtenido: %v", failAt, err)
			}
			assertPublicationContent(t, inventoryPath, "inventario anterior")
			assertPublicationContent(t, manifestPath, "manifiesto anterior")
			if _, err := os.Lstat(manifestTemporary + ".previous"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("quedó un respaldo innecesario: %v", err)
			}
		})
	}
}

func TestPublicationRemovesNewManifestWhenInventoryRenameFailsWithoutOldPair(t *testing.T) {
	directory := t.TempDir()
	inventoryPath := filepath.Join(directory, "inventory.jsonl")
	manifestPath := filepath.Join(directory, "manifest.json")
	inventoryTemporary := filepath.Join(directory, "inventory.tmp")
	manifestTemporary := filepath.Join(directory, "manifest.tmp")
	writePublicationFile(t, inventoryTemporary, "inventario nuevo")
	writePublicationFile(t, manifestTemporary, "manifiesto nuevo")
	renames := 0
	operations := operatingSystemPublication
	operations.rename = func(oldPath, newPath string) error {
		renames++
		if renames == 2 {
			return errors.New("fallo inventario")
		}
		return os.Rename(oldPath, newPath)
	}
	if err := publishPairWithOps(
		inventoryTemporary, inventoryPath,
		manifestTemporary, manifestPath,
		operations,
	); err == nil {
		t.Fatal("se esperaba fallo al publicar el inventario")
	}
	for _, path := range []string{inventoryPath, manifestPath} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("se publicó parte del par en %s: %v", path, err)
		}
	}
}

func TestPublicationKeepsRecoverableBackupIfRollbackRenameAlsoFails(t *testing.T) {
	directory := t.TempDir()
	inventoryPath := filepath.Join(directory, "inventory.jsonl")
	manifestPath := filepath.Join(directory, "manifest.json")
	inventoryTemporary := filepath.Join(directory, "inventory.tmp")
	manifestTemporary := filepath.Join(directory, "manifest.tmp")
	writePublicationFile(t, inventoryPath, "inventario anterior")
	writePublicationFile(t, manifestPath, "manifiesto anterior")
	writePublicationFile(t, inventoryTemporary, "inventario nuevo")
	writePublicationFile(t, manifestTemporary, "manifiesto nuevo")
	renames := 0
	operations := operatingSystemPublication
	operations.rename = func(oldPath, newPath string) error {
		renames++
		if renames == 2 || renames == 3 {
			return errors.New("fallo inyectado")
		}
		return os.Rename(oldPath, newPath)
	}
	if err := publishPairWithOps(
		inventoryTemporary, inventoryPath,
		manifestTemporary, manifestPath,
		operations,
	); err == nil {
		t.Fatal("se esperaba fallo de publicación y restauración")
	}
	assertPublicationContent(t, inventoryPath, "inventario anterior")
	assertPublicationContent(t, manifestTemporary+".previous", "manifiesto anterior")
}

func writePublicationFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertPublicationContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("contenido inesperado en %s: %q", path, content)
	}
}
