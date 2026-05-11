package orquestaruncontrol

import "testing"

func TestEvaluateRunControlAllowsRunningV0(t *testing.T) {
	got := EvaluateRunControlV0(RunControlStateV0{Status: RunControlStatusRunningV0})
	want := RunControlEvaluationV0{
		SchedulingAllowed: true,
		DispatchAllowed:   true,
	}
	assertRunControlEvaluationV0(t, got, want)
}

func TestEvaluateRunControlBlocksPausedWithoutTerminalV0(t *testing.T) {
	got := EvaluateRunControlV0(RunControlStateV0{Status: RunControlStatusPausedV0})
	assertRunControlEvaluationV0(t, got, RunControlEvaluationV0{})
}

func TestEvaluateRunControlRequiresCheckpointBeforeStopAgentsV0(t *testing.T) {
	statuses := []RunControlStatusV0{
		RunControlStatusStopRequestedV0,
		RunControlStatusCancelRequestedV0,
	}
	for _, status := range statuses {
		got := EvaluateRunControlV0(RunControlStateV0{Status: status})
		want := RunControlEvaluationV0{CheckpointRequired: true}
		assertRunControlEvaluationV0(t, got, want)
	}
}

func TestEvaluateRunControlAllowsStopAgentsAfterCheckpointV0(t *testing.T) {
	statuses := []RunControlStatusV0{
		RunControlStatusStopRequestedV0,
		RunControlStatusCancelRequestedV0,
	}
	for _, status := range statuses {
		got := EvaluateRunControlV0(RunControlStateV0{
			Status:             status,
			CheckpointRecorded: true,
		})
		want := RunControlEvaluationV0{StopAgentsAllowed: true}
		assertRunControlEvaluationV0(t, got, want)
	}
}

func TestEvaluateRunControlForcedBypassesCheckpointV0(t *testing.T) {
	statuses := []RunControlStatusV0{
		RunControlStatusStopRequestedV0,
		RunControlStatusCancelRequestedV0,
	}
	for _, status := range statuses {
		got := EvaluateRunControlV0(RunControlStateV0{
			Status: status,
			Forced: true,
		})
		want := RunControlEvaluationV0{StopAgentsAllowed: true}
		assertRunControlEvaluationV0(t, got, want)
	}
}

func TestEvaluateRunControlMarksTerminalStatusesV0(t *testing.T) {
	statuses := []RunControlStatusV0{
		RunControlStatusStoppedV0,
		RunControlStatusCanceledV0,
	}
	for _, status := range statuses {
		got := EvaluateRunControlV0(RunControlStateV0{Status: status})
		want := RunControlEvaluationV0{Terminal: true}
		assertRunControlEvaluationV0(t, got, want)
	}
}

func TestEvaluateRunControlNormalizesStatusV0(t *testing.T) {
	got := EvaluateRunControlV0(RunControlStateV0{Status: " RUNNING "})
	want := RunControlEvaluationV0{
		SchedulingAllowed: true,
		DispatchAllowed:   true,
	}
	assertRunControlEvaluationV0(t, got, want)
}

func TestIsTerminalRunControlStatusV0(t *testing.T) {
	if !IsTerminalRunControlStatusV0(" CANCELED ") {
		t.Fatalf("expected normalized canceled status to be terminal")
	}
	if IsTerminalRunControlStatusV0(RunControlStatusPausedV0) {
		t.Fatalf("paused must not be terminal")
	}
}

func assertRunControlEvaluationV0(t *testing.T, got RunControlEvaluationV0, want RunControlEvaluationV0) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected evaluation: got %#v want %#v", got, want)
	}
}
