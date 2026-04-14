package db

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	start := time.Now()
	bootstrapPromptDebugf("BuildLaunchBootstrapPrompt start agente=%s proyecto=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(proyecto.Slug))
	defer func() {
		bootstrapPromptDebugf("BuildLaunchBootstrapPrompt done agente=%s proyecto=%s duration=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(proyecto.Slug), time.Since(start).Round(time.Millisecond))
	}()

	workingDir := strings.TrimSpace(proyecto.RutaAbs)
	if plan != nil && strings.TrimSpace(plan.WorkingDir) != "" {
		workingDir = strings.TrimSpace(plan.WorkingDir)
	}

	if promptBootstrapCompacto(plan) {
		return buildCompactLaunchBootstrapPrompt(agente, proyecto, plan, workingDir, tareas)
	}

	lines := []string{
		fmt.Sprintf("Bootstrap de Orquesta para %s.", strings.TrimSpace(agente.Nombre)),
		fmt.Sprintf("Rol: %s. Proyecto: %s.", strings.TrimSpace(agente.Rol), strings.TrimSpace(proyecto.Slug)),
		fmt.Sprintf("Directorio de trabajo: %s.", workingDir),
		InstruccionDoctrinaCanonica(),
		"Fuente de verdad operativa: daemon/API de Orquesta. No uses acceso directo a BD en el flujo normal salvo diagnóstico o recuperación.",
		"Las tareas, propuestas, sesiones y runtimes vivos se consultan en Orquesta; la documentacion no sustituye el estado operativo.",
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
	stepStart := time.Now()
	if esSupervisor, supervisorOperativo, err := EsSupervisorAutonomiaOperativo(proyecto.ID, strings.TrimSpace(agente.Nombre)); err == nil {
		if esSupervisor {
			lines = append(lines, "Rol operativo en este proyecto: orquestador autónomo. Coordina a los demás agentes, reparte trabajo real y sigue hasta cerrar la app o dejar el siguiente frente útil en marcha.")
		} else if supervisorOperativo != nil && strings.TrimSpace(supervisorOperativo.Nombre) != "" {
			lines = append(lines, fmt.Sprintf("Supervisor operativo del proyecto: %s. Tu rol operativo aquí es programador y debes coordinarte con ese supervisor por Orquesta.", strings.TrimSpace(supervisorOperativo.Nombre)))
		}
	}
	bootstrapPromptDebugf("BuildLaunchBootstrapPrompt step=supervision_role duration=%s", time.Since(stepStart).Round(time.Millisecond))

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

func promptBootstrapCompacto(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	if plan.CanSendInput != nil && !*plan.CanSendInput &&
		strings.EqualFold(strings.TrimSpace(plan.MailboxDeliveryMode), runtimeagente.MailboxDeliveryBootstrapOnly) {
		return true
	}
	if planLooksLikeOrchestratedAgentCLI(plan) {
		return true
	}
	return false
}

func buildCompactLaunchBootstrapPrompt(agente *Agente, proyecto *Proyecto, plan *runtimeagente.LaunchPlan, workingDir string, tareas []*Tarea) string {
	if planLooksLikeOllamaCLI(plan) {
		return buildCompactLaunchBootstrapPromptOllama(agente, proyecto, workingDir, tareas)
	}
	lines := []string{
		fmt.Sprintf("Bootstrap de Orquesta para %s.", strings.TrimSpace(agente.Nombre)),
		fmt.Sprintf("Rol: %s. Proyecto: %s.", strings.TrimSpace(agente.Rol), strings.TrimSpace(proyecto.Slug)),
		fmt.Sprintf("Directorio: %s.", workingDir),
		"Doctrina: " + RutaDoctrinaCanonica + ".",
		"Primero lee `.orquesta-inbox.md` si existe; usalo como contrato operativo vigente antes de tocar codigo.",
		"Fuente de verdad operativa: daemon/API de Orquesta.",
		"Unidad de trabajo: microtarea cerrada.",
		"Trabaja solo dentro del alcance de la tarea activa y del mailbox actual.",
		"Implementa solo la funcion o el patch pedido. Respeta archivo, simbolo, write-set y tests obligatorios.",
		"No hagas rg global, broad scans ni recorridos del repo entero antes del primer patch pequeno dentro del write-set.",
		"Si el contexto visible de la sesión no coincide con la tarea activa o el mailbox actual, ignóralo.",
		"No propongas arquitectura, roadmap ni refactors globales salvo que la microtarea lo pida de forma explicita.",
		"No reabras frentes viejos ni reescribas módulos fuera del alcance inmediato.",
		"Si la tarea activa o la inbox ya fijan frente, simbolos, write-set y tests, ejecuta ese slice y no releas la BIBLIA completa.",
		"Si existe `.orquesta-inbox.md`, no releas doctrina ni busques otros frentes antes del primer patch pequeno verificable.",
		"Consulta solo el fragmento minimo de doctrina que necesites si aparece un bloqueo real o falta contrato operativo en la inbox/tarea.",
		"No expliques planes largos ni reabras decisiones de diseño ya tomadas.",
	}
	if !bootstrapCompactoTieneTareaActiva(tareas) {
		lines = append(lines,
			"Si todavía no tienes una microtarea cerrada, responde solo ACK-ESPERA y espera.",
			"Cuando llegue una microtarea, ejecuta solo ese cambio y devuelve evidencia breve; no hagas trabajo adicional.",
		)
	}
	if resumenTareas := resumirTareasBootstrapCompacto(tareas); resumenTareas != "" {
		lines = append(lines, resumenTareas)
	}
	lines = append(lines, "Empieza por la tarea asignada y evita tocar BD local salvo diagnóstico o recuperación.")
	lines = append(lines, "Si la acción es destructiva, irreversible o de riesgo alto, consulta antes.")
	return strings.Join(lines, "\n")
}

func bootstrapCompactoTieneTareaActiva(tareas []*Tarea) bool {
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso, TareaBloqueada:
			return true
		}
	}
	return false
}

func buildCompactLaunchBootstrapPromptOllama(agente *Agente, proyecto *Proyecto, workingDir string, tareas []*Tarea) string {
	lines := []string{
		"PROTOCOLO_ORQUESTA_MICRO",
		"NO_INTERPRETAR_COMO_PREGUNTA",
		"SALIDA_INMEDIATA=ACK-ESPERA",
		"ESPERA_MICROTAREA_CERRADA",
		"EJECUTA_SOLO_WRITE_SET",
		"SI_BLOQUEO=BLOQUEO: <motivo concreto>",
	}
	return strings.Join(lines, "\n")
}

func resumirTareasBootstrapCompacto(tareas []*Tarea) string {
	if len(tareas) == 0 {
		return "No hay una tarea activa única; consulta Orquesta antes de desviarte."
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso, TareaBloqueada:
			partes := []string{fmt.Sprintf("Tarea activa: #%d [%s] %s.", tarea.ID, tarea.Estado, strings.TrimSpace(tarea.Titulo))}
			if descripcion := resumirDescripcionTareaBootstrapCompacto(strings.TrimSpace(tarea.Descripcion)); descripcion != "" {
				partes = append(partes, "Alcance inmediato: "+descripcion+".")
			}
			return strings.Join(partes, " ")
		}
	}
	return "No hay una tarea activa única; consulta Orquesta antes de desviarte."
}

