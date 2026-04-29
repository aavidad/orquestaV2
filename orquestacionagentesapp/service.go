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
	"strconv"
	"strings"
	"sync"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/internal/controlruntime"
	"orquesta/runtimesapp"
)

type AgentPreparer interface {
	BuildPrepare(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error)
}

type RuntimeController interface {
	EnqueueAgentControl(req runtimesapp.AgentControlRequest) (int64, string, error)
	CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error)
	CreateStaleAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error)
	GetProject(ref string) (*db.Proyecto, error)
	ReconcileStaleRuntimeHandles() (int, error)
	ReconcileStaleRuntimeOrders() (int, error)
	PurgeInactiveRuntimeHandles(req runtimesapp.RuntimeHandlePurgeRequest) (*db.PurgaRuntimeHandlesResultado, error)
	PurgeTerminalRuntimeOrders(req runtimesapp.RuntimeOrderPurgeRequest) (*db.PurgaRuntimeOrdersResultado, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	GetRuntimeHandle(id int64) (*db.RuntimeHandle, error)
	GetRuntimeBySessionID(sessionID int64) (*db.RuntimeInstance, error)
	GetRuntimeHandleBySessionID(sessionID int64) (*db.RuntimeHandle, error)
	SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error)
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

type StartableWorkChecker interface {
	HasStartableAgentWork(agente string, proyectoID int64) (bool, error)
}

type RemoteRecoverySupport interface {
	ResolveSessionConnector(sesion *db.Sesion, runtime *db.RuntimeInstance, handle *db.RuntimeHandle) (*db.Conector, error)
	IsConnectorAvailable(conectorID int64) (bool, error)
	MarkProjectExternallyBlocked(proyectoID int64, motivo string) error
}

type RecoveryFlowSupport interface {
	SharedAccountAvailable(agente string) (bool, string, error)
	UsesSharedLocalPoolForReactivation(agente string, proyecto *db.Proyecto) bool
	PoolLocalActivationAllowed(agente string, proyectoSlug string) (bool, string, error)
	ProjectHasReactivableBacklog(proyectoID int64) (bool, error)
	ProjectIDFromAssignedWork(agente string) (int64, error)
	ActiveProjectID(agente string) (int64, error)
	OpenSessionProjectID(agente string) (int64, error)
	LatestSessionProjectID(agente string) (int64, error)
}

type Service struct {
	agents        AgentPreparer
	runtimes      RuntimeController
	autonomyStore AutonomyStore
	workChecker   StartableWorkChecker
	remoteSupport RemoteRecoverySupport
	recoveryFlow  RecoveryFlowSupport
}

var registrarAutonomyEventFn = db.RegistrarAutonomyEvent

var (
	startSessionHygieneMu       sync.Mutex
	startSessionHygieneRunning  bool
	startSessionHygieneLastRun  time.Time
	startSessionHygieneCooldown = 20 * time.Second
)

func NewService(agents AgentPreparer, runtimes RuntimeController) *Service {
	return &Service{agents: agents, runtimes: runtimes}
}

func (s *Service) SetAutonomyStore(store AutonomyStore) {
	if s == nil {
		return
	}
	s.autonomyStore = store
}

func (s *Service) SetStartableWorkChecker(checker StartableWorkChecker) {
	if s == nil {
		return
	}
	s.workChecker = checker
}

func (s *Service) SetRemoteRecoverySupport(support RemoteRecoverySupport) {
	if s == nil {
		return
	}
	s.remoteSupport = support
}

func (s *Service) SetRecoveryFlowSupport(support RecoveryFlowSupport) {
	if s == nil {
		return
	}
	s.recoveryFlow = support
}

func registrarAutonomyEventBestEffort(ev *db.AutonomyEvent) {
	if ev == nil {
		return
	}
	_ = registrarAutonomyEventFn(ev)
}

func ptrInt64Orq(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	value := v
	return &value
}

func recordWorkerRecoveryRequested(source, reason, action, agente string, proyectoID, runtimeID *int64, handle *db.RuntimeHandle, orderIDs map[string]int64) {
	delta := map[string]any{
		"last_autonomy_action": "worker_recovery_requested",
		"agente":               strings.TrimSpace(agente),
		"control_action":       strings.TrimSpace(action),
	}
	if handle != nil {
		delta["handle_state_before"] = strings.TrimSpace(handle.Estado)
	}
	for key, value := range orderIDs {
		if strings.TrimSpace(key) == "" || value <= 0 {
			continue
		}
		delta[key] = value
	}
	registrarAutonomyEventBestEffort(&db.AutonomyEvent{
		Kind:      "worker_recovery_requested",
		Actor:     "orquesta",
		ProjectID: proyectoID,
		RuntimeID: runtimeID,
		HandleID: func() *int64 {
			if handle == nil {
				return nil
			}
			return ptrInt64Orq(handle.ID)
		}(),
		Source:     strings.TrimSpace(source),
		Reason:     strings.TrimSpace(reason),
		StateDelta: delta,
	})
}

func tryBeginStartSessionHygiene(now time.Time) bool {
	startSessionHygieneMu.Lock()
	defer startSessionHygieneMu.Unlock()
	if startSessionHygieneRunning {
		return false
	}
	if !startSessionHygieneLastRun.IsZero() && now.Sub(startSessionHygieneLastRun) < startSessionHygieneCooldown {
		return false
	}
	startSessionHygieneRunning = true
	return true
}

func finishStartSessionHygiene(success bool, completedAt time.Time) {
	startSessionHygieneMu.Lock()
	defer startSessionHygieneMu.Unlock()
	startSessionHygieneRunning = false
	if success {
		startSessionHygieneLastRun = completedAt.UTC()
	}
}

func resetStartSessionHygieneStateForTests() {
	startSessionHygieneMu.Lock()
	defer startSessionHygieneMu.Unlock()
	startSessionHygieneRunning = false
	startSessionHygieneLastRun = time.Time{}
}

