package orquestacionagentesapp

import (
	"fmt"
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
