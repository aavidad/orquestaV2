package orquestagoal

import "testing"

func TestGoalObservationUnchangedV0ComparaFingerprintNormalizado(t *testing.T) {
	prev := GoalObservationFingerprintV0{
		RunRef:       " run-ref-goal-fingerprint-001 ",
		GoalRef:      " goal-ref-goal-fingerprint-001 ",
		AckFilesHash: " ack-hash-001 ",
		ProcessAlive: true,
		LastStatus:   " running ",
		EvidenceHash: " evidence-hash-001 ",
	}
	current := GoalObservationFingerprintV0{
		RunRef:       "run-ref-goal-fingerprint-001",
		GoalRef:      "goal-ref-goal-fingerprint-001",
		AckFilesHash: "ack-hash-001",
		ProcessAlive: true,
		LastStatus:   "running",
		EvidenceHash: "evidence-hash-001",
	}

	if !GoalObservationUnchangedV0(prev, current) {
		t.Fatalf("fingerprint equivalente debe ser unchanged")
	}
}

func TestGoalObservationUnchangedV0DetectaCambiosOperativos(t *testing.T) {
	base := GoalObservationFingerprintV0{
		RunRef:       "run-ref-goal-fingerprint-002",
		GoalRef:      "goal-ref-goal-fingerprint-002",
		AckFilesHash: "ack-hash-001",
		ProcessAlive: true,
		LastStatus:   GoalStatusRunningV0,
		EvidenceHash: "evidence-hash-001",
	}
	for _, tc := range []struct {
		name    string
		current GoalObservationFingerprintV0
	}{
		{name: "ack", current: GoalObservationFingerprintV0{RunRef: base.RunRef, GoalRef: base.GoalRef, AckFilesHash: "ack-hash-002", ProcessAlive: true, LastStatus: base.LastStatus, EvidenceHash: base.EvidenceHash}},
		{name: "process", current: GoalObservationFingerprintV0{RunRef: base.RunRef, GoalRef: base.GoalRef, AckFilesHash: base.AckFilesHash, ProcessAlive: false, LastStatus: base.LastStatus, EvidenceHash: base.EvidenceHash}},
		{name: "status", current: GoalObservationFingerprintV0{RunRef: base.RunRef, GoalRef: base.GoalRef, AckFilesHash: base.AckFilesHash, ProcessAlive: true, LastStatus: GoalStatusCompleteV0, EvidenceHash: base.EvidenceHash}},
		{name: "evidence", current: GoalObservationFingerprintV0{RunRef: base.RunRef, GoalRef: base.GoalRef, AckFilesHash: base.AckFilesHash, ProcessAlive: true, LastStatus: base.LastStatus, EvidenceHash: "evidence-hash-002"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if GoalObservationUnchangedV0(base, tc.current) {
				t.Fatalf("fingerprint cambiado no debe saltarse: %+v", tc.current)
			}
		})
	}
}
