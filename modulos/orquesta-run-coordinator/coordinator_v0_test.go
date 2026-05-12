package orquestaruncoordinator

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCoordinateRunsTickChoosesHighestPriorityV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-low", "app", 1),
		candidateV0("run-high", "app", 9),
	})

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-high"})
	assertRunRefsV0(t, rankedRefsV0(result.Ranked), []string{"run-high", "run-low"})
}

func TestCoordinateRunsTickSkipsPausedV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-paused", "app", 9),
		candidateV0("run-next", "app", 5),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-paused"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-paused",
		Status: orquestaruncontrol.RunControlStatusPausedV0,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-next"})
	assertRunRefsV0(t, skipRefsV0(result.Skips), []string{"run-paused"})
	if result.Skips[0].Status != string(orquestaruncontrol.RunControlStatusPausedV0) {
		t.Fatalf("skip status = %q", result.Skips[0].Status)
	}
}

func TestCoordinateRunsTickDrainsForcedStopRequestedV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-stop", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-stop"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-stop",
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: true,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-stop"})
	if len(result.Skips) != 0 {
		t.Fatalf("skips=%+v", result.Skips)
	}
	drainer := deps.Drainer.(*fakeDrainerV0)
	if len(drainer.requests) != 1 || drainer.requests[0].RunRef != "run-stop" {
		t.Fatalf("drain requests=%+v", drainer.requests)
	}
}

func TestCoordinateRunsTickSkipsStopRequestedConCheckpointPendienteV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-stop", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).states["run-stop"] = orquestaruncontrol.RunControlStateV0{
		RunRef: "run-stop",
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: false,
	}

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, skipRefsV0(result.Skips), []string{"run-stop"})
	if len(result.Executions) != 0 {
		t.Fatalf("executions=%+v", result.Executions)
	}
}

func TestCoordinateRunsTickDefaultsRunningWhenControlStateMissingV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-missing", "app", 9),
	})
	deps.ControlReader.(*fakeControlReaderV0).missing["run-missing"] = true

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-missing"})
}

func TestCoordinateRunsTickAllowsMissingControlReaderV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-no-control", "app", 9),
	})
	deps.ControlReader = nil

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-no-control"})
}

func TestCoordinateRunsTickRequiresReaderAndDrainerV0(t *testing.T) {
	_, err := CoordinateRunsTickV0(context.Background(), RunCoordinatorDepsV0{}, tickCommandV0(1))
	if err == nil {
		t.Fatalf("expected queue reader error")
	}
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
	})
	deps.Drainer = nil
	_, err = CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(1))
	if err == nil {
		t.Fatalf("expected drainer error")
	}
}

func TestCoordinateRunsTickExecutesUntilMaxRunsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
		candidateV0("run-c", "app", 7),
	})

	result, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(2))
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-a", "run-b"})
}

func TestCoordinateRunsTickSkipsExcludedRunsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
		candidateV0("run-b", "app", 8),
	})
	command := tickCommandV0(1)
	command.ExcludeRunRefs = []string{" run-a "}

	result, err := CoordinateRunsTickV0(context.Background(), deps, command)
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	assertRunRefsV0(t, executionRefsV0(result.Executions), []string{"run-b"})
	if len(result.Skips) != 1 ||
		result.Skips[0].RunRef != "run-a" ||
		result.Skips[0].Reason != "run_excluded" {
		t.Fatalf("skips=%+v", result.Skips)
	}
}

