package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
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
		agente, err := seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.SupervisorAgente), []string{"supervisor", "orquestador", "revisor", "reviewer", "admin", "programador"}, "")
		if err != nil {
			return total, err
		}
		if agente == nil {
			continue
		}
		if pendiente, err := existeRuntimeOrderAutonomiaPendiente(agente.Nombre, &proyecto.ID, "nudge", "supervisar_proyecto"); err != nil {
			return total, err
		} else if pendiente {
			continue
		}
		contexto, resumen := db.BuildProjectContextSummary(strings.TrimSpace(agente.Nombre), proyecto)
		inputJSON, decisionJSON := construirPayloadCicloAutonomia(policy, proyecto, agente, contexto, "supervision", map[string]any{
			"accion": "supervisar_proyecto",
			"motivo": resumen,
		})
		instruction := construirInstruccionSupervision(policy, proyecto, agente, resumen)
		if err := encolarNudgeAutonomiaDetallado(agente.Nombre, proyecto, "supervisar_proyecto", resumen, instruction, map[string]any{
			"objetivo_general":       strings.TrimSpace(policy.ObjetivoGeneral),
			"definition_of_done":     strings.TrimSpace(policy.DefinitionOfDoneJSON),
			"estado_autonomia":       strings.TrimSpace(string(policy.EstadoAutonomia)),
			"supervision_cycle_kind": "supervision",
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
		reviewer, err := seleccionarAgenteActivoProyectoPreferido(proyecto.ID, strings.TrimSpace(policy.ReviewerAgente), []string{"revisor", "reviewer", "supervisor", "orquestador", "admin", "programador"}, "")
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
		"Revisa el estado real del proyecto, detecta huecos, crea o reajusta tareas si falta trabajo y empuja el siguiente frente útil sin detenerte.",
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
