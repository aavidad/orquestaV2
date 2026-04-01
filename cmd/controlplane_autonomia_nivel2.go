package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/fabricaapp"
	"orquesta/gitgobernanza"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
	"orquesta/tareasapp"
)

func (dbAutomationService) ProcesarSupervisionAutonomaBatch() (int, error) {
	return procesarSupervisionAutonomaBatch()
}

func (dbAutomationService) ProcesarReviewGatesBatch() (int, error) {
	return procesarReviewGatesBatch()
}

func procesarSupervisionAutonomaBatch() (int, error) {
	policies, err := supervisionService.ListEnabledPolicies()
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	interval := time.Duration(controlPlaneConfigIntOrDefault("autonomia_supervision_interval_seconds", 300)) * time.Second
	total := 0
	for _, policy := range policies {
		if !supervisionDue(policy, now, interval) {
			continue
		}
		proyecto, err := db.GetProyecto(strconv.FormatInt(policy.ProyectoID, 10))
		if err != nil {
			return total, err
		}
		disponible, err := db.ProyectoDisponibleParaAutonomia(proyecto.ID)
		if err != nil {
			return total, err
		}
		if !disponible {
			continue
		}
		agente, _, err := resolverSupervisorAutonomiaOperativo(proyecto.ID, policy, "supervision_automatica", "")
		if err != nil {
			return total, err
		}
		if agente == nil {
			continue
		}
		autoCreate := autonomiaAutoCreateResult{}
		if res, err := asegurarTrabajoAutonomia(policy, proyecto, agente); err != nil {
			return total, err
		} else {
			autoCreate = res
		}
		if policy.LastSupervisionAt != nil && !autoCreate.Created {
			tieneTrabajoActivo, err := supervisorAutonomiaTieneTrabajoActivo(agente.Nombre, proyecto.ID)
			if err != nil {
				return total, err
			}
			if tieneTrabajoActivo {
				continue
			}
		}
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(agente.Nombre, &proyecto.ID, "nudge", "supervisar_proyecto"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		if reciente, err := existeRuntimeOrderAutonomiaReciente(agente.Nombre, &proyecto.ID, "nudge", "supervisar_proyecto", interval); err != nil {
			return total, err
		} else if reciente {
			continue
		}
		if policy.LastSupervisionAt != nil {
			operativo, err := supervisorAutonomiaYaOperativo(agente.Nombre, proyecto.ID)
			if err != nil {
				return total, err
			}
			if operativo {
				continue
			}
		}
		contexto, resumen := db.BuildProjectContextSummary(strings.TrimSpace(agente.Nombre), proyecto)
		decision := map[string]any{
			"accion": "supervisar_proyecto",
			"motivo": resumen,
		}
		if autoCreate.Created {
			decision["auto_created_work"] = true
			if autoCreate.SeedTaskID > 0 {
				decision["auto_created_task"] = true
				decision["auto_created_task_id"] = autoCreate.SeedTaskID
			}
			if autoCreate.PlanCreated > 0 {
				decision["auto_created_plan"] = true
				decision["auto_created_tasks_count"] = autoCreate.PlanCreated
				decision["auto_created_backlog"] = autoCreate.PlanBacklog
			}
		}
		inputJSON, decisionJSON := construirPayloadCicloAutonomia(policy, proyecto, agente, contexto, "supervision", decision)
		instruction := construirInstruccionSupervision(policy, proyecto, agente, resumen)
		if encolada, err := encolarNudgeAutonomiaDetallado(agente.Nombre, proyecto, "supervisar_proyecto", resumen, instruction, map[string]any{
			"objetivo_general":         strings.TrimSpace(policy.ObjetivoGeneral),
			"definition_of_done":       strings.TrimSpace(policy.DefinitionOfDoneJSON),
			"estado_autonomia":         strings.TrimSpace(string(policy.EstadoAutonomia)),
			"supervision_cycle_kind":   "supervision",
			"auto_create_tasks":        policy.AutoCreateTasks,
			"auto_created_work":        autoCreate.Created,
			"auto_created_task":        autoCreate.SeedTaskID > 0,
			"auto_created_task_id":     autoCreate.SeedTaskID,
			"auto_created_plan":        autoCreate.PlanCreated > 0,
			"auto_created_tasks_count": autoCreate.PlanCreated,
			"auto_created_backlog":     autoCreate.PlanBacklog,
		}); err != nil {
			return total, err
		} else if !encolada {
			continue
		}
		if _, err := supervisionService.RegisterCycle(proyecto.Slug, supervisionapp.CycleInput{
			Kind:         "supervision",
			Agente:       agente.Nombre,
			InputJSON:    inputJSON,
			DecisionJSON: decisionJSON,
			Resultado:    "encolado",
		}); err != nil {
			return total, err
		}
		if err := supervisionService.MarkSupervised(proyecto.Slug, now); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}

type autonomiaAutoCreateResult struct {
	Created     bool
	SeedTaskID  int64
	PlanCreated int
	PlanBacklog int
}

func asegurarTrabajoAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, agente *db.Agente) (autonomiaAutoCreateResult, error) {
	var zero autonomiaAutoCreateResult
	if policy == nil || !policy.Enabled || !policy.AutoCreateTasks || proyecto == nil || agente == nil {
		return zero, nil
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || terminado {
		return zero, err
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return zero, err
	}
	hasAnyTask := false
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		hasAnyTask = true
		switch tarea.Estado {
		case db.TareaCancelada, db.TareaCompletada:
			continue
		default:
			return zero, nil
		}
	}
	estadoAbierta := db.PropuestaAbierta
	propuestas, err := propuestasService.ListByProject(&estadoAbierta, proyecto.Slug)
	if err != nil {
		return zero, err
	}
	if len(propuestas) > 0 {
		return zero, nil
	}
	gates, err := reviewService.List(reviewapp.ListInput{ProyectoRef: proyecto.Slug, Limit: 20})
	if err != nil {
		return zero, err
	}
	if firstOpenGate(gates) != nil {
		return zero, nil
	}
	if !hasAnyTask {
		spec := construirSpecFactoryAutonomia(policy, proyecto)
		plan, err := newProjectAppFactory().Generate(spec)
		if err != nil {
			return zero, err
		}
		materialized, err := fabricaapp.Materialize(fabricaapp.DBStore{}, proyecto.ID, "orquesta", plan)
		if err != nil {
			return zero, err
		}
		primaryTaskID, err := arrancarPrimeraTareaPlanAutonomia(materialized.TaskIDs, strings.TrimSpace(agente.Nombre))
		if err != nil {
			return zero, err
		}
		return autonomiaAutoCreateResult{
			Created:     materialized.Created > 0,
			SeedTaskID:  primaryTaskID,
			PlanCreated: materialized.Created,
			PlanBacklog: materialized.Backlog,
		}, nil
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      "Autonomía: revisar backlog y abrir siguiente frente útil",
		Descripcion: construirDescripcionTareaSemillaAutonomia(policy, proyecto),
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Agente:      strings.TrimSpace(agente.Nombre),
		Proyecto:    proyecto.Slug,
		Notas:       "autonomia:auto_create_tasks",
	})
	if err != nil {
		return zero, err
	}
	if err := tareasService.Start(id, strings.TrimSpace(agente.Nombre)); err != nil {
		return zero, err
	}
	return autonomiaAutoCreateResult{
		Created:    true,
		SeedTaskID: id,
	}, nil
}

func construirSpecFactoryAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto) fabricaapp.AppSpec {
	texto := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(proyectoNombreAutonomia(proyecto)),
		strings.TrimSpace(proyectoSlugAutonomia(proyecto)),
		strings.TrimSpace(policy.ObjetivoGeneral),
		strings.TrimSpace(policy.DefinitionOfDoneJSON),
	}, " "))
	tipo := inferirTipoAppAutonomia(texto)
	if tipo == "web_api" {
		switch {
		case proyectoTieneArchivoAutonomia(proyecto, "package.json") && !proyectoTieneArchivoAutonomia(proyecto, "go.mod"):
			tipo = "web"
		case proyectoTieneArchivoAutonomia(proyecto, "go.mod") && !proyectoTieneArchivoAutonomia(proyecto, "package.json") && !proyectoTieneDirAutonomia(proyecto, "web", "ui", "frontend", "templates", "static"):
			tipo = "api"
		}
	}
	spec := fabricaapp.AppSpec{
		Nombre:      proyectoNombreAutonomia(proyecto),
		Descripcion: descripcionFactoryAutonomia(policy, proyecto),
		Tipo:        tipo,
		Frontend:    tipo == "web" || tipo == "web_api",
		API:         tipo == "api" || tipo == "web_api",
		Auth:        contieneAlguno(texto, "auth", "autentic", "oauth", "login", "permiso", "sesion"),
		Database: contieneAlguno(texto, "base de datos", "persistencia", "sqlite", "postgres", "mysql", "migracion", "migraciones") ||
			proyectoTieneDirAutonomia(proyecto, "db", "migrations", "schema") ||
			proyectoTieneArchivoAutonomia(proyecto, "orquesta.db"),
		Docker: contieneAlguno(texto, "docker", "compose", "container", "contenedor", "deploy", "kubernetes", "k8s") ||
			proyectoTieneArchivoAutonomia(proyecto, "Dockerfile", "docker-compose.yml", "compose.yml"),
		I18n: !contieneAlguno(texto, "sin i18n", "solo un idioma", "monolingue") ||
			proyectoTieneDirAutonomia(proyecto, "i18n", "locales", "translations"),
		Idiomas: []string{"es", "en"},
	}
	if tipo == "api" || tipo == "web_api" {
		spec.Database = true
		spec.Docker = true
	}
	return spec
}

func arrancarPrimeraTareaPlanAutonomia(taskIDs map[string]int64, agente string) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	for _, key := range []string{"briefing", "investigacion", "arquitectura", "frontend_base", "api_base", "cli_base"} {
		id := taskIDs[key]
		if id <= 0 {
			continue
		}
		if err := tareasService.Take(id, agente); err != nil {
			return 0, err
		}
		if err := tareasService.Start(id, agente); err != nil {
			return 0, err
		}
		return id, nil
	}
	return 0, nil
}

func proyectoNombreAutonomia(proyecto *db.Proyecto) string {
	if proyecto == nil {
		return "Proyecto"
	}
	if nombre := strings.TrimSpace(proyecto.Nombre); nombre != "" {
		return nombre
	}
	if slug := strings.TrimSpace(proyecto.Slug); slug != "" {
		return slug
	}
	return "Proyecto"
}

func proyectoSlugAutonomia(proyecto *db.Proyecto) string {
	if proyecto == nil {
		return ""
	}
	return strings.TrimSpace(proyecto.Slug)
}

func descripcionFactoryAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto) string {
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		return strings.TrimSpace(policy.ObjetivoGeneral)
	}
	return fmt.Sprintf("Backlog inicial generado automaticamente por Orquesta para %s.", proyectoNombreAutonomia(proyecto))
}

func inferirTipoAppAutonomia(texto string) string {
	switch {
	case contieneAlguno(texto, " cli ", " consola", "terminal", "linea de comandos", "comando "):
		return "cli"
	case contieneAlguno(texto, "web_api", "panel", "dashboard") || (contieneAlguno(texto, "web", "frontend", "ui") && contieneAlguno(texto, "api", "backend", "endpoint")):
		return "web_api"
	case contieneAlguno(texto, "frontend", "web", "ui", "panel"):
		return "web"
	case contieneAlguno(texto, "api", "backend", "endpoint", "servicio"):
		return "api"
	default:
		return "web_api"
	}
}

func contieneAlguno(texto string, needles ...string) bool {
	for _, needle := range needles {
		needle = strings.ToLower(strings.TrimSpace(needle))
		if needle != "" && strings.Contains(texto, needle) {
			return true
		}
	}
	return false
}

func proyectoTieneArchivoAutonomia(proyecto *db.Proyecto, names ...string) bool {
	if proyecto == nil || strings.TrimSpace(proyecto.RutaAbs) == "" {
		return false
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(proyecto.RutaAbs, filepath.Clean(name)))
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func proyectoTieneDirAutonomia(proyecto *db.Proyecto, names ...string) bool {
	if proyecto == nil || strings.TrimSpace(proyecto.RutaAbs) == "" {
		return false
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(proyecto.RutaAbs, filepath.Clean(name)))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func resolverContextoReviewProyecto(proyectoID int64) (*int64, *int64, error) {
	if proyectoID <= 0 {
		return nil, nil, nil
	}
	tareaID, err := seleccionarTareaReviewProyecto(proyectoID)
	if err != nil {
		return nil, nil, err
	}
	worktreeID, err := seleccionarWorktreeReviewProyecto(proyectoID, tareaID)
	if err != nil {
		return nil, nil, err
	}
	return tareaID, worktreeID, nil
}

func seleccionarTareaReviewProyecto(proyectoID int64) (*int64, error) {
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return nil, err
	}
	var seleccion *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		if !tareaAptaParaReview(tarea) {
			continue
		}
		if seleccion == nil || prioridadTareaReview(tarea) > prioridadTareaReview(seleccion) || (prioridadTareaReview(tarea) == prioridadTareaReview(seleccion) && tarea.ID > seleccion.ID) {
			seleccion = tarea
		}
	}
	if seleccion == nil {
		return nil, nil
	}
	return &seleccion.ID, nil
}

func tareaAptaParaReview(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	switch tarea.Estado {
	case db.TareaCompletada, db.TareaEnProgreso, db.TareaAsignada:
		return true
	default:
		return false
	}
}

func prioridadTareaReview(tarea *db.Tarea) int {
	if tarea == nil {
		return 0
	}
	switch tarea.Estado {
	case db.TareaCompletada:
		return 3
	case db.TareaEnProgreso:
		return 2
	case db.TareaAsignada:
		return 1
	default:
		return 0
	}
}

func seleccionarWorktreeReviewProyecto(proyectoID int64, tareaID *int64) (*int64, error) {
	state := coordinacion.WorktreeActive
	worktrees, err := db.ListarWorktreesCoordRaw(coordinacion.WorktreeFilter{ProjectID: &proyectoID, State: &state})
	if err != nil {
		return nil, err
	}
	var fallback *coordinacion.Worktree
	for _, worktree := range worktrees {
		if worktree == nil {
			continue
		}
		if fallback == nil || worktree.ID > fallback.ID {
			fallback = worktree
		}
		if tareaID != nil && worktree.TaskID != nil && *worktree.TaskID == *tareaID {
			id := worktree.ID
			return &id, nil
		}
	}
	if fallback == nil {
		return nil, nil
	}
	id := fallback.ID
	return &id, nil
}

func asegurarSolicitudMergeDesdeGateAprobado(proyecto *db.Proyecto, gate *reviewapp.Gate) (bool, error) {
	if proyecto == nil || gate == nil || gate.WorktreeID == nil || *gate.WorktreeID <= 0 {
		return false, nil
	}
	svc := gitgobernanza.NewService(gitgobernanza.Repository{})
	worktree, err := svc.GetWorktree(*gate.WorktreeID)
	if err != nil || worktree == nil {
		return false, err
	}
	sourceBranch := strings.TrimSpace(worktree.Branch)
	if sourceBranch == "" {
		return false, nil
	}
	targetBranch := strings.TrimSpace(worktree.BaseRef)
	if targetBranch == "" {
		targetBranch = "main"
	}
	existing, err := svc.ListRequests(proyecto.Slug, "")
	if err != nil {
		return false, err
	}
	for _, item := range existing {
		if item == nil {
			continue
		}
		if strings.TrimSpace(item.SourceBranch) != sourceBranch || strings.TrimSpace(item.TargetBranch) != targetBranch {
			continue
		}
		switch strings.TrimSpace(item.Estado) {
		case "rechazado", "cancelado", "fallido":
			continue
		default:
			return false, nil
		}
	}
	metadataJSON, err := json.Marshal(map[string]any{
		"auto_created": true,
		"source":       "review_gate_approved",
		"review_gate":  gate.ID,
		"worktree_id":  gate.WorktreeID,
		"tarea_id":     gate.TareaID,
	})
	if err != nil {
		return false, err
	}
	if _, err := svc.SaveRequest(gitgobernanza.SaveMergeRequestInput{
		ProyectoSlug: proyecto.Slug,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		RequestedBy:  valorConFallback(strings.TrimSpace(gate.ReviewerAgente), "orquesta"),
		Estado:       "aprobado",
		Notas:        fmt.Sprintf("Solicitud creada automáticamente tras aprobar review gate #%d.", gate.ID),
		MetadataJSON: string(metadataJSON),
	}); err != nil {
		return false, err
	}
	return true, nil
}

func construirDescripcionTareaSemillaAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto) string {
	partes := []string{
		"Tarea semilla creada automáticamente por Orquesta porque el proyecto autónomo no tenía ningún frente abierto.",
		"Objetivo: revisar backlog, contexto, transcript, propuestas y estado real del proyecto para abrir el siguiente frente útil sin intervención humana.",
	}
	if proyecto != nil && strings.TrimSpace(proyecto.Slug) != "" {
		partes = append(partes, "Proyecto: "+strings.TrimSpace(proyecto.Slug)+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		partes = append(partes, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		partes = append(partes, "Definition of done: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	partes = append(partes, "Si el proyecto está realmente terminado, no abras trabajo artificial: deja trazabilidad y permite el cierre autónomo.")
	return strings.Join(partes, " ")
}

func procesarReviewGatesBatch() (int, error) {
	policies, err := supervisionService.ListEnabledPolicies()
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	interval := time.Duration(controlPlaneConfigIntOrDefault("autonomia_review_interval_seconds", 300)) * time.Second
	total := 0
	for _, policy := range policies {
		if policy == nil || !policy.Enabled || !policy.ReviewRequired {
			continue
		}
		proyecto, err := db.GetProyecto(strconv.FormatInt(policy.ProyectoID, 10))
		if err != nil {
			return total, err
		}
		ready, resumenReady, err := proyectoSinTrabajoPendiente(proyecto)
		if err != nil {
			return total, err
		}
		if !ready {
			if policy.EstadoAutonomia == db.AutonomiaProyectoEsperandoReview {
				if err := persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoActiva); err != nil {
					return total, err
				}
			}
			continue
		}
		gates, err := reviewService.List(reviewapp.ListInput{ProyectoRef: proyecto.Slug, Limit: 20})
		if err != nil {
			return total, err
		}
		if approved := latestApprovedGate(gates); approved != nil {
			mergeCreated, err := asegurarSolicitudMergeDesdeGateAprobado(proyecto, approved)
			if err != nil {
				return total, err
			}
			when := now
			if approved.ResolvedAt != nil {
				when = approved.ResolvedAt.UTC()
			}
			if policy.LastReviewAt == nil || policy.LastReviewAt.Before(when) {
				if err := supervisionService.MarkReviewed(proyecto.Slug, when); err != nil {
					return total, err
				}
			}
			if policy.EstadoAutonomia == db.AutonomiaProyectoEsperandoReview {
				if err := persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoActiva); err != nil {
					return total, err
				}
			}
			if mergeCreated {
				total++
			}
			continue
		}
		if gate := firstOpenGate(gates); gate != nil {
			switch strings.TrimSpace(gate.Estado) {
			case reviewapp.GateStateChangesAsked:
				if err := persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoActiva); err != nil {
					return total, err
				}
				due, err := autonomiaCycleDue(proyecto.Slug, "review_feedback", now, interval)
				if err != nil {
					return total, err
				}
				if !due {
					continue
				}
				corrector, err := seleccionarAgenteCorreccionReview(proyecto.ID, policy, strings.TrimSpace(gate.ReviewerAgente))
				if err != nil {
					return total, err
				}
				if corrector == nil {
					continue
				}
				if pendiente, err := existeRuntimeOrderAutonomiaPendiente(corrector.Nombre, &proyecto.ID, "nudge", "resolver_review_feedback"); err != nil {
					return total, err
				} else if pendiente {
					continue
				}
				reopened, err := reviewService.Update(reviewapp.UpdateGateInput{
					ID:     gate.ID,
					Estado: reviewapp.GateStateInReview,
				})
				if err != nil {
					return total, err
				}
				gate = reopened
				contexto, resumenCtx := db.BuildProjectContextSummary(corrector.Nombre, proyecto)
				inputJSON, decisionJSON := construirPayloadCicloAutonomia(policy, proyecto, corrector, contexto, "review_feedback", map[string]any{
					"accion":   "resolver_review_feedback",
					"gate_id":  gate.ID,
					"reviewer": strings.TrimSpace(gate.ReviewerAgente),
					"motivo":   "review solicitó cambios",
				})
				instruction := construirInstruccionCorreccionReview(policy, proyecto, corrector.Nombre, gate)
				if encolada, err := encolarNudgeAutonomiaDetallado(corrector.Nombre, proyecto, "resolver_review_feedback", fmt.Sprintf("gate=%d; review solicitó cambios", gate.ID), instruction, map[string]any{
					"gate_id":            gate.ID,
					"reviewer_agente":    strings.TrimSpace(gate.ReviewerAgente),
					"objetivo_general":   strings.TrimSpace(policy.ObjetivoGeneral),
					"definition_of_done": strings.TrimSpace(policy.DefinitionOfDoneJSON),
					"review_findings":    strings.TrimSpace(gate.FindingsJSON),
				}); err != nil {
					return total, err
				} else if !encolada {
					continue
				}
				if _, err := supervisionService.RegisterCycle(proyecto.Slug, supervisionapp.CycleInput{
					Kind:         "review_feedback",
					Agente:       corrector.Nombre,
					InputJSON:    inputJSON,
					DecisionJSON: decisionJSON,
					Resultado:    "encolado",
				}); err != nil {
					return total, err
				}
				_ = resumenCtx
				total++
				continue
			case reviewapp.GateStateBlocked:
				if err := persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoEsperandoHumano); err != nil {
					return total, err
				}
				continue
			}
		}
		reviewer, activado, err := prepararAgentePreferidoAutonomia(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), "review_automatica")
		if err != nil {
			return total, err
		}
		if activado {
			continue
		}
		if reviewer == nil {
			reviewer, err = seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), []string{"revisor", "reviewer", "supervisor", "orquestador", "admin", "programador"}, "")
		}
		if err != nil {
			return total, err
		}
		if reviewer == nil {
			continue
		}
		reviewTaskID, reviewWorktreeID, err := resolverContextoReviewProyecto(proyecto.ID)
		if err != nil {
			return total, err
		}
		gate := firstOpenGate(gates)
		if gate == nil {
			created, err := reviewService.Create(reviewapp.CreateGateInput{
				ProyectoRef:     proyecto.Slug,
				TareaID:         reviewTaskID,
				WorktreeID:      reviewWorktreeID,
				RequestedBy:     "orquesta",
				ReviewerAgente:  reviewer.Nombre,
				SeverityMax:     "high",
				InitialFindings: "",
			})
			if err != nil {
				return total, err
			}
			gate = created
			total++
		} else if strings.TrimSpace(gate.ReviewerAgente) == "" || gate.TareaID == nil || gate.WorktreeID == nil {
			reviewerNombre := reviewer.Nombre
			update := reviewapp.UpdateGateInput{ID: gate.ID}
			if strings.TrimSpace(gate.ReviewerAgente) == "" {
				update.ReviewerAgente = &reviewerNombre
			}
			if gate.TareaID == nil {
				update.TareaID = reviewTaskID
			}
			if gate.WorktreeID == nil {
				update.WorktreeID = reviewWorktreeID
			}
			updated, err := reviewService.Update(update)
			if err != nil {
				return total, err
			}
			gate = updated
		}
		if err := persistirEstadoProyectoAutonomia(policy, db.AutonomiaProyectoEsperandoReview); err != nil {
			return total, err
		}
		reviewerNombre := strings.TrimSpace(gate.ReviewerAgente)
		if reviewerNombre == "" {
			reviewerNombre = reviewer.Nombre
		}
		due, err := autonomiaCycleDue(proyecto.Slug, "review", now, interval)
		if err != nil {
			return total, err
		}
		if !due {
			continue
		}
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(reviewerNombre, &proyecto.ID, "nudge", "ejecutar_review_gate"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		contexto, resumenCtx := db.BuildProjectContextSummary(reviewerNombre, proyecto)
		inputJSON, decisionJSON := construirPayloadCicloAutonomia(policy, proyecto, reviewer, contexto, "review", map[string]any{
			"accion":  "ejecutar_review_gate",
			"gate_id": gate.ID,
			"motivo":  resumenReady,
		})
		instruction := construirInstruccionReview(policy, proyecto, reviewerNombre, gate, resumenReady)
		if encolada, err := encolarNudgeAutonomiaDetallado(reviewerNombre, proyecto, "ejecutar_review_gate", fmt.Sprintf("gate=%d; %s", gate.ID, resumenReady), instruction, map[string]any{
			"gate_id":            gate.ID,
			"objetivo_general":   strings.TrimSpace(policy.ObjetivoGeneral),
			"definition_of_done": strings.TrimSpace(policy.DefinitionOfDoneJSON),
			"estado_autonomia":   strings.TrimSpace(string(policy.EstadoAutonomia)),
		}); err != nil {
			return total, err
		} else if !encolada {
			continue
		}
		if gate.Estado == reviewapp.GateStatePending {
			estado := reviewapp.GateStateInReview
			updated, err := reviewService.Update(reviewapp.UpdateGateInput{ID: gate.ID, Estado: estado})
			if err != nil {
				return total, err
			}
			gate = updated
		}
		if _, err := supervisionService.RegisterCycle(proyecto.Slug, supervisionapp.CycleInput{
			Kind:         "review",
			Agente:       reviewerNombre,
			InputJSON:    inputJSON,
			DecisionJSON: decisionJSON,
			Resultado:    "encolado",
		}); err != nil {
			return total, err
		}
		_ = resumenCtx
		total++
	}
	return total, nil
}