func recordWorkerRecoveryPausedExternal(source, reason, agente string, proyectoID, runtimeID *int64, handle *db.RuntimeHandle, orderIDs map[string]int64) {
	delta := map[string]any{
		"last_autonomy_action": "worker_recovery_paused_external",
		"agente":               strings.TrimSpace(agente),
		"control_action":       "pause",
		"external_block_kind":  "connector_unavailable",
	}
	if handle != nil {
		delta["handle_state_before"] = strings.TrimSpace(handle.Estado)
	}
	for key, value := range orderIDs {
		if strings.TrimSpace(key) == "" || value <= 0 {
			continue
		}
		delta[key] = value
	}
	registrarAutonomyEventBestEffort(&db.AutonomyEvent{
		Kind:      "worker_recovery_paused_external",
		Actor:     "orquesta",
		ProjectID: proyectoID,
		RuntimeID: runtimeID,
		HandleID: func() *int64 {
			if handle == nil {
				return nil
			}
			return ptrInt64Orq(handle.ID)
		}(),
		Source:     strings.TrimSpace(source),
		Reason:     strings.TrimSpace(reason),
		StateDelta: delta,
	})
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

type PostRemediationFollowupRequest struct {
	Agente          string
	Proyecto        *db.Proyecto
	TareaID         int64
	RemediationKind string
	OriginAgent     string
	VerificationKey string
	Motivo          string
	Instruction     string
	Extras          map[string]any
	Cooldown        time.Duration
}

type PostRemediationStatus struct {
	Found           bool
	OrderID         int64
	Agente          string
	ProyectoID      int64
	VerificationKey string
	RemediationKind string
	Estado          string
	DispatchState   string
	DeliveryState   string
	ReceiptSource   string
	BlockedReason   string
	RetryAfter      string
	UpdatedAt       time.Time
	Succeeded       bool
	Waiting         bool
	WorkConfirmed   bool
	AwaitingWork    bool
}

func (s *Service) EnqueueControl(req ControlRequest) (int64, string, error) {
	if s == nil || s.runtimes == nil {
		return 0, "", fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if err := s.runSessionHygieneForControl(req); err != nil {
		return 0, "", err
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

func (s *Service) RequestLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	if s == nil || s.runtimes == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.CreateLiveAgentHandoff(
		strings.TrimSpace(origen),
		strings.TrimSpace(destino),
		tareaID,
		strings.TrimSpace(motivo),
		strings.TrimSpace(resumenContinuidad),
		strings.TrimSpace(externalSessionID),
	)
}

func (s *Service) RequestStaleAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	if s == nil || s.runtimes == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	return s.runtimes.CreateStaleAgentHandoff(
		strings.TrimSpace(origen),
		strings.TrimSpace(destino),
		tareaID,
		strings.TrimSpace(motivo),
		strings.TrimSpace(resumenContinuidad),
		strings.TrimSpace(externalSessionID),
	)
}

func (s *Service) runSessionHygieneForControl(req ControlRequest) error {
	if s == nil || s.runtimes == nil {
		return fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if !strings.EqualFold(strings.TrimSpace(req.Accion), "start") {
		return nil
	}
	agente := strings.TrimSpace(req.Agente)
	proyecto := strings.TrimSpace(req.Proyecto)
	if agente == "" || proyecto == "" {
		return nil
	}
	if skip, err := s.shouldSkipSessionHygieneForStart(agente, proyecto); err != nil {
		return err
	} else if skip {
		return nil
	}
	actor := strings.TrimSpace(req.Por)
	if actor == "" {
		actor = "orquesta"
	}

	staleHandles := 0
	staleOrders := 0
	ranGlobalHygiene := false
	if tryBeginStartSessionHygiene(time.Now().UTC()) {
		ranGlobalHygiene = true
		globalHygieneOK := false
		defer func() {
			finishStartSessionHygiene(globalHygieneOK, time.Now().UTC())
		}()
		var err error
		staleHandles, err = s.runtimes.ReconcileStaleRuntimeHandles()
		if err != nil {
			return err
		}
		staleOrders, err = s.runtimes.ReconcileStaleRuntimeOrders()
		if err != nil {
			return err
		}
		globalHygieneOK = true
	}
	purgedOrders := 0
	if resultado, err := s.runtimes.PurgeTerminalRuntimeOrders(runtimesapp.RuntimeOrderPurgeRequest{
		Agente:   agente,
		Proyecto: proyecto,
		Estados:  []string{"completada", "fallida", "expirada", "cancelada"},
		Actor:    actor,
	}); err != nil {
		return err
	} else if resultado != nil {
		purgedOrders = resultado.Deleted
	}
	purgedHandles := 0
	if resultado, err := s.runtimes.PurgeInactiveRuntimeHandles(runtimesapp.RuntimeHandlePurgeRequest{
		Agente:   agente,
		Proyecto: proyecto,
		Estados:  []string{"cerrado", "fallido"},
		Actor:    actor,
	}); err != nil {
		return err
	} else if resultado != nil {
		purgedHandles = resultado.Deleted
	}
	if s.autonomyStore != nil && (ranGlobalHygiene || purgedHandles > 0 || purgedOrders > 0) && (staleHandles > 0 || staleOrders > 0 || purgedHandles > 0 || purgedOrders > 0) {
		s.autonomyStore.Audit(actor, "session_hygiene_start", "runtime", 0,
			fmt.Sprintf("agente=%s proyecto=%s stale_handles=%d stale_orders=%d purged_handles=%d purged_orders=%d",
				agente, proyecto, staleHandles, staleOrders, purgedHandles, purgedOrders))
	}
	return nil
}

func (s *Service) shouldSkipSessionHygieneForStart(agente, proyectoRef string) (bool, error) {
	if s == nil || s.runtimes == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	proyectoRef = strings.TrimSpace(proyectoRef)
	if agente == "" || proyectoRef == "" {
		return false, nil
	}
	proyecto, err := s.runtimes.GetProject(proyectoRef)
	if err != nil || proyecto == nil {
		return false, err
	}
	handle, err := s.runtimes.GetOperationalRuntimeHandleForProject(agente, &proyecto.ID)
	if err != nil || handle == nil {
		if err != nil {
			return false, err
		}
		return s.hasLiveRuntimeWorkForStartHygiene(agente, &proyecto.ID)
	}
	if db.RuntimeHandleSnapshotIsFresh(handle, time.Minute) {
		return true, nil
	}
	return s.hasLiveRuntimeWorkForStartHygiene(agente, &proyecto.ID)
}

func (s *Service) hasLiveRuntimeWorkForStartHygiene(agente string, proyectoID *int64) (bool, error) {
	if s == nil || s.runtimes == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID == nil || *proyectoID <= 0 {
		return false, nil
	}
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(order.Estado)) {
		case "pendiente", "notificada", "tomada", "ejecutando":
			return true, nil
		}
	}
	return false, nil
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

func (s *Service) HasRecentOperationalRuntimeOtherThanSession(sesion *db.Sesion) (bool, error) {
	if s == nil || s.runtimes == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil || sesion.ProyectoID == nil {
		return false, nil
	}
	handle, err := s.runtimes.GetOperationalRuntimeHandleForProject(strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil || handle == nil {
		return false, err
	}
	if handle.SesionID != nil && sesion.ID > 0 && *handle.SesionID == sesion.ID {
		return false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
		return false, nil
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return true, nil
	}
	meta := parseStringMap(strings.TrimSpace(handle.MetadataJSON))
	return strings.EqualFold(stringMapValue(meta, "driver"), "tmux_cli_session"), nil
}

func (s *Service) ShouldSupersedeSessionRecoveryWithRecentTMUX(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if handle.SesionID == nil || *handle.SesionID == sesion.ID {
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

func (s *Service) RecoverLocalFailedRuntimeSession(sesion *db.Sesion, proyecto *db.Proyecto, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (int, error) {
	if s == nil || s.runtimes == nil || s.workChecker == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil || proyecto == nil {
		return 0, nil
	}
	if s.ShouldSupersedeSessionRecoveryWithRecentTMUX(sesion, handle) {
		return 0, nil
	}
	var err error
	runtime, err = s.resolveCanonicalRuntimeForHandle(handle, strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	if !runtimeIsLocallyFailed(handle, runtime) {
		return 0, nil
	}
	if handle != nil {
		handle, err = s.runtimes.SyncSupervisedRuntimeHandle(handle, "autonomia_runtime_recovery")
		if err != nil {
			return 0, err
		}
	}
	runtime, err = s.resolveCanonicalRuntimeForHandle(handle, strings.TrimSpace(sesion.Agente), sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	if !runtimeIsLocallyFailed(handle, runtime) {
		return 0, nil
	}
	if pending, err := s.existsOpenAutonomyOrder(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "pause", "checkpoint", "start", "resume", "handoff"); err != nil {
		return 0, err
	} else if pending {
		return 0, nil
	}
	hasWork, err := s.workChecker.HasStartableAgentWork(strings.TrimSpace(sesion.Agente), proyecto.ID)
	if err != nil {
		return 0, err
	}
	if !hasWork {
		return 0, nil
	}
	perfil, modelo, razonamiento := resumePayloadExecutionProfile(sesion.ResumePayloadJSON)
	if strings.Contains(modelo, "gpt-5") || modelo == "" {
		modelo = ""
		razonamiento = ""
	}
	runtimeID := func() *int64 {
		if runtime != nil && runtime.ID > 0 {
			return &runtime.ID
		}
		if handle != nil && handle.RuntimeID != nil && *handle.RuntimeID > 0 {
			return handle.RuntimeID
		}
		return nil
	}()
	orderID, _, err := s.EnqueueControl(ControlRequest{
		Agente:       strings.TrimSpace(sesion.Agente),
		Proyecto:     strings.TrimSpace(proyecto.Slug),
		Accion:       "start",
		Modelo:       modelo,
		Razonamiento: razonamiento,
		Perfil:       perfil,
		Motivo:       "local_runtime_failed",
		Por:          "orquesta",
	})
	if err != nil {
		return 0, err
	}
	recordWorkerRecoveryRequested(
		"worker_recovery_local_failed",
		"local_runtime_failed",
		"start",
		strings.TrimSpace(sesion.Agente),
		sesion.ProyectoID,
		runtimeID,
		handle,
		map[string]int64{"runtime_order_id": orderID},
	)
	return 1, nil
}

func (s *Service) ReactivateProjectIfNeeded(agente string, proyecto *db.Proyecto, motivo string, handle *db.RuntimeHandle) (bool, error) {
	if s == nil || s.runtimes == nil || s.workChecker == nil || s.recoveryFlow == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" || proyecto == nil || proyecto.ID <= 0 {
		return false, nil
	}
	if s.recoveryFlow.UsesSharedLocalPoolForReactivation(agente, proyecto) {
		permite, poolSlug, err := s.recoveryFlow.PoolLocalActivationAllowed(agente, strings.TrimSpace(proyecto.Slug))
		if err != nil {
			return false, err
		}
		if !permite {
			if s.autonomyStore != nil {
				s.autonomyStore.Audit("orquesta", "reactivacion_pool_local_sin_capacidad", "agente", 0,
					fmt.Sprintf("agente=%s proyecto=%s pool=%s", agente, strings.TrimSpace(proyecto.Slug), poolSlug))
			}
			return false, nil
		}
	}
	if pending, err := s.existsOpenAutonomyOrder(agente, &proyecto.ID, "pause", "checkpoint", "resume", "start", "handoff"); err != nil {
		return false, err
	} else if pending {
		return false, nil
	}
	if reciente, err := s.existsRecentAutonomyOrder(agente, &proyecto.ID, "start", "", reactivationAttemptCooldown()); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	if reciente, err := s.existsRecentAutonomyOrder(agente, &proyecto.ID, "resume", "", reactivationAttemptCooldown()); err != nil {
		return false, err
	} else if reciente {
		return false, nil
	}
	tieneTrabajo, err := s.hasActionableReactivationWork(agente, proyecto.ID)
	if err != nil {
		return false, err
	}
	if !tieneTrabajo {
		return false, nil
	}
	if handle == nil {
		var err error
		handle, err = s.ResolveRecoveryHandle(agente, &proyecto.ID)
		if err != nil {
			return false, err
		}
	}
	if handle != nil {
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo":
			return false, nil
		case "pausado":
			accion := "resume"
			if db.RuntimeHandlePauseRequiresFreshStart(handle) {
				accion = "start"
			}
			orderID, _, err := s.EnqueueControl(ControlRequest{
				Agente:   agente,
				Proyecto: strings.TrimSpace(proyecto.Slug),
				Accion:   accion,
				Motivo:   strings.TrimSpace(motivo),
				Por:      "orquesta",
			})
			if err != nil {
				return false, err
			}
			recordWorkerRecoveryRequested(
				"worker_recovery_reactivate",
				strings.TrimSpace(motivo),
				accion,
				agente,
				&proyecto.ID,
				handle.RuntimeID,
				handle,
				map[string]int64{"runtime_order_id": orderID},
			)
			return true, nil
		case "fallido":
			orderID, _, err := s.EnqueueControl(ControlRequest{
				Agente:   agente,
				Proyecto: strings.TrimSpace(proyecto.Slug),
				Accion:   "start",
				Motivo:   strings.TrimSpace(motivo),
				Por:      "orquesta",
			})
			if err != nil {
				return false, err
			}
			recordWorkerRecoveryRequested(
				"worker_recovery_reactivate",
				strings.TrimSpace(motivo),
				"start",
				agente,
				&proyecto.ID,
				handle.RuntimeID,
				handle,
				map[string]int64{"runtime_order_id": orderID},
			)
			return true, nil
		}
	}
	orderID, _, err := s.EnqueueControl(ControlRequest{
		Agente:   agente,
		Proyecto: strings.TrimSpace(proyecto.Slug),
		Accion:   "start",
		Motivo:   strings.TrimSpace(motivo),
		Por:      "orquesta",
	})
	if err != nil {
		return false, err
	}
	recordWorkerRecoveryRequested(
		"worker_recovery_reactivate",
		strings.TrimSpace(motivo),
		"start",
		agente,
		&proyecto.ID,
		nil,
		handle,
		map[string]int64{"runtime_order_id": orderID},
	)
	return true, nil
}

func (s *Service) hasActionableReactivationWork(agente string, proyectoID int64) (bool, error) {
	if s == nil || s.workChecker == nil || proyectoID <= 0 {
		return false, nil
	}
	tieneTrabajo, err := s.workChecker.HasStartableAgentWork(agente, proyectoID)
	if err != nil {
		return false, err
	}
	if tieneTrabajo {
		return true, nil
	}
	return s.hasRecoverableAssignedProjectWork(agente, proyectoID)
}

func (s *Service) hasRecoverableAssignedProjectWork(agente string, proyectoID int64) (bool, error) {
	if s == nil || s.recoveryFlow == nil || proyectoID <= 0 {
		return false, nil
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	if taskProjectID, err := s.recoveryFlow.ProjectIDFromAssignedWork(agente); err != nil {
		return false, err
	} else if taskProjectID == proyectoID {
		return true, nil
	}
	return false, nil
}

func (s *Service) ResolveReactivationProject(agente string) (*db.Proyecto, error) {
	if s == nil || s.runtimes == nil || s.recoveryFlow == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	if proyectoID, err := s.recoveryFlow.ActiveProjectID(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "asignacion_activa")
	}
	if proyectoID, err := s.recoveryFlow.ProjectIDFromAssignedWork(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "tarea")
	}
	if proyectoID, err := s.projectIDFromPendingStartOrder(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "runtime_order_start")
	}
	if proyectoID, err := s.projectIDFromPendingMailbox(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "mailbox")
	}
	if proyectoID, err := s.recoveryFlow.OpenSessionProjectID(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "sesion_abierta")
	}
	if proyectoID, err := s.recoveryFlow.LatestSessionProjectID(agente); err != nil {
		return nil, err
	} else if proyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, proyectoID, "ultima_sesion")
	}
	if handle, err := s.ResolveRecoveryHandle(agente, nil); err != nil {
		return nil, err
	} else if handle != nil && handle.ProyectoID != nil && *handle.ProyectoID > 0 {
		return s.resolveReactivationProjectByID(agente, *handle.ProyectoID, "runtime_handle")
	}
	return nil, nil
}

