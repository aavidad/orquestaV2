package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorPreparaAutomejoraDesdeBacklogPlanV0(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 10, 0, 0, time.UTC)
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-backlog-t01",
		}, {
			RequestRef: "request-ref-backlog-t02",
		}, {
			RequestRef: "request-ref-backlog-t03",
		}},
		selfStarted: make(chan struct{}, 3),
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementMaxRequests: 2,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	if runtime.config.IdleSelfImprovementAfter != 60*time.Second {
		t.Fatalf("idle default=%s want=60s", runtime.config.IdleSelfImprovementAfter)
	}
	markNoExecutionSinceForTestV0(runtime, now)
	runtime.runSupervisorTickV0(context.Background())
	for i := 0; i < 2; i++ {
		select {
		case <-supervisor.selfStarted:
		case <-time.After(time.Second):
			t.Fatalf("automejora planificada no preparada: i=%d calls=%d", i, supervisor.selfCalls)
		}
	}
	if supervisor.planCalls != 1 ||
		supervisor.lastPlanRequest.MaxRequests != 2 ||
		supervisor.selfCalls != 2 ||
		supervisor.selfRequestRefs[0] != "request-ref-backlog-t01" ||
		supervisor.selfRequestRefs[1] != "request-ref-backlog-t02" {
		t.Fatalf("plan_calls=%d plan=%+v refs=%v calls=%d",
			supervisor.planCalls,
			supervisor.lastPlanRequest,
			supervisor.selfRequestRefs,
			supervisor.selfCalls,
		)
	}
}

type blockingSupervisorV0 struct {
	started, releaseCh, done chan struct{}
}

func newBlockingSupervisorV0() *blockingSupervisorV0 {
	return &blockingSupervisorV0{started: make(chan struct{}, 2), releaseCh: make(chan struct{}, 2), done: make(chan struct{}, 2)}
}

func (fake *blockingSupervisorV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.started <- struct{}{}
	<-fake.releaseCh
	fake.done <- struct{}{}
	return orquestarunsupervisor.RunSupervisorResultV0{StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0}, nil
}

func (fake *fakeSupervisorV0) PlanIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementPlanRequestV0,
) (IdleSelfImprovementPlanResultV0, error) {
	fake.planCalls++
	fake.lastPlanRequest = request
	if fake.planErr != nil {
		return IdleSelfImprovementPlanResultV0{}, fake.planErr
	}
	requests := append([]IdleSelfImprovementRequestV0(nil), fake.planRequests...)
	if fake.planRequests == nil {
		requests = []IdleSelfImprovementRequestV0{request.BaseRequest}
	}
	return IdleSelfImprovementPlanResultV0{
		Requests: requests,
	}, nil
}

func (fake *blockingSupervisorV0) release() {
	fake.releaseCh <- struct{}{}
}

func (fake *blockingSupervisorV0) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-fake.done:
	case <-time.After(time.Second):
		t.Fatalf("blocking supervisor did not finish")
	}
}
