package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
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
		agente, activado, err := prepararAgentePreferidoAutonomia(proyecto.ID, strings.TrimSpace(policy.SupervisorAgente), "supervision_automatica")
		if err != nil {
			return total, err
		}
		if activado {
			continue
		}
		if agente == nil {
			agente, err = seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.SupervisorAgente), []string{"supervisor", "orquestador", "revisor", "reviewer", "admin", "programador"}, "")
		}
		if err != nil {
			return total, err
		}
		if agente == nil {
			continue
		}
		taskCreated := false
		taskID := int64(0)
		if id, created, err := asegurarTareaSemillaAutonomia(policy, proyecto, agente); err != nil {
			return total, err
		} else if created {
			taskCreated = true
			taskID = id
		}
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(agente.Nombre, &proyecto.ID, "nudge", "supervisar_proyecto"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		contexto, resumen := db.BuildProjectContextSummary(strings.TrimSpace(agente.Nombre), proyecto)
		decision := map[string]any{
			"accion": "supervisar_proyecto",
			"motivo": resumen,
		}
		if taskCreated {
			decision["auto_created_task"] = true
			decision["auto_created_task_id"] = taskID
		}
		inputJSON, decisionJSON := construirPayloadCicloAutonomia(policy, proyecto, agente, contexto, "supervision", decision)
		instruction := construirInstruccionSupervision(policy, proyecto, agente, resumen)
		if err := encolarNudgeAutonomiaDetallado(agente.Nombre, proyecto, "supervisar_proyecto", resumen, instruction, map[string]any{
			"objetivo_general":       strings.TrimSpace(policy.ObjetivoGeneral),
			"definition_of_done":     strings.TrimSpace(policy.DefinitionOfDoneJSON),
			"estado_autonomia":       strings.TrimSpace(string(policy.EstadoAutonomia)),
			"supervision_cycle_kind": "supervision",
			"auto_create_tasks":      policy.AutoCreateTasks,
			"auto_created_task":      taskCreated,
			"auto_created_task_id":   taskID,
		}); err != nil {
			return total, err
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

func asegurarTareaSemillaAutonomia(policy *db.ProyectoAutonomia, proyecto *db.Proyecto, agente *db.Agente) (int64, bool, error) {
	if policy == nil || !policy.Enabled || !policy.AutoCreateTasks || proyecto == nil || agente == nil {
		return 0, false, nil
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil || terminado {
		return 0, false, err
	}
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return 0, false, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCancelada, db.TareaCompletada:
			continue
		default:
			return 0, false, nil
		}
	}
	estadoAbierta := db.PropuestaAbierta
	propuestas, err := propuestasService.ListByProject(&estadoAbierta, proyecto.Slug)
	if err != nil {
		return 0, false, err
	}
	if len(propuestas) > 0 {
		return 0, false, nil
	}
	gates, err := reviewService.List(reviewapp.ListInput{ProyectoRef: proyecto.Slug, Limit: 20})
	if err != nil {
		return 0, false, err
	}
	if firstOpenGate(gates) != nil {
		return 0, false, nil
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
		return 0, false, err
	}
	if err := tareasService.Start(id, strings.TrimSpace(agente.Nombre)); err != nil {
		return 0, false, err
	}
	return id, true, nil
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
				if err := encolarNudgeAutonomiaDetallado(corrector.Nombre, proyecto, "resolver_review_feedback", fmt.Sprintf("gate=%d; review solicitó cambios", gate.ID), instruction, map[string]any{
					"gate_id":            gate.ID,
					"reviewer_agente":    strings.TrimSpace(gate.ReviewerAgente),
					"objetivo_general":   strings.TrimSpace(policy.ObjetivoGeneral),
					"definition_of_done": strings.TrimSpace(policy.DefinitionOfDoneJSON),
					"review_findings":    strings.TrimSpace(gate.FindingsJSON),
				}); err != nil {
					return total, err
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
		gate := firstOpenGate(gates)
		if gate == nil {
			created, err := reviewService.Create(reviewapp.CreateGateInput{
				ProyectoRef:     proyecto.Slug,
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
		} else if strings.TrimSpace(gate.ReviewerAgente) == "" {
			reviewerNombre := reviewer.Nombre
			updated, err := reviewService.Update(reviewapp.UpdateGateInput{ID: gate.ID, ReviewerAgente: &reviewerNombre})
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
		if err := encolarNudgeAutonomiaDetallado(reviewerNombre, proyecto, "ejecutar_review_gate", fmt.Sprintf("gate=%d; %s", gate.ID, resumenReady), instruction, map[string]any{
			"gate_id":            gate.ID,
			"objetivo_general":   strings.TrimSpace(policy.ObjetivoGeneral),
			"definition_of_done": strings.TrimSpace(policy.DefinitionOfDoneJSON),
			"estado_autonomia":   strings.TrimSpace(string(policy.EstadoAutonomia)),
		}); err != nil {
			return total, err
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
	preferredAgent = strings.TrimSpace(preferredAgent)
	if preferredAgent == "" || proyectoID <= 0 {
		return nil, false, nil
	}
	agente, err := db.GetAgente(preferredAgent)
	if err != nil {
		return nil, false, err
	}
	if agente == nil || !agente.Habilitado {
		return nil, false, nil
	}
	if strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento") {
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
	return nil, true, nil
}

func seleccionarAgenteActivoProyectoPreferido(proyectoID int64, preferredAgent string, preferredRoles []string, exclude string) (*db.Agente, error) {
	sesiones, err := db.ListarSesionesActivas()
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
