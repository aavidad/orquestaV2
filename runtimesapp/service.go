/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimesapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/internal/controlruntime"
	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
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
	CloseRuntime(id int64) error
	CloseRuntimeHandles(agente string, proyectoID *int64) error
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	ListPassiveRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
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
	RegisterRuntimeTranscript(entry *db.RuntimeTranscriptEntry) (int64, error)
	ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error)
	CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	GetRuntimeOrder(id int64) (*db.RuntimeOrder, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	MarkRuntimeOrderState(id int64, estado, resultadoJSON, errorText string) error
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
	store                        Store
	afterEnqueueOrderHook        func(order *db.RuntimeOrder, orderID int64)
	afterCreateMailboxHook       func(msg *db.RuntimeMailboxMessage, mailboxID int64)
	afterRegisterTranscriptHook  func(entry *db.RuntimeTranscriptEntry, transcriptID int64)
	registradorEntregaGit        RegistradorEntregaGit
	registradorEntregaGitPremium RegistradorEntregaGitPremium
	materializadorEntrega        MaterializadorEntregaMicroprogramacion
	resolvedorWorktree           ResolvedorWorktreeActiva
	aseguradorWorktree           AseguradorWorktreeActiva
	taskCompleter                TaskCompleter
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

func (s *Service) SetAfterRegisterRuntimeTranscriptHook(hook func(entry *db.RuntimeTranscriptEntry, transcriptID int64)) {
	if s == nil {
		return
	}
	s.afterRegisterTranscriptHook = hook
}

type RegistradorEntregaGit interface {
	RegistrarEntregaGit(id int64, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*microprogramacionapp.ResultadoRegistrarEntregaGit, error)
}

type RegistradorEntregaGitPremium interface {
	RegistrarEntregaGitPremium(entrada EntradaRegistrarEntregaGitPremium) (*ResultadoRegistrarEntregaGitPremium, error)
}

type MaterializadorEntregaMicroprogramacion interface {
	MaterializarEntregaDesdeRespuesta(id int64, raizProyecto, respuesta, evidencia string) (*microprogramacionapp.ResultadoMaterializarEntrega, error)
}

type ResolvedorWorktreeActiva interface {
	ResolveActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error)
}

type AseguradorWorktreeActiva interface {
	EnsureActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error)
}

type TaskCompleter interface {
	Complete(id int64, agente, commit string) error
}

func (s *Service) SetRegistradorEntregaGit(registrador RegistradorEntregaGit) {
	if s == nil {
		return
	}
	s.registradorEntregaGit = registrador
}

func (s *Service) SetRegistradorEntregaGitPremium(registrador RegistradorEntregaGitPremium) {
	if s == nil {
		return
	}
	s.registradorEntregaGitPremium = registrador
}

func (s *Service) SetMaterializadorEntregaMicroprogramacion(materializador MaterializadorEntregaMicroprogramacion) {
	if s == nil {
		return
	}
	s.materializadorEntrega = materializador
}

func (s *Service) SetResolvedorWorktreeActiva(resolvedor ResolvedorWorktreeActiva) {
	if s == nil {
		return
	}
	s.resolvedorWorktree = resolvedor
}

func (s *Service) SetAseguradorWorktreeActiva(asegurador AseguradorWorktreeActiva) {
	if s == nil {
		return
	}
	s.aseguradorWorktree = asegurador
}

func (s *Service) SetTaskCompleter(completer TaskCompleter) {
	if s == nil {
		return
	}
	s.taskCompleter = completer
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
	TareaID      *int64
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
	TareaID      *int64 `json:"tarea_id,omitempty"`
}

type RuntimeHandlePurgeRequest struct {
	Agente   string
	Proyecto string
	Estados  []string
	Actor    string
}

type RuntimeHandleResidualCloseRequest struct {
	Agente   string
	Proyecto string
	Actor    string
}

type RuntimeHandleResidualCloseResult struct {
	Closed bool
}

type RuntimeOrderPurgeRequest struct {
	Agente           string
	Proyecto         string
	Estados          []string
	Tipos            []string
	OlderThanMinutes int
	Actor            string
}

type RuntimeResidualCloseRequest struct {
	Agente   string
	Proyecto string
	Actor    string
}

