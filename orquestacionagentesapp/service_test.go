package orquestacionagentesapp

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimesapp"
)

type stubAgentPreparer struct {
	input agentesapp.PrepareInput
	out   *agentesapp.PrepareOutput
	err   error
}

func (s *stubAgentPreparer) BuildPrepare(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error) {
	s.input = input
	return s.out, s.err
}

type stubRuntimeController struct {
	controlReq           runtimesapp.AgentControlRequest
	controlID            int64
	controlAction        string
	controlErr           error
	syncSource           string
	syncCount            int
	syncErr              error
	syncedHandle         *db.RuntimeHandle
	enqueuedRuntimeOrder *db.RuntimeOrder
	reconcileHandles     int
	reconcileHandlesErr  error
	reconcileOrders      int
	reconcileOrdersErr   error
	purgeHandlesReq      runtimesapp.RuntimeHandlePurgeRequest
	purgeHandlesResult   *db.PurgaRuntimeHandlesResultado
	purgeHandlesErr      error
	purgeOrdersReq       runtimesapp.RuntimeOrderPurgeRequest
	purgeOrdersResult    *db.PurgaRuntimeOrdersResultado
	purgeOrdersErr       error
	project              *db.Proyecto
	projectErr           error
	runtime              *db.RuntimeInstance
	runtimeErr           error
	sessionHandle        *db.RuntimeHandle
	sessionHandleErr     error
	listHandles          []*db.RuntimeHandle
	listHandlesErr       error
	orders               []*db.RuntimeOrder
	ordersErr            error
	mailbox              []*db.RuntimeMailboxMessage
	mailboxErr           error
	handleByID           map[int64]*db.RuntimeHandle
	activeHandle         *db.RuntimeHandle
	activeHandleErr      error
	operationalHandle    *db.RuntimeHandle
	operationalHandleErr error
	controlHandle        *db.RuntimeHandle
	controlHandleErr     error
	deliveryHandle       *db.RuntimeHandle
	deliveryHandleErr    error
	transcriptHandle     *db.RuntimeHandle
	transcriptHandleErr  error
}

func (s *stubRuntimeController) EnqueueAgentControl(req runtimesapp.AgentControlRequest) (int64, string, error) {
	s.controlReq = req
	return s.controlID, s.controlAction, s.controlErr
}

func (s *stubRuntimeController) GetProject(string) (*db.Proyecto, error) {
	return s.project, s.projectErr
}

func (s *stubRuntimeController) ReconcileStaleRuntimeHandles() (int, error) {
	return s.reconcileHandles, s.reconcileHandlesErr
}

func (s *stubRuntimeController) ReconcileStaleRuntimeOrders() (int, error) {
	return s.reconcileOrders, s.reconcileOrdersErr
}

func (s *stubRuntimeController) PurgeInactiveRuntimeHandles(req runtimesapp.RuntimeHandlePurgeRequest) (*db.PurgaRuntimeHandlesResultado, error) {
	s.purgeHandlesReq = req
	if s.purgeHandlesErr != nil {
		return nil, s.purgeHandlesErr
	}
	if s.purgeHandlesResult != nil {
		return s.purgeHandlesResult, nil
	}
	return &db.PurgaRuntimeHandlesResultado{}, nil
}

func (s *stubRuntimeController) PurgeTerminalRuntimeOrders(req runtimesapp.RuntimeOrderPurgeRequest) (*db.PurgaRuntimeOrdersResultado, error) {
	s.purgeOrdersReq = req
	if s.purgeOrdersErr != nil {
		return nil, s.purgeOrdersErr
	}
	if s.purgeOrdersResult != nil {
		return s.purgeOrdersResult, nil
	}
	return &db.PurgaRuntimeOrdersResultado{}, nil
}

func (s *stubRuntimeController) GetRuntime(int64) (*db.RuntimeInstance, error) {
	return s.runtime, s.runtimeErr
}

func (s *stubRuntimeController) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	if s.handleByID == nil {
		return nil, nil
	}
	return s.handleByID[id], nil
}

func (s *stubRuntimeController) GetRuntimeBySessionID(int64) (*db.RuntimeInstance, error) {
	return s.runtime, s.runtimeErr
}

func (s *stubRuntimeController) GetRuntimeHandleBySessionID(int64) (*db.RuntimeHandle, error) {
	return s.sessionHandle, s.sessionHandleErr
}

func (s *stubRuntimeController) SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error) {
	s.syncCount++
	s.syncSource = source
	if s.syncErr != nil {
		return nil, s.syncErr
	}
	if s.syncedHandle != nil {
		return s.syncedHandle, nil
	}
	return handle, nil
}

func (s *stubRuntimeController) GetPrimaryRuntimeForProject(string, *int64) (*db.RuntimeInstance, error) {
	return s.runtime, s.runtimeErr
}

func (s *stubRuntimeController) ListRuntimeHandles(*string) ([]*db.RuntimeHandle, error) {
	return s.listHandles, s.listHandlesErr
}

func (s *stubRuntimeController) GetActiveRuntimeHandleForProject(string, *int64) (*db.RuntimeHandle, error) {
	return s.activeHandle, s.activeHandleErr
}

func (s *stubRuntimeController) GetOperationalRuntimeHandleForProject(string, *int64) (*db.RuntimeHandle, error) {
	return s.operationalHandle, s.operationalHandleErr
}

func (s *stubRuntimeController) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	if order != nil {
		clone := *order
		s.enqueuedRuntimeOrder = &clone
	} else {
		s.enqueuedRuntimeOrder = nil
	}
	return s.controlID, s.controlErr
}

func (s *stubRuntimeController) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return s.orders, s.ordersErr
}

func (s *stubRuntimeController) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return s.mailbox, s.mailboxErr
}

func (s *stubRuntimeController) ResolveControlHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.controlHandle, s.controlHandleErr
}

