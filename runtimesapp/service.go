/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimesapp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	GetAgent(nombre string) (*db.Agente, error)
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error)
	PurgeInactiveRuntimeHandles(filtro db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error)
	PurgeTerminalRuntimeOrders(filtro db.FiltroPurgadoRuntimeOrders) (*db.PurgaRuntimeOrdersResultado, error)
	GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error)
	GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ListRuntimeEvents(filtro db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error)
	ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error)
	ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error)
	CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	MarkRuntimeMailboxDelivered(id int64) error
	MarkRuntimeMailboxConsumed(id int64) error
	CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error)
	LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
	UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error)
	GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type AgentControlRequest struct {
	Agente       string
	Proyecto     string
	Accion       string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
	Motivo       string
	Por          string
}

type AgentControlPayload struct {
	Accion       string `json:"accion"`
	Proyecto     string `json:"proyecto,omitempty"`
	Conector     string `json:"conector,omitempty"`
	Modelo       string `json:"modelo,omitempty"`
	Razonamiento string `json:"razonamiento,omitempty"`
	Perfil       string `json:"perfil,omitempty"`
	Motivo       string `json:"motivo,omitempty"`
	Por          string `json:"por,omitempty"`
}

type RuntimeHandlePurgeRequest struct {
	Agente   string
	Proyecto string
	Estados  []string
	Actor    string
}

type RuntimeOrderPurgeRequest struct {
	Agente           string
	Proyecto         string
	Estados          []string
	Tipos            []string
	OlderThanMinutes int
	Actor            string
}

func (s *Service) GetProject(ref string) (*db.Proyecto, error) {
	return s.store.GetProject(ref)
}

func (s *Service) GetAgent(nombre string) (*db.Agente, error) {
	return s.store.GetAgent(nombre)
}

func (s *Service) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return s.store.ListRuntimes(filtro)
}

func (s *Service) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return s.store.BuildRuntimeTree(filtro)
}

func (s *Service) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return s.store.GetRuntime(id)
}

func (s *Service) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	return s.store.GetRuntimeHandle(id)
}

func (s *Service) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	handles, err := s.store.ListRuntimeHandles(filtro)
	if err != nil {
		return nil, err
	}
	if filtro == nil || strings.TrimSpace(*filtro) == "" {
		return handles, nil
	}
	for i, handle := range handles {
		if handle == nil {
			continue
		}
		if refreshed, err := s.store.SyncSupervisedRuntimeHandle(handle, "runtimesapp_list_runtime_handles"); err == nil && refreshed != nil {
			handles[i] = refreshed
		}
	}
	return handles, nil
}

func (s *Service) PurgeInactiveRuntimeHandles(req RuntimeHandlePurgeRequest) (*db.PurgaRuntimeHandlesResultado, error) {
	agente := strings.TrimSpace(req.Agente)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "orquesta"
	}

	var (
		filtroAgente *string
		proyectoID   *int64
	)
	if agente != "" {
		if _, err := s.store.GetAgent(agente); err != nil {
			return nil, err
		}
		filtroAgente = &agente
	}
	if proyectoRef != "" {
		proyecto, err := s.store.GetProject(proyectoRef)
		if err != nil {
			return nil, err
		}
		proyectoID = &proyecto.ID
	}
	resultado, err := s.store.PurgeInactiveRuntimeHandles(db.FiltroPurgadoRuntimeHandles{
		Agente:     filtroAgente,
		ProyectoID: proyectoID,
		Estados:    req.Estados,
	})
	if err != nil {
		return nil, err
	}
	detalle := fmt.Sprintf("agente=%s proyecto=%s deleted=%d estados=%s", valueOrFallback(agente, "*"), valueOrFallback(proyectoRef, "*"), resultado.Deleted, strings.Join(resultado.Estados, ","))
	s.store.Audit(actor, "purgar_runtime_handles", "runtime_handle", 0, detalle)
	return resultado, nil
}

