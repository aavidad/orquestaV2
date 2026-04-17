/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package orquestacionagentesapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimesapp"
)

type AgentPreparer interface {
	BuildPrepare(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error)
}

type RuntimeController interface {
	EnqueueAgentControl(req runtimesapp.AgentControlRequest) (int64, string, error)
	GetProject(ref string) (*db.Proyecto, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	GetRuntimeBySessionID(sessionID int64) (*db.RuntimeInstance, error)
	GetRuntimeHandleBySessionID(sessionID int64) (*db.RuntimeHandle, error)
	GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error)
	GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error)
}

type AutonomyStore interface {
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
	GetPersistedAgentQuotaState(agente string) (string, error)
	GetSessionRuntimeHandle(sessionID int64) (*db.RuntimeHandle, error)
	ParkActiveSession(agente string, proyectoID *int64) error
	PauseAssignment(agente string, proyectoID int64, motivo string) error
}

type Service struct {
	agents        AgentPreparer
	runtimes      RuntimeController
	autonomyStore AutonomyStore
}

func NewService(agents AgentPreparer, runtimes RuntimeController) *Service {
	return &Service{agents: agents, runtimes: runtimes}
}

func (s *Service) SetAutonomyStore(store AutonomyStore) {
	if s == nil {
		return
	}
	s.autonomyStore = store
}

type ControlRequest struct {
	Agente       string
	Proyecto     string
	Accion       string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
	Motivo       string
	Por          string
	TareaID      *int64
}

type LaunchRequest struct {
	Agente       string
	Proyecto     string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
	Motivo       string
	Por          string
}

type LaunchResult struct {
	OrderID  int64
	Agente   string
	Proyecto agentesapp.ProjectBundle
	Conector agentesapp.ConnectorBundle
}

type NudgeRequest struct {
	Agente      string
	Proyecto    *db.Proyecto
	Accion      string
	Motivo      string
	Instruction string
	Extras      map[string]any
	Cooldown    time.Duration
}

func (s *Service) EnqueueControl(req ControlRequest) (int64, string, error) {
	if s == nil || s.runtimes == nil {
		return 0, "", fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.EnqueueAgentControl(runtimesapp.AgentControlRequest{
		Agente:       strings.TrimSpace(req.Agente),
		Proyecto:     strings.TrimSpace(req.Proyecto),
		Accion:       strings.TrimSpace(req.Accion),
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          strings.TrimSpace(req.Por),
		TareaID:      req.TareaID,
	})
}

func (s *Service) Launch(req LaunchRequest) (*LaunchResult, error) {
	if s == nil || s.agents == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	req.Agente = strings.TrimSpace(req.Agente)
	req.Proyecto = strings.TrimSpace(req.Proyecto)
	if req.Agente == "" || req.Proyecto == "" {
		return nil, fmt.Errorf("agente y proyecto son obligatorios")
	}
	prep, err := s.agents.BuildPrepare(agentesapp.PrepareInput{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
	})
	if err != nil {
		return nil, err
	}
	por := strings.TrimSpace(req.Por)
	if por == "" {
		por = "orquesta"
	}
	orderID, _, err := s.EnqueueControl(ControlRequest{
		Agente:       req.Agente,
		Proyecto:     req.Proyecto,
		Accion:       "start",
		Conector:     prep.Conector.Slug,
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          por,
	})
	if err != nil {
		return nil, err
	}
	return &LaunchResult{
		OrderID:  orderID,
		Agente:   prep.Agente,
		Proyecto: prep.Proyecto,
		Conector: prep.Conector,
	}, nil
}

func (s *Service) ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveControlHandle(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveDeliveryHandle(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.ResolveTranscriptDeliveryHandle(item)
}

func (s *Service) ResolveRecoveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	var (
		porProyecto *db.RuntimeHandle
		fallback    *db.RuntimeHandle
	)
	handles, err := s.runtimes.ListRuntimeHandles(&agente)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		if handle == nil || runtimeHandleRecoverySkipsLegacyATMUX(handle) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo", "pausado", "fallido":
		default:
			continue
		}
		if proyectoID != nil && *proyectoID > 0 && handle.ProyectoID != nil && *handle.ProyectoID == *proyectoID {
			if runtimeHandlePreferredForRecovery(handle, porProyecto) {
				porProyecto = handle
			}
			continue
		}
		if runtimeHandlePreferredForRecovery(handle, fallback) {
			fallback = handle
		}
	}
	if porProyecto != nil {
		return porProyecto, nil
	}
	return fallback, nil
}

