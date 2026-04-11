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
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	GetAgent(nombre string) (*db.Agente, error)
	SharedAccountActivationAllowed(agente string) (bool, string, error)
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	GetPrimaryRuntime(agente string) (*db.RuntimeInstance, error)
	GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error)
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	ListCanonicalRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error)
	PurgeInactiveRuntimeHandles(filtro db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error)
	PurgeTerminalRuntimeOrders(filtro db.FiltroPurgadoRuntimeOrders) (*db.PurgaRuntimeOrdersResultado, error)
	GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error)
	GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error)
	GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ListRuntimeEvents(filtro db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error)
	ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error)
	ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error)
	CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	GetRuntimeOrder(id int64) (*db.RuntimeOrder, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error)
	CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error)
	GetRuntimeMailbox(id int64) (*db.RuntimeMailboxMessage, error)
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
	store                  Store
	afterEnqueueOrderHook  func(order *db.RuntimeOrder, orderID int64)
	afterCreateMailboxHook func(msg *db.RuntimeMailboxMessage, mailboxID int64)
}

type RuntimeDescription struct {
	Runtime *db.RuntimeInstance             `json:"runtime"`
	Handle  *db.RuntimeHandle               `json:"handle,omitempty"`
	Worker  *runtimeagente.WorkerStatusView `json:"worker,omitempty"`
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetAfterEnqueueRuntimeOrderHook(hook func(order *db.RuntimeOrder, orderID int64)) {
	if s == nil {
		return
	}
	s.afterEnqueueOrderHook = hook
}

func (s *Service) SetAfterCreateRuntimeMailboxHook(hook func(msg *db.RuntimeMailboxMessage, mailboxID int64)) {
	if s == nil {
		return
	}
	s.afterCreateMailboxHook = hook
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

type MicroprogramacionDispatchRequest struct {
	AgenteDestino     string
	ProyectoID        *int64
	Mensaje           string
	EspecificacionID  int64
	ArchivoObjetivo   string
	SimboloObjetivo   string
	WriteSet          []string
	TestsObligatorios []string
	FormatoSalida     string
}

type MicroprogramacionDispatchResult struct {
	RuntimeOrderID int64
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

func (s *Service) DescribeRuntime(id int64) (*RuntimeDescription, error) {
	runtime, err := s.store.GetRuntime(id)
	if err != nil {
		return nil, err
	}
	handle, worker, err := s.resolveRuntimeStructuredWorker(runtime)
	if err != nil {
		return nil, err
	}
	return &RuntimeDescription{
		Runtime: runtime,
		Handle:  handle,
		Worker:  worker,
	}, nil
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

func (s *Service) resolveRuntimeStructuredWorker(runtime *db.RuntimeInstance) (*db.RuntimeHandle, *runtimeagente.WorkerStatusView, error) {
	if runtime == nil {
		return nil, nil, nil
	}
	if handle, err := s.preferredRuntimeHandle(runtime); err != nil {
		return nil, nil, err
	} else if handle != nil {
		if refreshed, err := s.store.SyncSupervisedRuntimeHandle(handle, "runtimesapp_describe_runtime"); err == nil && refreshed != nil {
			handle = refreshed
		}
		if runtimeHandleMatchesRuntime(runtime, handle) {
			snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
			if err != nil || snap == nil {
				return handle, nil, err
			}
			return handle, snap.View(time.Now().UTC(), time.Minute), nil
		}
	}
	handles, err := s.store.ListCanonicalRuntimeHandles(&runtime.Agente)
	if err != nil {
		return nil, nil, err
	}
	handle := selectBestRuntimeHandle(runtime, handles)
	if handle == nil {
		handles, err = s.store.ListRuntimeHandles(&runtime.Agente)
		if err != nil {
			return nil, nil, err
		}
		handle = selectBestRuntimeHandle(runtime, handles)
	}
	if handle == nil {
		return nil, nil, nil
	}
	if refreshed, err := s.store.SyncSupervisedRuntimeHandle(handle, "runtimesapp_describe_runtime"); err == nil && refreshed != nil {
		handle = refreshed
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return handle, nil, err
	}
	return handle, snap.View(time.Now().UTC(), time.Minute), nil
}

func (s *Service) preferredRuntimeHandle(runtime *db.RuntimeInstance) (*db.RuntimeHandle, error) {
	if runtime == nil {
		return nil, nil
	}
	if runtime.ProyectoID != nil && *runtime.ProyectoID > 0 {
		handle, err := s.store.GetOperationalRuntimeHandleForProject(strings.TrimSpace(runtime.Agente), runtime.ProyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
		handle, err = s.store.GetActiveRuntimeHandleForProject(strings.TrimSpace(runtime.Agente), runtime.ProyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	handle, err := s.store.GetOperationalRuntimeHandle(strings.TrimSpace(runtime.Agente))
	if err != nil {
		return nil, err
	}
	if handle != nil {
		return handle, nil
	}
	return s.store.GetActiveRuntimeHandle(strings.TrimSpace(runtime.Agente))
}

func runtimeHandleMatchesRuntime(runtime *db.RuntimeInstance, handle *db.RuntimeHandle) bool {
	if runtime == nil || handle == nil {
		return false
	}
	if runtime.ID > 0 && handle.RuntimeID != nil && *handle.RuntimeID == runtime.ID {
		return true
	}
	if runtime.SesionID != nil && handle.SesionID != nil && *handle.SesionID == *runtime.SesionID {
		return true
	}
	return false
}

func selectBestRuntimeHandle(runtime *db.RuntimeInstance, handles []*db.RuntimeHandle) *db.RuntimeHandle {
	var (
		best         *db.RuntimeHandle
		bestScore    int
		hasNonLegacy bool
	)
	for _, handle := range handles {
		if handle == nil || !strings.EqualFold(strings.TrimSpace(handle.Agente), strings.TrimSpace(runtime.Agente)) {
			continue
		}
		if !runtimeHandleIsLegacyProcessPTY(handle) {
			hasNonLegacy = true
			break
		}
	}
	for _, handle := range handles {
		if handle == nil || !strings.EqualFold(strings.TrimSpace(handle.Agente), strings.TrimSpace(runtime.Agente)) {
			continue
		}
		if hasNonLegacy && runtimeHandleIsLegacyProcessPTY(handle) {
			continue
		}
		score := runtimeHandleScore(runtime, handle)
		if best == nil || score > bestScore {
			best = handle
			bestScore = score
		}
	}
	return best
}

func runtimeHandleScore(runtime *db.RuntimeInstance, handle *db.RuntimeHandle) int {
	score := 0
	if runtime == nil || handle == nil {
		return score
	}
	if runtime.ID > 0 && handle.RuntimeID != nil && *handle.RuntimeID == runtime.ID {
		score += 1000
	}
	if runtime.SesionID != nil && handle.SesionID != nil && *handle.SesionID == *runtime.SesionID {
		score += 500
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo":
		score += 100
	case "pausado":
		score += 80
	case "degradado":
		score += 30
	case "fallido", "cerrado":
		score -= 100
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		score += 10
	}
	switch strings.ToLower(strings.TrimSpace(runtimeHandleDriver(handle))) {
	case "tmux_cli_session":
		score += 300
	case "process_pty_cli":
		score -= 150
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON); err == nil && snap != nil {
		view := snap.View(time.Now().UTC(), time.Minute)
		if view != nil {
			if strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") {
				score += 200
			}
			if view.Alive && strings.EqualFold(strings.TrimSpace(view.State), "running") {
				score += 40
			}
			if !view.Alive {
				score -= 20
			}
		}
	}
	return score
}

func runtimeHandleDriver(handle *db.RuntimeHandle) string {
	if handle == nil || strings.TrimSpace(handle.MetadataJSON) == "" {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		return ""
	}
	if value, ok := meta["driver"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func runtimeHandleIsLegacyProcessPTY(handle *db.RuntimeHandle) bool {
	return strings.EqualFold(strings.TrimSpace(runtimeHandleDriver(handle)), "process_pty_cli")
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

func (s *Service) GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return s.store.GetOperationalRuntimeHandle(agente)
}

func (s *Service) GetOperationalRuntimeHandleAgent(agente string) (*db.RuntimeHandle, error) {
	return s.GetOperationalRuntimeHandle(agente)
}

func (s *Service) GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.store.GetOperationalRuntimeHandleForProject(agente, proyectoID)
}

func (s *Service) GetOperationalRuntimeHandleAgentProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.GetOperationalRuntimeHandleForProject(agente, proyectoID)
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
	orderID, err := s.store.CreateRuntimeOrder(order)
	if err != nil {
		return 0, err
	}
	if s.afterEnqueueOrderHook != nil {
		s.afterEnqueueOrderHook(order, orderID)
	}
	return orderID, nil
}

func (s *Service) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return s.CreateRuntimeOrder(order)
}

func (s *Service) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return s.store.ListRuntimeOrders(filtro)
}

func (s *Service) GetRuntimeOrder(id int64) (*db.RuntimeOrder, error) {
	return s.store.GetRuntimeOrder(id)
}

func (s *Service) CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return s.store.CreateLiveAgentHandoff(
		strings.TrimSpace(origen),
		strings.TrimSpace(destino),
		tareaID,
		strings.TrimSpace(motivo),
		strings.TrimSpace(resumenContinuidad),
		strings.TrimSpace(externalSessionID),
	)
}

func (s *Service) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return s.store.CreateRuntimeMailbox(msg)
}

func (s *Service) DispatchMicroprogramacionInstruction(req MicroprogramacionDispatchRequest) (*MicroprogramacionDispatchResult, error) {
	agenteDestino := strings.TrimSpace(req.AgenteDestino)
	if agenteDestino == "" {
		return nil, fmt.Errorf("agente destino obligatorio")
	}
	if req.ProyectoID == nil || *req.ProyectoID <= 0 {
		return nil, fmt.Errorf("proyecto obligatorio para despachar microtarea")
	}
	mensaje, err := s.prepararMensajeMicroprogramacion(req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"to_agente": agenteDestino,
		"texto":     mensaje,
		"source":    "microprogramacion",
		"microprogramacion": map[string]any{
			"especificacion_id": req.EspecificacionID,
			"archivo_objetivo":  strings.TrimSpace(req.ArchivoObjetivo),
			"simbolo_objetivo":  strings.TrimSpace(req.SimboloObjetivo),
			"write_set":         req.WriteSet,
			"tests":             req.TestsObligatorios,
			"formato_salida":    strings.TrimSpace(req.FormatoSalida),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("serializando payload de microprogramacion: %w", err)
	}
	orderID, err := s.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      agenteDestino,
		ProyectoID:  req.ProyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: string(payload),
	})
	if err != nil {
		return nil, err
	}
	return &MicroprogramacionDispatchResult{RuntimeOrderID: orderID}, nil
}

func (s *Service) SendRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	id, err := s.CreateRuntimeMailbox(msg)
	if err != nil {
		return 0, err
	}
	if s.afterCreateMailboxHook != nil {
		s.afterCreateMailboxHook(msg, id)
	}
	return id, nil
}

