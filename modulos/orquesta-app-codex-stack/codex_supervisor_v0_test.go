package orquestaappcodexstack

import (
	"context"
	"reflect"
	"testing"
)

func TestCodexSupervisorV0LanzaPrimeroYContinuaHastaDoneV0(t *testing.T) {
	runtime := newFakeCodexSupervisorRuntimeV0("pending", "stopped", "done")

	result, err := SuperviseCodexV0(
		context.Background(),
		CodexSupervisorDepsV0{Runtime: runtime},
		CodexSupervisorCommandV0{
			MaxTicks:        5,
			ContinueMessage: "sigue",
		},
	)
	if err != nil {
		t.Fatalf("SuperviseCodexV0: %v", err)
	}

	wantCalls := []string{"launch", "continue:sigue", "continue:sigue"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("calls got %#v want %#v", runtime.calls, wantCalls)
	}
	if result.StopReason != CodexSupervisorStopDoneV0 {
		t.Fatalf("stop_reason=%q result=%+v", result.StopReason, result)
	}
	if result.Ticks != 3 {
		t.Fatalf("ticks=%d result=%+v", result.Ticks, result)
	}
}

func TestCodexSupervisorV0CortaPorMaxTicksSinDoneV0(t *testing.T) {
	runtime := newFakeCodexSupervisorRuntimeV0("pending", "stopped", "stopped", "stopped")

	result, err := SuperviseCodexV0(
		context.Background(),
		CodexSupervisorDepsV0{Runtime: runtime},
		CodexSupervisorCommandV0{
			MaxTicks:        3,
			ContinueMessage: "sigue",
		},
	)
	if err != nil {
		t.Fatalf("SuperviseCodexV0: %v", err)
	}

	wantCalls := []string{"launch", "continue:sigue", "continue:sigue"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("calls got %#v want %#v", runtime.calls, wantCalls)
	}
	if result.StopReason != CodexSupervisorStopMaxTicksV0 {
		t.Fatalf("stop_reason=%q result=%+v", result.StopReason, result)
	}
	if result.Ticks != 3 {
		t.Fatalf("ticks=%d result=%+v", result.Ticks, result)
	}
}

func TestCodexSupervisorV0UsaSiguePorDefectoV0(t *testing.T) {
	runtime := newFakeCodexSupervisorRuntimeV0("pending", "done")

	result, err := SuperviseCodexV0(
		context.Background(),
		CodexSupervisorDepsV0{Runtime: runtime},
		CodexSupervisorCommandV0{MaxTicks: 2},
	)
	if err != nil {
		t.Fatalf("SuperviseCodexV0: %v", err)
	}

	wantCalls := []string{"launch", "continue:sigue"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("calls got %#v want %#v", runtime.calls, wantCalls)
	}
	if result.StopReason != CodexSupervisorStopDoneV0 {
		t.Fatalf("stop_reason=%q result=%+v", result.StopReason, result)
	}
}

type fakeCodexSupervisorRuntimeV0 struct {
	states []CodexSupervisorRuntimeStateV0
	calls  []string
	next   int
}

func newFakeCodexSupervisorRuntimeV0(states ...string) *fakeCodexSupervisorRuntimeV0 {
	runtime := &fakeCodexSupervisorRuntimeV0{
		states: make([]CodexSupervisorRuntimeStateV0, 0, len(states)),
	}
	for _, state := range states {
		runtime.states = append(runtime.states, CodexSupervisorRuntimeStateV0(state))
	}
	return runtime
}

func (runtime *fakeCodexSupervisorRuntimeV0) LaunchV0(
	context.Context,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	runtime.calls = append(runtime.calls, "launch")
	return runtime.consumeV0(), nil
}

func (runtime *fakeCodexSupervisorRuntimeV0) ContinueV0(
	_ context.Context,
	message string,
) (CodexSupervisorRuntimeSnapshotV0, error) {
	runtime.calls = append(runtime.calls, "continue:"+message)
	return runtime.consumeV0(), nil
}

func (runtime *fakeCodexSupervisorRuntimeV0) consumeV0() CodexSupervisorRuntimeSnapshotV0 {
	if runtime.next >= len(runtime.states) {
		return CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0}
	}
	state := runtime.states[runtime.next]
	runtime.next++
	return CodexSupervisorRuntimeSnapshotV0{Status: state}
}