func (s *Service) RecoverDegradedRuntimeSession(sesion *db.Sesion) (int, error) {
	if s == nil || s.runtimes == nil || s.recoveryFlow == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil || sesion.ProyectoID == nil {
		return 0, nil
	}
	disponible, _, err := s.recoveryFlow.SharedAccountAvailable(strings.TrimSpace(sesion.Agente))
	if err != nil {
		return 0, err
	}
	if !disponible {
		return 0, nil
	}
	if reciente, err := s.HasRecentOperationalRuntimeOtherThanSession(sesion); err != nil {
		return 0, err
	} else if reciente {
		return 0, nil
	}
	proyecto, err := s.runtimes.GetProject(strconv.FormatInt(*sesion.ProyectoID, 10))
	if err != nil {
		return 0, err
	}
	handle, runtime, err := s.ResolveSessionRecoveryTarget(sesion)
	if err != nil {
		return 0, err
	}
	if handle == nil {
		reactivado, err := s.ReactivateProjectIfNeeded(strings.TrimSpace(sesion.Agente), proyecto, "local_runtime_missing", nil)
		if err != nil {
			return 0, err
		}
		if reactivado {
			return 1, nil
		}
		return 0, nil
	}
	if !isRemoteAutonomyTransport(handle.Transporte) {
		return s.RecoverLocalFailedRuntimeSession(sesion, proyecto, handle, runtime)
	}
	if !remoteRuntimeDegraded(handle, runtime) {
		return 0, nil
	}
	return s.RecoverRemoteDegradedRuntimeSession(sesion, proyecto, handle, runtime)
}