func supervisionDue(policy *db.ProyectoAutonomia, now time.Time, interval time.Duration) bool {
	if policy == nil || !policy.Enabled {
		return false
	}
	switch policy.EstadoAutonomia {
	case db.AutonomiaProyectoEsperandoHumano, db.AutonomiaProyectoEsperandoReview, db.AutonomiaProyectoCerrando, db.AutonomiaProyectoCerrado:
		return false
	}
	if policy.LastSupervisionAt == nil {
		return true
	}
	return now.Sub(policy.LastSupervisionAt.UTC()) >= interval
}

func supervisorAutonomiaYaOperativo(agente string, proyectoID int64) (bool, error) {
	if strings.TrimSpace(agente) == "" || proyectoID <= 0 {
		return false, nil
	}
	sesion, err := db.GetSesionActivaOperativa(strings.TrimSpace(agente), &proyectoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	handle, err := resolverHandleControlAgente(strings.TrimSpace(agente), &proyectoID)
	if err != nil {
		return false, err
	}
	return sesion != nil && handle != nil, nil
}

func supervisorAutonomiaTieneTrabajoActivo(agente string, proyectoID int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
	})
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			return true, nil
		}
	}
	return false, nil
}

func persistirEstadoProyectoAutonomia(policy *db.ProyectoAutonomia, estado db.EstadoAutonomiaProyecto) error {
	if policy == nil || policy.EstadoAutonomia == estado {
		return nil
	}
	cp := *policy
	cp.EstadoAutonomia = estado
	if _, err := db.UpsertProyectoAutonomia(&cp); err != nil {
		return err
	}
	policy.EstadoAutonomia = estado
	return nil
}

