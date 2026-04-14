/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package coordinacion

import (
	"os"
	"path/filepath"
	"strings"
)

type WorktreePathRef struct {
	Agent string
	Path  string
}

func CurrentPathInsideProjectWorktree(currentCWD, effectiveProjectPath string) bool {
	currentCWD = normalizeRouteSelectorPath(currentCWD)
	effectiveProjectPath = normalizeRouteSelectorPath(effectiveProjectPath)
	if currentCWD == "" || effectiveProjectPath == "" {
		return false
	}
	return routeSelectorPathWithin(filepath.Join(effectiveProjectPath, ".orquesta-worktrees"), currentCWD)
}

func WorktreePathUsable(path string) bool {
	path = normalizeRouteSelectorPath(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info != nil && info.IsDir()
}

func PathWithin(base, path string) bool {
	return routeSelectorPathWithin(base, path)
}

func SessionPathInsideActiveWorktree(agent, cwd string, worktrees []WorktreePathRef) bool {
	cwd = normalizeRouteSelectorPath(cwd)
	if cwd == "" {
		return false
	}
	agent = strings.TrimSpace(agent)
	for _, worktree := range worktrees {
		if agent != "" && strings.TrimSpace(worktree.Agent) != agent {
			continue
		}
		if !WorktreePathUsable(worktree.Path) {
			continue
		}
		if PathWithin(worktree.Path, cwd) {
			return true
		}
	}
	return false
}

func SelectUsableActiveWorktreePath(worktrees []WorktreePathRef, effectiveProjectPath string) string {
	for _, worktree := range worktrees {
		path := normalizeRouteSelectorPath(worktree.Path)
		if path == "" {
			continue
		}
		if effectiveProjectPath != "" && !ActiveWorktreePathCoherent(path, effectiveProjectPath) {
			continue
		}
		if !WorktreePathUsable(path) {
			continue
		}
		return path
	}
	return ""
}