func (s *Service) RecoverRemoteDegradedRuntimeSession(sesion *db.Sesion, proyecto *db.Proyecto, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (int, error) {
	if s == nil || s.runtimes == nil || s.remoteSupport == nil {
		return 0, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	if sesion == nil || sesion.ProyectoID == nil || proyecto == nil || handle == nil {
		return 0, nil
	}
	conector, err := s.remoteSupport.ResolveSessionConnector(sesion, runtime, handle)
	if err != nil {
		return 0, err
	}
	if conector != nil {
		disponible, err := s.remoteSupport.IsConnectorAvailable(conector.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			motivo := "conector:" + strings.TrimSpace(conector.Slug) + ":circuito_abierto"
			if err := s.remoteSupport.MarkProjectExternallyBlocked(*sesion.ProyectoID, motivo); err != nil {
				return 0, err
			}
			if satisfecha, err := s.PauseAutonomyAlreadySatisfied(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, sesion); err != nil {
				return 0, err
			} else if satisfecha {
				return 1, nil
			}
			pauseOrderID, _, err := s.EnqueueControl(ControlRequest{
				Agente:   strings.TrimSpace(sesion.Agente),
				Proyecto: strings.TrimSpace(proyecto.Slug),
				Accion:   "pause",
				Motivo:   motivo,
				Por:      "orquesta",
			})
			if err != nil {
				return 0, err
			}
			recordWorkerRecoveryPausedExternal(
				"worker_recovery_remote_degraded",
				motivo,
				strings.TrimSpace(sesion.Agente),
				sesion.ProyectoID,
				func() *int64 {
					if runtime == nil || runtime.ID <= 0 {
						return nil
					}
					return ptrInt64Orq(runtime.ID)
				}(),
				handle,
				map[string]int64{"pause_order_id": pauseOrderID},
			)
			return 1, nil
		}
	}
	if pendiente, err := s.existsOpenAutonomyOrder(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "pause", "checkpoint", "start", "resume", "handoff"); err != nil {
		return 0, err
	} else if pendiente {
		return 0, nil
	}
	if reciente, err := s.existsRecentAutonomyOrder(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "start", "", reactivationAttemptCooldown()); err != nil {
		return 0, err
	} else if reciente {
		return 0, nil
	}
	if reciente, err := s.existsRecentAutonomyOrder(strings.TrimSpace(sesion.Agente), sesion.ProyectoID, "resume", "", reactivationAttemptCooldown()); err != nil {
		return 0, err
	} else if reciente {
		return 0, nil
	}
	tieneTrabajo, err := s.hasActionableReactivationWork(strings.TrimSpace(sesion.Agente), *sesion.ProyectoID)
	if err != nil {
		return 0, err
	}
	if !tieneTrabajo {
		return 0, nil
	}
	checkpointPayload, err := json.Marshal(map[string]any{
		"checkpoint_kind": "remote_recovery",
		"resumen":         "Checkpoint automático antes de recuperación de runtime remoto degradado",
		"motivo":          "remote_runtime_degraded",
	})
	if err != nil {
		return 0, err
	}
	runtimeID := func() *int64 {
		if runtime != nil && runtime.ID > 0 {
			return &runtime.ID
		}
		return handle.RuntimeID
	}()
	checkpointOrderID, err := s.runtimes.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      strings.TrimSpace(sesion.Agente),
		ProyectoID:  sesion.ProyectoID,
		RuntimeID:   runtimeID,
		HandleID:    &handle.ID,
		Tipo:        "checkpoint",
		PayloadJSON: string(checkpointPayload),
	})
	if err != nil {
		return 0, err
	}
	if remoteRuntimeResumable(sesion, handle) {
		payload, err := json.Marshal(map[string]any{
			"accion":   "resume",
			"motivo":   "remote_runtime_degraded",
			"por":      "orquesta",
			"proyecto": strings.TrimSpace(proyecto.Slug),
		})
		if err != nil {
			return 0, err
		}
		resumeOrderID, err := s.runtimes.EnqueueRuntimeOrder(&db.RuntimeOrder{
			Agente:      strings.TrimSpace(sesion.Agente),
			ProyectoID:  sesion.ProyectoID,
			RuntimeID:   runtimeID,
			HandleID:    &handle.ID,
			Tipo:        "resume",
			PayloadJSON: string(payload),
		})
		if err != nil {
			return 0, err
		}
		recordWorkerRecoveryRequested(
			"worker_recovery_remote_degraded",
			"remote_runtime_degraded",
			"resume",
			strings.TrimSpace(sesion.Agente),
			sesion.ProyectoID,
			runtimeID,
			handle,
			map[string]int64{
				"checkpoint_order_id": checkpointOrderID,
				"resume_order_id":     resumeOrderID,
			},
		)
		return 2, nil
	}
	startOrderID, _, err := s.EnqueueControl(ControlRequest{
		Agente:   strings.TrimSpace(sesion.Agente),
		Proyecto: strings.TrimSpace(proyecto.Slug),
		Accion:   "start",
		Motivo:   "remote_runtime_degraded",
		Por:      "orquesta",
	})
	if err != nil {
		return 0, err
	}
	recordWorkerRecoveryRequested(
		"worker_recovery_remote_degraded",
		"remote_runtime_degraded",
		"start",
		strings.TrimSpace(sesion.Agente),
		sesion.ProyectoID,
		runtimeID,
		handle,
		map[string]int64{
			"checkpoint_order_id": checkpointOrderID,
			"start_order_id":      startOrderID,
		},
	)
	return 2, nil
}

