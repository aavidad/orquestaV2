package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultConfigInsertaSinDuplicar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	ensureDefaultConfig("clave_test_default", "uno")
	ensureDefaultConfig("clave_test_default", "dos")

	val, err := ConfigGet("clave_test_default")
	if err != nil {
		t.Fatalf("ConfigGet: %v", err)
	}
	if val != "uno" {
		t.Fatalf("valor inesperado: %s", val)
	}
}
