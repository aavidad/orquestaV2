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
	if path := lookPathInPathListV0(raw, os.Getenv(envCodexPathV0)); path != "" {
		return path
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		return raw
	}
	return path
}

func lookPathInPathListV0(command string, pathList string) string {
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsRune(command, os.PathSeparator) {
		return ""
	}
	for _, dir := range filepath.SplitList(pathList) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, command)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate
	}
	return ""
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
	if !filepath.IsAbs(codexCommandPathV0()) {
		return fmt.Errorf("codex_command_unavailable")
	}
	return nil
}
