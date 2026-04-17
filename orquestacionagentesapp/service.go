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
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error)
}

type AutonomyStore interface {
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

func runtimeHandleStatePaused(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado")
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