func reactivationAttemptCooldown() time.Duration {
	return 2 * time.Minute
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
	agente, err := db.CanonicalizeAgentName(req.Agente)
	if err != nil {
		return false, err
	}
	agente = strings.TrimSpace(agente)
	accion := strings.TrimSpace(req.Accion)
	if agente == "" || accion == "" || proyecto.ID <= 0 {
		return false, nil
	}
	if abierta, err := s.existsOpenAutonomyOrderMatchingAutonomyAction(agente, &proyecto.ID, "send_instruction", accion, req.Extras); err != nil {
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
			if pendiente, err := s.existsPendingAutonomyMailbox(agente, &proyecto.ID, accion, req.Extras); err != nil {
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

func (s *Service) EnqueuePostRemediationFollowup(req PostRemediationFollowupRequest) (bool, error) {
	if s == nil || s.runtimes == nil || s.autonomyStore == nil {
		return false, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	proyecto := req.Proyecto
	agente := strings.TrimSpace(req.Agente)
	if proyecto == nil || proyecto.ID <= 0 || agente == "" || req.TareaID <= 0 {
		return false, nil
	}
	verificationKey := strings.TrimSpace(req.VerificationKey)
	cooldown := req.Cooldown
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	if verificationKey != "" {
		if pendiente, err := s.existsPendingPostRemediationFollowup(agente, &proyecto.ID, verificationKey); err != nil {
			return false, err
		} else if pendiente {
			return false, nil
		}
		if reciente, err := s.existsRecentPostRemediationFollowup(agente, &proyecto.ID, verificationKey, cooldown); err != nil {
			return false, err
		} else if reciente {
			return false, nil
		}
	}
	extras := map[string]any{
		"post_remediation": true,
		"tarea_id":         req.TareaID,
	}
	if kind := strings.TrimSpace(req.RemediationKind); kind != "" {
		extras["remediation_kind"] = kind
	}
	if origin := strings.TrimSpace(req.OriginAgent); origin != "" {
		extras["origin_agent"] = origin
		extras["reasignada_desde"] = origin
	}
	if verificationKey != "" {
		extras["verification_key"] = verificationKey
	}
	for key, value := range req.Extras {
		extras[key] = value
	}
	motivo := strings.TrimSpace(req.Motivo)
	if motivo == "" {
		motivo = fmt.Sprintf("Tarea #%d reactivada automáticamente", req.TareaID)
	}
	return s.EnqueueAutonomyNudge(NudgeRequest{
		Agente:      agente,
		Proyecto:    proyecto,
		Accion:      "continuar_trabajo",
		Motivo:      motivo,
		Instruction: strings.TrimSpace(req.Instruction),
		Extras:      extras,
		Cooldown:    cooldown,
	})
}

func (s *Service) GetPostRemediationStatus(agente string, proyectoID *int64, verificationKey string) (*PostRemediationStatus, error) {
	if s == nil || s.runtimes == nil {
		return nil, fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	agente = strings.TrimSpace(agente)
	verificationKey = strings.TrimSpace(verificationKey)
	if agente == "" || proyectoID == nil || *proyectoID <= 0 || verificationKey == "" {
		return nil, nil
	}
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Tipos:      []string{"nudge"},
	})
	if err != nil {
		return nil, err
	}
	var best *db.RuntimeOrder
	bestAt := time.Time{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if !runtimeOrderHasAutonomyAction(order, "continuar_trabajo") || !runtimePayloadHasVerificationKey(order.PayloadJSON, verificationKey) {
			continue
		}
		moment := runtimeOrderMoment(order)
		if best == nil || moment.After(bestAt) {
			best = order
			bestAt = moment
		}
	}
	if best == nil {
		return nil, nil
	}
	result := parseStringMap(strings.TrimSpace(best.ResultadoJSON))
	status := &PostRemediationStatus{
		Found:           true,
		OrderID:         best.ID,
		Agente:          strings.TrimSpace(agente),
		ProyectoID:      *proyectoID,
		VerificationKey: verificationKey,
		RemediationKind: strings.TrimSpace(stringMapValue(result, "remediation_kind")),
		Estado:          strings.TrimSpace(best.Estado),
		DispatchState:   strings.TrimSpace(stringMapValue(result, "dispatch_state")),
		DeliveryState:   strings.TrimSpace(stringMapValue(result, "delivery_state")),
		ReceiptSource:   strings.TrimSpace(stringMapValue(result, "receipt_source")),
		BlockedReason:   strings.TrimSpace(stringMapValue(result, "blocked_reason")),
		RetryAfter:      strings.TrimSpace(stringMapValue(result, "retry_after")),
		UpdatedAt:       runtimeOrderMoment(best),
	}
	if status.RemediationKind == "" {
		status.RemediationKind = strings.TrimSpace(stringMapValue(parseStringMap(strings.TrimSpace(best.PayloadJSON)), "remediation_kind"))
	}
	switch {
	case strings.EqualFold(status.Estado, "completada") && status.ReceiptSource != "":
		status.WorkConfirmed = receiptSourceConfirmsWork(status.ReceiptSource)
		if status.WorkConfirmed {
			status.Succeeded = true
		} else {
			started, err := s.postRemediationWorkStartedFromQueue(agente, proyectoID, verificationKey)
			if err != nil {
				return nil, err
			}
			if started {
				status.WorkConfirmed = true
				status.Succeeded = true
			} else {
				status.Waiting = true
				status.AwaitingWork = true
			}
		}
	case strings.EqualFold(status.Estado, "fallida"), strings.EqualFold(status.DeliveryState, "blocked"):
		status.Waiting = false
	case strings.EqualFold(status.DispatchState, "notified"), strings.EqualFold(status.DispatchState, "pending"), strings.EqualFold(status.Estado, "pendiente"), strings.EqualFold(status.Estado, "ejecutando"):
		status.Waiting = true
	}
	return status, nil
}

func (s *Service) postRemediationWorkStartedFromQueue(agente string, proyectoID *int64, verificationKey string) (bool, error) {
	if s == nil || s.runtimes == nil || proyectoID == nil || *proyectoID <= 0 {
		return false, nil
	}
	agente = strings.TrimSpace(agente)
	verificationKey = strings.TrimSpace(verificationKey)
	if agente == "" || verificationKey == "" {
		return false, nil
	}
	handles, err := s.runtimes.ListRuntimeHandles(&agente)
	if err != nil {
		return false, err
	}
	match := controlruntime.WorkQueueMatch{
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		VerificationKey: verificationKey,
		Within:          2 * time.Hour,
	}
	for _, handle := range handles {
		if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID != *proyectoID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
			continue
		}
		if !db.RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) {
			continue
		}
		started, err := controlruntime.HasStartedWorkQueueEntryFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON), match)
		if err != nil {
			return false, err
		}
		if started {
			return true, nil
		}
	}
	return false, nil
}

