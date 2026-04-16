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