func (s *Service) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return s.store.ListRuntimeMailbox(filtro)
}

func (s *Service) GetRuntimeMailbox(id int64) (*db.RuntimeMailboxMessage, error) {
	return s.store.GetRuntimeMailbox(id)
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
	if accion == "start" || accion == "resume" {
		permite, ocupadoPor, err := s.store.SharedAccountActivationAllowed(agente)
		if err != nil {
			return 0, "", err
		}
		if !permite {
			if strings.TrimSpace(ocupadoPor) == "" {
				return 0, "", fmt.Errorf("la cuenta compartida observada de %s no admite más activaciones ahora", agente)
			}
			return 0, "", fmt.Errorf("la cuenta compartida observada de %s ya está ocupada por %s", agente, ocupadoPor)
		}
	}

	handle, err := s.resolveAgentControlHandle(agente, proyectoID)
	if err != nil {
		return 0, "", err
	}
	if accion == "start" && handle != nil {
		if fresh, err := s.store.GetRuntimeHandle(handle.ID); err != nil {
			return 0, "", err
		} else if fresh != nil {
			handle = fresh
		}
		if db.RuntimeHandlePauseRequiresFreshStart(handle) {
			handle = nil
		} else {
			canSupersede, err := s.canSupersedeStartHandle(agente, proyectoID, handle)
			if err != nil {
				return 0, "", err
			}
			if canSupersede {
				handle = nil
			} else {
				return 0, "", fmt.Errorf("el agente %s ya tiene un runtime handle activo", agente)
			}
		}
	}
	if handle != nil {
		handleID = &handle.ID
		runtime, err := s.runtimeForHandle(agente, proyectoID, handle)
		if err != nil {
			return 0, "", err
		}
		if runtime != nil && runtime.ID > 0 {
			runtimeID = &runtime.ID
		} else if handle.RuntimeID != nil {
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

func (Repository) SharedAccountActivationAllowed(agente string) (bool, string, error) {
	return db.CuentaCompartidaPermiteActivacionAgente(agente)
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

func (Repository) GetPrimaryRuntime(agente string) (*db.RuntimeInstance, error) {
	return db.GetRuntimePrincipalAgente(agente)
}

func (Repository) GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	return db.GetRuntimePrincipalAgenteProyecto(agente, proyectoID)
}

func (Repository) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandle(id)
}

func (Repository) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(filtro)
}

func (Repository) ListCanonicalRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesCanonicosRecientes(filtro)
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
	return db.GetRuntimeHandleCanonicoRecienteAgente(agente)
}

