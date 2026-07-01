package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func codexCommandPathV0() string {
	raw := strings.TrimSpace(os.Getenv(envCodexCommandV0))
	if raw == "" {
		raw = "codex"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		return raw
	}
	return path
}

func codeHomeDirV0() string {
	value := strings.TrimSpace(os.Getenv(envCodexCodeHomeV0))
	if value != "" {
		return value
	}
	value = strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if value != "" {
		return value
	}
	return filepath.Join(homeDirV0(), ".codex")
}

func homeDirV0() string {
	value := strings.TrimSpace(os.Getenv(envCodexHomeV0))
	if value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func validateCodexCommandAvailableV0() error {
	if !filepath.IsAbs(codexCommandPathV0()) {
		return fmt.Errorf("codex_command_unavailable")
	}
	return nil
}
