package orquestaappdirectorintake

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type directorTaskAreaV0 struct {
	Suffix   string
	Role     string
	Summary  string
	Capacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	WriteSet []string
}

const maxDirectorTaskAreasV0 = 4

func directorTasksFromAppSpecV0(spec AppDirectorInputSpecV0) []AppDirectorTaskV0 {
	appRef := appDirectorTaskRefPrefixV0(spec)
	areas := []directorTaskAreaV0{primaryDirectorTaskAreaV0(spec)}
	if spec.AgentPreferences.Autonomy != "alta" {
		return directorTasksForAreasV0(appRef, areas)
	}
	areas = append(areas, directorSpecializedAreasV0(spec)...)
	return directorTasksForAreasV0(appRef, compactDirectorTaskAreasV0(areas))
}

func primaryDirectorTaskAreaV0(spec AppDirectorInputSpecV0) directorTaskAreaV0 {
	return directorTaskAreaV0{
		Suffix:   "director",
		Role:     "director",
		Summary:  directorTaskSummaryV0(spec, "Dirigir solicitud de app con contexto pequeno."),
		Capacity: primaryDirectorCapacityV0(spec),
		WriteSet: primaryDirectorWriteSetV0(spec),
	}
}

func primaryDirectorCapacityV0(spec AppDirectorInputSpecV0) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if normalizeDirectorInputExecutionModeV0(spec.ExecutionMode) == AppDirectorExecutionModeDebugV0 {
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	}
	return orquestacoreworkflow.OrchestrationCapacityXHighV0
}

func primaryDirectorWriteSetV0(spec AppDirectorInputSpecV0) []string {
	paths := []string{
		"docs/arquitectura.md",
		"docs/plan_microtareas.md",
		"docs/decisiones.md",
		"docs/pruebas.md",
		"docs/pendientes.md",
	}
	switch normalizeDirectorInputRequestKindV0(spec.RequestKind) {
	case AppDirectorRequestKindDocumentarAppV0,
		AppDirectorRequestKindCrearAppCompletaV0,
		AppDirectorRequestKindPlanificarAppV0:
		paths = append(paths,
			"docs/manual_usuario.md",
			"docs/manual_desarrollador.md",
			"docs/manual_sistemas_deploy.md",
		)
	}
	return paths
}

func directorTaskSummaryV0(spec AppDirectorInputSpecV0, base string) string {
	kind := normalizeDirectorInputRequestKindV0(spec.RequestKind)
	mode := normalizeDirectorInputExecutionModeV0(spec.ExecutionMode)
	parts := []string{
		base,
		"request_kind=" + kind,
		"execution_mode=" + mode + ".",
	}
	if hints := directorTaskSummaryHintsV0(spec); len(hints) > 0 {
		parts = append(parts, "contexto_app: "+strings.Join(hints, "; ")+".")
	}
	return joinDirectorTaskSummaryWithinBudgetV0(
		compactDirectorTaskSummaryPartsV0(parts),
		maxDirectorTaskSummaryBytesV0,
	)
}

func directorTaskSummaryHintsV0(spec AppDirectorInputSpecV0) []string {
	hints := []string{
		directorTaskSummaryFieldV0("app", spec.App.Nombre),
		directorTaskSummaryFieldV0("objetivo", spec.App.Objetivo),
		directorTaskSummaryFieldV0("descripcion", spec.App.Descripcion),
		directorTaskSummaryFieldV0("tipo", spec.App.TipoApp),
		directorTaskSummaryListFieldV0("plataformas", spec.Platforms),
	}
	for _, need := range spec.Data.Needs {
		if hint := directorTaskSummaryFieldV0("datos", need); hint != "" {
			hints = append(hints, hint)
			break
		}
	}
	return compactDirectorTaskSummaryPartsV0(hints)
}

func directorSpecializedAreasV0(spec AppDirectorInputSpecV0) []directorTaskAreaV0 {
	areas := []directorTaskAreaV0{}
	if appNeedsWebDirectorAreaV0(spec) {
		areas = append(areas, directorTaskAreaV0{
			Suffix:   "web",
			Role:     "director_web",
			Summary:  directorTaskSummaryV0(spec, "Analizar experiencia web, pantallas y flujos de usuario."),
			Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			WriteSet: []string{"docs/web.md"},
		})
	}
	if appNeedsAPIDirectorAreaV0(spec) {
		areas = append(areas, directorTaskAreaV0{
			Suffix:   "api",
			Role:     "director_api",
			Summary:  directorTaskSummaryV0(spec, "Analizar interfaz publica, casos de uso y contratos."),
			Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			WriteSet: []string{"docs/api.md"},
		})
	}
	if spec.Data.PersistenceRequired {
		areas = append(areas, directorTaskAreaV0{
			Suffix:   "persistencia",
			Role:     "director_persistencia",
			Summary:  directorTaskSummaryV0(spec, "Definir persistencia por puerto y contrato de conector."),
			Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			WriteSet: []string{"docs/persistencia.md"},
		})
	}
	if spec.I18N.Enabled {
		areas = append(areas, directorTaskAreaV0{
			Suffix:   "i18n",
			Role:     "director_i18n",
			Summary:  directorTaskSummaryV0(spec, "Definir catalogos, locales y reglas de texto visible."),
			Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			WriteSet: []string{"docs/i18n.md"},
		})
	}
	if spec.Quality.Tests == "alta" {
		areas = append(areas, directorTaskAreaV0{
			Suffix:   "calidad",
			Role:     "director_calidad",
			Summary:  directorTaskSummaryV0(spec, "Definir estrategia de pruebas, seguridad y revision final."),
			Capacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			WriteSet: []string{"docs/calidad.md"},
		})
	}
	return areas
}

func directorTasksForAreasV0(appRef string, areas []directorTaskAreaV0) []AppDirectorTaskV0 {
	tasks := make([]AppDirectorTaskV0, 0, len(areas))
	for _, area := range areas {
		tasks = append(tasks, directorTaskForAreaV0(appRef, area))
	}
	return tasks
}

func compactDirectorTaskAreasV0(areas []directorTaskAreaV0) []directorTaskAreaV0 {
	seen := map[string]bool{}
	result := make([]directorTaskAreaV0, 0, len(areas))
	for _, area := range areas {
		suffix := safeDirectorIntakeRefPartV0(area.Suffix)
		if suffix == "" || seen[suffix] {
			continue
		}
		seen[suffix] = true
		result = append(result, area)
		if len(result) == maxDirectorTaskAreasV0 {
			return result
		}
	}
	return result
}

func appNeedsWebDirectorAreaV0(spec AppDirectorInputSpecV0) bool {
	return strings.Contains(spec.App.TipoApp, "web") ||
		spec.App.TipoApp == "mixed" ||
		directorIntakeStringInSetV0(spec.Platforms, "web")
}

func appNeedsAPIDirectorAreaV0(spec AppDirectorInputSpecV0) bool {
	return strings.Contains(spec.App.TipoApp, "api") ||
		spec.App.TipoApp == "mixed" ||
		spec.App.TipoApp == "automation" ||
		spec.App.TipoApp == "plugin"
}