func (Repository) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, proyectoID)
}

func (Repository) GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleOperativoRecienteAgente(agente)
}

func (Repository) GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, proyectoID)
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

func (Repository) CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return db.CrearHandoffAgenteVivo(origen, destino, tareaID, motivo, resumenContinuidad, externalSessionID)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) GetRuntimeOrder(id int64) (*db.RuntimeOrder, error) {
	return db.GetRuntimeOrder(id)
}

func (Repository) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return db.EnviarRuntimeMailbox(msg)
}

func (Repository) GetRuntimeMailbox(id int64) (*db.RuntimeMailboxMessage, error) {
	return db.GetRuntimeMailbox(id)
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

func (s *Service) ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if proyectoID != nil {
		handle, err := s.store.GetOperationalRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		handle, err = s.rehydrateHandle(handle)
		if err != nil {
			return nil, err
		}
		if handle != nil && runtimeHandleCanonicoParaControl(handle) {
			return handle, nil
		}
		handle, err = s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		handle, err = s.rehydrateHandle(handle)
		if err != nil {
			return nil, err
		}
		if handle != nil && runtimeHandleCanonicoParaControl(handle) {
			return handle, nil
		}
	}
	handle, err := s.store.GetOperationalRuntimeHandle(agente)
	if err != nil {
		return nil, err
	}
	handle, err = s.rehydrateHandle(handle)
	if err != nil {
		return nil, err
	}
	if handle != nil && runtimeHandleCanonicoParaControl(handle) {
		return handle, nil
	}
	handle, err = s.store.GetActiveRuntimeHandle(agente)
	if err != nil {
		return nil, err
	}
	handle, err = s.rehydrateHandle(handle)
	if err != nil {
		return nil, err
	}
	if !runtimeHandleCanonicoParaControl(handle) {
		return nil, nil
	}
	return handle, nil
}