func receiptSourceConfirmsWork(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "last_progress", "worker_activity", "tmux_transcript_activity", "transcript_patch", "tmux_pane_patch", "git_worktree":
		return true
	default:
		return false
	}
}

func (s *Service) sessionRuntimeHandle(sesion *db.Sesion) (*db.RuntimeHandle, error) {
	if s == nil || s.autonomyStore == nil || sesion == nil || sesion.ID <= 0 {
		return nil, nil
	}
	return s.autonomyStore.GetSessionRuntimeHandle(sesion.ID)
}

func (s *Service) existsPendingAutonomyOrder(agente string, proyectoID *int64, tipo string, accion string) (bool, error) {
	if strings.TrimSpace(accion) != "" {
		if pendiente, err := s.existsPendingAutonomyDurableAction(agente, proyectoID, strings.TrimSpace(accion), nil); err != nil {
			return false, err
		} else if pendiente {
			return true, nil
		}
	}
	estado := "pendiente"
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
		Tipos:      []string{strings.TrimSpace(tipo)},
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil {
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
	if strings.TrimSpace(accion) != "" {
		if reciente, err := s.existsRecentAutonomyDurableAction(agente, proyectoID, strings.TrimSpace(accion), nil, within); err != nil {
			return false, err
		} else if reciente {
			return true, nil
		}
	}
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Tipos:      []string{strings.TrimSpace(tipo)},
	})
	if err != nil {
		return false, err
	}
	cutoff := time.Now().UTC().Add(-within)
	for _, order := range orders {
		if order == nil {
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

func (s *Service) resolveReactivationProjectByID(agente string, proyectoID int64, origen string) (*db.Proyecto, error) {
	if proyectoID <= 0 {
		return nil, nil
	}
	proyecto, err := s.runtimes.GetProject(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, fmt.Errorf("agente %s con proyecto de reactivacion inconsistente (%s=%d)", strings.TrimSpace(agente), strings.TrimSpace(origen), proyectoID)
	}
	return proyecto, nil
}

func (s *Service) projectIDFromPendingMailbox(agente string) (int64, error) {
	estado := "pendiente"
	mailbox, err := s.runtimes.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente: &agente,
		Estado:   &estado,
	})
	if err != nil {
		return 0, err
	}
	bestProjectID := int64(0)
	bestID := int64(0)
	for _, msg := range mailbox {
		if msg == nil || msg.ProyectoID == nil || *msg.ProyectoID <= 0 {
			continue
		}
		if bestProjectID == 0 || msg.ID > bestID {
			bestProjectID = *msg.ProyectoID
			bestID = msg.ID
		}
	}
	return bestProjectID, nil
}

