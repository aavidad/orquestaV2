package cmd

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type apiProyectoCockpitResponse struct {
	Cockpit *apiProyectoCockpit `json:"cockpit"`
}

type apiProyectoCockpit struct {
	Proyecto                *db.Proyecto               `json:"proyecto,omitempty"`
	TareasPorEstado         map[string]int             `json:"tareas_por_estado"`
	TareasActivas           []tareaLite                `json:"tareas_activas,omitempty"`
	TareasReservadas        []tareaLite                `json:"tareas_reservadas,omitempty"`
	AgentesActivos          []apiProyectoCockpitAgente `json:"agentes_activos,omitempty"`
	AutonomyEvents          int                        `json:"autonomy_events"`
	AutonomyByKind          map[string]int             `json:"autonomy_by_kind,omitempty"`
	AutonomyLastAt          *time.Time                 `json:"autonomy_last_at,omitempty"`
	Autonomy                []autonomyEventSummary     `json:"autonomy,omitempty"`
	AutonomyHighlights      []string                   `json:"autonomy_highlights,omitempty"`
	IntegrationRisk         string                     `json:"integration_risk,omitempty"`
	IntegrationRiskScore    int                        `json:"integration_risk_score,omitempty"`
	IntegrationHighlights   []string                   `json:"integration_highlights,omitempty"`
	MailboxPendiente        []apiOpenClawMailboxLite   `json:"mailbox_pendiente,omitempty"`
	WorktreeDrift           []apiOpenClawWorktreeDrift `json:"worktree_drift,omitempty"`
	AsignacionesActivas     int                        `json:"asignaciones_activas"`
	PropuestasAbiertas      int                        `json:"propuestas_abiertas"`
	ReviewGatesAbiertas     int                        `json:"review_gates_abiertas"`
	RuntimeMailboxPendiente int                        `json:"runtime_mailbox_pendiente"`
	RuntimeOrdersAbiertas   int                        `json:"runtime_orders_abiertas"`
}

type apiProyectoCockpitAgente struct {
	Nombre string `json:"nombre"`
	Rol    string `json:"rol,omitempty"`
}

func buildProyectoCockpit(ref string) (*apiProyectoCockpit, error) {
	proyecto, err := db.GetProyectoConRutaEfectiva(strings.TrimSpace(ref), "")
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, nil
	}

	cockpit := &apiProyectoCockpit{
		Proyecto:        proyecto,
		TareasPorEstado: map[string]int{},
		AutonomyByKind:  map[string]int{},
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		estado := strings.TrimSpace(string(tarea.Estado))
		if estado == "" {
			estado = "desconocido"
		}
		cockpit.TareasPorEstado[estado]++
		item := tareaLite{
			ID:        tarea.ID,
			Titulo:    strings.TrimSpace(tarea.Titulo),
			Agente:    safeStringPtr(tarea.Agente),
			Estado:    tarea.Estado,
			Modulo:    strings.TrimSpace(tarea.Modulo),
			Prioridad: tarea.Prioridad,
		}
		switch tarea.Estado {
		case db.TareaEnProgreso:
			cockpit.TareasActivas = append(cockpit.TareasActivas, item)
		case db.TareaAsignada:
			cockpit.TareasReservadas = append(cockpit.TareasReservadas, item)
		}
	}

	estadoAsignacion := db.AsignacionActiva
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{
		ProyectoID: &proyecto.ID,
		Estado:     &estadoAsignacion,
	})
	if err != nil {
		return nil, err
	}
	cockpit.AsignacionesActivas = len(asignaciones)

	agentesActivos, err := listarAgentesActivosProyecto(proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.AgentesActivos = agentesActivos

	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	relevantAgents := projectRelevantAgentNames(proyecto.ID, tareas, status.AgentesActivos)
	cockpit.MailboxPendiente, err = buildProyectoPendingMailbox(proyecto.ID, relevantAgents)
	if err != nil {
		return nil, err
	}
	drift, err := buildOpenClawWorktreeDriftFromAPIStatus(status)
	if err != nil {
		return nil, err
	}
	cockpit.WorktreeDrift = filterOpenClawWorktreeDriftByAgents(drift, relevantAgents)

	cockpit.PropuestasAbiertas, err = db.CountProjectOpenProposals(proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.ReviewGatesAbiertas, err = db.CountProjectOpenReviewGates(proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.RuntimeMailboxPendiente, err = db.CountProjectPendingRuntimeMailbox(proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.RuntimeOrdersAbiertas, err = db.CountProjectOpenRuntimeOrders(proyecto.ID, "pendiente", "tomada", "ejecutando")
	if err != nil {
		return nil, err
	}
	autonomy, err := buildProjectAutonomyEventSummaries(proyecto.ID, time.Now().UTC().Add(-24*time.Hour), 50)
	if err != nil {
		return nil, err
	}
	cockpit.Autonomy, cockpit.AutonomyByKind, cockpit.AutonomyLastAt, cockpit.AutonomyEvents = compactAutonomyEventSummaries(autonomy, 8)
	cockpit.AutonomyHighlights = buildAutonomyHighlights(cockpit.AutonomyByKind, cockpit.Autonomy, cockpit.AutonomyLastAt, 3)
	if highlights, risk := buildWorkspaceBlockingHighlights(cockpit); risk > 0 {
		cockpit.IntegrationRiskScore = risk
		cockpit.IntegrationRisk = workspaceIntegrationRiskLabel(risk)
		cockpit.IntegrationHighlights = compactProjectControlIntegrationHighlights(highlights)
	}

	return cockpit, nil
}

func projectRelevantAgentNames(proyectoID int64, tareas []*db.Tarea, activos []*db.Agente) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		if agente := strings.ToLower(strings.TrimSpace(safeStringPtr(tarea.Agente))); agente != "" {
			out[agente] = struct{}{}
		}
	}
	for _, agente := range activos {
		if agente == nil {
			continue
		}
		if nombre := strings.ToLower(strings.TrimSpace(agente.Nombre)); nombre != "" {
			out[nombre] = struct{}{}
		}
	}
	sesiones, err := db.ListarSesionesActivasOperativas()
	if err == nil {
		for _, sesion := range sesiones {
			if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
				continue
			}
			if agente := strings.ToLower(strings.TrimSpace(sesion.Agente)); agente != "" {
				out[agente] = struct{}{}
			}
		}
	}
	return out
}

