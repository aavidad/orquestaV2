// Este fichero simula sustituciones de padres y acredita el anclaje por descriptor.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRootParentReplacementAfterAnchorCannotRedirectRead(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "padre")
	if err := os.MkdirAll(filepath.Join(parent, "raiz"), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(parent, "raiz", "propio"), "propio", 0o600)
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(outside, "raiz"), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(outside, "raiz", "ajeno"), "ajeno", 0o600)
	root := rootOption{
		alias: "historia", mode: modeMetadata, path: filepath.Join(parent, "raiz"),
	}
	root.afterAnchor = func() {
		if err := os.Rename(parent, filepath.Join(base, "padre-original")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, parent); err != nil {
			t.Fatal(err)
		}
	}
	anchor, err := anchorRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := readDirectoryEntries(int(anchor.file.Fd()), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "propio" {
		t.Fatalf("la lectura fue redirigida: %#v", entries)
	}
	if err := anchor.file.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestOutputParentReplacementCannotRedirectEffects(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "salida")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	redirected := filepath.Join(base, "redirigida")
	if err := os.Mkdir(redirected, 0o700); err != nil {
		t.Fatal(err)
	}
	jsonl := filepath.Join(parent, "censo.jsonl")
	manifest := filepath.Join(parent, "censo.json")
	anchor, err := openOutputAnchorWithHook(jsonl, manifest, func() {
		if err := os.Rename(parent, filepath.Join(base, "salida-original")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(redirected, parent); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err := createAt(anchor.fd(), "prueba")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "salida-original", "prueba")); err != nil {
		t.Fatalf("el efecto no quedó en el directorio anclado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(redirected, "prueba")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("el efecto fue redirigido: %v", err)
	}
	if err := anchor.close(); err != nil {
		t.Fatal(err)
	}
}