func resumirDescripcionTareaBootstrapCompacto(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if raw == "" {
		return ""
	}
	raw = strings.TrimSpace(strings.TrimRight(raw, ".;"))
	const maxRunes = 360
	prioritized := resumirDescripcionTareaBootstrapPrioritaria(raw, maxRunes)
	if prioritized != "" {
		return prioritized
	}
	runes := []rune(raw)
	if len(runes) <= maxRunes {
		return raw
	}
	return string(runes[:maxRunes-1]) + "…"
}

func resumirDescripcionTareaBootstrapPrioritaria(raw string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	clauses := splitBootstrapTaskClauses(raw)
	if len(clauses) == 0 {
		return ""
	}
	priorityMatchers := []string{
		"simbolos foco:",
		"write-set",
		"write_set",
		"tests minimos",
		"tests mínimos",
		"regla arquitectonica de este frente:",
		"regla arquitectónica de este frente:",
	}
	seen := map[string]struct{}{}
	selected := make([]string, 0, len(clauses))
	appendIfFits := func(clause string) bool {
		clause = compactarBootstrapTaskClause(strings.TrimSpace(strings.TrimRight(clause, ".;")))
		if clause == "" {
			return false
		}
		if _, ok := seen[clause]; ok {
			return true
		}
		candidate := clause
		if len(selected) > 0 {
			candidate = strings.Join(append(append([]string(nil), selected...), clause), ". ")
		}
		if len([]rune(candidate)) > maxRunes {
			return false
		}
		selected = append(selected, clause)
		seen[clause] = struct{}{}
		return true
	}
	for _, matcher := range priorityMatchers {
		for _, clause := range clauses {
			lower := strings.ToLower(strings.TrimSpace(clause))
			if !strings.Contains(lower, matcher) {
				continue
			}
			if !appendIfFits(clause) {
				break
			}
		}
	}
	if len(selected) == 0 {
		return ""
	}
	resumen := strings.Join(selected, ". ")
	resumen = strings.TrimSpace(strings.TrimRight(resumen, ".;"))
	if resumen == "" {
		return ""
	}
	return resumen
}

