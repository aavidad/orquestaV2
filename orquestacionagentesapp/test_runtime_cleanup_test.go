package orquestacionagentesapp

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type stubTestRuntimeOps struct {
	activity      []bool
	activityErr   error
	activityCalls int
	wakeOrders    bool
	wakeMailbox   bool
	wakeWarm      bool
	wakeErr       error
}

func (s *stubTestRuntimeOps) HasLiveRuntimeActivity(string, string, int) (bool, error) {
	if s.activityErr != nil {
		return false, s.activityErr
	}
	idx := s.activityCalls
	s.activityCalls++
	if idx >= len(s.activity) {
		if len(s.activity) == 0 {
			return false, nil
		}
		return s.activity[len(s.activity)-1], nil
	}
	return s.activity[idx], nil
}

func (s *stubTestRuntimeOps) WakeRuntimeBatches(orders, mailbox, warm bool) error {
	if s.wakeErr != nil {
		return s.wakeErr
	}
	s.wakeOrders = orders
	s.wakeMailbox = mailbox
	s.wakeWarm = warm
	return nil
}

type stubTestRuntimeControlOps struct {
	stubTestRuntimeOps
	controlReq ControlRequest
	controlErr error
}

func (s *stubTestRuntimeControlOps) EnqueueStopControl(req ControlRequest) error {
	if s.controlErr != nil {
		return s.controlErr
	}
	s.controlReq = req
	return nil
}

type stubTestRuntimeEnvironmentOps struct {
	stubTestRuntimeControlOps
	steps            []string
	cleanupFrontErr  error
	mailboxErr       error
	ordersErr        error
	closeHandlesErr  error
	purgeHandlesErr  error
	closeRuntimesErr error
	keepIDs          []int64
	mailboxKinds     []string
	cleanupAgente    string
	cleanupProyecto  string
	cleanupActor     string
}

func (s *stubTestRuntimeEnvironmentOps) CleanupTaskFront(agente, proyecto string, keepIDs []int64) error {
	if s.cleanupFrontErr != nil {
		return s.cleanupFrontErr
	}
	s.steps = append(s.steps, "front")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.keepIDs = append([]int64(nil), keepIDs...)
	return nil
}

func (s *stubTestRuntimeEnvironmentOps) ClearPendingRuntimeMailbox(agente, proyecto string, kinds []string) error {
	if s.mailboxErr != nil {
		return s.mailboxErr
	}
	s.steps = append(s.steps, "mailbox")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.mailboxKinds = append([]string(nil), kinds...)
	return nil
}

func (s *stubTestRuntimeEnvironmentOps) PurgeTerminalRuntimeOrders(agente, proyecto, actor string) error {
	if s.ordersErr != nil {
		return s.ordersErr
	}
	s.steps = append(s.steps, "orders")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.cleanupActor = actor
	return nil
}

func (s *stubTestRuntimeEnvironmentOps) CloseResidualRuntimeHandles(agente, proyecto, actor string) error {
	if s.closeHandlesErr != nil {
		return s.closeHandlesErr
	}
	s.steps = append(s.steps, "close_handles")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.cleanupActor = actor
	return nil
}

func (s *stubTestRuntimeEnvironmentOps) PurgeClosedRuntimeHandles(agente, proyecto, actor string) error {
	if s.purgeHandlesErr != nil {
		return s.purgeHandlesErr
	}
	s.steps = append(s.steps, "purge_handles")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.cleanupActor = actor
	return nil
}

func (s *stubTestRuntimeEnvironmentOps) CloseResidualRuntimes(agente, proyecto, actor string) error {
	if s.closeRuntimesErr != nil {
		return s.closeRuntimesErr
	}
	s.steps = append(s.steps, "close_runtimes")
	s.cleanupAgente = agente
	s.cleanupProyecto = proyecto
	s.cleanupActor = actor
	return nil
}

