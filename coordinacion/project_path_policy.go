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
)

type RepoMarkerResolver func(path string) (bool, error)

func CandidateEffectiveProjectPath(currentCWD string, currentInsideActiveWorktree bool, hasRepoMarkers RepoMarkerResolver) string {
	currentCWD = normalizeRouteSelectorPath(currentCWD)
	if currentCWD == "" || currentInsideActiveWorktree {
		return ""
	}
	root, _ := ResolveWorkRoot(currentCWD, hasRepoMarkers)
	return root
}

func ResolveWorkRoot(path string, hasRepoMarkers RepoMarkerResolver) (string, bool) {
	path = normalizeRouteSelectorPath(path)
	if path == "" || hasRepoMarkers == nil {
		return "", false
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		path = filepath.Dir(path)
	}
	current := path
	for {
		ok, err := hasRepoMarkers(current)
		if err == nil && ok {
			if filepath.Dir(current) == current {
				return "", false
			}
			return current, true
		}
		next := filepath.Dir(current)
		if next == current {
			return "", false
		}
		current = next
	}
}

func WorkPathBelongsToProject(currentCWD, effectiveProjectPath, activeWorktreePath string) bool {
	currentCWD = normalizeRouteSelectorPath(currentCWD)
	effectiveProjectPath = normalizeRouteSelectorPath(effectiveProjectPath)
	activeWorktreePath = normalizeRouteSelectorPath(activeWorktreePath)
	if currentCWD == "" {
		return false
	}
	if effectiveProjectPath != "" && PathWithin(effectiveProjectPath, currentCWD) {
		return true
	}
	if activeWorktreePath != "" && PathWithin(activeWorktreePath, currentCWD) {
		return true
	}
	return false
}