func (s *Service) ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	resolveFresh := func(handle *db.RuntimeHandle, err error) (*db.RuntimeHandle, error) {
		if err != nil || handle == nil {
			return nil, err
		}
		return s.rehydrateHandle(handle)
	}
	if proyectoID != nil && *proyectoID > 0 {
		if handle, err := resolveFresh(s.store.GetOperationalRuntimeHandleForProject(agente, proyectoID)); err != nil {
			return nil, err
		} else if handle != nil {
			return handle, nil
		}
	}
	if handle, err := resolveFresh(s.store.GetOperationalRuntimeHandle(agente)); err != nil {
		return nil, err
	} else if handle != nil {
		return handle, nil
	}
	if proyectoID != nil && *proyectoID > 0 {
		if handle, err := resolveFresh(s.GetOperationalRuntimeHandleAgentProject(agente, proyectoID)); err != nil {
			return nil, err
		} else if handle != nil {
			return handle, nil
		}
		if handle, err := resolveFresh(s.GetActiveRuntimeHandleAgentProject(agente, proyectoID)); err != nil {
			return nil, err
		} else if handle != nil {
			return handle, nil
		}
	}
	if handle, err := resolveFresh(s.GetOperationalRuntimeHandleAgent(agente)); err != nil {
		return nil, err
	} else if handle != nil {
		return handle, nil
	}
	if handle, err := resolveFresh(s.GetActiveRuntimeHandleAgent(agente)); err != nil {
		return nil, err
	} else if handle != nil {
		return handle, nil
	}
	handles, err := s.store.ListRuntimeHandles(&agente)
	if err != nil {
		return nil, err
	}
	var best *db.RuntimeHandle
	bestMoment := time.Time{}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if proyectoID != nil && *proyectoID > 0 {
			if handle.ProyectoID == nil || *handle.ProyectoID != *proyectoID {
				continue
			}
		}
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo", "pausado", "fallido":
		default:
			continue
		}
		moment := handle.CreatedAt.UTC()
		if handle.UpdatedAt.After(moment) {
			moment = handle.UpdatedAt.UTC()
		}
		if handle.LastSeenAt != nil && handle.LastSeenAt.After(moment) {
			moment = handle.LastSeenAt.UTC()
		}
		if best == nil || moment.After(bestMoment) || (moment.Equal(bestMoment) && handle.ID > best.ID) {
			best = handle
			bestMoment = moment
		}
	}
	if best == nil {
		return nil, nil
	}
	return s.rehydrateHandle(best)
}