type RuntimeResidualCloseResult struct {
	Closed    int
	ClosedIDs []int64
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

type AgentNudgeRequest struct {
	Agente      string
	Proyecto    string
	Accion      string
	Motivo      string
	Instruction string
	Actor       string
	Metadata    map[string]any
}

type AgentNudgeResult struct {
	RuntimeOrderID int64
}

type ResultadoEntregaGitMicroprogramacionActiva struct {
	Contexto        *ContextoEntregaMicroprogramacion
	Entrega         *microprogramacionapp.ResultadoRegistrarEntregaGit
	ContextoPremium *ContextoEntregaGitPremium
	EntregaPremium  *ResultadoRegistrarEntregaGitPremium
}

type ContextoEntregaMicroprogramacion struct {
	RuntimeOrderID   int64
	EspecificacionID int64
	ArchivoObjetivo  string
	SimboloObjetivo  string
	WriteSet         []string
	FormatoSalida    string
	WorktreeID       *int64
	RutaWorktree     string
	BranchWorktree   string
	BaseRefWorktree  string
}

type EntradaRegistrarEntregaGitPremium struct {
	Agente                  string
	ProyectoID              *int64
	ProyectoSlug            string
	SolicitadoPor           string
	Evidencia               string
	Carril                  string
	TareaObjetivoID         int64
	WriteSet                []string
	PreferenciaWorktreeID   *int64
	PreferenciaRutaWorktree string
	PreferenciaBranch       string
	PreferenciaBaseRef      string
}

type ResultadoRegistrarEntregaGitPremium struct {
	WorktreeID         int64
	RutaWorktree       string
	SourceBranch       string
	TargetBranch       string
	HeadCommit         string
	ArchivosEntregados []string
	GitMergeID         int64
}

type ContextoEntregaGitPremium struct {
	RuntimeOrderID  int64
	Carril          string
	TareaObjetivoID int64
	WriteSet        []string
	WorktreeID      *int64
	RutaWorktree    string
	BranchWorktree  string
	BaseRefWorktree string
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
	synced := 0
	for i, handle := range handles {
		if handle == nil {
			continue
		}
		if synced >= 1 && !runtimeHandleNeedsFreshSync(handle) {
			continue
		}
		if refreshed, err := s.store.SyncSupervisedRuntimeHandle(handle, "runtimesapp_list_runtime_handles"); err == nil && refreshed != nil {
			handles[i] = refreshed
		}
		synced++
	}
	return handles, nil
}

func (s *Service) ListRuntimeHandlesCompact(filtro *string) ([]*db.RuntimeHandle, error) {
	if filtro == nil || strings.TrimSpace(*filtro) == "" {
		return s.store.ListRuntimeHandles(filtro)
	}
	handles, err := s.store.ListPassiveRuntimeHandles(filtro)
	if err != nil {
		return nil, err
	}
	if len(handles) > 0 {
		return handles, nil
	}
	return s.ListRuntimeHandles(filtro)
}

func runtimeHandleNeedsFreshSync(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "running", "ready", "starting", "paused", "fallido", "degradado":
		return true
	default:
		return false
	}
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
		if !runtimepolicy.RuntimeHandleUsaLegacyProcessPTY(mapFromJSONRuntimeHandle(handle.MetadataJSON)) {
			hasNonLegacy = true
			break
		}
	}
	for _, handle := range handles {
		if handle == nil || !strings.EqualFold(strings.TrimSpace(handle.Agente), strings.TrimSpace(runtime.Agente)) {
			continue
		}
		if hasNonLegacy && runtimepolicy.RuntimeHandleUsaLegacyProcessPTY(mapFromJSONRuntimeHandle(handle.MetadataJSON)) {
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

func (s *Service) CloseResidualRuntimeHandles(req RuntimeHandleResidualCloseRequest) (*RuntimeHandleResidualCloseResult, error) {
	agente := strings.TrimSpace(req.Agente)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	if agente == "" && proyectoRef == "" {
		return nil, fmt.Errorf("agente o proyecto son obligatorios")
	}
	var proyectoID *int64
	if agente != "" {
		if _, err := s.store.GetAgent(agente); err != nil {
			return nil, err
		}
	}
	if proyectoRef != "" {
		proyecto, err := s.store.GetProject(proyectoRef)
		if err != nil {
			return nil, err
		}
		proyectoID = &proyecto.ID
	}
	if err := s.store.CloseRuntimeHandles(agente, proyectoID); err != nil {
		return nil, err
	}
	detalle := fmt.Sprintf("agente=%s proyecto=%s", valueOrFallback(agente, "*"), valueOrFallback(proyectoRef, "*"))
	s.store.Audit(actor, "cerrar_runtime_handles_residuales", "runtime_handle", 0, detalle)
	return &RuntimeHandleResidualCloseResult{Closed: true}, nil
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

func (s *Service) CloseResidualRuntimes(req RuntimeResidualCloseRequest) (*RuntimeResidualCloseResult, error) {
	agente := strings.TrimSpace(req.Agente)
	proyectoRef := strings.TrimSpace(req.Proyecto)
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	if agente == "" && proyectoRef == "" {
		return nil, fmt.Errorf("agente o proyecto son obligatorios")
	}

	var (
		filtroAgente *string
		proyectoID   *int64
		activos      = true
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
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{
		Agente:     filtroAgente,
		ProyectoID: proyectoID,
		Activos:    &activos,
	})
	if err != nil {
		return nil, err
	}
	resultado := &RuntimeResidualCloseResult{}
	for _, runtime := range runtimes {
		if runtime == nil || runtime.ID <= 0 || strings.EqualFold(strings.TrimSpace(runtime.LogicalState), "cerrado") {
			continue
		}
		if err := s.store.CloseRuntime(runtime.ID); err != nil {
			return nil, err
		}
		resultado.Closed++
		resultado.ClosedIDs = append(resultado.ClosedIDs, runtime.ID)
	}
	detalle := fmt.Sprintf("agente=%s proyecto=%s closed=%d ids=%v", valueOrFallback(agente, "*"), valueOrFallback(proyectoRef, "*"), resultado.Closed, resultado.ClosedIDs)
	s.store.Audit(actor, "cerrar_runtimes_residuales", "runtime_instance", 0, detalle)
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

func (s *Service) RegistrarSalidaObservada(agente string, proyectoID *int64, texto string) (int64, error) {
	agente = strings.TrimSpace(agente)
	texto = strings.TrimSpace(texto)
	if agente == "" {
		return 0, fmt.Errorf("agente obligatorio")
	}
	if texto == "" {
		return 0, fmt.Errorf("texto obligatorio")
	}
	handle, err := s.ResolveDeliveryHandle(agente, proyectoID)
	if err != nil {
		return 0, err
	}
	if handle == nil {
		return 0, fmt.Errorf("sin handle entregable para %s", agente)
	}
	runtime, err := s.runtimeForHandle(agente, proyectoID, handle)
	if err != nil {
		return 0, err
	}
	if runtime == nil {
		return 0, fmt.Errorf("sin runtime activo para %s", agente)
	}
	entry := &db.RuntimeTranscriptEntry{
		RuntimeID:  runtime.ID,
		HandleID:   &handle.ID,
		Agente:     agente,
		ProyectoID: proyectoID,
		Stream:     "pty_out",
		Text:       texto,
	}
	id, err := s.store.RegisterRuntimeTranscript(entry)
	if err != nil {
		return 0, err
	}
	if s.afterRegisterTranscriptHook != nil {
		s.afterRegisterTranscriptHook(entry, id)
	}
	return id, nil
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

func (s *Service) MarcarRuntimeOrderDispatchNotificado(runtimeOrderID int64, detalle string) error {
	order, err := s.store.GetRuntimeOrder(runtimeOrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("runtime order no encontrada")
	}
	reason := strings.TrimSpace(detalle)
	if reason == "" {
		reason = "dispatch notificado pendiente de entrega"
	}
	resultadoJSON := fusionarResultadoDispatchDurable(order.ResultadoJSON, EntradaResultadoDispatchDurable{
		EstadoDispatch: DispatchNotificada,
		EstadoEntrega:  "notified",
		UltimaRazon:    reason,
	})
	resultadoJSON = mergeRuntimeOrderResultJSONApp(resultadoJSON, map[string]any{
		"ok":                   false,
		"deferred":             true,
		"deferred_reason":      reason,
		"delivery_notified_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	return s.store.MarkRuntimeOrderState(runtimeOrderID, "pendiente", resultadoJSON, reason)
}

func (s *Service) MarcarRuntimeOrderDispatchFallido(runtimeOrderID int64, detalle string) error {
	order, err := s.store.GetRuntimeOrder(runtimeOrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("runtime order no encontrada")
	}
	reason := strings.TrimSpace(detalle)
	if reason == "" {
		reason = "dispatch fallido"
	}
	resultadoJSON := fusionarResultadoDispatchDurable(order.ResultadoJSON, EntradaResultadoDispatchDurable{
		EstadoDispatch: DispatchFallida,
		EstadoEntrega:  "failed",
		UltimaRazon:    reason,
	})
	resultadoJSON = mergeRuntimeOrderResultJSONApp(resultadoJSON, map[string]any{
		"ok":                 false,
		"delivery_failed_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	return s.store.MarkRuntimeOrderState(runtimeOrderID, "fallida", resultadoJSON, reason)
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

func (s *Service) CancelRuntimeOrder(id int64, actor, motivo string) error {
	order, err := s.store.GetRuntimeOrder(id)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("runtime order no encontrada")
	}
	switch strings.ToLower(strings.TrimSpace(order.Estado)) {
	case "completada", "fallida", "cancelada", "expirada":
		return nil
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "orquesta"
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		motivo = "cancelada manualmente"
	}
	resultadoJSON := mergeRuntimeOrderResultJSONApp(order.ResultadoJSON, map[string]any{
		"ok":           false,
		"cancelled_at": time.Now().UTC().Format(time.RFC3339Nano),
		"cancelled_by": actor,
		"reason":       motivo,
	})
	if err := s.store.MarkRuntimeOrderState(id, "cancelada", resultadoJSON, motivo); err != nil {
		return err
	}
	s.store.Audit(actor, "cancelar_runtime_order", "runtime_order", id, motivo)
	return nil
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

func (s *Service) EnqueueAgentNudge(req AgentNudgeRequest) (*AgentNudgeResult, error) {
	agente := strings.TrimSpace(req.Agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	proyectoRef := strings.TrimSpace(req.Proyecto)
	if proyectoRef == "" {
		return nil, fmt.Errorf("proyecto obligatorio")
	}
	proyecto, err := s.store.GetProject(proyectoRef)
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto no encontrado")
	}
	accion := strings.TrimSpace(req.Accion)
	if accion == "" {
		return nil, fmt.Errorf("accion obligatoria")
	}
	var (
		handleID  *int64
		runtimeID *int64
	)
	handle, err := s.ResolveDeliveryHandle(agente, &proyecto.ID)
	if err != nil {
		return nil, err
	}
	if handle != nil {
		handleID = &handle.ID
		if runtime, err := s.runtimeForHandle(agente, &proyecto.ID, handle); err != nil {
			return nil, err
		} else if runtime != nil && runtime.ID > 0 {
			runtimeID = &runtime.ID
		} else if handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
	}
	payload := map[string]any{
		"from_agente": "server",
		"to_agente":   agente,
		"kind":        "pipeline_local",
		"accion":      accion,
		"texto":       strings.TrimSpace(req.Motivo),
	}
	if instruction := strings.TrimSpace(req.Instruction); instruction != "" {
		payload["instruction"] = instruction
	}
	for key, value := range req.Metadata {
		payload[strings.TrimSpace(key)] = value
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	orderID, err := s.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  &proyecto.ID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "nudge",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return nil, err
	}
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	s.store.Audit(actor, "pipeline_nudge", "runtime_order", orderID, fmt.Sprintf("agente=%s proyecto=%s accion=%s", agente, proyecto.Slug, accion))
	return &AgentNudgeResult{RuntimeOrderID: orderID}, nil
}

func (s *Service) DispatchMicroprogramacionInstruction(req MicroprogramacionDispatchRequest) (*MicroprogramacionDispatchResult, error) {
	agenteDestino := strings.TrimSpace(req.AgenteDestino)
	if agenteDestino == "" {
		return nil, fmt.Errorf("agente destino obligatorio")
	}
	formatoSalida := s.resolverFormatoSalidaEfectivoMicroprogramacion(req)
	if req.ProyectoID == nil || *req.ProyectoID <= 0 {
		return nil, fmt.Errorf("proyecto obligatorio para despachar microtarea")
	}
	req.FormatoSalida = formatoSalida
	mensaje, err := s.prepararMensajeMicroprogramacion(req)
	if err != nil {
		return nil, err
	}
	microprogramacion := map[string]any{
		"especificacion_id": req.EspecificacionID,
		"archivo_objetivo":  strings.TrimSpace(req.ArchivoObjetivo),
		"simbolo_objetivo":  strings.TrimSpace(req.SimboloObjetivo),
		"write_set":         req.WriteSet,
		"tests":             req.TestsObligatorios,
		"formato_salida":    strings.TrimSpace(formatoSalida),
	}
	if worktree := s.resolveWorktreePayloadMicroprogramacion(req); worktree != nil {
		microprogramacion["worktree_id"] = worktree.ID
		microprogramacion["ruta_worktree"] = strings.TrimSpace(worktree.RutaAbs)
		microprogramacion["branch_worktree"] = strings.TrimSpace(worktree.Branch)
		microprogramacion["base_ref_worktree"] = strings.TrimSpace(worktree.BaseRef)
	}
	payload, err := json.Marshal(map[string]any{
		"to_agente":         agenteDestino,
		"texto":             mensaje,
		"source":            "microprogramacion",
		"microprogramacion": microprogramacion,
	})
	if err != nil {
		return nil, fmt.Errorf("serializando payload de microprogramacion: %w", err)
	}
	if err := s.supersederMicroprogramacionAbiertaEquivalente(agenteDestino, req.ProyectoID, microprogramacion); err != nil {
		return nil, err
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

func (s *Service) supersederMicroprogramacionAbiertaEquivalente(agente string, proyectoID *int64, micro map[string]any) error {
	if s == nil {
		return nil
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Limit:      50,
	})
	if err != nil {
		return err
	}
	for _, order := range orders {
		if !runtimeOrderMicroprogramacionEquivalente(order, micro) {
			continue
		}
		payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
		if err != nil {
			return err
		}
		if mailboxID := int64Any(payload["mailbox_id"]); mailboxID > 0 {
			if err := s.store.MarkRuntimeMailboxConsumed(mailboxID); err != nil {
				return err
			}
		}
		resultadoJSON := mergeRuntimeOrderResultJSONApp(order.ResultadoJSON, map[string]any{
			"ok":                false,
			"superseded":        true,
			"superseded_by_new": true,
			"mailbox_id":        int64Any(payload["mailbox_id"]),
			"archivo_objetivo":  strings.TrimSpace(stringAny(micro["archivo_objetivo"])),
			"simbolo_objetivo":  strings.TrimSpace(stringAny(micro["simbolo_objetivo"])),
			"delivery_state":    "superseded",
			"superseded_at":     time.Now().UTC().Format(time.RFC3339Nano),
			"write_set":         stringSliceAny(micro["write_set"]),
			"especificacion_id": int64Any(micro["especificacion_id"]),
		})
		if err := s.store.MarkRuntimeOrderState(order.ID, "cancelada", resultadoJSON, "microprogramacion supersedida por una orden nueva equivalente"); err != nil {
			return err
		}
	}
	return nil
}

func runtimeOrderMicroprogramacionEquivalente(order *db.RuntimeOrder, micro map[string]any) bool {
	if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(order.Estado)) {
	case "completada", "fallida", "cancelada", "expirada":
		return false
	}
	payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
	if err != nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(stringAny(payload["source"])), "microprogramacion") {
		return false
	}
	actual, _ := payload["microprogramacion"].(map[string]any)
	if actual == nil {
		return false
	}
	if strings.TrimSpace(stringAny(actual["archivo_objetivo"])) != strings.TrimSpace(stringAny(micro["archivo_objetivo"])) {
		return false
	}
	if strings.TrimSpace(stringAny(actual["simbolo_objetivo"])) != strings.TrimSpace(stringAny(micro["simbolo_objetivo"])) {
		return false
	}
	return runtimeOrderMicroprogramacionMismoWriteSet(stringSliceAny(actual["write_set"]), stringSliceAny(micro["write_set"]))
}

func runtimeOrderMicroprogramacionMismoWriteSet(actual, esperado []string) bool {
	if len(actual) != len(esperado) {
		return false
	}
	normalizar := func(items []string) []string {
		out := make([]string, 0, len(items))
		for _, item := range items {
			if ruta := strings.TrimSpace(item); ruta != "" {
				out = append(out, ruta)
			}
		}
		sort.Strings(out)
		return out
	}
	a := normalizar(actual)
	b := normalizar(esperado)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Service) resolverFormatoSalidaEfectivoMicroprogramacion(req MicroprogramacionDispatchRequest) string {
	formato := strings.TrimSpace(req.FormatoSalida)
	if formato == "" {
		return formato
	}
	if agenteUsaMicroprogramacionInline(req.AgenteDestino) {
		if microprogramacionapp.FormatoSalidaUsaGitWorktree(formato) &&
			!s.agenteUsaPoolLocalCompartido(req.AgenteDestino, req.ProyectoID) {
			return "ficheros+evidencia"
		}
		if !microprogramacionapp.FormatoSalidaUsaGitWorktree(formato) && !formatoSalidaUsaBloquesArchivo(formato) {
			return "ficheros+evidencia"
		}
	}
	return formato
}

func (s *Service) agenteUsaPoolLocalCompartido(agente string, proyectoID *int64) bool {
	if s == nil {
		return false
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false
	}
	if proyectoID != nil && *proyectoID > 0 {
		runtime, err := s.store.GetPrimaryRuntimeForProject(agente, proyectoID)
		if err == nil && runtimeConectorEsPoolLocalCompartido(runtime) {
			return true
		}
	}
	runtime, err := s.store.GetPrimaryRuntime(agente)
	if err != nil {
		return false
	}
	return runtimeConectorEsPoolLocalCompartido(runtime)
}

func runtimeConectorEsPoolLocalCompartido(runtime *db.RuntimeInstance) bool {
	if runtime == nil {
		return false
	}
	conector := strings.ToLower(strings.TrimSpace(runtime.Connector))
	return conector == "ollama_pool_local" || conector == "ollama-pool-local"
}

func (s *Service) resolveWorktreePayloadMicroprogramacion(req MicroprogramacionDispatchRequest) *db.Worktree {
	if s == nil || req.ProyectoID == nil || *req.ProyectoID <= 0 || !microprogramacionapp.FormatoSalidaUsaGitWorktree(strings.TrimSpace(req.FormatoSalida)) {
		return nil
	}
	proyecto, err := s.store.GetProject(fmt.Sprintf("%d", *req.ProyectoID))
	if err != nil || proyecto == nil || strings.TrimSpace(proyecto.Slug) == "" {
		return nil
	}
	proyectoSlug := strings.TrimSpace(proyecto.Slug)
	agente := strings.TrimSpace(req.AgenteDestino)
	if s.resolvedorWorktree != nil {
		worktree, err := s.resolvedorWorktree.ResolveActiveWorktree(proyectoSlug, agente)
		if err == nil && worktree != nil {
			return worktree
		}
	}
	if s.aseguradorWorktree != nil {
		worktree, err := s.aseguradorWorktree.EnsureActiveWorktree(proyectoSlug, agente)
		if err == nil && worktree != nil {
			return worktree
		}
	}
	return nil
}

func (s *Service) ResolverContextoEntregaMicroprogramacion(agente string, proyectoID *int64) (*ContextoEntregaMicroprogramacion, error) {
	contextos, err := s.resolverContextosEntregaMicroprogramacion(agente, proyectoID, nil)
	if err != nil || len(contextos) == 0 {
		return nil, err
	}
	return contextos[0], nil
}

func (s *Service) ResolverContextoEntregaMicroprogramacionParaRespuesta(agente string, proyectoID *int64, respuesta string) (*ContextoEntregaMicroprogramacion, error) {
	archivos := microprogramacionapp.ExtraerArchivosEntrega(respuesta)
	contextos, err := s.resolverContextosEntregaMicroprogramacion(agente, proyectoID, archivos)
	if err != nil || len(contextos) == 0 {
		return nil, err
	}
	return contextos[0], nil
}

func (s *Service) resolverContextosEntregaMicroprogramacion(agente string, proyectoID *int64, archivos []microprogramacionapp.ArchivoEntrega) ([]*ContextoEntregaMicroprogramacion, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Limit:      50,
	})
	if err != nil {
		return nil, err
	}
	var candidatos []*ContextoEntregaMicroprogramacion
	for _, order := range orders {
		ctx, err := contextoEntregaMicroprogramacionDesdeOrder(order)
		if err != nil {
			return nil, err
		}
		if ctx == nil {
			continue
		}
		if !contextoEntregaMicroprogramacionCoincideArchivos(ctx, archivos) {
			continue
		}
		candidatos = append(candidatos, ctx)
	}
	sort.Slice(candidatos, func(i, j int) bool {
		return candidatos[i].RuntimeOrderID > candidatos[j].RuntimeOrderID
	})
	return candidatos, nil
}

func (s *Service) CompletarEntregasMicroprogramacionParaRespuesta(agente string, proyectoID *int64, respuesta, receiptSource string) (int, error) {
	contextos, err := s.resolverContextosEntregaMicroprogramacion(strings.TrimSpace(agente), proyectoID, microprogramacionapp.ExtraerArchivosEntrega(respuesta))
	if err != nil {
		return 0, err
	}
	completadas := 0
	vistos := map[int64]struct{}{}
	for _, ctx := range contextos {
		if ctx == nil || ctx.RuntimeOrderID <= 0 {
			continue
		}
		if _, ok := vistos[ctx.RuntimeOrderID]; ok {
			continue
		}
		vistos[ctx.RuntimeOrderID] = struct{}{}
		if err := s.CompletarEntregaMicroprogramacion(ctx.RuntimeOrderID, receiptSource); err != nil {
			return completadas, err
		}
		completadas++
	}
	return completadas, nil
}

func (s *Service) MaterializarEntregaMicroprogramacionActivaDesdeRespuesta(agente string, proyectoID *int64, proyectoSlug, respuesta, evidencia, receiptSource string) (*microprogramacionapp.ResultadoMaterializarEntrega, error) {
	if s.materializadorEntrega == nil {
		return nil, fmt.Errorf("materializador de entrega no configurado")
	}
	ctx, err := s.ResolverContextoEntregaMicroprogramacionParaRespuesta(strings.TrimSpace(agente), proyectoID, respuesta)
	if err != nil {
		return nil, err
	}
	if ctx == nil || ctx.EspecificacionID <= 0 {
		return nil, nil
	}
	if microprogramacionapp.FormatoSalidaUsaGitWorktree(ctx.FormatoSalida) {
		return nil, nil
	}
	proyectoRef := strings.TrimSpace(proyectoSlug)
	if proyectoRef == "" && proyectoID != nil && *proyectoID > 0 {
		proyectoRef = strconv.FormatInt(*proyectoID, 10)
	}
	if strings.TrimSpace(proyectoRef) == "" {
		return nil, fmt.Errorf("proyecto obligatorio para materializar entrega")
	}
	proyecto, err := s.store.GetProject(proyectoRef)
	if err != nil {
		return nil, err
	}
	if proyecto == nil || strings.TrimSpace(proyecto.RutaAbs) == "" {
		return nil, fmt.Errorf("proyecto sin ruta absoluta para materializar entrega")
	}
	resultado, err := s.materializadorEntrega.MaterializarEntregaDesdeRespuesta(
		ctx.EspecificacionID,
		strings.TrimSpace(proyecto.RutaAbs),
		respuesta,
		strings.TrimSpace(evidencia),
	)
	if err != nil {
		return nil, err
	}
	if _, err := s.CompletarEntregasMicroprogramacionParaRespuesta(agente, proyectoID, respuesta, strings.TrimSpace(receiptSource)); err != nil {
		return nil, err
	}
	return resultado, nil
}

func (s *Service) ReencolarCorreccionEntregaMicroprogramacion(runtimeOrderID int64, detalle string) (int64, error) {
	if runtimeOrderID <= 0 {
		return 0, fmt.Errorf("runtime order obligatoria")
	}
	order, err := s.store.GetRuntimeOrder(runtimeOrderID)
	if err != nil {
		return 0, err
	}
	if order == nil {
		return 0, fmt.Errorf("runtime order no encontrada")
	}
	if strings.TrimSpace(order.Tipo) != "send_instruction" {
		return 0, fmt.Errorf("runtime order %d no es send_instruction", runtimeOrderID)
	}
	payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
	if err != nil {
		return 0, err
	}
	if !strings.EqualFold(strings.TrimSpace(stringAny(payload["source"])), "microprogramacion") {
		return 0, fmt.Errorf("runtime order %d no es de microprogramacion", runtimeOrderID)
	}
	micro, _ := payload["microprogramacion"].(map[string]any)
	if micro == nil {
		return 0, fmt.Errorf("runtime order %d sin metadata de microprogramacion", runtimeOrderID)
	}
	intentoCorreccion := int64Any(payload["correccion_intento"])
	if intentoCorreccion >= 1 {
		return 0, nil
	}
	motivo := strings.TrimSpace(detalle)
	if motivo == "" {
		motivo = "entrega invalida rechazada por Orquesta"
	}
	textoBase := strings.TrimSpace(stringAny(payload["texto"]))
	if textoBase == "" {
		return 0, fmt.Errorf("runtime order %d sin texto base", runtimeOrderID)
	}
	textoCorreccion := strings.TrimSpace(textoBase + "\n\nCORRECCION_OBLIGATORIA:\nLa entrega anterior fue rechazada por Orquesta.\nERROR_EXACTO: " + motivo + "\nReescribe solo los ficheros del WRITE_SET. Devuelve de nuevo una entrega completa valida que compile.")
	nuevoPayload := map[string]any{}
	for key, value := range payload {
		nuevoPayload[key] = value
	}
	delete(nuevoPayload, "mailbox_id")
	delete(nuevoPayload, "mailbox_kind")
	nuevoPayload["texto"] = textoCorreccion
	nuevoPayload["correccion_intento"] = intentoCorreccion + 1
	nuevoPayload["correccion_motivo"] = motivo
	if err := s.supersederMicroprogramacionAbiertaEquivalente(strings.TrimSpace(order.Agente), order.ProyectoID, micro); err != nil {
		return 0, err
	}
	raw, err := json.Marshal(nuevoPayload)
	if err != nil {
		return 0, fmt.Errorf("serializando payload de correccion: %w", err)
	}
	orderID, err := s.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(order.Agente),
		ProyectoID:  order.ProyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: string(raw),
	})
	if err != nil {
		return 0, err
	}
	return orderID, nil
}

func (s *Service) CompletarEntregaMicroprogramacion(runtimeOrderID int64, receiptSource string) error {
	if runtimeOrderID <= 0 {
		return fmt.Errorf("runtime order obligatoria")
	}
	order, err := s.store.GetRuntimeOrder(runtimeOrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("runtime order no encontrada")
	}
	switch strings.TrimSpace(order.Tipo) {
	case "send_instruction":
		return s.completarEntregaSendInstruction(order, receiptSource)
	case "start", "resume", "handoff":
		return s.completarEntregaBootstrapPremium(order, receiptSource)
	default:
		return fmt.Errorf("runtime order %d no soporta cierre de entrega", runtimeOrderID)
	}
}

func (s *Service) completarEntregaSendInstruction(order *db.RuntimeOrder, receiptSource string) error {
	payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
	if err != nil {
		return err
	}
	if mailboxID := int64Any(payload["mailbox_id"]); mailboxID > 0 {
		if err := s.store.MarkRuntimeMailboxConsumed(mailboxID); err != nil {
			return err
		}
	}
	resultadoJSON := mergeRuntimeOrderResultJSONApp(order.ResultadoJSON, map[string]any{
		"ok":                  true,
		"mailbox_id":          int64Any(payload["mailbox_id"]),
		"mailbox_only":        int64Any(payload["mailbox_id"]) > 0,
		"delivery_receipt_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	resultadoJSON = fusionarResultadoDispatchDurable(resultadoJSON, EntradaResultadoDispatchDurable{
		EstadoDispatch: DispatchEntregada,
		EstadoEntrega:  "delivered",
		ReceiptSource:  strings.TrimSpace(receiptSource),
		UltimaRazon:    "entrega valida confirmada por la app",
	})
	return s.store.MarkRuntimeOrderState(order.ID, "completada", resultadoJSON, "")
}

func (s *Service) completarEntregaBootstrapPremium(order *db.RuntimeOrder, receiptSource string) error {
	mailboxIDs, ok := mailboxIDsBootstrapLeaseApp(order.ResultadoJSON)
	if !ok {
		return fmt.Errorf("runtime order %d sin bootstrap lease valida", order.ID)
	}
	for _, mailboxID := range mailboxIDs {
		if mailboxID <= 0 {
			continue
		}
		if err := s.store.MarkRuntimeMailboxConsumed(mailboxID); err != nil {
			return err
		}
	}
	firstMailboxID := int64(0)
	if len(mailboxIDs) > 0 {
		firstMailboxID = mailboxIDs[0]
	}
	resultadoJSON := mergeRuntimeOrderResultJSONApp(order.ResultadoJSON, map[string]any{
		"ok":                  true,
		"mailbox_id":          firstMailboxID,
		"mailbox_ids":         mailboxIDs,
		"mailbox_only":        len(mailboxIDs) > 0,
		"delivery_receipt_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	resultadoJSON = fusionarResultadoDispatchDurable(resultadoJSON, EntradaResultadoDispatchDurable{
		EstadoDispatch: DispatchEntregada,
		EstadoEntrega:  "delivered",
		ReceiptSource:  strings.TrimSpace(receiptSource),
		UltimaRazon:    "entrega bootstrap validada por git/worktree",
	})
	return s.store.MarkRuntimeOrderState(order.ID, "completada", resultadoJSON, "")
}

func (s *Service) RegistrarEntregaGitMicroprogramacionActiva(agente string, proyectoID *int64, proyectoSlug, evidencia, solicitadoPor string) (*ResultadoEntregaGitMicroprogramacionActiva, error) {
	return s.registrarEntregaGitMicroprogramacionActiva(agente, proyectoID, proyectoSlug, evidencia, solicitadoPor, microprogramacionapp.EntradaRegistrarEntregaGit{})
}

func (s *Service) RegistrarEntregaGitMicroprogramacionActivaPreferente(agente string, proyectoID *int64, proyectoSlug, evidencia, solicitadoPor string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*ResultadoEntregaGitMicroprogramacionActiva, error) {
	return s.registrarEntregaGitMicroprogramacionActiva(agente, proyectoID, proyectoSlug, evidencia, solicitadoPor, entrada)
}

func (s *Service) registrarEntregaGitMicroprogramacionActiva(agente string, proyectoID *int64, proyectoSlug, evidencia, solicitadoPor string, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*ResultadoEntregaGitMicroprogramacionActiva, error) {
	ctx, err := s.ResolverContextoEntregaMicroprogramacion(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		if s.registradorEntregaGit == nil {
			return nil, fmt.Errorf("registrador de entrega git no configurado")
		}
		if !microprogramacionapp.FormatoSalidaUsaGitWorktree(ctx.FormatoSalida) {
			return nil, nil
		}
		entrada.Agente = strings.TrimSpace(agente)
		entrada.ProyectoID = proyectoID
		entrada.ProyectoSlug = strings.TrimSpace(proyectoSlug)
		entrada.Evidencia = strings.TrimSpace(evidencia)
		entrada.SolicitadoPor = strings.TrimSpace(solicitadoPor)
		if entrada.PreferenciaWorktreeID == nil && ctx.WorktreeID != nil && *ctx.WorktreeID > 0 {
			entrada.PreferenciaWorktreeID = ctx.WorktreeID
		}
		if strings.TrimSpace(entrada.PreferenciaRutaWorktree) == "" {
			entrada.PreferenciaRutaWorktree = strings.TrimSpace(ctx.RutaWorktree)
		}
		if strings.TrimSpace(entrada.PreferenciaBranch) == "" {
			entrada.PreferenciaBranch = strings.TrimSpace(ctx.BranchWorktree)
		}
		if strings.TrimSpace(entrada.PreferenciaBaseRef) == "" {
			entrada.PreferenciaBaseRef = strings.TrimSpace(ctx.BaseRefWorktree)
		}
		resultado, err := s.registradorEntregaGit.RegistrarEntregaGit(ctx.EspecificacionID, entrada)
		if err != nil {
			return nil, err
		}
		if err := s.CompletarEntregaMicroprogramacion(ctx.RuntimeOrderID, "git_worktree"); err != nil {
			return nil, err
		}
		return &ResultadoEntregaGitMicroprogramacionActiva{
			Contexto: ctx,
			Entrega:  resultado,
		}, nil
	}
	return s.registrarEntregaGitPremiumActiva(
		agente,
		proyectoID,
		proyectoSlug,
		evidencia,
		solicitadoPor,
		EntradaRegistrarEntregaGitPremium{
			Agente:                  strings.TrimSpace(agente),
			ProyectoID:              proyectoID,
			ProyectoSlug:            strings.TrimSpace(proyectoSlug),
			SolicitadoPor:           strings.TrimSpace(solicitadoPor),
			Evidencia:               strings.TrimSpace(evidencia),
			PreferenciaWorktreeID:   entrada.PreferenciaWorktreeID,
			PreferenciaRutaWorktree: strings.TrimSpace(entrada.PreferenciaRutaWorktree),
			PreferenciaBranch:       strings.TrimSpace(entrada.PreferenciaBranch),
			PreferenciaBaseRef:      strings.TrimSpace(entrada.PreferenciaBaseRef),
		},
	)
}

func (s *Service) IntentarRegistrarEntregaGitMicroprogramacionActiva(agente string, proyectoID *int64, proyectoSlug, evidencia, solicitadoPor string) (*ResultadoEntregaGitMicroprogramacionActiva, error) {
	resultado, err := s.RegistrarEntregaGitMicroprogramacionActiva(agente, proyectoID, proyectoSlug, evidencia, solicitadoPor)
	if err == nil || resultado != nil {
		return resultado, err
	}
	if entregaGitSinCambios(err) {
		return nil, nil
	}
	return nil, err
}

func (s *Service) registrarEntregaGitPremiumActiva(agente string, proyectoID *int64, proyectoSlug, evidencia, solicitadoPor string, entrada EntradaRegistrarEntregaGitPremium) (*ResultadoEntregaGitMicroprogramacionActiva, error) {
	if s.registradorEntregaGitPremium == nil {
		return nil, nil
	}
	ctx, err := s.ResolverContextoEntregaGitPremium(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, nil
	}
	entrada.Agente = strings.TrimSpace(agente)
	entrada.ProyectoID = proyectoID
	entrada.ProyectoSlug = strings.TrimSpace(proyectoSlug)
	entrada.Evidencia = strings.TrimSpace(evidencia)
	entrada.SolicitadoPor = strings.TrimSpace(solicitadoPor)
	entrada.Carril = strings.TrimSpace(ctx.Carril)
	entrada.TareaObjetivoID = ctx.TareaObjetivoID
	if len(entrada.WriteSet) == 0 && len(ctx.WriteSet) > 0 {
		entrada.WriteSet = append([]string(nil), ctx.WriteSet...)
	}
	if entrada.PreferenciaWorktreeID == nil && ctx.WorktreeID != nil && *ctx.WorktreeID > 0 {
		entrada.PreferenciaWorktreeID = ctx.WorktreeID
	}
	if strings.TrimSpace(entrada.PreferenciaRutaWorktree) == "" {
		entrada.PreferenciaRutaWorktree = strings.TrimSpace(ctx.RutaWorktree)
	}
	if strings.TrimSpace(entrada.PreferenciaBranch) == "" {
		entrada.PreferenciaBranch = strings.TrimSpace(ctx.BranchWorktree)
	}
	if strings.TrimSpace(entrada.PreferenciaBaseRef) == "" {
		entrada.PreferenciaBaseRef = strings.TrimSpace(ctx.BaseRefWorktree)
	}
	resultado, err := s.registradorEntregaGitPremium.RegistrarEntregaGitPremium(entrada)
	if err != nil {
		return nil, err
	}
	if resultado == nil {
		return nil, nil
	}
	if err := s.completarTareaEntregaGitPremiumSiCorresponde(ctx, strings.TrimSpace(agente), resultado); err != nil {
		return nil, err
	}
	if err := s.CompletarEntregaMicroprogramacion(ctx.RuntimeOrderID, "git_worktree"); err != nil {
		return nil, err
	}
	return &ResultadoEntregaGitMicroprogramacionActiva{
		ContextoPremium: ctx,
		EntregaPremium:  resultado,
	}, nil
}

func (s *Service) completarTareaEntregaGitPremiumSiCorresponde(ctx *ContextoEntregaGitPremium, agente string, resultado *ResultadoRegistrarEntregaGitPremium) error {
	if s == nil || s.taskCompleter == nil || ctx == nil || ctx.TareaObjetivoID <= 0 {
		return nil
	}
	if err := s.taskCompleter.Complete(ctx.TareaObjetivoID, strings.TrimSpace(agente), strings.TrimSpace(resultado.HeadCommit)); err != nil {
		if tareaNoEncontrada(err) {
			return nil
		}
		return err
	}
	return nil
}

func tareaNoEncontrada(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "tarea #") &&
		strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no encontrada")
}

func entregaGitSinCambios(err error) bool {
	if err == nil {
		return false
	}
	texto := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(texto, "no existe entrega git capturada"):
		return true
	case strings.Contains(texto, "no contiene archivos modificados"):
		return true
	case strings.Contains(texto, "no incluye el archivo objetivo"):
		return true
	}
	return false
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

func contextoEntregaMicroprogramacionDesdeOrder(order *db.RuntimeOrder) (*ContextoEntregaMicroprogramacion, error) {
	if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" || strings.TrimSpace(order.PayloadJSON) == "" {
		return nil, nil
	}
	payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stringMapValue(payload, "source")) != "microprogramacion" {
		return nil, nil
	}
	micro, _ := payload["microprogramacion"].(map[string]any)
	if micro == nil {
		return nil, nil
	}
	especificacionID := int64Any(micro["especificacion_id"])
	if especificacionID <= 0 {
		return nil, nil
	}
	ctx := &ContextoEntregaMicroprogramacion{
		RuntimeOrderID:   order.ID,
		EspecificacionID: especificacionID,
		ArchivoObjetivo:  strings.TrimSpace(stringAny(micro["archivo_objetivo"])),
		SimboloObjetivo:  strings.TrimSpace(stringAny(micro["simbolo_objetivo"])),
		WriteSet:         stringSliceAny(micro["write_set"]),
		FormatoSalida:    strings.TrimSpace(stringAny(micro["formato_salida"])),
		RutaWorktree:     strings.TrimSpace(stringAny(micro["ruta_worktree"])),
		BranchWorktree:   strings.TrimSpace(stringAny(micro["branch_worktree"])),
		BaseRefWorktree:  strings.TrimSpace(stringAny(micro["base_ref_worktree"])),
	}
	if worktreeID := int64Any(micro["worktree_id"]); worktreeID > 0 {
		ctx.WorktreeID = &worktreeID
	}
	return ctx, nil
}

func (s *Service) ResolverContextoEntregaGitPremium(agente string, proyectoID *int64) (*ContextoEntregaGitPremium, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Limit:      20,
	})
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "encolada", "ejecutando", "completada":
		default:
			continue
		}
		ctx, err := s.contextoEntregaGitPremiumDesdeOrder(order)
		if err != nil {
			return nil, err
		}
		if ctx != nil {
			return ctx, nil
		}
	}
	return nil, nil
}