func (s *Service) ResolveSessionRecoveryTarget(sesion *db.Sesion) (*db.RuntimeHandle, *db.RuntimeInstance, error) {
	if s == nil || s.runtimes == nil {
		return nil, nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil {
		return nil, nil, nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	proyectoID := sesion.ProyectoID
	var (
		handle        *db.RuntimeHandle
		sessionHandle *db.RuntimeHandle
		runtime       *db.RuntimeInstance
		err           error
	)
	if sesion.ID > 0 {
		handle, err = s.runtimes.GetRuntimeHandleBySessionID(sesion.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
		sessionHandle = handle
		runtime, err = s.runtimes.GetRuntimeBySessionID(sesion.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
	}
	if handle == nil {
		handle, err = s.ResolveRecoveryHandle(agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
	} else if agente != "" {
		candidate, err := s.ResolveRecoveryHandle(agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
		if runtimeHandlePreferredForRecovery(candidate, handle) {
			handle = candidate
		}
	}
	if handle != nil && runtime != nil && sessionHandle != nil && handle.ID != sessionHandle.ID {
		runtime = nil
	} else if handle != nil && runtime != nil && handle.SesionID != nil && sesion.ID > 0 && *handle.SesionID != sesion.ID {
		runtime = nil
	}
	if runtime == nil {
		runtime, err = s.resolveCanonicalRuntimeForHandle(handle, agente, proyectoID)
		if err != nil {
			return nil, nil, err
		}
	}
	if runtime == nil && sesion.ID > 0 {
		runtime, err = s.runtimes.GetRuntimeBySessionID(sesion.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
	}
	return handle, runtime, nil
}

func (s *Service) PauseAutonomyAlreadySatisfied(agente string, proyectoID *int64, sesion *db.Sesion) (bool, error) {
	if s == nil || s.runtimes == nil || s.autonomyStore == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	if pending, err := s.existsPendingAutonomyOrder(agente, proyectoID, "pause", ""); err != nil {
		return false, err
	} else if pending {
		return true, nil
	}
	if recent, err := s.existsRecentAutonomyOrder(agente, proyectoID, "pause", "", 2*time.Minute); err != nil {
		return false, err
	} else if recent {
		return true, nil
	}
	if sesion != nil && strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		return true, nil
	}
	handleSesion, err := s.sessionRuntimeHandle(sesion)
	if err != nil {
		return false, err
	}
	estadoCuota, err := s.autonomyStore.GetPersistedAgentQuotaState(agente)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(estadoCuota)) {
	case "enfriamiento", "agotado":
		if runtimeHandleStatePaused(handleSesion) || handleSesion == nil {
			return true, nil
		}
		handle, err := s.ResolveControlHandle(agente, proyectoID)
		if err != nil {
			return false, err
		}
		if handle == nil || strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
			return true, nil
		}
	}
	if runtimeHandleStatePaused(handleSesion) {
		return true, nil
	}
	handle, err := s.ResolveControlHandle(agente, proyectoID)
	if err != nil {
		return false, err
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return true, nil
	}
	return false, nil
}

func (s *Service) PauseAutonomySessionIfNeeded(sesion *db.Sesion, proyecto *db.Proyecto, motivo string) (int, error) {
	if s == nil || s.runtimes == nil || s.autonomyStore == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil || sesion.ProyectoID == nil || proyecto == nil {
		return 0, nil
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return 0, nil
	}
	if strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		if err := s.autonomyStore.ParkActiveSession(sesion.Agente, sesion.ProyectoID); err != nil {
			return 0, err
		}
		if err := s.autonomyStore.PauseAssignment(sesion.Agente, proyecto.ID, motivo); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if satisfied, err := s.PauseAutonomyAlreadySatisfied(sesion.Agente, &proyecto.ID, sesion); err != nil {
		return 0, err
	} else if satisfied {
		return 0, nil
	}
	if _, _, err := s.EnqueueControl(ControlRequest{
		Agente:   strings.TrimSpace(sesion.Agente),
		Proyecto: strings.TrimSpace(proyecto.Slug),
		Accion:   "pause",
		Motivo:   motivo,
		Por:      "orquesta",
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func (s *Service) EnqueueAutonomyNudge(req NudgeRequest) (bool, error) {
	if s == nil || s.runtimes == nil || s.autonomyStore == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	proyecto := req.Proyecto
	if proyecto == nil {
		return false, nil
	}
	agente := strings.TrimSpace(req.Agente)
	accion := strings.TrimSpace(req.Accion)
	if agente == "" || accion == "" || proyecto.ID <= 0 {
		return false, nil
	}
	if abierta, err := s.existsOpenAutonomyOrder(agente, &proyecto.ID, "send_instruction"); err != nil {
		return false, err
	} else if abierta {
		return false, nil
	}
	cooldown := req.Cooldown
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	if reciente, err := s.existsRecentAutonomyOrder(agente, &proyecto.ID, "nudge", "", cooldown); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	var runtimeID *int64
	var handleID *int64
	handle, err := s.ResolveDeliveryHandle(agente, &proyecto.ID)
	if err != nil {
		return false, err
	}
	if handle != nil {
		handleID = &handle.ID
		runtimeID, err = s.resolveCanonicalRuntimeIDForHandle(handle, agente, &proyecto.ID)
		if err != nil {
			return false, err
		}
		if runtimeID == nil && handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
		if !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
			if pendiente, err := s.existsPendingAutonomyMailbox(agente, &proyecto.ID, accion); err != nil {
				return false, err
			} else if pendiente {
				return false, nil
			}
		}
	}
	payload := map[string]any{
		"from_agente": "server",
		"to_agente":   agente,
		"kind":        "autonomia",
		"accion":      accion,
		"texto":       strings.TrimSpace(req.Motivo),
	}
	if text := strings.TrimSpace(req.Instruction); text != "" {
		payload["instruction"] = text
	}
	for key, value := range req.Extras {
		payload[key] = value
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	orderID, err := s.runtimes.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  &proyecto.ID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "nudge",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return false, err
	}
	s.autonomyStore.Audit("orquesta", "autonomia_nudge", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s accion=%s", agente, strings.TrimSpace(proyecto.Slug), accion))
	return true, nil
}

func (s *Service) sessionRuntimeHandle(sesion *db.Sesion) (*db.RuntimeHandle, error) {
	if s == nil || s.autonomyStore == nil || sesion == nil || sesion.ID <= 0 {
		return nil, nil
	}
	return s.autonomyStore.GetSessionRuntimeHandle(sesion.ID)
}

func (s *Service) existsPendingAutonomyOrder(agente string, proyectoID *int64, tipo string, accion string) (bool, error) {
	estado := "pendiente"
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if strings.TrimSpace(accion) == "" {
			return true, nil
		}
		if runtimeOrderHasAutonomyAction(order, accion) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) existsRecentAutonomyOrder(agente string, proyectoID *int64, tipo string, accion string, within time.Duration) (bool, error) {
	if within <= 0 {
		return false, nil
	}
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
	})
	if err != nil {
		return false, err
	}
	cutoff := time.Now().UTC().Add(-within)
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		if strings.TrimSpace(accion) != "" && !runtimeOrderHasAction(order, accion) {
			continue
		}
		if runtimeOrderMoment(order).Before(cutoff) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) existsOpenAutonomyOrder(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *Service) existsPendingAutonomyMailbox(agente string, proyectoID *int64, accion string) (bool, error) {
	estado := "pendiente"
	mailbox, err := s.runtimes.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, msg := range mailbox {
		if msg == nil || strings.TrimSpace(msg.Kind) != "autonomia" {
			continue
		}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(msg.PayloadJSON), &payload)
		if strings.TrimSpace(stringMapValue(payload, "accion")) == strings.TrimSpace(accion) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) resolveCanonicalRuntimeIDForHandle(handle *db.RuntimeHandle, agente string, proyectoID *int64) (*int64, error) {
	runtime, err := s.resolveCanonicalRuntimeForHandle(handle, agente, proyectoID)
	if err != nil || runtime == nil || runtime.ID <= 0 {
		return nil, err
	}
	return &runtime.ID, nil
}

func (s *Service) resolveCanonicalRuntimeForHandle(handle *db.RuntimeHandle, agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	if handle != nil {
		if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
			runtime, err := s.runtimes.GetRuntime(*handle.RuntimeID)
			if err == nil && runtime != nil {
				return runtime, nil
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		if handle.SesionID != nil && *handle.SesionID > 0 {
			runtime, err := s.runtimes.GetRuntimeBySessionID(*handle.SesionID)
			if err == nil && runtime != nil {
				return runtime, nil
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	runtime, err := s.runtimes.GetPrimaryRuntimeForProject(agente, proyectoID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return runtime, nil
}

func runtimeHandleStatePaused(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado")
}

func runtimeHandleRecoverySkipsLegacyATMUX(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return false
	}
	meta := parseStringMap(strings.TrimSpace(handle.MetadataJSON))
	driver := strings.ToLower(stringMapValue(meta, "driver"))
	tmuxSession := stringMapValue(meta, "tmux_session")
	return driver == "" && tmuxSession == ""
}

func runtimeHandlePreferredForRecovery(candidate, current *db.RuntimeHandle) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateTMUX := runtimeHandleTMUXRecentForRecovery(candidate)
	currentTMUX := runtimeHandleTMUXRecentForRecovery(current)
	if candidateTMUX != currentTMUX {
		return candidateTMUX
	}
	candidateFresh := db.RuntimeHandleSnapshotIsFresh(candidate, time.Minute)
	currentFresh := db.RuntimeHandleSnapshotIsFresh(current, time.Minute)
	if candidateFresh != currentFresh {
		return candidateFresh
	}
	return runtimeHandleRecoveryRecency(candidate).After(runtimeHandleRecoveryRecency(current))
}

func runtimeHandleTMUXRecentForRecovery(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if !db.RuntimeHandleSnapshotIsFresh(handle, time.Minute) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return true
	}
	meta := parseStringMap(strings.TrimSpace(handle.MetadataJSON))
	if strings.EqualFold(stringMapValue(meta, "driver"), "tmux_cli_session") {
		return true
	}
	return stringMapValue(meta, "tmux_session") != ""
}

func runtimeHandleRecoveryRecency(handle *db.RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return handle.LastSeenAt.UTC()
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt.UTC()
	}
	return handle.CreatedAt.UTC()
}

func runtimeOrderHasAutonomyAction(order *db.RuntimeOrder, accion string) bool {
	if order == nil {
		return false
	}
	payload := strings.ToLower(strings.TrimSpace(order.PayloadJSON))
	accion = strings.ToLower(strings.TrimSpace(accion))
	return strings.Contains(payload, `"kind":"autonomia"`) &&
		strings.Contains(payload, fmt.Sprintf(`"accion":"%s"`, accion))
}

func runtimeOrderHasAction(order *db.RuntimeOrder, accion string) bool {
	if order == nil {
		return false
	}
	accion = strings.TrimSpace(strings.ToLower(accion))
	if accion == "" {
		return false
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(order.PayloadJSON)), &payload); err == nil {
		if text, ok := payload["accion"].(string); ok && strings.EqualFold(strings.TrimSpace(text), accion) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(order.PayloadJSON), fmt.Sprintf(`"accion":"%s"`, accion))
}

func runtimeOrderMoment(order *db.RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	if order.FinishedAt != nil && !order.FinishedAt.IsZero() {
		return order.FinishedAt.UTC()
	}
	if order.StartedAt != nil && !order.StartedAt.IsZero() {
		return order.StartedAt.UTC()
	}
	if !order.UpdatedAt.IsZero() {
		return order.UpdatedAt.UTC()
	}
	return order.CreatedAt.UTC()
}

func stringMapValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if raw, ok := payload[key]; ok {
		if text, ok := raw.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func parseStringMap(raw string) map[string]any {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return nil
	}
	return payload
}
