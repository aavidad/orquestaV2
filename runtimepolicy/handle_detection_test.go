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