func (s *Service) contextoEntregaGitPremiumDesdeOrder(order *db.RuntimeOrder) (*ContextoEntregaGitPremium, error) {
	if order == nil {
		return nil, nil
	}
	switch strings.TrimSpace(order.Tipo) {
	case "send_instruction":
		return contextoEntregaGitPremiumDesdeSendInstructionOrder(order)
	case "start", "resume", "handoff":
		return s.contextoEntregaGitPremiumDesdeBootstrapOrder(order)
	default:
		return nil, nil
	}
}

func contextoEntregaGitPremiumDesdeSendInstructionOrder(order *db.RuntimeOrder) (*ContextoEntregaGitPremium, error) {
	if order == nil || strings.TrimSpace(order.PayloadJSON) == "" {
		return nil, nil
	}
	payload, err := decodePayloadMicroprogramacion(order.PayloadJSON)
	if err != nil {
		return nil, err
	}
	return contextoEntregaGitPremiumDesdePayload(order.ID, payload), nil
}

func (s *Service) contextoEntregaGitPremiumDesdeBootstrapOrder(order *db.RuntimeOrder) (*ContextoEntregaGitPremium, error) {
	mailboxIDs, ok := mailboxIDsBootstrapLeaseApp(order.ResultadoJSON)
	if !ok {
		return nil, nil
	}
	for _, mailboxID := range mailboxIDs {
		msg, err := s.store.GetRuntimeMailbox(mailboxID)
		if err != nil {
			return nil, err
		}
		if msg == nil || strings.TrimSpace(msg.PayloadJSON) == "" {
			continue
		}
		payload, err := decodePayloadMicroprogramacion(msg.PayloadJSON)
		if err != nil {
			return nil, err
		}
		if ctx := contextoEntregaGitPremiumDesdePayload(order.ID, payload); ctx != nil {
			return ctx, nil
		}
	}
	return nil, nil
}