func (s *Service) ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	if item == nil {
		return nil, nil
	}
	if item.HandleID != nil && *item.HandleID > 0 {
		handle, err := s.store.GetRuntimeHandle(*item.HandleID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	agente := strings.TrimSpace(item.Agente)
	if agente != "" && item.RuntimeID > 0 {
		handles, err := s.store.ListRuntimeHandles(&agente)
		if err != nil {
			return nil, err
		}
		for _, handle := range handles {
			if handle == nil || handle.RuntimeID == nil || *handle.RuntimeID != item.RuntimeID {
				continue
			}
			return s.rehydrateHandle(handle)
		}
	}
	return s.ResolveDeliveryHandle(agente, item.ProyectoID)
}

func (s *Service) resolveAgentControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	if proyectoID != nil {
		handle, err := s.store.GetOperationalRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
		handle, err = s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	handle, err := s.store.GetOperationalRuntimeHandle(agente)
	if err != nil {
		return nil, err
	}
	if handle != nil {
		return handle, nil
	}
	return s.store.GetActiveRuntimeHandle(agente)
}

func (s *Service) canSupersedeStartHandle(agente string, proyectoID *int64, handle *db.RuntimeHandle) (bool, error) {
	if handle == nil {
		return false, nil
	}
	if refreshed, err := s.store.SyncSupervisedRuntimeHandle(handle, "runtimesapp_can_supersede_start"); err != nil {
		return false, err
	} else if refreshed != nil {
		handle = refreshed
	}
	runtime, err := s.runtimeForHandle(agente, proyectoID, handle)
	if err != nil {
		return false, err
	}
	if runtimeHandleShouldSupersedeStart(handle, runtime) {
		return true, nil
	}
	estado := "pendiente"
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
		Limit:      10,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if strings.TrimSpace(order.Tipo) == "stop" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) runtimeForHandle(agente string, proyectoID *int64, handle *db.RuntimeHandle) (*db.RuntimeInstance, error) {
	var (
		runtime *db.RuntimeInstance
		err     error
	)
	if proyectoID != nil {
		runtime, err = s.store.GetPrimaryRuntimeForProject(agente, proyectoID)
		if err != nil {
			return nil, err
		}
	}
	if runtime == nil {
		runtime, err = s.store.GetPrimaryRuntime(agente)
		if err != nil {
			return nil, err
		}
	}
	if runtime == nil && handle != nil && handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtime, err = s.store.GetRuntime(*handle.RuntimeID)
		if err != nil {
			return nil, err
		}
	}
	return runtime, nil
}

