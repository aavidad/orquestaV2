package db

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDBNoIntroduceDependenciasInternasFueraDeLaFronteraPermitida(t *testing.T) {
	t.Helper()

	permitidos := map[string]map[string]struct{}{
		"orquesta/coordinacion": {
			"coordinacion_backend.go": {},
			"diagnostico.go":          {},
			"locks.go":                {},
			"proyectos.go":            {},
			"worktree_coherence.go":   {},
			"worktrees.go":            {},
		},
		"orquesta/i18n": {
			"lenguaje.go": {},
		},
		"orquesta/internal/controlruntime": {
			"controlplane_entities.go": {},
		},
		"orquesta/internal/observabilidadruntime": {
			"runtimes.go": {},
		},
		"orquesta/microprogramacionapp": {
			"microprogramacionapp_adapter.go": {},
		},
		"orquesta/runtimeagente": {
			"controlplane_entities.go":    {},
			"resume_context_sanitizer.go": {},
			"runtime_bootstrap_prompt.go": {},
		},
		"orquesta/sesionesapp": {
			"sesiones.go":            {},
			"sesionesapp_adapter.go": {},
		},
		"orquesta/storage": {
			"backend.go":                   {},
			"backend_mysql.go":             {},
			"backend_postgres.go":          {},
			"backend_sqlite.go":            {},
			"db.go":                        {},
			"sqlwrap.go":                   {},
			"verificacion_persistencia.go": {},
		},
		"orquesta/tareaspolicy": {
			"tareas.go": {},
		},
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir db: %v", err)
	}

	observadosPorImport := map[string][]string{}
	var violaciones []string

	for _, entry := range entries {
		nombre := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(nombre, ".go") || strings.HasSuffix(nombre, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Clean(nombre), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", nombre, err)
		}
		for _, spec := range file.Imports {
			ruta := strings.Trim(spec.Path.Value, `"`)
			if !strings.HasPrefix(ruta, "orquesta/") {
				continue
			}
			observadosPorImport[ruta] = append(observadosPorImport[ruta], nombre)
			permitidosEnRuta, ok := permitidos[ruta]
			if !ok {
				violaciones = append(violaciones, fmt.Sprintf("%s importa %s fuera de la frontera permitida de db/", nombre, ruta))
				continue
			}
			if _, ok := permitidosEnRuta[nombre]; !ok {
				violaciones = append(violaciones, fmt.Sprintf("%s importa %s pero ese acoplamiento solo esta permitido en %v", nombre, ruta, keysSorted(permitidosEnRuta)))
			}
		}
	}

	if len(violaciones) > 0 {
		sort.Strings(violaciones)
		t.Fatalf("dependencias internas nuevas o fuera de frontera en db/:\n%s", strings.Join(violaciones, "\n"))
	}

	for ruta, permitidosEnRuta := range permitidos {
		observados := uniqueSorted(observadosPorImport[ruta])
		esperados := keysSorted(permitidosEnRuta)
		if strings.Join(observados, ",") != strings.Join(esperados, ",") {
			t.Fatalf("allowlist de imports internos desactualizado para %s.\nobservados: %v\npermitidos: %v", ruta, observados, esperados)
		}
	}
}

func keysSorted(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	sort.Strings(items)
	out := items[:0]
	var prev string
	for i, item := range items {
		if i == 0 || item != prev {
			out = append(out, item)
			prev = item
		}
	}
	return out
}
