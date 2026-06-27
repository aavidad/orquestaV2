package orquestaruntime

import (
	"path/filepath"
	"strings"
)

func processRuntimeCommandIsShellV0(commandPath string) bool {
	base := strings.ToLower(filepath.Base(commandPath))
	switch base {
	case "sh", "bash", "dash", "zsh", "fish",
		"cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func processRuntimeEnvEntryAllowedV0(item string) bool {
	key, value, ok := strings.Cut(item, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return false
	}
	key = strings.TrimSpace(key)
	if !processRuntimeEnvKeyAllowedV0(key) {
		return false
	}
	if strings.ToUpper(key) == "PATH" {
		return processRuntimePathEnvValueAllowedV0(value)
	}
	if processRuntimeEnvKeyForbiddenV0(key) {
		return false
	}
	if strings.ContainsAny(value, `/\`) {
		return false
	}
	return !processRuntimeUnsafeValueV0(value)
}

func processRuntimeEnvKeyAllowedV0(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	return upper == "PATH" ||
		upper == "LANG" ||
		upper == "TZ" ||
		strings.HasPrefix(upper, "LC_") ||
		strings.HasPrefix(upper, "ORQUESTA_")
}

func processRuntimeEnvKeyForbiddenV0(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range []string{
		"HOME",
		"PATH",
		"OAUTH",
		"TOKEN",
		"SECRET",
		"API_KEY",
		"CREDENTIAL",
		"PASSWORD",
	} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func processRuntimePathEnvValueAllowedV0(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
		return false
	}
	for _, dir := range filepath.SplitList(value) {
		if strings.TrimSpace(dir) == "" {
			return false
		}
		if !filepath.IsAbs(dir) {
			return false
		}
		if looksLikeSecret(dir) || processRuntimeContainsCredentialMarkerV0(dir) {
			return false
		}
	}
	return true
}