func seleccionarAgenteActivoProyecto(proyectoID int64, preferredRoles []string, exclude string) (*db.Agente, error) {
	return seleccionarAgenteActivoProyectoPreferido(proyectoID, "", preferredRoles, exclude)
}

func prepararAgentePreferidoAutonomia(proyectoID int64, preferredAgent, activationReason string) (*db.Agente, bool, error) {
	agente, activado, err := asegurarAgenteAutonomiaOperativo(proyectoID, preferredAgent, activationReason)
	if err != nil {
		return nil, false, err
	}
	if activado {
		return nil, true, nil
	}
	return agente, false, nil
}

func asegurarAgenteAutonomiaOperativo(proyectoID int64, preferredAgent, activationReason string) (*db.Agente, bool, error) {
	preferredAgent = strings.TrimSpace(preferredAgent)
	if preferredAgent == "" || proyectoID <= 0 {
		return nil, false, nil
	}
	agente, err := resolverAgenteAutonomiaOperativo(preferredAgent)
	if err != nil {
		return nil, false, err
	}
	if agente == nil || !agente.Habilitado {
		return nil, false, nil
	}
	preferredAgent = strings.TrimSpace(agente.Nombre)
	estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
	if estadoCuota != "" && estadoCuota != "activo" {
		return nil, false, nil
	}
	proyectoActivoID, err := db.ObtenerProyectoActivoAgente(preferredAgent)
	if err != nil {
		return nil, false, err
	}
	if proyectoActivoID != 0 && proyectoActivoID != proyectoID {
		return nil, false, nil
	}
	if sesion, err := db.GetSesionActiva(preferredAgent, &proyectoID); err != nil && err != sql.ErrNoRows {
		return nil, false, err
	} else if sesion != nil {
		return agente, false, nil
	}
	if handle, err := db.GetRuntimeHandleActivoAgenteProyecto(preferredAgent, &proyectoID); err != nil {
		return nil, false, err
	} else if handle != nil {
		return agente, false, nil
	}
	if err := db.ActivarAsignacion(preferredAgent, proyectoID, activationReason); err != nil {
		return nil, false, err
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return nil, false, err
	}
	if proyecto == nil {
		return nil, false, nil
	}
	if err := db.EncolarStartAutomaticoSiHaceFalta(preferredAgent, proyecto, activationReason); err != nil {
		return nil, false, err
	}
	return agente, true, nil
}

func resolverAgenteAutonomiaOperativo(nombre string) (*db.Agente, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return nil, nil
	}
	agente, err := db.GetAgente(nombre)
	if err == nil || err != sql.ErrNoRows {
		return agente, err
	}
	agentes, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	var candidato *db.Agente
	for _, item := range agentes {
		if item == nil || !strings.EqualFold(strings.TrimSpace(item.Nombre), nombre) {
			continue
		}
		if candidato != nil {
			return nil, fmt.Errorf("agente ambiguo para autonomia: %s", nombre)
		}
		candidato = item
	}
	if candidato == nil {
		return nil, sql.ErrNoRows
	}
	return candidato, nil
}

func resolverSupervisorAutonomiaOperativo(proyectoID int64, policy *db.ProyectoAutonomia, activationReason, exclude string) (*db.Agente, bool, error) {
	if policy == nil || !policy.Enabled || proyectoID <= 0 {
		return nil, false, nil
	}
	preferred := strings.TrimSpace(policy.SupervisorAgente)
	if preferred != "" && !strings.EqualFold(preferred, strings.TrimSpace(exclude)) {
		if agente, activado, err := asegurarAgenteAutonomiaOperativo(proyectoID, preferred, activationReason); err != nil {
			return nil, false, err
		} else if agente != nil {
			return agente, activado, nil
		}
	}
	candidato, err := db.SeleccionarSupervisorAutonomiaOperativo(proyectoID, exclude)
	if err != nil || candidato == nil {
		return nil, false, err
	}
	if preferred != "" && strings.EqualFold(strings.TrimSpace(candidato.Nombre), preferred) {
		return nil, false, nil
	}
	return asegurarAgenteAutonomiaOperativo(proyectoID, strings.TrimSpace(candidato.Nombre), activationReason)
}