func (s *Service) PurgeTerminalRuntimeOrders(req RuntimeOrderPurgeRequest) (*db.PurgaRuntimeOrdersResultado, error) {
	agente := strings.TrimSpace(req.Agente)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	if req.OlderThanMinutes < 0 {
		return nil, fmt.Errorf("older_than_minutes no puede ser negativo")
	}

	var (
		filtroAgente  *string
		proyectoID    *int64
		createdBefore *time.Time
	)
	if agente != "" {
		if _, err := s.store.GetAgent(agente); err != nil {
			return nil, err
		}
		filtroAgente = &agente
	}
	if proyectoRef != "" {
		proyecto, err := s.store.GetProject(proyectoRef)
		if err != nil {
			return nil, err
		}
		proyectoID = &proyecto.ID
	}
	if req.OlderThanMinutes > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(req.OlderThanMinutes) * time.Minute)
		createdBefore = &cutoff
	}
	resultado, err := s.store.PurgeTerminalRuntimeOrders(db.FiltroPurgadoRuntimeOrders{
		Agente:        filtroAgente,
		ProyectoID:    proyectoID,
		Estados:       req.Estados,
		Tipos:         req.Tipos,
		CreatedBefore: createdBefore,
	})
	if err != nil {
		return nil, err
	}
	detalle := fmt.Sprintf("agente=%s proyecto=%s deleted=%d estados=%s tipos=%s older_than_minutes=%d", valueOrFallback(agente, "*"), valueOrFallback(proyectoRef, "*"), resultado.Deleted, strings.Join(resultado.Estados, ","), strings.Join(resultado.Tipos, ","), req.OlderThanMinutes)
	s.store.Audit(actor, "purgar_runtime_orders", "runtime_order", 0, detalle)
	return resultado, nil
}

func (s *Service) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return s.store.GetActiveRuntimeHandle(agente)
}

func (s *Service) GetActiveRuntimeHandleAgent(agente string) (*db.RuntimeHandle, error) {
	return s.GetActiveRuntimeHandle(agente)
}

func (s *Service) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
}

func (s *Service) GetActiveRuntimeHandleAgentProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.GetActiveRuntimeHandleForProject(agente, proyectoID)
}

func (s *Service) ListRuntimeEvents(filtro db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error) {
	return s.store.ListRuntimeEvents(filtro)
}

func (s *Service) ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	return s.store.ListRuntimeTranscript(filtro)
}

func (s *Service) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return s.store.ListRuntimeSamples(runtimeID, limit)
}

func (s *Service) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return s.store.CreateRuntimeOrder(order)
}

func (s *Service) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return s.CreateRuntimeOrder(order)
}

func (s *Service) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return s.store.ListRuntimeOrders(filtro)
}

func (s *Service) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return s.store.CreateRuntimeMailbox(msg)
}

func (s *Service) SendRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return s.CreateRuntimeMailbox(msg)
}

func (s *Service) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return s.store.ListRuntimeMailbox(filtro)
}

func (s *Service) MarkRuntimeMailboxDelivered(id int64) error {
	return s.store.MarkRuntimeMailboxDelivered(id)
}

func (s *Service) MarkRuntimeMailboxConsumed(id int64) error {
	return s.store.MarkRuntimeMailboxConsumed(id)
}

func (s *Service) CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error) {
	return s.store.CreateRuntimeCheckpoint(checkpoint)
}

func (s *Service) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return s.store.ListRuntimeCheckpoints(filtro)
}

func (s *Service) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return s.store.GetRuntimeCheckpoint(id)
}

func (s *Service) LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error) {
	return s.store.LatestRuntimeCheckpoint(agente, proyectoID)
}

func (s *Service) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return s.store.ListMemoryEntities(filtro)
}

func (s *Service) UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error) {
	return s.store.UpsertMemoryEntity(entidad)
}

func (s *Service) GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error) {
	return s.store.GetMemoryEntity(nombre, proyectoID)
}

