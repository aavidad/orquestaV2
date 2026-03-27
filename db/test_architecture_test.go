package db

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDBTestsNoIntroducenNuevasAperturasDirectas(t *testing.T) {
	t.Helper()

	permitidos := map[string]struct{}{
		"mcp_views_test.go":                      {},
		"ops_view_test.go":                       {},
		"orquestacion_test.go":                   {},
		"post_migrations_test.go":                {},
		"postgres_bootstrap_integration_test.go": {},
		"runtime_legacy_migration_test.go":       {},
		"runtime_observability_test.go":          {},
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir db: %v", err)
	}

	var encontrados []string
	for _, entry := range entries {
		nombre := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(nombre, "_test.go") {
			continue
		}
		if nombre == "test_architecture_test.go" {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(nombre))
		if err != nil {
			t.Fatalf("read %s: %v", nombre, err)
		}
		if !strings.Contains(string(raw), "Open()") {
			continue
		}
		encontrados = append(encontrados, nombre)
		if _, ok := permitidos[nombre]; !ok {
			t.Fatalf("test con Open() directo no permitido: %s; usa prepararDBTemporal(...) o justifica la excepción en este allowlist", nombre)
		}
	}

	sort.Strings(encontrados)
	var allow []string
	for nombre := range permitidos {
		allow = append(allow, nombre)
	}
	sort.Strings(allow)
	if strings.Join(encontrados, ",") != strings.Join(allow, ",") {
		t.Fatalf("allowlist de Open() directo desactualizado.\nencontrados: %v\npermitidos: %v", encontrados, allow)
	}
}
