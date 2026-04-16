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
		name    string
		estado  string
		want    bool
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
