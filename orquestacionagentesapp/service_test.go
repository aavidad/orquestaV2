package orquestacionagentesapp

import (
	"fmt"
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

func (s *stubRuntimeController) EnqueueRuntimeOrder(*db.RuntimeOrder) (int64, error) {
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

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
