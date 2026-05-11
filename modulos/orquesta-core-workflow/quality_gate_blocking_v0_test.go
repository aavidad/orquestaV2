package orquestacoreworkflow

import (
	"reflect"
	"testing"
)

func TestPendingBlockingQualityGateRefsForSubjectV0DetectsBlockedPending(t *testing.T) {
	run := OrchestrationRunV0{
		QualityGates: []string{
			qualityGateBlockingProjectionV0("quality-gate-blocked-pending", "task-001", QualityGateDecisionBlockedV0),
		},
	}

	got := PendingBlockingQualityGateRefsForSubjectV0(run, "task-001")
	want := []string{"quality-gate-blocked-pending"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending blockers=%v, want %v", got, want)
	}
	if !QualityGateSubjectHasPendingBlockersV0(run, "task-001") {
		t.Fatalf("subject should have pending blockers")
	}
}

func TestPendingBlockingQualityGateRefsForSubjectV0AcceptedSameSubjectResolves(t *testing.T) {
	run := OrchestrationRunV0{
		QualityGates: []string{
			qualityGateBlockingProjectionV0("quality-gate-rework", "task-001", QualityGateDecisionReworkRequiredV0),
			qualityGateBlockingProjectionV0("quality-gate-accepted", "task-001", QualityGateDecisionAcceptedV0),
		},
	}

	if got := PendingBlockingQualityGateRefsForSubjectV0(run, "task-001"); len(got) != 0 {
		t.Fatalf("pending blockers=%v, want none", got)
	}
}

func TestPendingBlockingQualityGateRefsForSubjectV0AcceptedOtherSubjectDoesNotResolve(t *testing.T) {
	run := OrchestrationRunV0{
		QualityGates: []string{
			qualityGateBlockingProjectionV0("quality-gate-ask-director", "task-001", QualityGateDecisionAskDirectorV0),
			qualityGateBlockingProjectionV0("quality-gate-accepted-other", "task-002", QualityGateDecisionAcceptedV0),
		},
	}

	got := PendingBlockingQualityGateRefsForSubjectV0(run, "task-001")
	want := []string{"quality-gate-ask-director"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending blockers=%v, want %v", got, want)
	}
}

func TestPendingBlockingQualityGateRefsForSubjectV0InvalidProjectionDoesNotPanic(t *testing.T) {
	run := OrchestrationRunV0{
		QualityGates: []string{
			"invalid-quality-gate-projection",
			qualityGateBlockingProjectionV0("quality-gate-valid-blocked", "task-001", QualityGateDecisionBlockedV0),
		},
	}

	got := PendingBlockingQualityGateRefsForSubjectV0(run, "task-001")
	want := []string{"quality-gate-valid-blocked"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending blockers=%v, want %v", got, want)
	}
}

func qualityGateBlockingProjectionV0(gateRef string, subjectRef string, decision QualityGateDecisionV0) string {
	return qualityGateProjectionRefV0(QualityGateRecordedPayloadV0{
		RunRef:     "run-001",
		GateRef:    gateRef,
		PhaseID:    string(OrchestrationPhaseProgramacionV0),
		SubjectRef: subjectRef,
		Decision:   decision,
		Summary:    "quality gate projection for blocking policy",
	})
}
