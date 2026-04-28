package db

import (
	"strings"
	"sync"
	"time"

	"orquesta/runtimeagente"
)

var runtimeTranscriptHotIdleState struct {
	mu              sync.Mutex
	failedHandleDue map[int64]time.Time
}

func runtimeTranscriptResolveLogPath(handle *RuntimeHandle, meta map[string]any) string {
	if handle == nil {
		return ""
	}
	if logPath := strings.TrimSpace(stringFromMap(meta, "log_path", "")); logPath != "" {
		return logPath
	}
	snapshot, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snapshot == nil {
		return ""
	}
	candidates := []string{}
	if snapshot.Status != nil {
		candidates = append(candidates, strings.TrimSpace(snapshot.Status.LogPath))
	}
	if snapshot.Manifest != nil {
		candidates = append(candidates, strings.TrimSpace(snapshot.Manifest.LogPath))
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		meta["log_path"] = candidate
		return candidate
	}
	return ""
}

func runtimeTranscriptHotIdleSkip(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil || handle.ID <= 0 || !runtimeTranscriptHandleAdmiteIdle(handle) {
		return false
	}
	runtimeTranscriptHotIdleState.mu.Lock()
	defer runtimeTranscriptHotIdleState.mu.Unlock()
	if runtimeTranscriptHotIdleState.failedHandleDue == nil {
		return false
	}
	due := runtimeTranscriptHotIdleState.failedHandleDue[handle.ID]
	return !due.IsZero() && now.Before(due)
}

func runtimeTranscriptHotIdleRemember(handle *RuntimeHandle, now time.Time) {
	if handle == nil || handle.ID <= 0 {
		return
	}
	if !runtimeTranscriptHandleAdmiteIdle(handle) {
		runtimeTranscriptHotIdleClear(handle)
		return
	}
	runtimeTranscriptHotIdleState.mu.Lock()
	defer runtimeTranscriptHotIdleState.mu.Unlock()
	if runtimeTranscriptHotIdleState.failedHandleDue == nil {
		runtimeTranscriptHotIdleState.failedHandleDue = map[int64]time.Time{}
	}
	runtimeTranscriptHotIdleState.failedHandleDue[handle.ID] = now.Add(runtimeTranscriptFailedIdleCooldown())
}

func runtimeTranscriptHotIdleClear(handle *RuntimeHandle) {
	if handle == nil || handle.ID <= 0 {
		return
	}
	runtimeTranscriptHotIdleState.mu.Lock()
	defer runtimeTranscriptHotIdleState.mu.Unlock()
	if runtimeTranscriptHotIdleState.failedHandleDue == nil {
		return
	}
	delete(runtimeTranscriptHotIdleState.failedHandleDue, handle.ID)
}

func runtimeTranscriptFailedIdleCooldown() time.Duration {
	return 5 * time.Minute
}

func runtimeTranscriptHandleAdmiteIdle(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(handle.Estado), "fallido")
}

func resetRuntimeTranscriptHotIdleState() {
	runtimeTranscriptHotIdleState.mu.Lock()
	defer runtimeTranscriptHotIdleState.mu.Unlock()
	runtimeTranscriptHotIdleState.failedHandleDue = nil
}
