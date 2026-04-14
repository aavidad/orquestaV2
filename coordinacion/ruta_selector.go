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

func PreferredWorkPath(currentCWD, activeWorktreePath, effectiveProjectPath string, currentInsideProjectWorktree bool) string {
	currentCWD = normalizeRouteSelectorPath(currentCWD)
	activeWorktreePath = normalizeRouteSelectorPath(activeWorktreePath)
	effectiveProjectPath = normalizeRouteSelectorPath(effectiveProjectPath)

	if activeWorktreePath != "" {
		if activeWorktreePath == currentCWD || routeSelectorPathWithin(activeWorktreePath, currentCWD) {
			return currentCWD
		}
		return activeWorktreePath
	}
	if currentInsideProjectWorktree {
		return currentCWD
	}
	if effectiveProjectPath != "" && currentCWD == effectiveProjectPath {
		return currentCWD
	}
	if effectiveProjectPath != "" {
		return effectiveProjectPath
	}
	return currentCWD
}

func ActiveWorktreePathCoherent(path, effectiveProjectPath string) bool {
	path = normalizeRouteSelectorPath(path)
	effectiveProjectPath = normalizeRouteSelectorPath(effectiveProjectPath)
	if path == "" {
		return false
	}
	if effectiveProjectPath == "" {
		return true
	}
	return routeSelectorPathWithin(effectiveProjectPath, path)
}

func normalizeRouteSelectorPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func routeSelectorPathWithin(base, path string) bool {
	base = normalizeRouteSelectorPath(base)
	path = normalizeRouteSelectorPath(path)
	if base == "" || path == "" {
		return false
	}
	if base == path {
		return true
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	prefixOutside := ".." + string(os.PathSeparator)
	return rel != ".." && !strings.HasPrefix(rel, prefixOutside)
}
