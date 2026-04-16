package runtimepolicy

import "testing"

func TestRuntimeHandleDeliveryContextScore(t *testing.T) {
	t.Parallel()

	meta := `{"rendered_command":"codex-perfil x","driver":"tmux_cli_session","working_dir":"/tmp"}`
	caps := `{"mailbox_delivery_mode":"interactive"}`
	got := RuntimeHandleDeliveryContextScore(meta, caps)
	if got != 4 {
		t.Fatalf("unexpected delivery context score: %d", got)
	}
}

func TestRuntimeHandleDeliveryContextScoreFallbacks(t *testing.T) {
	t.Parallel()

	if got := RuntimeHandleDeliveryContextScore("", ""); got != 0 {
		t.Fatalf("unexpected score for empty metadata/caps: %d", got)
	}
	if got := RuntimeHandleDeliveryContextScore("not-json", "not-json"); got != 0 {
		t.Fatalf("unexpected score for invalid json: %d", got)
	}
}

func TestRuntimeHandleHasDeliveryContext(t *testing.T) {
	t.Parallel()

	meta := `{"driver":"tmux_cli_session"}`
	if !RuntimeHandleHasDeliveryContext(meta, "{}") {
		t.Fatal("expected delivery context to be detected")
	}
	if RuntimeHandleHasDeliveryContext("{}", "{}") {
		t.Fatal("expected empty context to be false")
	}
}