func compactarBootstrapTaskClause(clause string) string {
	clause = strings.TrimSpace(strings.TrimRight(clause, ".;"))
	if clause == "" {
		return ""
	}
	lower := strings.ToLower(clause)
	if strings.HasPrefix(lower, "tests minimos") || strings.HasPrefix(lower, "tests mínimos") {
		clause = strings.Replace(clause, "Tests minimos del slice:", "Tests minimos:", 1)
		clause = strings.Replace(clause, "Tests mínimos del slice:", "Tests mínimos:", 1)
		clause = strings.ReplaceAll(clause, " -count=1", "")
		clause = strings.ReplaceAll(clause, " y go test ", " ; go test ")
		clause = strings.Join(strings.Fields(clause), " ")
	}
	return clause
}

func splitBootstrapTaskClauses(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "\n", ". ")
	partes := strings.Split(raw, ". ")
	out := make([]string, 0, len(partes))
	for _, parte := range partes {
		parte = strings.TrimSpace(strings.TrimRight(parte, ".;"))
		if parte == "" {
			continue
		}
		out = append(out, parte)
	}
	return out
}

func planLooksLikeOrchestratedAgentCLI(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(plan.Driver), "cli") {
		return false
	}
	parts := make([]string, 0, 1+len(plan.Args))
	if cmd := strings.ToLower(strings.TrimSpace(plan.Comando)); cmd != "" {
		parts = append(parts, cmd)
	}
	for _, arg := range plan.Args {
		arg = strings.ToLower(strings.TrimSpace(arg))
		if arg != "" {
			parts = append(parts, arg)
		}
	}
	for _, part := range parts {
		base := strings.ToLower(strings.TrimSpace(filepath.Base(part)))
		if strings.Contains(part, "codex-perfil") || strings.Contains(part, "claude-perfil") || strings.Contains(part, "gemini-perfil") ||
			strings.Contains(base, "codex-perfil") || strings.Contains(base, "claude-perfil") || strings.Contains(base, "gemini-perfil") {
			return true
		}
		if part == "codex" || part == "claude" || part == "gemini" || part == "ollama" ||
			base == "codex" || base == "claude" || base == "gemini" || base == "ollama" {
			return true
		}
		if strings.Contains(part, "ollama-cli") || strings.Contains(part, "ollama-perfil") ||
			strings.Contains(base, "ollama-cli") || strings.Contains(base, "ollama-perfil") {
			return true
		}
	}
	return false
}

func planLooksLikeOllamaCLI(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	parts := make([]string, 0, 1+len(plan.Args))
	if cmd := strings.ToLower(strings.TrimSpace(plan.Comando)); cmd != "" {
		parts = append(parts, cmd)
	}
	for _, arg := range plan.Args {
		arg = strings.ToLower(strings.TrimSpace(arg))
		if arg != "" {
			parts = append(parts, arg)
		}
	}
	for _, part := range parts {
		base := strings.ToLower(strings.TrimSpace(filepath.Base(part)))
		if part == "ollama" || base == "ollama" ||
			strings.Contains(part, "ollama-cli") || strings.Contains(part, "ollama-perfil") ||
			strings.Contains(base, "ollama-cli") || strings.Contains(base, "ollama-perfil") {
			return true
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(plan.RemoteConfigJSON)), "ollama") {
		return true
	}
	return false
}

func bootstrapPromptDebugf(format string, args ...any) {
	if !preparePromptDebugEnabled() {
		return
	}
	log.Printf("orquesta[prepare-prompt] "+format, args...)
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