func TestStopAgentRuntimeIfActiveNoopWhenAgentEmpty(t *testing.T) {
	t.Parallel()

	svc := NewTestRuntimeCleanupService(NewService(nil, &stubRuntimeController{}), &stubTestRuntimeOps{})
	result, err := svc.StopAgentRuntimeIfActive(StopAgentRuntimeIfActiveInput{})
	if err != nil {
		t.Fatalf("StopAgentRuntimeIfActive: %v", err)
	}
	if result.HadLiveActivity || result.StopEnqueued || result.Stopped {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestStopAgentRuntimeIfActiveNoopWhenAlreadyStopped(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeOps{activity: []bool{false}}
	svc := NewTestRuntimeCleanupService(NewService(nil, &stubRuntimeController{}), ops)
	result, err := svc.StopAgentRuntimeIfActive(StopAgentRuntimeIfActiveInput{Agente: "Codex1"})
	if err != nil {
		t.Fatalf("StopAgentRuntimeIfActive: %v", err)
	}
	if result.HadLiveActivity || result.StopEnqueued || result.Stopped {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestStopAgentRuntimeIfActiveEnqueuesStopAndWaits(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeOps{activity: []bool{true, false}}
	runtimes := &stubRuntimeController{controlID: 41, controlAction: "stop"}
	orchestrator := NewService(nil, runtimes)
	svc := NewTestRuntimeCleanupService(orchestrator, ops)

	result, err := svc.StopAgentRuntimeIfActive(StopAgentRuntimeIfActiveInput{
		Agente:       "Codex2",
		Actor:        "orquesta",
		Reason:       "runtime limpiar-pruebas",
		Timeout:      200 * time.Millisecond,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("StopAgentRuntimeIfActive: %v", err)
	}
	if !result.HadLiveActivity || !result.StopEnqueued || !result.Stopped {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if runtimes.controlReq.Agente != "Codex2" || runtimes.controlReq.Accion != "stop" {
		t.Fatalf("control inesperado: %+v", runtimes.controlReq)
	}
	if !ops.wakeOrders || ops.wakeMailbox || !ops.wakeWarm {
		t.Fatalf("wake inesperado: %+v", ops)
	}
}

func TestStopAgentRuntimeIfActiveUsesControlOpsWhenAvailable(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeControlOps{stubTestRuntimeOps: stubTestRuntimeOps{activity: []bool{true, false}}}
	svc := NewTestRuntimeCleanupService(nil, ops)

	result, err := svc.StopAgentRuntimeIfActive(StopAgentRuntimeIfActiveInput{
		Agente:       "CodexAPI",
		Actor:        "orquesta",
		Reason:       "runtime limpiar-pruebas",
		Timeout:      200 * time.Millisecond,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("StopAgentRuntimeIfActive: %v", err)
	}
	if !result.HadLiveActivity || !result.StopEnqueued || !result.Stopped {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if ops.controlReq.Agente != "CodexAPI" || ops.controlReq.Accion != "stop" {
		t.Fatalf("control inesperado: %+v", ops.controlReq)
	}
}

func TestStopAgentRuntimeIfActivePropagatesWakeError(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeOps{activity: []bool{true}, wakeErr: fmt.Errorf("wake failed")}
	svc := NewTestRuntimeCleanupService(NewService(nil, &stubRuntimeController{}), ops)
	_, err := svc.StopAgentRuntimeIfActive(StopAgentRuntimeIfActiveInput{
		Agente:       "Codex3",
		Timeout:      100 * time.Millisecond,
		PollInterval: time.Millisecond,
	})
	if err == nil || err.Error() != "wake failed" {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestCleanupTestRuntimeEnvironmentRunsSequentialCleanup(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeEnvironmentOps{
		stubTestRuntimeControlOps: stubTestRuntimeControlOps{
			stubTestRuntimeOps: stubTestRuntimeOps{activity: []bool{true, false}},
		},
	}
	svc := NewTestRuntimeCleanupService(nil, ops)

	result, err := svc.CleanupTestRuntimeEnvironment(CleanupTestRuntimeEnvironmentInput{
		Agente:       "Codex2",
		Proyecto:     "orquestador",
		KeepTaskIDs:  []int64{3, 9},
		MailboxKinds: []string{"nudge"},
		Actor:        "orquesta",
		Reason:       "runtime limpiar-pruebas",
		StopTimeout:  200 * time.Millisecond,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("CleanupTestRuntimeEnvironment: %v", err)
	}
	if !result.RuntimeStopAttempted || !result.FrontCleaned || !result.MailboxCleared || !result.OrdersPurged || !result.ResidualHandlesClosed || !result.ClosedHandlesPurged || !result.ResidualRuntimesClosed {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if ops.controlReq.Agente != "Codex2" || ops.controlReq.Accion != "stop" {
		t.Fatalf("control inesperado: %+v", ops.controlReq)
	}
	if got := strings.Join(ops.steps, ","); got != "front,mailbox,orders,close_handles,purge_handles,close_runtimes" {
		t.Fatalf("secuencia inesperada: %s", got)
	}
	if got := fmt.Sprint(ops.keepIDs); got != "[3 9]" {
		t.Fatalf("keepIDs inesperados: %v", ops.keepIDs)
	}
	if got := fmt.Sprint(ops.mailboxKinds); got != "[nudge]" {
		t.Fatalf("mailboxKinds inesperados: %v", ops.mailboxKinds)
	}
}

func TestCleanupTestRuntimeEnvironmentSkipsTaskFrontWithoutProject(t *testing.T) {
	t.Parallel()

	ops := &stubTestRuntimeEnvironmentOps{}
	svc := NewTestRuntimeCleanupService(nil, ops)

	result, err := svc.CleanupTestRuntimeEnvironment(CleanupTestRuntimeEnvironmentInput{
		Agente:       "Codex2",
		StopTimeout:  50 * time.Millisecond,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("CleanupTestRuntimeEnvironment: %v", err)
	}
	if result.FrontCleaned {
		t.Fatalf("no deberia limpiar frente sin proyecto: %+v", result)
	}
	if got := strings.Join(ops.steps, ","); got != "mailbox,orders,close_handles,purge_handles,close_runtimes" {
		t.Fatalf("secuencia inesperada: %s", got)
	}
}