func (s *Service) EnqueueAgentControl(req AgentControlRequest) (int64, string, error) {
	agente := strings.TrimSpace(req.Agente)
	if agente == "" {
		return 0, "", fmt.Errorf("agente obligatorio")
	}
	accion, err := normalizeAgentControlAction(req.Accion)
	if err != nil {
		return 0, "", err
	}
	if _, err := s.store.GetAgent(agente); err != nil {
		return 0, "", err
	}

	var (
		proyectoID *int64
		handleID   *int64
		runtimeID  *int64
	)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	if accion == "start" && proyectoRef == "" {
		return 0, "", fmt.Errorf("debes indicar proyecto para arrancar el agente")
	}
	if proyectoRef != "" {
		proyecto, err := s.store.GetProject(proyectoRef)
		if err != nil {
			return 0, "", err
		}
		proyectoID = &proyecto.ID
	}

	handle, err := s.resolveAgentControlHandle(agente, proyectoID)
	if err != nil {
		return 0, "", err
	}
	if accion == "start" && handle != nil {
		return 0, "", fmt.Errorf("el agente %s ya tiene un runtime handle activo", agente)
	}
	if handle != nil {
		handleID = &handle.ID
		if handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
	}

	payloadJSON, err := json.Marshal(AgentControlPayload{
		Accion:       accion,
		Proyecto:     proyectoRef,
		Conector:     strings.TrimSpace(req.Conector),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		Perfil:       strings.TrimSpace(req.Perfil),
		Motivo:       strings.TrimSpace(req.Motivo),
		Por:          valueOrFallback(strings.TrimSpace(req.Por), "orquesta"),
	})
	if err != nil {
		return 0, "", err
	}

	orderID, err := s.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  proyectoID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        accion,
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return 0, "", err
	}

	detalle := fmt.Sprintf("%s agente=%s proyecto=%s", accion, agente, valueOrFallback(proyectoRef, ""))
	if motivo := strings.TrimSpace(req.Motivo); motivo != "" {
		detalle += " motivo=" + motivo
	}
	s.store.Audit(valueOrFallback(strings.TrimSpace(req.Por), "orquesta"), "control_agente_"+accion, "runtime_order", orderID, detalle)
	return orderID, accion, nil
}

type Repository struct{}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}

func (Repository) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return db.ListarRuntimes(filtro)
}

func (Repository) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return db.ConstruirArbolRuntimes(filtro)
}

func (Repository) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return db.GetRuntime(id)
}

func (Repository) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandle(id)
}

func (Repository) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(filtro)
}

func (Repository) SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error) {
	handle, _, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, source)
	return handle, err
}

func (Repository) PurgeInactiveRuntimeHandles(filtro db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error) {
	return db.PurgarRuntimeHandlesInactivos(filtro)
}

func (Repository) PurgeTerminalRuntimeOrders(filtro db.FiltroPurgadoRuntimeOrders) (*db.PurgaRuntimeOrdersResultado, error) {
	return db.PurgarRuntimeOrdersTerminales(filtro)
}

func (Repository) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleActivoAgente(agente)
}

func (Repository) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleActivoAgenteProyecto(agente, proyectoID)
}

func (Repository) ListRuntimeEvents(filtro db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error) {
	return db.ListarRuntimeEvents(filtro)
}

func (Repository) ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	return db.ListarRuntimeTranscript(filtro)
}

func (Repository) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return db.ListarMuestrasRuntime(runtimeID, limit)
}

func (Repository) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return db.EncolarRuntimeOrder(order)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return db.EnviarRuntimeMailbox(msg)
}

func (Repository) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return db.ListarRuntimeMailbox(filtro)
}

func (Repository) MarkRuntimeMailboxDelivered(id int64) error {
	return db.MarcarRuntimeMailboxEntregado(id)
}

func (Repository) MarkRuntimeMailboxConsumed(id int64) error {
	return db.MarcarRuntimeMailboxConsumido(id)
}

func (Repository) CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error) {
	return db.CrearRuntimeCheckpoint(checkpoint)
}

func (Repository) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return db.ListarRuntimeCheckpoints(filtro)
}

func (Repository) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return db.GetRuntimeCheckpoint(id)
}

func (Repository) LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error) {
	return db.UltimoRuntimeCheckpoint(agente, proyectoID)
}

func (Repository) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return db.ListarEntidadesMemoria(filtro)
}

func (Repository) UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error) {
	return db.UpsertEntidadMemoria(entidad)
}

func (Repository) GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error) {
	return db.GetEntidadMemoria(nombre, proyectoID)
}

func (Repository) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func (s *Service) resolveAgentControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if proyectoID != nil {
		handle, err := s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	return s.store.GetActiveRuntimeHandle(agente)
}

func normalizeAgentControlAction(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "start", "arrancar", "arranque", "iniciar":
		return "start", nil
	case "pause", "pausar", "pausa":
		return "pause", nil
	case "resume", "continuar", "reanudar":
		return "resume", nil
	case "stop", "detener", "parar":
		return "stop", nil
	default:
		return "", fmt.Errorf("acción de control no soportada: %s", strings.TrimSpace(v))
	}
}

func valueOrFallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