func (s *Service) projectIDFromPendingStartOrder(agente string) (int64, error) {
	if s == nil || s.runtimes == nil {
		return 0, nil
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	bestProjectID := int64(0)
	bestID := int64(0)
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente: &agente,
			Estado: &estado,
			Tipos:  []string{"start"},
		})
		if err != nil {
			return 0, err
		}
		for _, order := range orders {
			if order == nil || order.ProyectoID == nil || *order.ProyectoID <= 0 {
				continue
			}
			if bestProjectID == 0 || order.ID > bestID {
				bestProjectID = *order.ProyectoID
				bestID = order.ID
			}
		}
	}
	return bestProjectID, nil
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
			Tipos:      append([]string(nil), tipos...),
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order != nil {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Service) existsOpenAutonomyOrderMatchingAutonomyAction(agente string, proyectoID *int64, tipo string, accion string, extras map[string]any) (bool, error) {
	accion = strings.TrimSpace(accion)
	if strings.TrimSpace(tipo) == "" {
		return false, nil
	}
	if accion == "" {
		return s.existsOpenAutonomyOrder(agente, proyectoID, tipo)
	}
	estados := []string{"pendiente", "tomada", "ejecutando"}
	for _, estado := range estados {
		estado := estado
		orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
			Tipos:      []string{strings.TrimSpace(tipo)},
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if runtimeOrderMatchesAutonomyActionAndContext(order, accion, extras) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Service) existsPendingAutonomyMailbox(agente string, proyectoID *int64, accion string, extras map[string]any) (bool, error) {
	if pendiente, err := s.existsPendingAutonomyDurableAction(agente, proyectoID, accion, extras); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
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
		if autonomyMailboxMatchesActionAndContext(payload, accion, extras) {
			return true, nil
		}
	}
	return false, nil
}

func autonomyMailboxMatchesActionAndContext(payload map[string]any, accion string, extras map[string]any) bool {
	if strings.TrimSpace(stringMapValue(payload, "accion")) != strings.TrimSpace(accion) {
		return false
	}
	if verificationKey := strings.TrimSpace(stringMapValue(extras, "verification_key")); verificationKey != "" {
		if !strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "verification_key")), verificationKey) {
			return false
		}
	}
	if tareaID := int64MapValue(extras, "tarea_id"); tareaID > 0 {
		if int64MapValue(payload, "tarea_id") != tareaID {
			return false
		}
	}
	return true
}

func (s *Service) existsPendingPostRemediationFollowup(agente string, proyectoID *int64, verificationKey string) (bool, error) {
	if strings.TrimSpace(verificationKey) == "" {
		return false, nil
	}
	if pendiente, err := s.existsPendingAutonomyOrderWithVerificationKey(agente, proyectoID, "continuar_trabajo", verificationKey); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
	return s.existsPendingAutonomyMailboxWithVerificationKey(agente, proyectoID, "continuar_trabajo", verificationKey)
}

func (s *Service) existsRecentPostRemediationFollowup(agente string, proyectoID *int64, verificationKey string, within time.Duration) (bool, error) {
	if strings.TrimSpace(verificationKey) == "" || within <= 0 {
		return false, nil
	}
	if reciente, err := s.existsRecentAutonomyDurableAction(agente, proyectoID, "continuar_trabajo", map[string]any{"verification_key": verificationKey}, within); err != nil {
		return false, err
	} else if reciente {
		return true, nil
	}
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Tipos:      []string{"nudge"},
	})
	if err != nil {
		return false, err
	}
	cutoff := time.Now().UTC().Add(-within)
	for _, order := range orders {
		if order == nil {
			continue
		}
		if runtimeOrderMoment(order).Before(cutoff) {
			continue
		}
		if !runtimeOrderHasAutonomyAction(order, "continuar_trabajo") || !runtimePayloadHasVerificationKey(order.PayloadJSON, verificationKey) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) existsPendingAutonomyOrderWithVerificationKey(agente string, proyectoID *int64, accion, verificationKey string) (bool, error) {
	if pendiente, err := s.existsPendingAutonomyDurableAction(agente, proyectoID, accion, map[string]any{"verification_key": verificationKey}); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
	estado := "pendiente"
	orders, err := s.runtimes.ListRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
		Tipos:      []string{"nudge"},
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "nudge" {
			continue
		}
		if !runtimeOrderHasAutonomyAction(order, accion) || !runtimePayloadHasVerificationKey(order.PayloadJSON, verificationKey) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) existsPendingAutonomyMailboxWithVerificationKey(agente string, proyectoID *int64, accion, verificationKey string) (bool, error) {
	if pendiente, err := s.existsPendingAutonomyDurableAction(agente, proyectoID, accion, map[string]any{"verification_key": verificationKey}); err != nil {
		return false, err
	} else if pendiente {
		return true, nil
	}
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
		if strings.TrimSpace(stringMapValue(payload, "accion")) != strings.TrimSpace(accion) {
			continue
		}
		if strings.TrimSpace(stringMapValue(payload, "verification_key")) != strings.TrimSpace(verificationKey) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *Service) existsPendingAutonomyDurableAction(agente string, proyectoID *int64, accion string, extras map[string]any) (bool, error) {
	match, metaRaw, err := s.autonomyDurableActionMatch(agente, proyectoID, accion, extras, 0)
	if err != nil || strings.TrimSpace(metaRaw) == "" {
		return false, err
	}
	return controlruntime.HasPendingWorkQueueEntryFromMetadataJSON(metaRaw, match)
}