func contextoEntregaGitPremiumDesdePayload(runtimeOrderID int64, payload map[string]any) *ContextoEntregaGitPremium {
	isPipelineLocal := strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "source")), "pipeline_local") ||
		strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "kind")), "pipeline_local") ||
		strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "mailbox_kind")), "pipeline_local")
	if !isPipelineLocal {
		return nil
	}
	carril := strings.ToLower(strings.TrimSpace(stringMapValue(payload, "carril")))
	switch carril {
	case "premium_worktree", "revision_diff":
	default:
		return nil
	}
	ctx := &ContextoEntregaGitPremium{
		RuntimeOrderID:  runtimeOrderID,
		Carril:          carril,
		TareaObjetivoID: int64Any(payload["tarea_objetivo_id"]),
		WriteSet:        stringSliceAny(payload["write_set"]),
		RutaWorktree:    strings.TrimSpace(stringAny(payload["ruta_worktree"])),
		BranchWorktree:  strings.TrimSpace(stringAny(payload["branch_worktree"])),
		BaseRefWorktree: strings.TrimSpace(stringAny(payload["base_ref_worktree"])),
	}
	if worktreeID := int64Any(payload["worktree_id"]); worktreeID > 0 {
		ctx.WorktreeID = &worktreeID
	}
	return ctx
}

