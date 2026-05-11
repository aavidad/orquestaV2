package orquestaruncontrol

import (
	"errors"
	"testing"
)

func TestRunControlStateNotFoundErrorV0EsTipado(t *testing.T) {
	err := RunControlStateNotFoundErrorV0{RunRef: " run-1 "}
	var typed RunControlStateNotFoundErrorV0
	if !errors.As(err, &typed) ||
		typed.RunRef != " run-1 " ||
		err.Error() != "run_control_state_not_found: run-1" {
		t.Fatalf("err=%v typed=%+v", err, typed)
	}
}

func TestDefaultRunControlStateV0EsRunning(t *testing.T) {
	state := DefaultRunControlStateV0(" run-1 ")
	if state.RunRef != "run-1" || state.Status != RunControlStatusRunningV0 {
		t.Fatalf("state=%+v", state)
	}
}
