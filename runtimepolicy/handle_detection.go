package runtimepolicy

import (
	"orquesta/internal/controlruntime"
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

func RuntimeHandleUsaTMUXPreferredCLI(meta map[string]any) bool {
	if RuntimeHandleUsaLegacyCLITMUXPreferred(meta) {
		return true
	}
	if meta == nil {
		return false
	}
	for _, candidate := range []string{
		mapValueString(meta, "rendered_command", ""),
		mapValueString(meta, "wrapped_command", ""),
		mapValueString(meta, "herramienta", ""),
		mapValueString(meta, "conector", ""),
		mapValueString(meta, "profile_status_wrapper", ""),
	} {
		if RuntimeOrderUsaCLITMUXPreferred(candidate) {
			return true
		}
	}
	return false
}

func RuntimeOrderUsaCLITMUXPreferred(refs ...string) bool {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(ref) {
			return true
		}
		lower := strings.ToLower(ref)
		if strings.Contains(lower, "codex") || strings.Contains(lower, "claude") || strings.Contains(lower, "gemini") {
			return true
		}
	}
	return false
}

func RuntimeHandleEsTMUXCanonico(meta map[string]any, transporte string) bool {
	if strings.EqualFold(strings.TrimSpace(transporte), "tmux") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(mapValueString(meta, "driver", "")), "tmux_cli_session") {
		return true
	}
	return strings.TrimSpace(mapValueString(meta, "tmux_session", "")) != "" ||
		strings.TrimSpace(mapValueString(meta, "tmux_pane_id", "")) != ""
}

func RuntimeHandleTMUXSessionRef(meta map[string]any, transporte, handleKind, handleRef string) string {
	session := strings.TrimSpace(mapValueString(meta, "tmux_session", ""))
	pane := strings.TrimSpace(mapValueString(meta, "tmux_pane_id", ""))
	switch {
	case session != "" && pane != "":
		return session + "/" + pane
	case session != "":
		return session
	case pane != "":
		return pane
	case strings.EqualFold(strings.TrimSpace(transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(handleKind), "session"):
		return strings.TrimSpace(handleRef)
	default:
		return ""
	}
}

func RuntimeHandleUsaLegacyCLITMUXPreferred(meta map[string]any) bool {
	if !RuntimeHandleUsaLegacyProcessPTY(meta) {
		return false
	}
	for _, candidate := range []string{
		mapValueString(meta, "rendered_command", ""),
		mapValueString(meta, "wrapped_command", ""),
		mapValueString(meta, "herramienta", ""),
		mapValueString(meta, "conector", ""),
		mapValueString(meta, "profile_status_wrapper", ""),
	} {
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(candidate) {
			return true
		}
	}
	return false
}

func RuntimeHandleUsaLegacyProcessPTY(meta map[string]any) bool {
	return strings.EqualFold(mapValueString(meta, "driver", ""), "process_pty_cli")
}