func (s *Service) existsRecentAutonomyDurableAction(agente string, proyectoID *int64, accion string, extras map[string]any, within time.Duration) (bool, error) {
	match, metaRaw, err := s.autonomyDurableActionMatch(agente, proyectoID, accion, extras, within)
	if err != nil || strings.TrimSpace(metaRaw) == "" {
		return false, err
	}
	return controlruntime.HasRecentWorkQueueEntryFromMetadataJSON(metaRaw, match)
}

func (s *Service) autonomyDurableActionMatch(agente string, proyectoID *int64, accion string, extras map[string]any, within time.Duration) (controlruntime.WorkQueueMatch, string, error) {
	match := controlruntime.WorkQueueMatch{
		Kind:   "autonomia",
		Action: strings.TrimSpace(accion),
		Within: within,
	}
	if match.Action == "" {
		return match, "", nil
	}
	if extras != nil {
		match.VerificationKey = strings.TrimSpace(stringMapValue(extras, "verification_key"))
		match.TaskID = int64MapValue(extras, "tarea_id")
	}
	metaRaw, err := s.deliveryHandleMetadata(agente, proyectoID)
	if err != nil {
		return match, "", err
	}
	return match, metaRaw, nil
}

func (s *Service) deliveryHandleMetadata(agente string, proyectoID *int64) (string, error) {
	if s == nil || s.runtimes == nil {
		return "", fmt.Errorf("servicio de orquestacion de agentes no inicializado")
	}
	handle, err := s.ResolveDeliveryHandle(strings.TrimSpace(agente), proyectoID)
	if err != nil || handle == nil {
		return "", err
	}
	return strings.TrimSpace(handle.MetadataJSON), nil
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

func runtimeOrderMatchesAutonomyActionAndContext(order *db.RuntimeOrder, accion string, extras map[string]any) bool {
	if !runtimeOrderHasAutonomyAction(order, accion) {
		return false
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(order.PayloadJSON)), &payload); err != nil {
		return false
	}
	return autonomyMailboxMatchesActionAndContext(payload, accion, extras)
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

func runtimePayloadHasVerificationKey(payloadJSON, verificationKey string) bool {
	verificationKey = strings.TrimSpace(verificationKey)
	if verificationKey == "" {
		return false
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(payloadJSON)), &payload); err == nil {
		return strings.EqualFold(strings.TrimSpace(stringMapValue(payload, "verification_key")), verificationKey)
	}
	return strings.Contains(strings.ToLower(payloadJSON), fmt.Sprintf(`"verification_key":"%s"`, strings.ToLower(verificationKey)))
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

func int64MapValue(payload map[string]any, key string) int64 {
	if payload == nil {
		return 0
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

func parseStringMap(raw string) map[string]any {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return nil
	}
	return payload
}

func resumePayloadExecutionProfile(raw string) (string, string, string) {
	payload := parseStringMap(raw)
	if payload == nil {
		return "", "", ""
	}
	perfil, ok := payload["perfil_ejecucion"].(map[string]any)
	if !ok {
		return "", "", ""
	}
	return stringMapValue(perfil, "perfil_tarea"),
		stringMapValue(perfil, "modelo"),
		stringMapValue(perfil, "razonamiento")
}

func runtimeIsLocallyFailed(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle == nil || strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") || strings.EqualFold(strings.TrimSpace(handle.Transporte), "mcp_http") || strings.EqualFold(strings.TrimSpace(handle.Transporte), "otro") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "fallido" || process == "fallido" || process == "crashed" || process == "exited"
}

func remoteRuntimeDegraded(handle *db.RuntimeHandle, runtime *db.RuntimeInstance) bool {
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido") {
		return true
	}
	if runtime == nil {
		return false
	}
	logical := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	process := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	return logical == "degradado" || process == "remote_status_error"
}

func remoteRuntimeResumable(sesion *db.Sesion, handle *db.RuntimeHandle) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if !isRemoteAutonomyTransport(handle.Transporte) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(sesion.Herramienta), "ollama_pool_local") ||
		strings.EqualFold(strings.TrimSpace(sesion.ConectorSlug), "ollama_pool_local") {
		return false
	}
	meta := parseStringMap(strings.TrimSpace(handle.MetadataJSON))
	if strings.EqualFold(stringMapValue(meta, "driver"), "ollama_pool_local") {
		return false
	}
	if strings.TrimSpace(sesion.ExternalSessionID) != "" {
		return true
	}
	if stringMapValue(meta, "external_session_id") != "" {
		return true
	}
	if strings.TrimSpace(handle.HandleKind) == "process" {
		return false
	}
	ref := strings.TrimSpace(handle.HandleRef)
	if ref == "" {
		return false
	}
	return ref != strconv.FormatInt(sesion.ID, 10)
}

func isRemoteAutonomyTransport(transport string) bool {
	switch strings.TrimSpace(transport) {
	case "api", "mcp_http", "otro":
		return true
	default:
		return false
	}
}
