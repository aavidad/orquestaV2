package orquesta_test

import (
	"strings"
	"testing"
)

func TestEstadoVivoStatusSurfacesDoNotImportStateStoresDirectly(t *testing.T) {
	paquetesStatus := []string{
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-app-gateway",
		"orquesta/modulos/orquesta-mcp",
	}
	forbiddenImports := []string{
		"orquesta/modulos/orquesta-agent-process-registry",
		"orquesta/modulos/orquesta-agent-process-registry-memory",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-memory",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-persistence",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-run-memory",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-runtime-codex-delivery",
	}

	// PROHIBIDO anadir entradas; si tu cambio necesita anadir una, tu diseno
	// es incorrecto: pasa por orquesta-estado-vivo. Esta allowlist solo puede
	// menguar cuando se retire una dependencia directa existente.
	allowlistActual := map[string][]string{
		"orquesta/modulos/orquesta-app-codex-stack": {
			"orquesta/modulos/orquesta-runtime-codex-delivery",
		},
	}

	for _, pkg := range paquetesStatus {
		imports := packageImportsForBoundaryTest(t, pkg)
		importSet := boundaryStringSetV0(imports)
		allowed := boundaryStringSetV0(allowlistActual[pkg])
		for _, allowedImport := range allowlistActual[pkg] {
			if !importSet[allowedImport] {
				t.Fatalf("%s tiene allowlist stale para %s; elimina la entrada en vez de mantener permisos muertos", pkg, allowedImport)
			}
		}
		for _, imported := range imports {
			if !estadoVivoBoundaryForbiddenImportV0(imported, forbiddenImports) {
				continue
			}
			if allowed[imported] {
				continue
			}
			t.Fatalf("%s importa estado directo %s; deriva vida desde orquesta-estado-vivo", pkg, imported)
		}
	}
}

func estadoVivoBoundaryForbiddenImportV0(imported string, forbiddenImports []string) bool {
	imported = strings.TrimSpace(imported)
	for _, forbidden := range forbiddenImports {
		forbidden = strings.TrimSpace(forbidden)
		if imported == forbidden || strings.HasPrefix(imported, forbidden+"/") {
			return true
		}
	}
	return false
}

func boundaryStringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}
