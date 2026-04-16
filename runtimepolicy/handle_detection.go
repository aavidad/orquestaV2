package runtimepolicy

import (
	"path/filepath"
	"strings"
)

func RuntimeHandleLooksLikeCodexCLIRef(ref string) bool {
	lower := strings.ToLower(strings.TrimSpace(ref))
	if lower == "" {
		return false
	}
	switch lower {
	case "codex", "codex-cli":
		return true
	}
	base := filepath.Base(strings.Trim(lower, "'\""))
	return strings.HasPrefix(base, "codex-perfil")
}

func RuntimeHandleLooksLikeCodexCommand(rendered string) bool {
	lower := strings.ToLower(strings.TrimSpace(rendered))
	if lower == "" {
		return false
	}
	if lower == "codex" || strings.HasPrefix(lower, "codex ") || strings.Contains(lower, "codex-perfil") {
		return true
	}
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = fields[0]
	}
	first = strings.Trim(first, "'\"")
	base := filepath.Base(first)
	return base == "codex" || base == "codex-cli" || strings.HasPrefix(base, "codex-perfil")
}