func mailboxIDsBootstrapLeaseApp(raw string) ([]int64, bool) {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return nil, false
	}
	leaseState := strings.ToLower(strings.TrimSpace(stringMapValue(payload, "lease_state")))
	switch leaseState {
	case "waiting_for_evidence", "delivered", "acked":
	default:
		return nil, false
	}
	ids := int64SliceAny(payload["mailbox_ids"])
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}

func decodePayloadMicroprogramacion(raw string) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("payload_json inválido: %w", err)
	}
	return payload, nil
}

func stringMapValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	return stringAny(payload[key])
}

func stringAny(v any) string {
	s, _ := v.(string)
	return s
}

func int64Any(v any) int64 {
	switch value := v.(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func int64SliceAny(v any) []int64 {
	items, _ := v.([]any)
	if len(items) == 0 {
		return nil
	}
	resultado := make([]int64, 0, len(items))
	for _, item := range items {
		if id := int64Any(item); id > 0 {
			resultado = append(resultado, id)
		}
	}
	return resultado
}

func stringSliceAny(v any) []string {
	switch typed := v.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s := strings.TrimSpace(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	items, _ := v.([]any)
	if len(items) == 0 {
		return nil
	}
	resultado := make([]string, 0, len(items))
	for _, item := range items {
		if s := strings.TrimSpace(stringAny(item)); s != "" {
			resultado = append(resultado, s)
		}
	}
	return resultado
}

func contextoEntregaMicroprogramacionCoincideArchivos(ctx *ContextoEntregaMicroprogramacion, archivos []microprogramacionapp.ArchivoEntrega) bool {
	if ctx == nil {
		return false
	}
	if len(archivos) == 0 {
		return true
	}
	if len(ctx.WriteSet) == 0 {
		return false
	}
	writeSet := make(map[string]struct{}, len(ctx.WriteSet))
	for _, ruta := range ctx.WriteSet {
		ruta = strings.TrimSpace(ruta)
		if ruta != "" {
			writeSet[ruta] = struct{}{}
		}
	}
	if len(writeSet) == 0 {
		return false
	}
	for _, archivo := range archivos {
		ruta := strings.TrimSpace(archivo.RutaRelativa)
		if ruta == "" {
			continue
		}
		if _, ok := writeSet[ruta]; !ok {
			return false
		}
	}
	return true
}

func mergeRuntimeOrderResultJSONApp(base string, values map[string]any) string {
	resultado := map[string]any{}
	if strings.TrimSpace(base) != "" {
		_ = json.Unmarshal([]byte(base), &resultado)
	}
	for key, value := range values {
		resultado[key] = value
	}
	raw, err := json.Marshal(resultado)
	if err != nil {
		return strings.TrimSpace(base)
	}
	return string(raw)
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

func (s *Service) ClearRuntimeMailbox(filter db.FiltroRuntimeMailbox) ([]int64, error) {
	items, err := s.store.ListRuntimeMailbox(filter)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item == nil || item.ID <= 0 {
			continue
		}
		if err := s.store.MarkRuntimeMailboxConsumed(item.ID); err != nil {
			return ids, err
		}
		ids = append(ids, item.ID)
	}
	return ids, nil
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
	if accion == "resume" && handle == nil {
		handle, err = s.resolveAgentControlResumeHandle(agente, proyectoID)
		if err != nil {
			return 0, "", err
		}
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
		if accion == "resume" && agentControlResumeRequiresFreshStart(handle) {
			accion = "start"
			handleID = nil
			runtimeID = nil
		}
	} else if accion == "resume" {
		accion = "start"
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
		TareaID:      req.TareaID,
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

func agentControlResumeRequiresFreshStart(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return true
	}
	if db.RuntimeHandlePauseRequiresFreshStart(handle) {
		return true
	}
	meta := mapFromJSONRuntimeHandle(handle.MetadataJSON)
	if strings.EqualFold(mapStringValueRuntimeHandle(meta, "driver"), "tmux_cli_session") &&
		!db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		return true
	}
	return false
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

func (Repository) CloseRuntime(id int64) error {
	return db.MarcarRuntimeCerrado(id)
}

func (Repository) CloseRuntimeHandles(agente string, proyectoID *int64) error {
	return db.MarcarRuntimeHandlesCerrados(agente, proyectoID)
}

func (Repository) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandle(id)
}

func (Repository) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(filtro)
}

func (Repository) ListPassiveRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesPasivos(filtro)
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

func (Repository) RegisterRuntimeTranscript(entry *db.RuntimeTranscriptEntry) (int64, error) {
	return db.RegistrarRuntimeTranscript(entry)
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

func (Repository) MarkRuntimeOrderState(id int64, estado, resultadoJSON, errorText string) error {
	return db.MarcarRuntimeOrderEstado(id, estado, resultadoJSON, errorText)
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
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
		handle, err = s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if handle != nil {
			return handle, nil
		}
	}
	handle, err := s.store.GetOperationalRuntimeHandle(agente)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if handle != nil {
		return handle, nil
	}
	handle, err = s.store.GetActiveRuntimeHandle(agente)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return handle, nil
}

func (s *Service) resolveAgentControlResumeHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	handles, err := s.store.ListCanonicalRuntimeHandles(&agente)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if strings.TrimSpace(handle.Agente) != strings.TrimSpace(agente) {
			continue
		}
		if proyectoID != nil {
			if handle.ProyectoID == nil || *handle.ProyectoID != *proyectoID {
				continue
			}
		}
		if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
			db.RuntimeHandlePauseRequiresFreshStart(handle) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
			return handle, nil
		}
		meta := mapFromJSONRuntimeHandle(handle.MetadataJSON)
		if strings.TrimSpace(mapStringValueRuntimeHandle(meta, "external_session_id")) != "" {
			return handle, nil
		}
	}
	return nil, nil
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
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if runtime == nil {
		runtime, err = s.store.GetPrimaryRuntime(agente)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if runtime == nil && handle != nil && handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtime, err = s.store.GetRuntime(*handle.RuntimeID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return runtime, nil
}

func runtimeHandleShouldSupersedeStart(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil {
		return false
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON)); err == nil && snap != nil {
		state := strings.ToLower(strings.TrimSpace(snap.EffectiveState()))
		switch state {
		case "stale", "stopped", "failed", "exited", "closed":
			return true
		}
		if snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) || !snap.Alive() {
			return true
		}
		switch state {
		case "starting":
			return false
		}
	}
	if runtimeHandleTMUXSessionMissing(handle) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "pausado":
	default:
		return true
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
	if handle == nil {
		return false
	}
	meta := mapFromJSONRuntimeHandle(handle.MetadataJSON)
	if !runtimepolicy.RuntimeHandleIsLegacyControlPlane(handle.Estado, handle.Transporte, handle.HandleKind, meta) {
		return false
	}
	return runtimepolicy.RuntimeHandleUsaLegacyCLITMUXPreferred(meta)
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
