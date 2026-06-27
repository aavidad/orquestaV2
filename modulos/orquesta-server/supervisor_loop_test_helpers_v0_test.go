package orquestaserver

import (
	"context"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type fakeSupervisorV0 struct {
	calls                 int
	selfCalls             int
	planCalls             int
	lastCommand           orquestarunsupervisor.RunSupervisorCommandV0
	lastSelfRequest       IdleSelfImprovementRequestV0
	lastPlanRequest       IdleSelfImprovementPlanRequestV0
	results               []fakeSupervisorResultV0
	planRequests          []IdleSelfImprovementRequestV0
	planErr               error
	blocker               IdleSelfImprovementBlockerResultV0
	retryableRunRefs      []string
	retryableRequestRefs  []string
	retryableEvidenceRefs []string
	filterCalls           int
	lastFilterRequest     IdleSelfImprovementRequestFilterRequestV0
	filterRequests        []IdleSelfImprovementRequestV0
	filterErr             error
	selfRequestRefs       []string
	selfResults           []IdleSelfImprovementResultV0
	selfErrs              []error
	selfStarted           chan struct{}
	selfRelease           chan struct{}
}

func (fake *fakeSupervisorV0) PrepareIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	fake.selfCalls++
	index := fake.selfCalls - 1
	fake.lastSelfRequest = request
	fake.selfRequestRefs = append(fake.selfRequestRefs, request.RequestRef)
	if fake.selfStarted != nil {
		fake.selfStarted <- struct{}{}
	}
	if fake.selfRelease != nil {
		<-fake.selfRelease
	}
	if index < len(fake.selfErrs) && fake.selfErrs[index] != nil {
		return IdleSelfImprovementResultV0{}, fake.selfErrs[index]
	}
	if index < len(fake.selfResults) {
		return fake.selfResults[index], nil
	}
	return IdleSelfImprovementResultV0{
		Accepted: true, RunRef: "run-ref-idle-self-improvement-001", RequestRef: request.RequestRef, Status: "ok",
	}, nil
}

func (fake *fakeSupervisorV0) IdleSelfImprovementBlockersV0(
	context.Context,
	IdleSelfImprovementBlockerRequestV0,
) (IdleSelfImprovementBlockerResultV0, error) {
	return fake.blocker, nil
}

func (fake *fakeSupervisorV0) RetryableIdleSelfImprovementRunRefsV0(
	context.Context,
	IdleSelfImprovementRunFreshnessRequestV0,
) (IdleSelfImprovementRunFreshnessResultV0, error) {
	return IdleSelfImprovementRunFreshnessResultV0{
		RetryableRunRefs: append([]string(nil), fake.retryableRunRefs...),
		RetryableRequestRefs: append(
			[]string(nil),
			fake.retryableRequestRefs...,
		),
		EvidenceRefs: append([]string(nil), fake.retryableEvidenceRefs...),
	}, nil
}

func (fake *fakeSupervisorV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request IdleSelfImprovementRequestFilterRequestV0,
) (IdleSelfImprovementRequestFilterResultV0, error) {
	fake.filterCalls++
	fake.lastFilterRequest = request
	if fake.filterErr != nil {
		return IdleSelfImprovementRequestFilterResultV0{}, fake.filterErr
	}
	if fake.filterRequests != nil {
		return IdleSelfImprovementRequestFilterResultV0{Requests: fake.filterRequests}, nil
	}
	return IdleSelfImprovementRequestFilterResultV0{Requests: request.Requests}, nil
}

type fakeSupervisorResultV0 struct {
	result orquestarunsupervisor.RunSupervisorResultV0
	err    error
}

func (fake *fakeSupervisorV0) RunGlobalSupervisorV0(
	_ context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	fake.lastCommand = command
	if len(fake.results) >= fake.calls {
		next := fake.results[fake.calls-1]
		return next.result, next.err
	}
	return orquestarunsupervisor.RunSupervisorResultV0{
		Ticks:           []orquestarunsupervisor.RunSupervisorTickSummaryV0{{TickNumber: 1}},
		TotalExecutions: 1,
		TotalSkips:      2,
		StopReason:      orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

type memoryStateStoreV0 struct{ last StateV0 }

func (store *memoryStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.last = state
	return nil
}
func (store *memoryStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	return store.last, nil
}

type fixedClockV0 struct{ now time.Time }

func (clock fixedClockV0) Now() time.Time { return clock.now }

func markNoExecutionSinceForTestV0(runtime *RuntimeV0, now time.Time) {
	runtime.tracker.MarkSupervisorV0(runtime.config.SupervisorCommand, orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, now.Add(-61*time.Second))
}

func waitRuntimeAsyncWorkForTestV0(t *testing.T, runtime *RuntimeV0) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !runtime.waitAsyncWorkV0(ctx) {
		t.Fatalf("runtime async work sigue en vuelo")
	}
}

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