func (s *stubRuntimeController) ResolveDeliveryHandle(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.deliveryHandle, s.deliveryHandleErr
}

func (s *stubRuntimeController) ResolveTranscriptDeliveryHandle(item *db.RuntimeTranscriptEntry) (*db.RuntimeHandle, error) {
	return s.transcriptHandle, s.transcriptHandleErr
}

type stubAutonomyStore struct {
	quotaState       string
	quotaErr         error
	sessionHandle    *db.RuntimeHandle
	sessionHandleErr error
	auditAction      string
	auditEntity      string
	auditEntityID    int64
	auditDetail      string
	parkedAgent      string
	parkedProjectID  *int64
	pauseAgent       string
	pauseProjectID   int64
	pauseReason      string
	pauseErr         error
}

type stubStartableWorkChecker struct {
	ok  bool
	err error
}

func (s *stubStartableWorkChecker) HasStartableAgentWork(string, int64) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.ok, nil
}

type stubRemoteRecoverySupport struct {
	conector       *db.Conector
	resolveErr     error
	available      bool
	availableErr   error
	blockProjectID int64
	blockReason    string
	blockErr       error
}

func (s *stubRemoteRecoverySupport) ResolveSessionConnector(*db.Sesion, *db.RuntimeInstance, *db.RuntimeHandle) (*db.Conector, error) {
	if s.resolveErr != nil {
		return nil, s.resolveErr
	}
	return s.conector, nil
}

func (s *stubRemoteRecoverySupport) IsConnectorAvailable(int64) (bool, error) {
	if s.availableErr != nil {
		return false, s.availableErr
	}
	return s.available, nil
}

func (s *stubRemoteRecoverySupport) MarkProjectExternallyBlocked(proyectoID int64, motivo string) error {
	if s.blockErr != nil {
		return s.blockErr
	}
	s.blockProjectID = proyectoID
	s.blockReason = motivo
	return nil
}

type stubRecoveryFlowSupport struct {
	available        bool
	availableReason  string
	availableErr     error
	usePool          bool
	poolAllowed      bool
	poolSlug         string
	poolErr          error
	backlog          bool
	backlogErr       error
	taskProjectID    int64
	taskProjectErr   error
	activeProjectID  int64
	activeProjectErr error
	openProjectID    int64
	openProjectErr   error
	latestProjectID  int64
	latestProjectErr error
}

func (s *stubRecoveryFlowSupport) SharedAccountAvailable(string) (bool, string, error) {
	if s.availableErr != nil {
		return false, "", s.availableErr
	}
	return s.available, s.availableReason, nil
}

func (s *stubRecoveryFlowSupport) UsesSharedLocalPoolForReactivation(string, *db.Proyecto) bool {
	return s.usePool
}

func (s *stubRecoveryFlowSupport) PoolLocalActivationAllowed(string, string) (bool, string, error) {
	if s.poolErr != nil {
		return false, "", s.poolErr
	}
	return s.poolAllowed, s.poolSlug, nil
}

func (s *stubRecoveryFlowSupport) ProjectHasReactivableBacklog(int64) (bool, error) {
	if s.backlogErr != nil {
		return false, s.backlogErr
	}
	return s.backlog, nil
}

func (s *stubRecoveryFlowSupport) ProjectIDFromAssignedWork(string) (int64, error) {
	if s.taskProjectErr != nil {
		return 0, s.taskProjectErr
	}
	return s.taskProjectID, nil
}

func (s *stubRecoveryFlowSupport) ActiveProjectID(string) (int64, error) {
	if s.activeProjectErr != nil {
		return 0, s.activeProjectErr
	}
	return s.activeProjectID, nil
}

func (s *stubRecoveryFlowSupport) OpenSessionProjectID(string) (int64, error) {
	if s.openProjectErr != nil {
		return 0, s.openProjectErr
	}
	return s.openProjectID, nil
}

func (s *stubRecoveryFlowSupport) LatestSessionProjectID(string) (int64, error) {
	if s.latestProjectErr != nil {
		return 0, s.latestProjectErr
	}
	return s.latestProjectID, nil
}

func (s *stubAutonomyStore) GetPersistedAgentQuotaState(string) (string, error) {
	return s.quotaState, s.quotaErr
}

func (s *stubAutonomyStore) GetSessionRuntimeHandle(int64) (*db.RuntimeHandle, error) {
	return s.sessionHandle, s.sessionHandleErr
}

func (s *stubAutonomyStore) ParkActiveSession(agente string, proyectoID *int64) error {
	s.parkedAgent = agente
	s.parkedProjectID = proyectoID
	return nil
}

func (s *stubAutonomyStore) PauseAssignment(agente string, proyectoID int64, motivo string) error {
	if s.pauseErr != nil {
		return s.pauseErr
	}
	s.pauseAgent = agente
	s.pauseProjectID = proyectoID
	s.pauseReason = motivo
	return nil
}

func (s *stubAutonomyStore) Audit(_ string, accion, entidad string, entidadID int64, detalle string) {
	s.auditAction = accion
	s.auditEntity = entidad
	s.auditEntityID = entidadID
	s.auditDetail = detalle
}