func seleccionarAgenteActivoProyectoPreferido(proyectoID int64, preferredAgent string, preferredRoles []string, exclude string) (*db.Agente, error) {
	sesiones, err := db.ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	type candidato struct {
		agente *db.Agente
		score  int
	}
	var candidatos []candidato
	seen := map[string]struct{}{}
	exclude = strings.TrimSpace(exclude)
	preferredAgent = strings.TrimSpace(preferredAgent)
	for _, sesion := range sesiones {
		if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
			continue
		}
		nombre := strings.TrimSpace(sesion.Agente)
		if nombre == "" || nombre == exclude {
			continue
		}
		if _, ok := seen[nombre]; ok {
			continue
		}
		seen[nombre] = struct{}{}
		agente, err := db.GetAgente(nombre)
		if err != nil {
			return nil, err
		}
		if agente == nil || !agente.Habilitado {
			continue
		}
		if preferredAgent != "" && strings.EqualFold(nombre, preferredAgent) {
			return agente, nil
		}
		candidatos = append(candidatos, candidato{agente: agente, score: roleScore(agente.Rol, preferredRoles)})
	}
	sort.Slice(candidatos, func(i, j int) bool {
		if candidatos[i].score != candidatos[j].score {
			return candidatos[i].score < candidatos[j].score
		}
		return strings.TrimSpace(candidatos[i].agente.Nombre) < strings.TrimSpace(candidatos[j].agente.Nombre)
	})
	if len(candidatos) == 0 {
		return nil, nil
	}
	return candidatos[0].agente, nil
}

func roleScore(role string, preferredRoles []string) int {
	role = strings.ToLower(strings.TrimSpace(role))
	for idx, candidate := range preferredRoles {
		if role == strings.ToLower(strings.TrimSpace(candidate)) {
			return idx
		}
	}
	return len(preferredRoles) + 1
}

func autonomiaCycleDue(projectRef string, kind string, now time.Time, interval time.Duration) (bool, error) {
	if interval <= 0 {
		return true, nil
	}
	list, err := supervisionService.ListCycles(projectRef, &kind, 1)
	if err != nil {
		return false, err
	}
	if len(list) == 0 || list[0] == nil {
		return true, nil
	}
	return now.Sub(list[0].CreatedAt.UTC()) >= interval, nil
}

func firstOpenGate(gates []*reviewapp.Gate) *reviewapp.Gate {
	for _, gate := range gates {
		if gate == nil {
			continue
		}
		switch strings.TrimSpace(gate.Estado) {
		case reviewapp.GateStatePending, reviewapp.GateStateInReview, reviewapp.GateStateChangesAsked, reviewapp.GateStateBlocked:
			return gate
		}
	}
	return nil
}

func latestApprovedGate(gates []*reviewapp.Gate) *reviewapp.Gate {
	var approved []*reviewapp.Gate
	for _, gate := range gates {
		if gate == nil || strings.TrimSpace(gate.Estado) != reviewapp.GateStateApproved {
			continue
		}
		approved = append(approved, gate)
	}
	sort.Slice(approved, func(i, j int) bool {
		return approved[i].ID > approved[j].ID
	})
	if len(approved) == 0 {
		return nil
	}
	return approved[0]
}

func seleccionarAgenteCorreccionReview(proyectoID int64, policy *db.ProyectoAutonomia, reviewer string) (*db.Agente, error) {
	preferredSupervisor := ""
	if policy != nil {
		preferredSupervisor = strings.TrimSpace(policy.SupervisorAgente)
	}
	preferredRoles := []string{"supervisor", "orquestador", "programador", "admin", "revisor", "reviewer"}
	agente, err := seleccionarAgenteActivoProyectoPreferido(proyectoID, preferredSupervisor, preferredRoles, reviewer)
	if err != nil {
		return nil, err
	}
	if agente != nil {
		return agente, nil
	}
	if strings.TrimSpace(reviewer) == "" {
		return nil, nil
	}
	return seleccionarAgenteActivoProyectoPreferido(proyectoID, preferredSupervisor, preferredRoles, "")
}

func proyectoSinTrabajoPendiente(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return false, "", err
	}
	total := 0
	abiertas := 0
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCancelada, db.TareaCompletada:
			total++
		default:
			total++
			abiertas++
		}
	}
	if total == 0 || abiertas > 0 {
		return false, "", nil
	}
	estadoAbierta := db.PropuestaAbierta
	propuestas, err := propuestasService.ListByProject(&estadoAbierta, proyecto.Slug)
	if err != nil {
		return false, "", err
	}
	if len(propuestas) > 0 {
		return false, "", nil
	}
	return true, fmt.Sprintf("sin trabajo pendiente (%d tarea(s) terminales, 0 propuestas abiertas)", total), nil
}

func proyectoReviewAutonomoCompletado(proyecto *db.Proyecto) (bool, string, error) {
	if proyecto == nil {
		return false, "", nil
	}
	policy, err := supervisionService.GetProjectPolicy(proyecto.Slug)
	if err != nil || policy == nil || !policy.Enabled || !policy.ReviewRequired {
		return true, "", err
	}
	gates, err := reviewService.List(reviewapp.ListInput{ProyectoRef: proyecto.Slug, Limit: 20})
	if err != nil {
		return false, "", err
	}
	if gate := firstOpenGate(gates); gate != nil {
		return false, fmt.Sprintf("esperando review gate #%d (%s)", gate.ID, strings.TrimSpace(gate.Estado)), nil
	}
	if gate := latestApprovedGate(gates); gate != nil {
		return true, fmt.Sprintf("review gate #%d aprobado", gate.ID), nil
	}
	return false, "esperando apertura/aprobación de review gate", nil
}

func construirPayloadCicloAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, agente *db.Agente, contexto map[string]any, kind string, decision map[string]any) (string, string) {
	input := map[string]any{
		"kind":              strings.TrimSpace(kind),
		"proyecto_id":       proyecto.ID,
		"proyecto_slug":     strings.TrimSpace(proyecto.Slug),
		"agente":            strings.TrimSpace(agente.Nombre),
		"agente_rol":        strings.TrimSpace(agente.Rol),
		"contexto_proyecto": contexto,
	}
	if policy != nil {
		input["objetivo_general"] = strings.TrimSpace(policy.ObjetivoGeneral)
		input["definition_of_done_json"] = strings.TrimSpace(policy.DefinitionOfDoneJSON)
		input["estado_autonomia"] = strings.TrimSpace(string(policy.EstadoAutonomia))
	}
	rawInput, _ := json.Marshal(input)
	rawDecision, _ := json.Marshal(decision)
	return string(rawInput), string(rawDecision)
}

func construirInstruccionSupervision(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, agente *db.Agente, resumen string) string {
	parts := []string{
		"Orquesta: actúa como supervisor autónomo del proyecto y mantén el trabajo alineado con la planificación aprobada.",
		"Revisa el estado real del proyecto y empuja el siguiente frente útil sin detenerte.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		parts = append(parts, "Definition of done JSON: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	if strings.TrimSpace(resumen) != "" {
		parts = append(parts, "Contexto actual: "+strings.TrimSpace(resumen)+".")
	}
	if policy != nil && policy.AutoCreateTasks {
		parts = append(parts, "Si falta trabajo, crea o reajusta tareas de forma autónoma para que el proyecto no se quede sin backlog útil.")
	} else {
		parts = append(parts, "Si detectas huecos de backlog, documenta el frente faltante y replanifica sin crear tareas nuevas automáticamente.")
	}
	parts = append(parts, "Respeta gobernanza efectiva, tests y arquitectura. Si el cambio es delicado, crea checkpoint y sigue.")
	return strings.Join(parts, " ")
}

func construirInstruccionReview(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, reviewer string, gate *reviewapp.Gate, resumen string) string {
	parts := []string{
		"Orquesta: actúa como revisor autónomo del proyecto.",
		"Inspecciona el código, valida arquitectura, tests, i18n y cumplimiento de reglas; documenta findings y aprueba o pide cambios sin bloquear por defecto.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if gate != nil {
		parts = append(parts, fmt.Sprintf("Review gate: #%d.", gate.ID))
	}
	if strings.TrimSpace(reviewer) != "" {
		parts = append(parts, "Revisor: "+strings.TrimSpace(reviewer)+".")
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		parts = append(parts, "Definition of done JSON: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	if strings.TrimSpace(resumen) != "" {
		parts = append(parts, "Contexto actual: "+strings.TrimSpace(resumen)+".")
	}
	parts = append(parts, "Si detectas defectos reales, pide cambios concretos; si está correcto, aprueba y deja trazabilidad.")
	return strings.Join(parts, " ")
}

func construirInstruccionCorreccionReview(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, agente string, gate *reviewapp.Gate) string {
	parts := []string{
		"Orquesta: aplica de forma autónoma los cambios pedidos en la revisión y deja el proyecto listo para re-review sin esperar a un humano.",
		"Corrige los findings reales, crea o reajusta tareas si hacen falta, ejecuta los tests pertinentes y preserva la arquitectura y la gobernanza efectiva.",
	}
	if proyecto != nil {
		parts = append(parts, fmt.Sprintf("Proyecto: %s.", strings.TrimSpace(proyecto.Slug)))
	}
	if strings.TrimSpace(agente) != "" {
		parts = append(parts, "Agente responsable: "+strings.TrimSpace(agente)+".")
	}
	if gate != nil {
		parts = append(parts, fmt.Sprintf("Review gate: #%d.", gate.ID))
		if strings.TrimSpace(gate.ReviewerAgente) != "" {
			parts = append(parts, "Revisor origen: "+strings.TrimSpace(gate.ReviewerAgente)+".")
		}
		if strings.TrimSpace(gate.FindingsJSON) != "" {
			parts = append(parts, "Findings actuales: "+strings.TrimSpace(gate.FindingsJSON)+".")
		}
	}
	if policy != nil && strings.TrimSpace(policy.ObjetivoGeneral) != "" {
		parts = append(parts, "Objetivo general: "+strings.TrimSpace(policy.ObjetivoGeneral)+".")
	}
	if policy != nil && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "" && strings.TrimSpace(policy.DefinitionOfDoneJSON) != "{}" {
		parts = append(parts, "Definition of done JSON: "+strings.TrimSpace(policy.DefinitionOfDoneJSON)+".")
	}
	parts = append(parts, "Cuando cierres este frente, deja el estado listo para que el reviewer pueda retomar la verificación final.")
	return strings.Join(parts, " ")
}

func controlPlaneConfigIntOrDefault(clave string, fallback int) int {
	v, err := db.ConfigGet(clave)
	if err != nil {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