func runtimeHandleShouldSupersedeStart(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil {
		return false
	}
	if runtimeHandleTMUXSessionMissing(handle) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "pausado":
	default:
		return true
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON)); err == nil && snap != nil {
		view := snap.View(time.Now().UTC(), time.Minute)
		if view != nil {
			switch strings.ToLower(strings.TrimSpace(view.State)) {
			case "stale", "stopped", "failed", "exited", "closed":
				return true
			}
			if view.HeartbeatStale || !view.Alive {
				return true
			}
		}
	}
	if runtime == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "fallido", "degradado":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(runtime.ProcessState)) {
	case "fallido", "crashed", "exited", "remote_status_error":
		return true
	}
	return false
}

func runtimeHandleTMUXSessionMissing(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	known, exists := controlruntime.TMUXSessionExistsMetadata(strings.TrimSpace(handle.MetadataJSON))
	return known && !exists
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

func (s *Service) rehydrateHandle(handle *db.RuntimeHandle) (*db.RuntimeHandle, error) {
	if handle == nil || handle.ID <= 0 {
		return handle, nil
	}
	fresh, err := s.store.GetRuntimeHandle(handle.ID)
	if err != nil {
		return nil, err
	}
	if fresh != nil {
		return fresh, nil
	}
	return nil, nil
}

func runtimeHandleCanonicoParaControl(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if runtimeHandleEsCandidatoLegacyATMUX(handle) {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil || snap == nil {
		return true
	}
	view := snap.View(time.Now().UTC(), 2*time.Minute)
	if view == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "stale", "stopped", "failed", "exited", "closed":
		return false
	}
	if view.HeartbeatStale || !view.Alive {
		return false
	}
	return true
}

func runtimeHandleEsCandidatoLegacyATMUX(handle *db.RuntimeHandle) bool {
	if !runtimeHandleEsLegacyControlPlane(handle) {
		return false
	}
	meta := mapFromJSONRuntimeHandle(handle.MetadataJSON)
	for _, candidate := range []string{
		mapStringValueRuntimeHandle(meta, "rendered_command"),
		mapStringValueRuntimeHandle(meta, "wrapped_command"),
		mapStringValueRuntimeHandle(meta, "herramienta"),
		mapStringValueRuntimeHandle(meta, "conector"),
		mapStringValueRuntimeHandle(meta, "profile_status_wrapper"),
	} {
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(candidate) {
			return true
		}
	}
	return false
}

func runtimeHandleEsLegacyControlPlane(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.Transporte) != "cli" || strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "pausado":
	default:
		return false
	}
	meta := mapFromJSONRuntimeHandle(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(mapStringValueRuntimeHandle(meta, "driver")), "process_pty_cli") {
		return false
	}
	return true
}

func mapFromJSONRuntimeHandle(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func mapStringValueRuntimeHandle(meta map[string]any, key string) string {
	if len(meta) == 0 {
		return ""
	}
	value, _ := meta[key].(string)
	return strings.TrimSpace(value)
}