func TestLaunchBuildsPrepareAndEnqueuesStart(t *testing.T) {
	t.Parallel()

	agents := &stubAgentPreparer{
		out: &agentesapp.PrepareOutput{
			Agente:   "Codex1",
			Proyecto: agentesapp.ProjectBundle{ID: 7, Slug: "orquestador"},
			Conector: agentesapp.ConnectorBundle{Slug: "codex-cli"},
		},
	}
	runtimes := &stubRuntimeController{controlID: 33, controlAction: "start"}
	service := NewService(agents, runtimes)

	result, err := service.Launch(LaunchRequest{
		Agente:       " Codex1 ",
		Proyecto:     " orquestador ",
		Conector:     " codex-cli ",
		Modelo:       " gpt-5.4 ",
		Razonamiento: " high ",
		Perfil:       " implementacion ",
		Motivo:       " cerrar core ",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.OrderID != 33 || result.Agente != "Codex1" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if agents.input.Agente != "Codex1" || agents.input.Proyecto != "orquestador" {
		t.Fatalf("prepare no normalizado: %+v", agents.input)
	}
	if runtimes.controlReq.Accion != "start" || runtimes.controlReq.Conector != "codex-cli" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
	if runtimes.controlReq.Por != "orquesta" {
		t.Fatalf("actor por defecto inesperado: %+v", runtimes.controlReq)
	}
}

func TestEnqueueControlNormalizesPayload(t *testing.T) {
	t.Parallel()

	runtimes := &stubRuntimeController{controlID: 7, controlAction: "pause"}
	service := NewService(nil, runtimes)

	orderID, accion, err := service.EnqueueControl(ControlRequest{
		Agente:       " Codex2 ",
		Proyecto:     " core ",
		Accion:       " pause ",
		Conector:     " codex ",
		Modelo:       " gpt-5.4 ",
		Razonamiento: " medium ",
		Perfil:       " review ",
		Motivo:       " mantenimiento ",
		Por:          " admin ",
	})
	if err != nil {
		t.Fatalf("EnqueueControl: %v", err)
	}
	if orderID != 7 || accion != "pause" {
		t.Fatalf("respuesta inesperada: %d %s", orderID, accion)
	}
	if runtimes.controlReq.Agente != "Codex2" || runtimes.controlReq.Proyecto != "core" {
		t.Fatalf("payload no normalizado: %+v", runtimes.controlReq)
	}
}

func TestEnqueueControlStartRunsSessionHygieneBeforeEnqueue(t *testing.T) {
	t.Parallel()

	store := &stubAutonomyStore{}
	runtimes := &stubRuntimeController{
		controlID:        12,
		controlAction:    "start",
		reconcileHandles: 2,
		reconcileOrders:  3,
		purgeHandlesResult: &db.PurgaRuntimeHandlesResultado{
			Deleted: 1,
		},
		purgeOrdersResult: &db.PurgaRuntimeOrdersResultado{
			Deleted: 4,
		},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)

	orderID, accion, err := service.EnqueueControl(ControlRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		Accion:   "start",
		Por:      "server",
	})
	if err != nil {
		t.Fatalf("EnqueueControl: %v", err)
	}
	if orderID != 12 || accion != "start" {
		t.Fatalf("respuesta inesperada: %d %s", orderID, accion)
	}
	if store.auditAction != "session_hygiene_start" || store.auditEntity != "runtime" {
		t.Fatalf("auditoria de hygiene inesperada: %+v", store)
	}
	if got := store.auditDetail; got == "" || !strings.Contains(got, "stale_handles=2") || !strings.Contains(got, "stale_orders=3") || !strings.Contains(got, "purged_handles=1") || !strings.Contains(got, "purged_orders=4") {
		t.Fatalf("detalle de hygiene inesperado: %q", got)
	}
	if runtimes.purgeHandlesReq.Agente != "Codex2" || runtimes.purgeHandlesReq.Proyecto != "orquestador" {
		t.Fatalf("purga handles inesperada: %+v", runtimes.purgeHandlesReq)
	}
	if got := strings.Join(runtimes.purgeHandlesReq.Estados, ","); got != "cerrado,fallido" {
		t.Fatalf("estados purge handles inesperados: %s", got)
	}
	if runtimes.purgeOrdersReq.Agente != "Codex2" || runtimes.purgeOrdersReq.Proyecto != "orquestador" {
		t.Fatalf("purga orders inesperada: %+v", runtimes.purgeOrdersReq)
	}
	if got := strings.Join(runtimes.purgeOrdersReq.Estados, ","); got != "completada,fallida,expirada,cancelada" {
		t.Fatalf("estados purge orders inesperados: %s", got)
	}
}

func TestEnqueueControlNonStartSkipsSessionHygiene(t *testing.T) {
	t.Parallel()

	store := &stubAutonomyStore{}
	runtimes := &stubRuntimeController{
		controlID:        13,
		controlAction:    "pause",
		reconcileHandles: 9,
		reconcileOrders:  9,
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)

	_, _, err := service.EnqueueControl(ControlRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		Accion:   "pause",
		Por:      "server",
	})
	if err != nil {
		t.Fatalf("EnqueueControl: %v", err)
	}
	if store.auditAction == "session_hygiene_start" {
		t.Fatalf("no deberia auditar hygiene en accion no-start: %+v", store)
	}
}

