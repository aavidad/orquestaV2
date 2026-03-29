package db

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"orquesta/runtimeagente"
)

func BuildLaunchBootstrapPromptForContext(agente *Agente, proyecto *Proyecto, plan *runtimeagente.LaunchPlan) (string, error) {
	if agente == nil || proyecto == nil {
		return "", nil
	}
	catalogo, err := ResolveGovernanceCatalogForContext(strings.TrimSpace(agente.Rol), &proyecto.ID, strings.TrimSpace(agente.Nombre))
	if err != nil {
		return "", err
	}
	memoria, err := ListarEntidadesMemoria(FiltroEntidadesMemoria{ProyectoID: &proyecto.ID})
	if err != nil {
		return "", err
	}
	tareas, err := ListarTareas(FiltroTareas{
		Agente:     strPtrRuntime(strings.TrimSpace(agente.Nombre)),
		ProyectoID: &proyecto.ID,
	})
	if err != nil {
		return "", err
	}
	propuestas, err := PropuestasPendientesVotoProyecto(strings.TrimSpace(agente.Nombre), &proyecto.ID)
	if err != nil {
		return "", err
	}
	_, resumenProyecto := BuildProjectContextSummary(strings.TrimSpace(agente.Nombre), proyecto)
	_, resumenGobernanza := BuildGovernanceContextSummaryForContext(strings.TrimSpace(agente.Rol), &proyecto.ID, strings.TrimSpace(agente.Nombre))
	return BuildLaunchBootstrapPrompt(agente, proyecto, plan, catalogo, memoria, tareas, propuestas, resumenProyecto, resumenGobernanza), nil
}

func BuildLaunchBootstrapPrompt(agente *Agente, proyecto *Proyecto, plan *runtimeagente.LaunchPlan, catalogo *GovernanceCatalog, memoria []*EntidadMemoria, tareas []*Tarea, propuestas []*Propuesta, resumenProyecto, resumenGobernanza string) string {
	if agente == nil || proyecto == nil {
		return ""
	}

	workingDir := strings.TrimSpace(proyecto.RutaAbs)
	if plan != nil && strings.TrimSpace(plan.WorkingDir) != "" {
		workingDir = strings.TrimSpace(plan.WorkingDir)
	}

	lines := []string{
		fmt.Sprintf("Bootstrap de Orquesta para %s.", strings.TrimSpace(agente.Nombre)),
		fmt.Sprintf("Rol: %s. Proyecto: %s.", strings.TrimSpace(agente.Rol), strings.TrimSpace(proyecto.Slug)),
		fmt.Sprintf("Directorio de trabajo: %s.", workingDir),
		"Fuente de verdad operativa: daemon/API de Orquesta. No uses acceso directo a BD en el flujo normal salvo diagnóstico o recuperación.",
		fmt.Sprintf("Este bootstrap ya equivale a orquesta sesion inicio %s y la sesión actual ya está abierta por Orquesta; no ejecutes sesion inicio de nuevo salvo recuperación explícita.", strings.TrimSpace(agente.Nombre)),
		"Si la acción es destructiva, irreversible o de riesgo alto, consulta antes.",
	}

	if plan != nil {
		perfil := strings.TrimSpace(plan.PerfilTarea)
		modelo := strings.TrimSpace(plan.Modelo)
		razonamiento := strings.TrimSpace(plan.Razonamiento)
		if perfil != "" || modelo != "" || razonamiento != "" {
			lines = append(lines, fmt.Sprintf(
				"Perfil de ejecución: perfil=%s modelo=%s razonamiento=%s.",
				defaultBootstrapPromptValue(perfil),
				defaultBootstrapPromptValue(modelo),
				defaultBootstrapPromptValue(razonamiento),
			))
		}
		if continuity := strings.TrimSpace(plan.ContinuityPrompt); continuity != "" {
			lines = append(lines, "Continuidad disponible: "+continuity)
		}
	}

	if resumenProyecto = strings.TrimSpace(resumenProyecto); resumenProyecto != "" {
		lines = append(lines, ensurePromptSentence("Contexto operativo: "+resumenProyecto))
	}
	if resumenGobernanza = strings.TrimSpace(resumenGobernanza); resumenGobernanza != "" {
		lines = append(lines, ensurePromptSentence("Gobernanza efectiva: "+resumenGobernanza))
	}
	if resumenTareas := resumirTareasBootstrap(tareas); resumenTareas != "" {
		lines = append(lines, resumenTareas)
	}
	if resumenPropuestas := resumirPropuestasBootstrap(propuestas); resumenPropuestas != "" {
		lines = append(lines, resumenPropuestas)
	}
	if resumenReglas := resumirReglasBootstrap(catalogo); resumenReglas != "" {
		lines = append(lines, resumenReglas)
	}
	if resumenSkills := resumirSkillsBootstrap(catalogo); resumenSkills != "" {
		lines = append(lines, resumenSkills)
	}
	if resumenWorkflows := resumirWorkflowsBootstrap(catalogo); resumenWorkflows != "" {
		lines = append(lines, resumenWorkflows)
	}
	if resumenMemoria := resumirMemoriaBootstrap(memoria); resumenMemoria != "" {
		lines = append(lines, resumenMemoria)
	}

	lines = append(lines, "Empieza por la tarea asignada o, si no hay una única clara, consulta Orquesta antes de desviarte.")
	return strings.Join(lines, "\n")
}