func TestCoordinateRunsTickPropagatesLimitAndTraceFieldsV0(t *testing.T) {
	deps := coordinatorDepsV0([]orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 9),
	})
	queue := deps.QueueReader.(*fakeQueueReaderV0)
	drainer := deps.Drainer.(*fakeDrainerV0)
	occurredAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	_, err := CoordinateRunsTickV0(context.Background(), deps, RunCoordinatorTickCommandV0{
		QueueRef:      "queue-main",
		AppRefs:       []string{"app"},
		QueueLimit:    25,
		MaxRuns:       1,
		OccurredAt:    occurredAt,
		CorrelationID: "corr-123",
		DrainLimits: RunDrainLimitsV0{
			MaxExternalWaits: 1,
		},
		RankingPolicy: orquestarunqueue.DefaultRunQueueRankingPolicyV0(occurredAt),
	})
	if err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if queue.request.Limit != 25 {
		t.Fatalf("queue limit = %d", queue.request.Limit)
	}
	if len(drainer.requests) != 1 {
		t.Fatalf("drain requests = %d", len(drainer.requests))
	}
	if !drainer.requests[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %s", drainer.requests[0].OccurredAt)
	}
	if drainer.requests[0].CorrelationID != "corr-123" {
		t.Fatalf("correlation_id = %q", drainer.requests[0].CorrelationID)
	}
	if drainer.requests[0].Limits.MaxExternalWaits != 1 {
		t.Fatalf("limits = %+v", drainer.requests[0].Limits)
	}
}

func TestCoordinateRunsTickDoesNotMutateCandidatesV0(t *testing.T) {
	candidates := []orquestarunqueue.RunSchedulingCandidateV0{
		candidateV0("run-a", "app", 1),
		candidateV0("run-b", "app", 9),
	}
	candidates[0].EvidenceRefs = []string{"ev-a"}
	want := cloneCandidatesV0(candidates)
	deps := coordinatorDepsV0(candidates)

	if _, err := CoordinateRunsTickV0(context.Background(), deps, tickCommandV0(2)); err != nil {
		t.Fatalf("coordinate tick: %v", err)
	}

	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates mutated\nwant: %#v\ngot:  %#v", want, candidates)
	}
}

type fakeQueueReaderV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
	request    orquestarunqueue.RunQueueReadRequestV0
}

func (fake *fakeQueueReaderV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.request = request
	return fake.candidates, nil
}

type fakeControlReaderV0 struct {
	states  map[string]orquestaruncontrol.RunControlStateV0
	missing map[string]bool
}

func (fake *fakeControlReaderV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if fake.missing[request.RunRef] {
		return orquestaruncontrol.RunControlStateV0{},
			orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: request.RunRef}
	}
	if state, ok := fake.states[request.RunRef]; ok {
		return state, nil
	}
	return orquestaruncontrol.DefaultRunControlStateV0(request.RunRef), nil
}

type fakeDrainerV0 struct {
	requests []RunDrainRequestV0
}

func (fake *fakeDrainerV0) DrainRunV0(
	_ context.Context,
	request RunDrainRequestV0,
) (RunDrainResultV0, error) {
	if request.RunRef == "" {
		return RunDrainResultV0{}, errors.New("missing run ref")
	}
	fake.requests = append(fake.requests, request)
	return RunDrainResultV0{RunRef: request.RunRef, AppRef: request.AppRef, Outcome: "drained"}, nil
}

func coordinatorDepsV0(candidates []orquestarunqueue.RunSchedulingCandidateV0) RunCoordinatorDepsV0 {
	return RunCoordinatorDepsV0{
		QueueReader: &fakeQueueReaderV0{candidates: candidates},
		ControlReader: &fakeControlReaderV0{
			states:  map[string]orquestaruncontrol.RunControlStateV0{},
			missing: map[string]bool{},
		},
		Drainer: &fakeDrainerV0{},
	}
}

func tickCommandV0(maxRuns int) RunCoordinatorTickCommandV0 {
	now := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	return RunCoordinatorTickCommandV0{
		MaxRuns:       maxRuns,
		OccurredAt:    now,
		CorrelationID: "corr-test",
		RankingPolicy: orquestarunqueue.DefaultRunQueueRankingPolicyV0(now),
	}
}

func candidateV0(runRef string, appRef string, priority int) orquestarunqueue.RunSchedulingCandidateV0 {
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        runRef,
		AppRef:        appRef,
		Status:        "queued",
		PriorityScore: priority,
		UpdatedAt:     time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
	}
}

func cloneCandidatesV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) []orquestarunqueue.RunSchedulingCandidateV0 {
	cloned := append([]orquestarunqueue.RunSchedulingCandidateV0(nil), candidates...)
	for index := range cloned {
		cloned[index].EvidenceRefs = append([]string(nil), cloned[index].EvidenceRefs...)
	}
	return cloned
}
