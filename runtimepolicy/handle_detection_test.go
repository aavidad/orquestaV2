package runtimepolicy

import "testing"

func TestRuntimeHandleLooksLikeCodexCLIRef(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"vacio", "", false},
		{"whitespace", "   ", false},
		{"codex", "codex", true},
		{"codex-cli", "codex-cli", true},
		{"codex-perfil", "codex-perfil", true},
		{"codex-perfil quote", " '/home/x/bin/codex-perfil' ", true},
		{"no codex", "/usr/bin/other", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleLooksLikeCodexCLIRef(tc.raw); got != tc.want {
				t.Fatalf("RuntimeHandleLooksLikeCodexCLIRef(%q) = %v; want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleLooksLikeCodexCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"vacio", "", false},
		{"whitespace", "   ", false},
		{"codex exacto", "codex", true},
		{"codex con espacio", "codex --help", true},
		{"codex-perfil", "codex-perfil Codex7", true},
		{"codex-perfil envuelto", "'/home/x/bin/codex-perfil' Codex7", true},
		{"codex-cli", "codex-cli", true},
		{"sin codex", "/usr/bin/other", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleLooksLikeCodexCommand(tc.raw); got != tc.want {
				t.Fatalf("RuntimeHandleLooksLikeCodexCommand(%q) = %v; want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleUsaTMUXPreferredCLI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{"nil", nil, false},
		{"legacy process_pty", map[string]any{"driver": "process_pty_cli", "rendered_command": "codex-perfil Codex1"}, true},
		{"rendered command tmux", map[string]any{"rendered_command": "cat"}, false},
		{"rendered command codex", map[string]any{"rendered_command": "codex --help"}, true},
		{"herramienta claude", map[string]any{"herramienta": "claude-code"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleUsaTMUXPreferredCLI(tc.meta); got != tc.want {
				t.Fatalf("RuntimeHandleUsaTMUXPreferredCLI(%v) = %v; want %v", tc.meta, got, tc.want)
			}
		})
	}
}

func TestRuntimeOrderUsaCLITMUXPreferred(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		refs []string
		want bool
	}{
		{"vacio", nil, false},
		{"sin datos", []string{"", "  "}, false},
		{"codex-cli", []string{"codex-cli"}, true},
		{"claude", []string{"claude-code"}, true},
		{"gemini", []string{"  gemini --help"}, true},
		{"ollama run", []string{"ollama run llama3"}, true},
		{"no tmux", []string{"bash"}, false},
		{"codex en segunda ref", []string{"bash", "codex"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeOrderUsaCLITMUXPreferred(tc.refs...); got != tc.want {
				t.Fatalf("RuntimeOrderUsaCLITMUXPreferred(%v) = %v; want %v", tc.refs, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleEsTMUXCanonico(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		meta       map[string]any
		transporte string
		want       bool
	}{
		{"nil transport", nil, "", false},
		{"transport tmux", nil, "tmux", true},
		{"driver tmux_cli_session", map[string]any{"driver": "tmux_cli_session"}, "", true},
		{"tmux_session", map[string]any{"tmux_session": "s1"}, "", true},
		{"tmux_pane_id", map[string]any{"tmux_pane_id": "p1"}, "", true},
		{"sin match", map[string]any{"driver": "process_pty_cli"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleEsTMUXCanonico(tc.meta, tc.transporte); got != tc.want {
				t.Fatalf("RuntimeHandleEsTMUXCanonico(%v, %q) = %v; want %v", tc.meta, tc.transporte, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleTMUXSessionRef(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		meta       map[string]any
		transporte string
		handleKind string
		handleRef  string
		want       string
	}{
		{"ambos campos", map[string]any{"tmux_session": "sess", "tmux_pane_id": "pane"}, "", "", "", "sess/pane"},
		{"solo session", map[string]any{"tmux_session": "sess"}, "", "", "", "sess"},
		{"solo pane", map[string]any{"tmux_pane_id": "pane"}, "", "", "", "pane"},
		{"canonico tmux session", map[string]any{}, "tmux", "process", "  /tmp/ref ", "/tmp/ref"},
		{"canonico session kind", map[string]any{}, "", "session", "ref-session", "ref-session"},
		{"ninguno", map[string]any{}, "", "", "process", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleTMUXSessionRef(tc.meta, tc.transporte, tc.handleKind, tc.handleRef); got != tc.want {
				t.Fatalf("RuntimeHandleTMUXSessionRef(%v, %q, %q, %q) = %q; want %q", tc.meta, tc.transporte, tc.handleKind, tc.handleRef, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleUsaLegacyCLITMUXPreferred(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{"nil", nil, false},
		{"missing driver", map[string]any{"driver": "process_pty_cli"}, false},
		{"codex in rendered command", map[string]any{"driver": "process_pty_cli", "rendered_command": "codex"}, true},
		{"claude in herramienta", map[string]any{"driver": "process_pty_cli", "herramienta": "claude"}, true},
		{"other driver", map[string]any{"driver": "bash", "rendered_command": "codex"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleUsaLegacyCLITMUXPreferred(tc.meta); got != tc.want {
				t.Fatalf("RuntimeHandleUsaLegacyCLITMUXPreferred(%v) = %v; want %v", tc.meta, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleUsaLegacyProcessPTY(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{"nil", nil, false},
		{"legacy driver", map[string]any{"driver": "process_pty_cli"}, true},
		{"other driver", map[string]any{"driver": "tmux_cli_session"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleUsaLegacyProcessPTY(tc.meta); got != tc.want {
				t.Fatalf("RuntimeHandleUsaLegacyProcessPTY(%v) = %v; want %v", tc.meta, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleIsLegacyControlPlane(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		estado     string
		transporte string
		handleKind string
		meta       map[string]any
		want       bool
	}{
		{"activo cli process process_pty_cli", "activo", "cli", "process", map[string]any{"driver": "process_pty_cli"}, true},
		{"pausado cli process process_pty_cli", "pausado", "cli", "process", map[string]any{"driver": "process_pty_cli"}, true},
		{"activo cli process otro driver", "activo", "cli", "process", map[string]any{"driver": "tmux_cli_session"}, false},
		{"activo tmux process process_pty_cli", "activo", "tmux", "process", map[string]any{"driver": "process_pty_cli"}, false},
		{"activo cli session process_pty_cli", "activo", "cli", "session", map[string]any{"driver": "process_pty_cli"}, false},
		{"caido cli process process_pty_cli", "cerrado", "cli", "process", map[string]any{"driver": "process_pty_cli"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleIsLegacyControlPlane(tc.estado, tc.transporte, tc.handleKind, tc.meta); got != tc.want {
				t.Fatalf("RuntimeHandleIsLegacyControlPlane(%q, %q, %q, %v) = %v; want %v", tc.estado, tc.transporte, tc.handleKind, tc.meta, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleNeedsFreshSync(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		estado string
		want   bool
	}{
		{"activo", "activo", true},
		{"running", "running", true},
		{"ready", "ready", true},
		{"starting", "starting", true},
		{"paused", "paused", true},
		{"fallido", "fallido", true},
		{"degradado", "degradado", true},
		{"fallido con espacios", "  fallido  ", true},
		{"cerrado", "cerrado", false},
		{"vacio", "", false},
		{"blank", "   ", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleNeedsFreshSync(tc.estado); got != tc.want {
				t.Fatalf("RuntimeHandleNeedsFreshSync(%q) = %v; want %v", tc.estado, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleTMUXSessionMissing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		metadataJSON string
		want         bool
	}{
		{"sin metadata", "", false},
		{"metadata sin session", `{"worker":"x"}`, false},
		{"metadata tmux incompleta", `{"tmux_session":"", "tmux_pane_id":""}`, false},
		{"metadata session missing", `{"tmux_session":"", "tmux_pane_id":"", "child_pid":"123"}`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleTMUXSessionMissing(tc.metadataJSON); got != tc.want {
				t.Fatalf("RuntimeHandleTMUXSessionMissing(%q) = %v; want %v", tc.metadataJSON, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleMatchesRuntime(t *testing.T) {
	t.Parallel()

	runtimeID := int64(15)
	sesionID := int64(55)
	otroSesion := int64(99)
	cases := []struct {
		name            string
		runtimeID       int64
		runtimeSesionID *int64
		handleRuntimeID *int64
		handleSesionID  *int64
		want            bool
	}{
		{"match por runtime id", runtimeID, nil, &runtimeID, nil, true},
		{"match por sesion", runtimeID, &sesionID, nil, &sesionID, true},
		{"match por runtime id y sesiones distintas", runtimeID, &sesionID, &runtimeID, &otroSesion, true},
		{"sin coincidencia", runtimeID, &sesionID, ptrInt64(runtimeID + 1), &otroSesion, false},
		{"runtime id sin sesiones nula", runtimeID, nil, ptrInt64(runtimeID + 1), &sesionID, false},
		{"sin runtime id", 0, nil, &runtimeID, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleMatchesRuntime(tc.runtimeID, tc.runtimeSesionID, tc.handleRuntimeID, tc.handleSesionID); got != tc.want {
				t.Fatalf("RuntimeHandleMatchesRuntime(%d, %+v, %+v, %+v) = %v; want %v", tc.runtimeID, tc.runtimeSesionID, tc.handleRuntimeID, tc.handleSesionID, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		metadataJSON string
		want         string
	}{
		{"empty", "", ""},
		{"invalid json", "{bad", ""},
		{"driver en metadata", `{"driver":"tmux_cli_session"}`, "tmux_cli_session"},
		{"driver con espacios", `{"driver":"  process_pty_cli  "}`, "process_pty_cli"},
		{"sin driver", `{"worker":"x"}`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleDriver(tc.metadataJSON); got != tc.want {
				t.Fatalf("RuntimeHandleDriver(%q) = %q; want %q", tc.metadataJSON, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleEstadoScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		estado string
		want   int
	}{
		{"activo", "activo", 100},
		{"pausado", "pausado", 80},
		{"degradado", "degradado", 30},
		{"fallido", "fallido", -100},
		{"cerrado", "cerrado", -100},
		{"otro", "otro", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleEstadoScore(tc.estado); got != tc.want {
				t.Fatalf("RuntimeHandleEstadoScore(%q) = %d; want %d", tc.estado, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleDriverScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		driver string
		want   int
	}{
		{"tmux", "tmux_cli_session", 300},
		{"process", "process_pty_cli", -150},
		{"otro", "bash", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleDriverScore(tc.driver); got != tc.want {
				t.Fatalf("RuntimeHandleDriverScore(%q) = %d; want %d", tc.driver, got, tc.want)
			}
		})
	}
}

func TestRuntimeHandleObservedWorkerSnapshotScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		driver string
		alive  bool
		state  string
		want   int
	}{
		{"tmux alive running", "tmux_cli_session", true, "running", 240},
		{"tmux dead", "tmux_cli_session", false, "running", 180},
		{"other alive running", "process_pty_cli", true, "running", 40},
		{"other alive stopped", "process_pty_cli", true, "stopped", 0},
		{"other dead", "process_pty_cli", false, "stopped", -20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeHandleObservedWorkerSnapshotScore(tc.driver, tc.alive, tc.state); got != tc.want {
				t.Fatalf("RuntimeHandleObservedWorkerSnapshotScore(%q, %v, %q) = %d; want %d", tc.driver, tc.alive, tc.state, got, tc.want)
			}
		})
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