func resumirTareasBootstrap(tareas []*Tarea) string {
	if len(tareas) == 0 {
		return "No hay tareas activas asignadas en este proyecto."
	}
	items := make([]string, 0, 3)
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso, TareaBloqueada:
		default:
			continue
		}
		items = append(items, fmt.Sprintf("#%d [%s] %s", tarea.ID, tarea.Estado, strings.TrimSpace(tarea.Titulo)))
		if len(items) == 3 {
			break
		}
	}
	if len(items) == 0 {
		return "No hay tareas activas asignadas en este proyecto."
	}
	return "Tareas activas: " + strings.Join(items, " | ") + "."
}

func resumirPropuestasBootstrap(propuestas []*Propuesta) string {
	if len(propuestas) == 0 {
		return ""
	}
	items := make([]string, 0, 3)
	for _, propuesta := range propuestas {
		if propuesta == nil || strings.TrimSpace(propuesta.Codigo) == "" {
			continue
		}
		items = append(items, strings.TrimSpace(propuesta.Codigo))
		if len(items) == 3 {
			break
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "Propuestas pendientes de voto: " + strings.Join(items, ", ") + "."
}

func resumirReglasBootstrap(catalogo *GovernanceCatalog) string {
	if catalogo == nil || len(catalogo.Reglas) == 0 {
		return ""
	}
	items := make([]string, 0, len(catalogo.Reglas))
	for _, regla := range catalogo.Reglas {
		if regla == nil {
			continue
		}
		items = append(items, fmt.Sprintf("[%s] %s: %s", strings.TrimSpace(regla.Categoria), strings.TrimSpace(regla.Titulo), strings.TrimSpace(regla.Descripcion)))
	}
	if len(items) == 0 {
		return ""
	}
	return "Reglas efectivas:\n- " + strings.Join(items, "\n- ")
}

func resumirSkillsBootstrap(catalogo *GovernanceCatalog) string {
	if catalogo == nil || len(catalogo.Skills) == 0 {
		return ""
	}
	items := make([]string, 0, 6)
	for _, skill := range catalogo.Skills {
		if skill == nil || strings.TrimSpace(skill.Nombre) == "" {
			continue
		}
		line := strings.TrimSpace(skill.Nombre)
		if when := strings.TrimSpace(skill.CuandoUsar); when != "" {
			line += ": " + when
		}
		items = append(items, line)
		if len(items) == 6 {
			break
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "Skills relevantes:\n- " + strings.Join(items, "\n- ")
}

func resumirWorkflowsBootstrap(catalogo *GovernanceCatalog) string {
	if catalogo == nil || len(catalogo.Workflows) == 0 {
		return ""
	}
	items := make([]string, 0, 4)
	for _, workflow := range catalogo.Workflows {
		if workflow == nil || strings.TrimSpace(workflow.Nombre) == "" {
			continue
		}
		line := strings.TrimSpace(workflow.Nombre)
		if desc := strings.TrimSpace(workflow.Descripcion); desc != "" {
			line += ": " + desc
		}
		if pasos := resumirPasosWorkflowBootstrap(workflow.Pasos, 3); pasos != "" {
			line += " Pasos: " + pasos
		}
		items = append(items, line)
		if len(items) == 4 {
			break
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "Workflows aplicables:\n- " + strings.Join(items, "\n- ")
}

func resumirPasosWorkflowBootstrap(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || max <= 0 {
		return ""
	}
	var pasos []string
	if err := json.Unmarshal([]byte(raw), &pasos); err != nil {
		return ""
	}
	resumen := make([]string, 0, max)
	for _, paso := range pasos {
		paso = strings.TrimSpace(paso)
		if paso == "" {
			continue
		}
		resumen = append(resumen, paso)
		if len(resumen) == max {
			break
		}
	}
	return strings.Join(resumen, " | ")
}

func resumirMemoriaBootstrap(memoria []*EntidadMemoria) string {
	if len(memoria) == 0 {
		return ""
	}
	items := make([]string, 0, 5)
	for _, entidad := range memoria {
		if entidad == nil || strings.TrimSpace(entidad.Nombre) == "" {
			continue
		}
		valor := truncarBootstrap(strings.TrimSpace(entidad.ValorJSON), 120)
		items = append(items, fmt.Sprintf("%s [%s]: %s", strings.TrimSpace(entidad.Nombre), strings.TrimSpace(entidad.Tipo), valor))
		if len(items) == 5 {
			break
		}
	}
	if len(items) == 0 {
		return ""
	}
	return "Memoria compartida:\n- " + strings.Join(items, "\n- ")
}

func truncarBootstrap(raw string, max int) string {
	if max <= 0 || len(raw) <= max {
		return raw
	}
	if max <= 3 {
		return raw[:max]
	}
	return raw[:max-3] + "..."
}

func defaultBootstrapPromptValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "auto"
	}
	return strconv.Quote(v)
}

func ensurePromptSentence(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch raw[len(raw)-1] {
	case '.', '!', '?':
		return raw
	default:
		return raw + "."
	}
}