func TestPauseAutonomyAlreadySatisfiedByRecentPendingOrder(t *testing.T) {
	t.Parallel()

	runtimes := &stubRuntimeController{
		orders: []*db.RuntimeOrder{{ID: 9, Tipo: "pause", Estado: "pendiente", CreatedAt: time.Now().UTC()}},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(&stubAutonomyStore{})

	ok, err := service.PauseAutonomyAlreadySatisfied("Codex1", nil, &db.Sesion{Agente: "Codex1", Estado: "activa"})
	if err != nil {
		t.Fatalf("PauseAutonomyAlreadySatisfied: %v", err)
	}
	if !ok {
		t.Fatal("debería considerar satisfecha la pausa por orden pendiente")
	}
}

func TestPauseAutonomySessionIfNeededParksPausedSession(t *testing.T) {
	t.Parallel()

	proyectoID := int64(7)
	store := &stubAutonomyStore{}
	service := NewService(nil, &stubRuntimeController{})
	service.SetAutonomyStore(store)

	count, err := service.PauseAutonomySessionIfNeeded(&db.Sesion{
		ID:         3,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Estado:     "pausada",
	}, &db.Proyecto{ID: proyectoID, Slug: "orquestador"}, "cierre")
	if err != nil {
		t.Fatalf("PauseAutonomySessionIfNeeded: %v", err)
	}
	if count != 1 {
		t.Fatalf("count inesperado: %d", count)
	}
	if store.parkedAgent != "Codex1" || store.pauseProjectID != proyectoID || store.pauseReason != "cierre" {
		t.Fatalf("persistencia inesperada: %+v", store)
	}
}

func TestPauseAutonomySessionIfNeededEnqueuesPauseWhenNotSatisfied(t *testing.T) {
	t.Parallel()

	proyectoID := int64(11)
	runtimes := &stubRuntimeController{controlID: 44, controlAction: "pause"}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(&stubAutonomyStore{})

	count, err := service.PauseAutonomySessionIfNeeded(&db.Sesion{
		ID:         5,
		Agente:     "Codex2",
		ProyectoID: &proyectoID,
		Estado:     "activa",
	}, &db.Proyecto{ID: proyectoID, Slug: "core"}, "bloqueo")
	if err != nil {
		t.Fatalf("PauseAutonomySessionIfNeeded: %v", err)
	}
	if count != 1 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "pause" || runtimes.controlReq.Proyecto != "core" || runtimes.controlReq.Motivo != "bloqueo" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
}

func TestPauseAutonomySessionIfNeededReturnsZeroWhenAlreadySatisfied(t *testing.T) {
	t.Parallel()

	proyectoID := int64(13)
	runtimes := &stubRuntimeController{
		orders: []*db.RuntimeOrder{{ID: 1, Tipo: "pause", Estado: "pendiente", CreatedAt: time.Now().UTC()}},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(&stubAutonomyStore{})

	count, err := service.PauseAutonomySessionIfNeeded(&db.Sesion{
		ID:         7,
		Agente:     "Codex3",
		ProyectoID: &proyectoID,
		Estado:     "activa",
	}, &db.Proyecto{ID: proyectoID, Slug: "ops"}, "mantenimiento")
	if err != nil {
		t.Fatalf("PauseAutonomySessionIfNeeded: %v", err)
	}
	if count != 0 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no debería encolar control: %+v", runtimes.controlReq)
	}
}

func TestPauseAutonomyAlreadySatisfiedDetectsCooldownStateWithoutHandle(t *testing.T) {
	t.Parallel()

	service := NewService(nil, &stubRuntimeController{})
	service.SetAutonomyStore(&stubAutonomyStore{quotaState: "agotado"})

	ok, err := service.PauseAutonomyAlreadySatisfied("Codex4", nil, &db.Sesion{Agente: "Codex4", Estado: "activa"})
	if err != nil {
		t.Fatalf("PauseAutonomyAlreadySatisfied: %v", err)
	}
	if !ok {
		t.Fatal("debería considerar satisfecha la pausa con cuota agotada y sin handle")
	}
}

func TestPauseAutonomySessionIfNeededRequiresAutonomyStore(t *testing.T) {
	t.Parallel()

	service := NewService(nil, &stubRuntimeController{})
	_, err := service.PauseAutonomySessionIfNeeded(&db.Sesion{}, &db.Proyecto{}, "motivo")
	if err == nil {
		t.Fatal("esperaba error sin autonomy store")
	}
	if got := err.Error(); got == "" {
		t.Fatal("error vacío")
	}
}

func TestPauseAutonomySessionIfNeededPropagatesPauseAssignmentError(t *testing.T) {
	t.Parallel()

	proyectoID := int64(21)
	store := &stubAutonomyStore{pauseErr: fmt.Errorf("pause failed")}
	service := NewService(nil, &stubRuntimeController{})
	service.SetAutonomyStore(store)

	_, err := service.PauseAutonomySessionIfNeeded(&db.Sesion{
		ID:         9,
		Agente:     "Codex5",
		ProyectoID: &proyectoID,
		Estado:     "pausada",
	}, &db.Proyecto{ID: proyectoID, Slug: "infra"}, "stop")
	if err == nil || err.Error() != "pause failed" {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestEnqueueAutonomyNudgeCreatesRuntimeOrderAndAudit(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	runtimeID := int64(44)
	handleID := int64(9)
	store := &stubAutonomyStore{}
	runtimes := &stubRuntimeController{
		controlID: 77,
		deliveryHandle: &db.RuntimeHandle{
			ID:        handleID,
			RuntimeID: &runtimeID,
			Estado:    "activo",
		},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)

	ok, err := service.EnqueueAutonomyNudge(NudgeRequest{
		Agente:      "Codex1",
		Proyecto:    &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		Accion:      "continuar_trabajo",
		Motivo:      "seguir",
		Instruction: "continua",
		Extras:      map[string]any{"tarea_id": int64(88)},
	})
	if err != nil {
		t.Fatalf("EnqueueAutonomyNudge: %v", err)
	}
	if !ok {
		t.Fatal("debería encolar nudge")
	}
	if store.auditAction != "autonomia_nudge" || store.auditEntityID != 77 {
		t.Fatalf("audit inesperada: %+v", store)
	}
}

func TestEnqueueAutonomyNudgeSkipsWhenMailboxPending(t *testing.T) {
	t.Parallel()

	proyectoID := int64(32)
	runtimes := &stubRuntimeController{
		deliveryHandle: &db.RuntimeHandle{
			ID:               3,
			Estado:           "activo",
			CapabilitiesJSON: `{"can_send_input":false}`,
		},
		mailbox: []*db.RuntimeMailboxMessage{{
			ID:          1,
			Kind:        "autonomia",
			PayloadJSON: `{"accion":"continuar_trabajo"}`,
		}},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(&stubAutonomyStore{})

	ok, err := service.EnqueueAutonomyNudge(NudgeRequest{
		Agente:   "Codex2",
		Proyecto: &db.Proyecto{ID: proyectoID, Slug: "core"},
		Accion:   "continuar_trabajo",
		Motivo:   "seguir",
	})
	if err != nil {
		t.Fatalf("EnqueueAutonomyNudge: %v", err)
	}
	if ok {
		t.Fatal("no debería encolar con mailbox pendiente")
	}
}

func TestEnqueuePostRemediationFollowupAddsVerificationMetadata(t *testing.T) {
	t.Parallel()

	proyectoID := int64(33)
	runtimeID := int64(45)
	handleID := int64(10)
	store := &stubAutonomyStore{}
	runtimes := &stubRuntimeController{
		controlID: 91,
		deliveryHandle: &db.RuntimeHandle{
			ID:        handleID,
			RuntimeID: &runtimeID,
			Estado:    "activo",
		},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)

	ok, err := service.EnqueuePostRemediationFollowup(PostRemediationFollowupRequest{
		Agente:          "Codex2",
		Proyecto:        &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		TareaID:         41,
		RemediationKind: "reassign",
		OriginAgent:     "Codex1",
		VerificationKey: "reassign:41:Codex1:Codex2",
		Motivo:          "seguir frente",
		Instruction:     "continúa con la tarea reasignada y deja evidencia de avance",
	})
	if err != nil {
		t.Fatalf("EnqueuePostRemediationFollowup: %v", err)
	}
	if !ok {
		t.Fatal("debería encolar follow-up post-remediation")
	}
	if runtimes.enqueuedRuntimeOrder == nil {
		t.Fatal("faltaba runtime order encolada")
	}
	payload := runtimes.enqueuedRuntimeOrder.PayloadJSON
	for _, fragment := range []string{
		`"accion":"continuar_trabajo"`,
		`"post_remediation":true`,
		`"tarea_id":41`,
		`"remediation_kind":"reassign"`,
		`"origin_agent":"Codex1"`,
		`"verification_key":"reassign:41:Codex1:Codex2"`,
	} {
		if !strings.Contains(payload, fragment) {
			t.Fatalf("payload sin %s: %s", fragment, payload)
		}
	}
}

func TestEnqueuePostRemediationFollowupDedupesByVerificationKey(t *testing.T) {
	t.Parallel()

	proyectoID := int64(34)
	now := time.Now().UTC()
	store := &stubAutonomyStore{}
	runtimes := &stubRuntimeController{
		deliveryHandle: &db.RuntimeHandle{ID: 11, RuntimeID: ptrInt64(46), Estado: "activo"},
		orders: []*db.RuntimeOrder{{
			ID:          17,
			Agente:      "Codex2",
			ProyectoID:  &proyectoID,
			Tipo:        "nudge",
			Estado:      "pendiente",
			PayloadJSON: `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"reassign:41:Codex1:Codex2"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}},
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)

	ok, err := service.EnqueuePostRemediationFollowup(PostRemediationFollowupRequest{
		Agente:          "Codex2",
		Proyecto:        &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		TareaID:         41,
		RemediationKind: "reassign",
		OriginAgent:     "Codex1",
		VerificationKey: "reassign:41:Codex1:Codex2",
		Instruction:     "continúa con la tarea reasignada y deja evidencia de avance",
	})
	if err != nil {
		t.Fatalf("EnqueuePostRemediationFollowup: %v", err)
	}
	if ok {
		t.Fatal("no debería duplicar follow-up con la misma verification_key")
	}
	if runtimes.enqueuedRuntimeOrder != nil {
		t.Fatalf("no debería encolar runtime order nueva: %+v", runtimes.enqueuedRuntimeOrder)
	}
}

func TestGetPostRemediationStatusReturnsLatestSuccessfulReceipt(t *testing.T) {
	t.Parallel()

	proyectoID := int64(35)
	now := time.Now().UTC()
	service := NewService(nil, &stubRuntimeController{
		orders: []*db.RuntimeOrder{
			{
				ID:            18,
				Agente:        "Codex2",
				ProyectoID:    &proyectoID,
				Tipo:          "nudge",
				Estado:        "pendiente",
				PayloadJSON:   `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"reassign:41:Codex1:Codex2","remediation_kind":"reassign"}`,
				ResultadoJSON: `{"dispatch_state":"notified","delivery_state":"notified","verification_key":"reassign:41:Codex1:Codex2","remediation_kind":"reassign"}`,
				UpdatedAt:     now.Add(-time.Minute),
			},
			{
				ID:            19,
				Agente:        "Codex2",
				ProyectoID:    &proyectoID,
				Tipo:          "nudge",
				Estado:        "completada",
				PayloadJSON:   `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"reassign:41:Codex1:Codex2","remediation_kind":"reassign"}`,
				ResultadoJSON: `{"dispatch_state":"delivered","delivery_state":"delivered","receipt_source":"tmux_pane_activity","verification_key":"reassign:41:Codex1:Codex2","remediation_kind":"reassign"}`,
				UpdatedAt:     now,
			},
		},
	})

	status, err := service.GetPostRemediationStatus("Codex2", &proyectoID, "reassign:41:Codex1:Codex2")
	if err != nil {
		t.Fatalf("GetPostRemediationStatus: %v", err)
	}
	if status == nil || !status.Found {
		t.Fatalf("faltaba status: %+v", status)
	}
	if status.OrderID != 19 || !status.Succeeded || status.Waiting {
		t.Fatalf("status inesperado: %+v", status)
	}
	if status.ReceiptSource != "tmux_pane_activity" || status.RemediationKind != "reassign" {
		t.Fatalf("status sin receipt/remediation correctos: %+v", status)
	}
}

func TestGetPostRemediationStatusReturnsWaitingForNotifiedOrder(t *testing.T) {
	t.Parallel()

	proyectoID := int64(36)
	now := time.Now().UTC()
	service := NewService(nil, &stubRuntimeController{
		orders: []*db.RuntimeOrder{{
			ID:            20,
			Agente:        "Codex2",
			ProyectoID:    &proyectoID,
			Tipo:          "nudge",
			Estado:        "pendiente",
			PayloadJSON:   `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"reassign:42:Codex1:Codex2","remediation_kind":"reassign"}`,
			ResultadoJSON: `{"dispatch_state":"notified","delivery_state":"notified","verification_key":"reassign:42:Codex1:Codex2","remediation_kind":"reassign","retry_after":"2030-01-01T00:00:00Z"}`,
			UpdatedAt:     now,
		}},
	})

	status, err := service.GetPostRemediationStatus("Codex2", &proyectoID, "reassign:42:Codex1:Codex2")
	if err != nil {
		t.Fatalf("GetPostRemediationStatus: %v", err)
	}
	if status == nil || !status.Found {
		t.Fatalf("faltaba status: %+v", status)
	}
	if status.Succeeded || !status.Waiting {
		t.Fatalf("status debería seguir esperando: %+v", status)
	}
	if status.DispatchState != "notified" || status.RetryAfter == "" {
		t.Fatalf("status sin dispatch/retry correctos: %+v", status)
	}
}

func TestResolveRecoveryHandlePrefersOperationalHandle(t *testing.T) {
	t.Parallel()

	service := NewService(nil, &stubRuntimeController{
		listHandles: []*db.RuntimeHandle{{ID: 44, Estado: "activo"}},
	})

	handle, err := service.ResolveRecoveryHandle("Codex1", nil)
	if err != nil {
		t.Fatalf("ResolveRecoveryHandle: %v", err)
	}
	if handle == nil || handle.ID != 44 {
		t.Fatalf("handle inesperado: %+v", handle)
	}
}

func TestResolveSessionRecoveryTargetUsesSessionHandleAndRuntime(t *testing.T) {
	t.Parallel()

	proyectoID := int64(7)
	service := NewService(nil, &stubRuntimeController{
		sessionHandle: &db.RuntimeHandle{ID: 12, SesionID: ptrInt64(9), RuntimeID: ptrInt64(15)},
		runtime:       &db.RuntimeInstance{ID: 15, Agente: "Codex1", ProyectoID: &proyectoID},
	})

	handle, runtime, err := service.ResolveSessionRecoveryTarget(&db.Sesion{
		ID:         9,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("ResolveSessionRecoveryTarget: %v", err)
	}
	if handle == nil || handle.ID != 12 {
		t.Fatalf("handle inesperado: %+v", handle)
	}
	if runtime == nil || runtime.ID != 15 {
		t.Fatalf("runtime inesperado: %+v", runtime)
	}
}

func TestHasRecentOperationalRuntimeOtherThanSessionDetectsTMUXHandle(t *testing.T) {
	t.Parallel()

	proyectoID := int64(9)
	service := NewService(nil, &stubRuntimeController{
		operationalHandle: &db.RuntimeHandle{
			ID:         77,
			SesionID:   ptrInt64(10),
			Estado:     "activo",
			Transporte: "tmux",
		},
	})

	ok, err := service.HasRecentOperationalRuntimeOtherThanSession(&db.Sesion{
		ID:         5,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("HasRecentOperationalRuntimeOtherThanSession: %v", err)
	}
	if !ok {
		t.Fatal("debería detectar runtime operativo distinto de la sesión")
	}
}

func TestHasRecentOperationalRuntimeOtherThanSessionIgnoresSameSession(t *testing.T) {
	t.Parallel()

	proyectoID := int64(9)
	service := NewService(nil, &stubRuntimeController{
		operationalHandle: &db.RuntimeHandle{
			ID:         77,
			SesionID:   ptrInt64(5),
			Estado:     "activo",
			Transporte: "tmux",
		},
	})

	ok, err := service.HasRecentOperationalRuntimeOtherThanSession(&db.Sesion{
		ID:         5,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("HasRecentOperationalRuntimeOtherThanSession: %v", err)
	}
	if ok {
		t.Fatal("no debería considerar distinta la misma sesión")
	}
}

func TestShouldSupersedeSessionRecoveryWithRecentTMUX(t *testing.T) {
	t.Parallel()

	ok := NewService(nil, nil).ShouldSupersedeSessionRecoveryWithRecentTMUX(
		&db.Sesion{ID: 5},
		&db.RuntimeHandle{
			SesionID:   ptrInt64(10),
			Estado:     "activo",
			Transporte: "tmux",
			HandleKind: "session",
			LastSeenAt: ptrTime(time.Now().UTC()),
		},
	)
	if !ok {
		t.Fatal("debería suplantar la recuperación con handle tmux reciente")
	}
}

func TestShouldSupersedeSessionRecoveryWithRecentTMUXIgnoresSameSession(t *testing.T) {
	t.Parallel()

	ok := NewService(nil, nil).ShouldSupersedeSessionRecoveryWithRecentTMUX(
		&db.Sesion{ID: 5},
		&db.RuntimeHandle{
			SesionID:   ptrInt64(5),
			Estado:     "activo",
			Transporte: "tmux",
			HandleKind: "session",
			LastSeenAt: ptrTime(time.Now().UTC()),
		},
	)
	if ok {
		t.Fatal("no debería suplantar si el handle pertenece a la misma sesión")
	}
}

func TestRecoverLocalFailedRuntimeSessionEnqueuesStartWithPersistedProfile(t *testing.T) {
	t.Parallel()

	proyectoID := int64(13)
	runtimeID := int64(21)
	runtimes := &stubRuntimeController{
		runtime: &db.RuntimeInstance{
			ID:           runtimeID,
			Agente:       "Codex1",
			ProyectoID:   &proyectoID,
			LogicalState: "fallido",
			ProcessState: "fallido",
		},
		controlID:     4,
		controlAction: "start",
	}
	service := NewService(nil, runtimes)
	service.SetStartableWorkChecker(&stubStartableWorkChecker{ok: true})

	count, err := service.RecoverLocalFailedRuntimeSession(
		&db.Sesion{
			ID:                9,
			Agente:            "Codex1",
			ProyectoID:        &proyectoID,
			ResumePayloadJSON: `{"perfil_ejecucion":{"perfil_tarea":"implementacion","modelo":"gemma4:26b","razonamiento":"medium"}}`,
		},
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		&db.RuntimeHandle{ID: 3, Estado: "fallido", Transporte: "cli", HandleKind: "process", RuntimeID: &runtimeID},
		nil,
	)
	if err != nil {
		t.Fatalf("RecoverLocalFailedRuntimeSession: %v", err)
	}
	if count != 1 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "start" || runtimes.controlReq.Motivo != "local_runtime_failed" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
	if runtimes.controlReq.Perfil != "implementacion" || runtimes.controlReq.Modelo != "gemma4:26b" || runtimes.controlReq.Razonamiento != "medium" {
		t.Fatalf("perfil persistido inesperado: %+v", runtimes.controlReq)
	}
}

func TestRecoverLocalFailedRuntimeSessionSkipsWhenPendingOrderExists(t *testing.T) {
	t.Parallel()

	proyectoID := int64(13)
	runtimeID := int64(21)
	runtimes := &stubRuntimeController{
		runtime: &db.RuntimeInstance{
			ID:           runtimeID,
			Agente:       "Codex1",
			ProyectoID:   &proyectoID,
			LogicalState: "fallido",
			ProcessState: "fallido",
		},
		orders: []*db.RuntimeOrder{{
			ID:     5,
			Tipo:   "pause",
			Estado: "pendiente",
		}},
	}
	service := NewService(nil, runtimes)
	service.SetStartableWorkChecker(&stubStartableWorkChecker{ok: true})

	count, err := service.RecoverLocalFailedRuntimeSession(
		&db.Sesion{ID: 9, Agente: "Codex1", ProyectoID: &proyectoID},
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		&db.RuntimeHandle{ID: 3, Estado: "fallido", Transporte: "cli", HandleKind: "process", RuntimeID: &runtimeID},
		nil,
	)
	if err != nil {
		t.Fatalf("RecoverLocalFailedRuntimeSession: %v", err)
	}
	if count != 0 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no deberia encolar start: %+v", runtimes.controlReq)
	}
}

func TestRecoverLocalFailedRuntimeSessionSkipsHealthyRuntimeWithoutSync(t *testing.T) {
	t.Parallel()

	proyectoID := int64(13)
	runtimeID := int64(21)
	runtimes := &stubRuntimeController{
		runtime: &db.RuntimeInstance{
			ID:           runtimeID,
			Agente:       "Codex1",
			ProyectoID:   &proyectoID,
			LogicalState: "activo",
			ProcessState: "running",
		},
	}
	service := NewService(nil, runtimes)
	service.SetStartableWorkChecker(&stubStartableWorkChecker{ok: true})

	count, err := service.RecoverLocalFailedRuntimeSession(
		&db.Sesion{ID: 9, Agente: "Codex1", ProyectoID: &proyectoID},
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		&db.RuntimeHandle{ID: 3, Estado: "activo", Transporte: "cli", HandleKind: "process", RuntimeID: &runtimeID},
		nil,
	)
	if err != nil {
		t.Fatalf("RecoverLocalFailedRuntimeSession: %v", err)
	}
	if count != 0 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.syncCount != 0 {
		t.Fatalf("no deberia sincronizar handle sano, syncs=%d", runtimes.syncCount)
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no deberia encolar start: %+v", runtimes.controlReq)
	}
}

func TestRecoverRemoteDegradedRuntimeSessionEnqueuesCheckpointAndResume(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	runtimeID := int64(44)
	runtimes := &stubRuntimeController{
		runtime:       &db.RuntimeInstance{ID: runtimeID, Agente: "Codex1", ProyectoID: &proyectoID},
		controlID:     8,
		controlAction: "start",
	}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(&stubAutonomyStore{})
	service.SetRemoteRecoverySupport(&stubRemoteRecoverySupport{
		conector:  &db.Conector{ID: 5, Slug: "codex-remote"},
		available: true,
	})

	count, err := service.RecoverRemoteDegradedRuntimeSession(
		&db.Sesion{ID: 9, Agente: "Codex1", ProyectoID: &proyectoID, ExternalSessionID: "sess-1"},
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		&db.RuntimeHandle{ID: 3, Estado: "fallido", Transporte: "api", HandleKind: "session", HandleRef: "remote-1", RuntimeID: &runtimeID},
		&db.RuntimeInstance{ID: runtimeID, Agente: "Codex1", ProyectoID: &proyectoID, LogicalState: "degradado", ProcessState: "remote_status_error"},
	)
	if err != nil {
		t.Fatalf("RecoverRemoteDegradedRuntimeSession: %v", err)
	}
	if count != 2 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no deberia encolar control start: %+v", runtimes.controlReq)
	}
}

func TestRecoverRemoteDegradedRuntimeSessionPausesWhenConnectorUnavailable(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	store := &stubAutonomyStore{}
	support := &stubRemoteRecoverySupport{
		conector:  &db.Conector{ID: 5, Slug: "codex-remote"},
		available: false,
	}
	runtimes := &stubRuntimeController{controlID: 8, controlAction: "pause"}
	service := NewService(nil, runtimes)
	service.SetAutonomyStore(store)
	service.SetRemoteRecoverySupport(support)

	count, err := service.RecoverRemoteDegradedRuntimeSession(
		&db.Sesion{ID: 9, Agente: "Codex1", ProyectoID: &proyectoID, Estado: "activa"},
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		&db.RuntimeHandle{ID: 3, Estado: "fallido", Transporte: "api", HandleKind: "session", HandleRef: "remote-1"},
		&db.RuntimeInstance{ID: 44, Agente: "Codex1", ProyectoID: &proyectoID, LogicalState: "degradado", ProcessState: "remote_status_error"},
	)
	if err != nil {
		t.Fatalf("RecoverRemoteDegradedRuntimeSession: %v", err)
	}
	if count != 1 {
		t.Fatalf("count inesperado: %d", count)
	}
	if support.blockProjectID != proyectoID || support.blockReason != "conector:codex-remote:circuito_abierto" {
		t.Fatalf("bloqueo inesperado: %+v", support)
	}
	if runtimes.controlReq.Accion != "pause" || runtimes.controlReq.Motivo != "conector:codex-remote:circuito_abierto" {
		t.Fatalf("pause inesperada: %+v", runtimes.controlReq)
	}
}

func TestRecoverDegradedRuntimeSessionReactivatesWhenHandleMissing(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	flow := &stubRecoveryFlowSupport{available: true}
	runtimes := &stubRuntimeController{
		project:       &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		controlID:     8,
		controlAction: "start",
	}
	service := NewService(nil, runtimes)
	service.SetRecoveryFlowSupport(flow)
	service.SetStartableWorkChecker(&stubStartableWorkChecker{ok: true})

	count, err := service.RecoverDegradedRuntimeSession(&db.Sesion{
		ID:         9,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("RecoverDegradedRuntimeSession: %v", err)
	}
	if count != 1 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "start" || runtimes.controlReq.Motivo != "local_runtime_missing" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
}

func TestRecoverDegradedRuntimeSessionSkipsWhenSharedAccountUnavailable(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	flow := &stubRecoveryFlowSupport{available: false}
	runtimes := &stubRuntimeController{
		project: &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
	}
	service := NewService(nil, runtimes)
	service.SetRecoveryFlowSupport(flow)

	count, err := service.RecoverDegradedRuntimeSession(&db.Sesion{
		ID:         9,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("RecoverDegradedRuntimeSession: %v", err)
	}
	if count != 0 {
		t.Fatalf("count inesperado: %d", count)
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no deberia intentar reactivar: %+v", runtimes.controlReq)
	}
}

func TestReactivateProjectIfNeededSkipsWhenHandleAlreadyActive(t *testing.T) {
	t.Parallel()

	proyectoID := int64(31)
	runtimes := &stubRuntimeController{}
	service := NewService(nil, runtimes)
	service.SetStartableWorkChecker(&stubStartableWorkChecker{ok: true})
	service.SetRecoveryFlowSupport(&stubRecoveryFlowSupport{available: true})

	ok, err := service.ReactivateProjectIfNeeded(
		"Codex1",
		&db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		"reanimacion_automatica",
		&db.RuntimeHandle{ID: 5, Estado: "activo"},
	)
	if err != nil {
		t.Fatalf("ReactivateProjectIfNeeded: %v", err)
	}
	if ok {
		t.Fatal("no deberia reactivar con handle ya activo")
	}
	if runtimes.controlReq.Accion != "" {
		t.Fatalf("no deberia encolar control: %+v", runtimes.controlReq)
	}
}

func TestResolveReactivationProjectPrefersActiveAssignment(t *testing.T) {
	t.Parallel()

	runtimes := &stubRuntimeController{
		project: &db.Proyecto{ID: 31, Slug: "orquestador"},
		mailbox: []*db.RuntimeMailboxMessage{{ID: 9, ProyectoID: ptrInt64(55)}},
	}
	service := NewService(nil, runtimes)
	service.SetRecoveryFlowSupport(&stubRecoveryFlowSupport{
		activeProjectID: 31,
		taskProjectID:   44,
		openProjectID:   66,
		latestProjectID: 77,
	})

	proyecto, err := service.ResolveReactivationProject("Codex1")
	if err != nil {
		t.Fatalf("ResolveReactivationProject: %v", err)
	}
	if proyecto == nil || proyecto.ID != 31 {
		t.Fatalf("proyecto inesperado: %+v", proyecto)
	}
}

func TestResolveReactivationProjectFallsBackToMailbox(t *testing.T) {
	t.Parallel()

	runtimes := &stubRuntimeController{
		project: &db.Proyecto{ID: 55, Slug: "mailbox-project"},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 1, ProyectoID: ptrInt64(44)},
			{ID: 3, ProyectoID: ptrInt64(55)},
		},
	}
	service := NewService(nil, runtimes)
	service.SetRecoveryFlowSupport(&stubRecoveryFlowSupport{})

	proyecto, err := service.ResolveReactivationProject("Codex1")
	if err != nil {
		t.Fatalf("ResolveReactivationProject: %v", err)
	}
	if proyecto == nil || proyecto.ID != 55 {
		t.Fatalf("proyecto inesperado: %+v", proyecto)
	}
}

func TestResolveReactivationProjectFallsBackToRecoveryHandleProject(t *testing.T) {
	t.Parallel()

	proyectoID := int64(88)
	runtimes := &stubRuntimeController{
		project:     &db.Proyecto{ID: proyectoID, Slug: "handle-project"},
		listHandles: []*db.RuntimeHandle{{ID: 5, Estado: "pausado", ProyectoID: &proyectoID}},
	}
	service := NewService(nil, runtimes)
	service.SetRecoveryFlowSupport(&stubRecoveryFlowSupport{})

	proyecto, err := service.ResolveReactivationProject("Codex1")
	if err != nil {
		t.Fatalf("ResolveReactivationProject: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoID {
		t.Fatalf("proyecto inesperado: %+v", proyecto)
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
