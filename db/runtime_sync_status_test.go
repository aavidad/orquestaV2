package db

import "testing"

func TestErrorObservacionLocalIgnorable(t *testing.T) {
	casosTrue := []error{
		nil,
	}
	for _, err := range casosTrue {
		if err != nil {
			continue
		}
		if errorObservacionLocalIgnorable(err) {
			t.Fatalf("nil no debe ser ignorable")
		}
	}

	ignorables := []error{
		errString("tmux kill-session: can't find session: orq-codex1-141251"),
		errString("tmux kill-session: no server running on /tmp/tmux-1000/default"),
		errString("tmux kill-session: error connecting to /tmp/tmux-1000/default (No such file or directory)"),
		errString("tmux session missing"),
		errString("failed to connect to server"),
	}
	for _, err := range ignorables {
		if !errorObservacionLocalIgnorable(err) {
			t.Fatalf("deberia ignorar err=%v", err)
		}
	}

	if errorObservacionLocalIgnorable(errString("permiso denegado")) {
		t.Fatal("no deberia ignorar errores no relacionados con tmux")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
