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
	if !filepath.IsAbs(raw) {
		return ""
	}
	return raw
}

func codeHomeDirV0() string {
	value := strings.TrimSpace(os.Getenv(envCodexCodeHomeV0))
	if value != "" {
		return value
	}
	value = strings.TrimSpace(os.Getenv(envCodexHomeV0))
	if value != "" {
		return value
	}
	value = strings.TrimSpace(os.Getenv(envCodexCodeHomeLegacyV0))
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
	command := codexCommandPathV0()
	if !filepath.IsAbs(command) {
		return fmt.Errorf("codex_command_unavailable")
	}
	version, err := exec.Command(command, "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) == "" {
		return fmt.Errorf("codex_command_version_unavailable")
	}
	return nil
}
