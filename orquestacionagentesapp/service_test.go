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
	controlReq          runtimesapp.AgentControlRequest
	controlID           int64
	controlAction       string
	controlErr          error
	project             *db.Proyecto
	projectErr          error
	runtime             *db.RuntimeInstance
	runtimeErr          error
	orders              []*db.RuntimeOrder
	ordersErr           error
	mailbox             []*db.RuntimeMailboxMessage
	mailboxErr          error
	handleByID          map[int64]*db.RuntimeHandle
	controlHandle       *db.RuntimeHandle
	controlHandleErr    error
	deliveryHandle      *db.RuntimeHandle
	deliveryHandleErr   error
	transcriptHandle    *db.RuntimeHandle
	transcriptHandleErr error
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

func (s *stubRuntimeController) GetPrimaryRuntimeForProject(string, *int64) (*db.RuntimeInstance, error) {
	return s.runtime, s.runtimeErr
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