func buildProyectoPendingMailbox(proyectoID int64, relevantAgents map[string]struct{}) ([]apiOpenClawMailboxLite, error) {
	if proyectoID <= 0 {
		return nil, nil
	}
	estado := "pendiente"
	items, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		return nil, err
	}
	type agg struct {
		count             int
		kinds             map[string]bool
		supervisorActions map[string]bool
		oldest            *time.Time
	}
	byAgent := map[string]*agg{}
	for _, item := range items {
		if item == nil {
			continue
		}
		agente := strings.TrimSpace(item.ToAgente)
		if agente == "" {
			continue
		}
		if len(relevantAgents) > 0 {
			if _, ok := relevantAgents[strings.ToLower(agente)]; !ok {
				continue
			}
		}
		entry := byAgent[agente]
		if entry == nil {
			entry = &agg{kinds: map[string]bool{}, supervisorActions: map[string]bool{}}
			byAgent[agente] = entry
		}
		entry.count++
		if entry.oldest == nil || item.CreatedAt.Before(*entry.oldest) {
			ts := item.CreatedAt
			entry.oldest = &ts
		}
		if kind := strings.TrimSpace(item.Kind); kind != "" {
			entry.kinds[kind] = true
		}
		var payload struct {
			SupervisorAction string `json:"supervisor_action"`
		}
		if strings.TrimSpace(item.PayloadJSON) != "" && item.PayloadJSON != "{}" {
			if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err == nil {
				if action := strings.TrimSpace(payload.SupervisorAction); action != "" {
					entry.supervisorActions[action] = true
				}
			}
		}
	}
	out := make([]apiOpenClawMailboxLite, 0, len(byAgent))
	for agente, entry := range byAgent {
		kinds := make([]string, 0, len(entry.kinds))
		for kind := range entry.kinds {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		supervisorActions := make([]string, 0, len(entry.supervisorActions))
		for action := range entry.supervisorActions {
			supervisorActions = append(supervisorActions, action)
		}
		sort.Strings(supervisorActions)
		oldestAge := 0
		if entry.oldest != nil && !entry.oldest.IsZero() {
			oldestAge = int(time.Since(*entry.oldest).Minutes())
			if oldestAge < 0 {
				oldestAge = 0
			}
		}
		out = append(out, apiOpenClawMailboxLite{
			Agente:               agente,
			Count:                entry.count,
			Kinds:                kinds,
			KindsCSV:             strings.Join(kinds, ", "),
			SupervisorActions:    supervisorActions,
			SupervisorActionsCSV: strings.Join(supervisorActions, ", "),
			OldestAgeMin:         oldestAge,
		})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Agente) < strings.ToLower(out[j].Agente) })
	return out, nil
}

func filterOpenClawWorktreeDriftByAgents(items []apiOpenClawWorktreeDrift, relevantAgents map[string]struct{}) []apiOpenClawWorktreeDrift {
	if len(items) == 0 {
		return nil
	}
	if len(relevantAgents) == 0 {
		return items
	}
	out := make([]apiOpenClawWorktreeDrift, 0, len(items))
	for _, item := range items {
		if _, ok := relevantAgents[strings.ToLower(strings.TrimSpace(item.Agente))]; !ok {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func listarAgentesActivosProyecto(proyectoID int64) ([]apiProyectoCockpitAgente, error) {
	sesiones, err := db.ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	nombresProyecto := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
			continue
		}
		nombre := strings.TrimSpace(sesion.Agente)
		if nombre != "" {
			nombresProyecto[nombre] = struct{}{}
		}
	}
	if len(nombresProyecto) == 0 {
		return nil, nil
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	items := make([]apiProyectoCockpitAgente, 0, len(status.AgentesActivos))
	for _, agente := range status.AgentesActivos {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre == "" || !containsProyectoAgente(nombresProyecto, nombre) {
			continue
		}
		items = append(items, apiProyectoCockpitAgente{
			Nombre: nombre,
			Rol:    strings.TrimSpace(agente.Rol),
		})
	}
	if len(items) == 0 {
		names := make([]string, 0, len(nombresProyecto))
		for nombre := range nombresProyecto {
			names = append(names, strings.TrimSpace(nombre))
		}
		sort.Strings(names)
		items = make([]apiProyectoCockpitAgente, 0, len(names))
		for _, nombre := range names {
			if nombre == "" {
				continue
			}
			items = append(items, apiProyectoCockpitAgente{Nombre: nombre})
		}
	}
	return items, nil
}

func containsProyectoAgente(nombres map[string]struct{}, nombre string) bool {
	_, ok := nombres[strings.TrimSpace(nombre)]
	return ok
}

func safeStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
