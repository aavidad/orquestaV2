package orquestadirectortickinput

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildDirectorSchedulerTickInputV0ConstruyeSnapshotCanonico(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	run.CapacityRequests = []string{"capacity-ref-001"}
	run.CapacityDecisions = []string{"capacity-ref-001#capacity_decision:decision-ref-001"}
	run.ConcurrencyGates = []string{"gate-ref-001#decision:allow_request_agent#plan:plan-ref-001"}
	run.Deliveries = []string{"delivery-ref-001"}
	run.Reviews = []string{"review-request-ref-001"}
	run.ReviewResults = []string{
		"review-result-ref-001#review_result:changes_requested#review_request:review-request-ref-001#delivery:delivery-ref-001",
	}
	run.ReworkRequests = []string{
		"rework-request-ref-001#review_result:review-result-ref-001#review_request:review-request-ref-001#delivery:delivery-ref-001",
	}
	request := DirectorTickInputBuildRequestV0{
		TickRef:           "tick-ref-001",
		OccurredAt:        "2026-05-06T12:00:00Z",
		Run:               run,
		PendingOutboxRefs: []string{"outbox-ref-001", " outbox-ref-001 "},
		EvidenceRefs:      []string{"evidence-ref-001"},
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	if input.RunRef != run.RunID || input.Snapshot.CurrentPhaseID != string(run.CurrentPhase) {
		t.Fatalf("unexpected snapshot: %+v", input.Snapshot)
	}
	if !reflect.DeepEqual(input.Snapshot.CapacityDecisions, []string{"capacity-ref-001"}) {
		t.Fatalf("capacity decisions=%v", input.Snapshot.CapacityDecisions)
	}
	if !reflect.DeepEqual(input.Snapshot.ConcurrencyGates, []string{"gate-ref-001"}) {
		t.Fatalf("concurrency gates=%v", input.Snapshot.ConcurrencyGates)
	}
	if !reflect.DeepEqual(input.Snapshot.ReworkRequests, run.ReworkRequests) {
		t.Fatalf("rework requests=%v", input.Snapshot.ReworkRequests)
	}
	if !reflect.DeepEqual(input.Snapshot.PendingOutboxRefs, []string{"outbox-ref-001"}) {
		t.Fatalf("pending outbox=%v", input.Snapshot.PendingOutboxRefs)
	}
}

func TestBuildDirectorSchedulerTickInputV0DerivaQualityGatesBloqueantesPendientes(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	run.QualityGates = []string{
		"quality-gate-ref-blocked-cleared#decision:blocked#subject:subject-ref-shared",
		"quality-gate-ref-blocked-pending#decision:blocked#subject:subject-ref-pending",
		"quality-gate-ref-accepted#decision:accepted#subject:subject-ref-shared",
	}
	request := DirectorTickInputBuildRequestV0{
		TickRef:    "tick-ref-quality-gate-001",
		OccurredAt: "2026-05-06T12:00:00Z",
		Run:        run,
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	want := []string{"quality-gate-ref-blocked-pending"}
	if !reflect.DeepEqual(input.Snapshot.BlockingQualityGateRefs, want) {
		t.Fatalf("blocking quality gates=%v, want %v", input.Snapshot.BlockingQualityGateRefs, want)
	}
}

func TestBuildDirectorSchedulerTickInputV0RejectsIncompleto(t *testing.T) {
	_, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var publicErr DirectorTickInputBuildErrorV0
	if !errors.As(err, &publicErr) || publicErr.Code != ErrDirectorTickInputBuildInvalidoV0 {
		t.Fatalf("unexpected error: %v", err)
	}
}
