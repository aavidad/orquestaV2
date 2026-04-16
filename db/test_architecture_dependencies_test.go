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

type importSeam struct {
	required bool
	files    map[string]struct{}
}

func TestDBNoIntroduceDependenciasInternasFueraDeLaFronteraPermitida(t *testing.T) {
	t.Helper()

	permitidos := map[string]importSeam{
		"orquesta/coordinacion": {
			required: true,
			files: map[string]struct{}{
				"coordinacion_backend.go": {},
				"diagnostico.go":          {},
				"locks.go":                {},
				"project_context.go":      {},
				"proyectos.go":            {},
				"worktrees.go":            {},
			},
		},
		"orquesta/i18n": {
			required: true,
			files: map[string]struct{}{
				"lenguaje.go": {},
			},
		},
		"orquesta/internal/controlruntime": {
			required: true,
			files: map[string]struct{}{
				"controlplane_entities.go": {},
			},
		},
		"orquesta/internal/observabilidadruntime": {
			required: true,
			files: map[string]struct{}{
				"runtimes.go": {},
			},
		},
		"orquesta/microprogramacionapp": {
			required: true,
			files: map[string]struct{}{
				"microprogramacionapp_adapter.go": {},
			},
		},
		"orquesta/runtimeagente": {
			required: true,
			files: map[string]struct{}{
				"controlplane_entities.go":    {},
				"resume_context_sanitizer.go": {},
				"runtime_bootstrap_prompt.go": {},
			},
		},
		"orquesta/sesionesapp": {
			required: true,
			files: map[string]struct{}{
				"presupuestos_sesion.go": {},
				"sesiones.go":            {},
				"sesionesapp_adapter.go": {},
			},
		},
		"orquesta/storage": {
			required: true,
			files: map[string]struct{}{
				"backend.go":                   {},
				"backend_mysql.go":             {},
				"backend_postgres.go":          {},
				"backend_sqlite.go":            {},
				"db.go":                        {},
				"sqlwrap.go":                   {},
				"verificacion_persistencia.go": {},
			},
		},
		"orquesta/gobernanzapolicy": {
			required: true,
			files: map[string]struct{}{
				"governance_overrides.go": {},
				"reglas.go":               {},
			},
		},
		"orquesta/planificadorpolicy": {
			required: true,
			files: map[string]struct{}{
				"autonomia_proyecto.go": {},
				"planificador.go":       {},
			},
		},
		"orquesta/runtimepolicy": {
			required: true,
			files: map[string]struct{}{
				"controlplane_entities.go":    {},
				"runtime_bootstrap_prompt.go": {},
				"runtime_transcript.go":       {},
			},
		},
		"orquesta/runtimesapp": {
			required: false,
			files: map[string]struct{}{
				"service.go": {},
				"trace.go":   {},
			},
		},
		"orquesta/autonomiapolicy": {
			required: true,
			files: map[string]struct{}{
				"autonomia_proyecto.go":             {},
				"autonomia_supervisor_operativo.go": {},
			},
		},
		"orquesta/skillspolicy": {
			required: true,
			files: map[string]struct{}{
				"skills_catalog.go": {},
			},
		},
		"orquesta/tareaspolicy": {
			required: true,
			files: map[string]struct{}{
				"tareas.go": {},
			},
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
			if len(permitidosEnRuta.files) > 0 {
				if _, ok := permitidosEnRuta.files[nombre]; !ok {
					violaciones = append(violaciones, fmt.Sprintf("%s importa %s pero ese acoplamiento solo esta permitido en %v", nombre, ruta, keysSortedFromSet(permitidosEnRuta.files)))
				}
				continue
			}
		}
	}

	if len(violaciones) > 0 {
		sort.Strings(violaciones)
		t.Fatalf("dependencias internas nuevas o fuera de frontera en db/:\n%s", strings.Join(violaciones, "\n"))
	}

	for ruta, permitidosEnRuta := range permitidos {
		if !permitidosEnRuta.required {
			continue
		}
		observados := uniqueSorted(observadosPorImport[ruta])
		esperados := keysSortedFromSet(permitidosEnRuta.files)
		if strings.Join(observados, ",") != strings.Join(esperados, ",") {
			t.Fatalf("allowlist de imports internos desactualizado para %s.\nobservados: %v\npermitidos: %v", ruta, observados, esperados)
		}
	}
}

func keysSortedFromSet(m map[string]struct{}) []string {
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
