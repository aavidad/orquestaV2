package orquestaruntimecodexappserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func (backend serverCodexAppServerTmuxBackendV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	if !command.CleanupGoalBackends {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	evidenceRefs := compactStringsV0(append(
		append([]string(nil), command.EvidenceRefs...),
		"evidence-ref-codex-app-server-tmux-cleanup-requested",
	))
	cleaned := 0
	if backend.detectTmuxResidueV0(ctx).Active {
		if err := backend.shutdownTmuxSessionForCleanupV0(ctx); err != nil {
			return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, err
		}
		if !backend.detectTmuxResidueV0(ctx).Active {
			cleaned++
			evidenceRefs = append(evidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned")
		}
	}
	seenMarkers := map[string]struct{}{}
	currentMarker := filepath.Clean(backend.tmuxOwnerMarkerPathV0())
	for _, markerPath := range backend.tmuxOwnerMarkerScanPathsV0() {
		markerPath = filepath.Clean(strings.TrimSpace(markerPath))
		if markerPath == "" || markerPath == currentMarker {
			continue
		}
		if _, exists := seenMarkers[markerPath]; exists {
			continue
		}
		seenMarkers[markerPath] = struct{}{}
		if backend.cleanupTmuxOwnerMarkerPathV0(ctx, markerPath) {
			cleaned++
			evidenceRefs = append(evidenceRefs, "evidence-ref-codex-app-server-tmux-owner-cleaned")
		}
	}
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
		CleanedWorkCount: cleaned,
		EvidenceRefs:     compactStringsV0(evidenceRefs),
	}, nil
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupTmuxOwnerMarkerPathV0(
	ctx context.Context,
	markerPath string,
) bool {
	marker, ok := readCodexAppServerTmuxOwnerMarkerPathV0(markerPath)
	if !ok {
		return false
	}
	sessionName := strings.TrimSpace(marker.SessionName)
	if sessionName == "" {
		return false
	}
	markerDir := filepath.Dir(markerPath)
	candidate := backend
	candidate.SessionName = sessionName
	if marker.generationMarkerV0() {
		candidate.SocketPath = strings.TrimSpace(marker.SocketPath)
		if filepath.Dir(filepath.Clean(candidate.SocketPath)) != filepath.Clean(markerDir) {
			return false
		}
	} else {
		candidate.SocketPath = filepath.Join(markerDir, "s.sock")
	}
	if !candidate.tmuxConfiguredOrphanCleanupAllowedV0() {
		return false
	}
	if err := candidate.shutdownTmuxSessionForCleanupV0(ctx); err != nil {
		return false
	}
	if marker.generationMarkerV0() {
		// Generation cleanup already removed exactly its socket and marker.
		// Never glob a directory that may now host a newer generation.
		return !candidate.detectTmuxOwnerMarkerResiduePathV0(ctx, markerPath)
	}
	cleanupTmuxOwnedSocketsInDirV0(markerDir)
	_ = os.Remove(markerPath)
	return !candidate.detectTmuxOwnerMarkerResiduePathV0(ctx, markerPath)
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxOwnerMarkerResiduePathV0(
	ctx context.Context,
	markerPath string,
) bool {
	marker, ok := readCodexAppServerTmuxOwnerMarkerPathV0(markerPath)
	if !ok {
		return false
	}
	sessionName := strings.TrimSpace(marker.SessionName)
	if sessionName == "" {
		return false
	}
	session, process := backend.detectTmuxSessionResidueByNameV0(ctx, sessionName)
	return session || process
}

func cleanupTmuxOwnedSocketsInDirV0(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." {
		return
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.sock"))
	if err != nil {
		return
	}
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		_ = os.Remove(match)
	}
}
