package runtimeagente

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func resetWorkerSnapshotCacheForTest() {
	workerSnapshotCache.mu.Lock()
	defer workerSnapshotCache.mu.Unlock()
	workerSnapshotCache.items = map[string]cachedWorkerSnapshot{}
}

func TestLoadWorkerSnapshotFromMetadataJSON(t *testing.T) {
	tmp := t.TempDir()
	now := time.Date(2026, 4, 3, 9, 12, 0, 0, time.UTC)

	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(manifestPath, WorkerManifest{
		Version:             1,
		Agent:               "Codex8",
		Project:             "orquestador",
		Driver:              "tmux_cli_session",
		Transport:           "tmux",
		TmuxSession:         "orq-codex8-093000",
		TmuxPaneID:          "%3",
		ChildPID:            4242,
		ExternalSessionID:   "sess-123",
		MailboxDeliveryMode: "session_resume",
	})
	write(statusPath, WorkerStatus{
		State:               "running",
		UpdatedAt:           now.Format(time.RFC3339),
		Alive:               true,
		ExternalSessionID:   "sess-123",
		MailboxDeliveryMode: "session_resume",
		LastProgressAt:      now.Add(3 * time.Second).Format(time.RFC3339),
	})
	write(heartbeatPath, WorkerHeartbeat{
		Alive:             true,
		HeartbeatAt:       now.Add(5 * time.Second).Format(time.RFC3339),
		ExternalSessionID: "sess-123",
		LastProgressAt:    now.Add(4 * time.Second).Format(time.RFC3339),
	})

	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	snap, err := LoadWorkerSnapshotFromMetadataJSON(string(meta))
	if err != nil {
		t.Fatalf("load worker snapshot: %v", err)
	}
	if snap == nil || snap.Manifest == nil || snap.Status == nil || snap.Heartbeat == nil {
		t.Fatalf("snapshot incompleto: %+v", snap)
	}
	if got := snap.EffectiveState(); got != "running" {
		t.Fatalf("estado efectivo inesperado: %q", got)
	}
	if !snap.Alive() {
		t.Fatalf("worker deberia salir vivo")
	}
	if got := snap.ExternalSessionID(); got != "sess-123" {
		t.Fatalf("external_session_id inesperado: %q", got)
	}
	if got := snap.SessionRef(); got != "sess-123" {
		t.Fatalf("session ref inesperado: %q", got)
	}
	if got := snap.MailboxDeliveryMode(); got != "session_resume" {
		t.Fatalf("mailbox delivery inesperado: %q", got)
	}
	if got := snap.Driver(); got != "tmux_cli_session" {
		t.Fatalf("driver inesperado: %q", got)
	}
	if got := snap.Transport(); got != "tmux" {
		t.Fatalf("transport inesperado: %q", got)
	}
	if got := snap.RuntimeRef(); got != "orq-codex8-093000/%3" {
		t.Fatalf("runtime ref inesperado: %q", got)
	}
	if got := snap.ChildPID(); got != 4242 {
		t.Fatalf("child pid inesperado: %d", got)
	}
	if hb := snap.HeartbeatTime(); hb == nil || hb.UTC().Format(time.RFC3339) != now.Add(5*time.Second).Format(time.RFC3339) {
		t.Fatalf("heartbeat inesperado: %+v", hb)
	}
	if progress := snap.LastProgressTime(); progress == nil || progress.UTC().Format(time.RFC3339) != now.Add(3*time.Second).Format(time.RFC3339) {
		t.Fatalf("last progress inesperado: %+v", progress)
	}
}

func TestLoadWorkerSnapshotFromMetadataJSONCompletaTMUXDesdeRuntimeManifest(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	runtimeManifestPath := filepath.Join(tmp, "runtime.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(manifestPath, WorkerManifest{
		Version:             1,
		Agent:               "Codex7",
		RuntimeManifestPath: runtimeManifestPath,
	})
	write(statusPath, WorkerStatus{State: "running", Alive: true})
	write(heartbeatPath, WorkerHeartbeat{Alive: true})
	write(runtimeManifestPath, map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-111552",
		"tmux_pane_id": "%0",
	})

	meta, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	snap, err := LoadWorkerSnapshotFromMetadataJSON(string(meta))
	if err != nil {
		t.Fatalf("load worker snapshot: %v", err)
	}
	if snap == nil || snap.Manifest == nil {
		t.Fatalf("snapshot incompleto: %+v", snap)
	}
	if got := snap.Driver(); got != "tmux_cli_session" {
		t.Fatalf("driver inesperado: %q", got)
	}
	if got := snap.Transport(); got != "tmux" {
		t.Fatalf("transport inesperado: %q", got)
	}
	if got := snap.RuntimeRef(); got != "orq-codex7-111552/%0" {
		t.Fatalf("runtime ref inesperado: %q", got)
	}
	if got := snap.SessionRef(); got != "orq-codex7-111552/%0" {
		t.Fatalf("session ref inesperado: %q", got)
	}
}

func TestLoadWorkerSnapshotFromMetadataJSONUsaCacheCaliente(t *testing.T) {
	resetWorkerSnapshotCacheForTest()
	prevTTL := workerSnapshotCacheTTL
	workerSnapshotCacheTTL = time.Hour
	defer func() {
		workerSnapshotCacheTTL = prevTTL
		resetWorkerSnapshotCacheForTest()
	}()

	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(manifestPath, WorkerManifest{Version: 1, Agent: "CodexHot"})
	write(statusPath, WorkerStatus{State: "running", Alive: true})
	write(heartbeatPath, WorkerHeartbeat{Alive: true})

	metaRaw, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	first, err := LoadWorkerSnapshotFromMetadataJSON(string(metaRaw))
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if first == nil || first.Status == nil || first.Status.State != "running" {
		t.Fatalf("snapshot inicial inesperado: %+v", first)
	}

	write(statusPath, WorkerStatus{State: "blocked_quota", Alive: true})

	second, err := LoadWorkerSnapshotFromMetadataJSON(string(metaRaw))
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if second == nil || second.Status == nil {
		t.Fatalf("snapshot cacheado inesperado: %+v", second)
	}
	if got := second.Status.State; got != "running" {
		t.Fatalf("deberia servir snapshot cacheado, got=%q", got)
	}
}

func TestLoadWorkerSnapshotFromMetadataJSONCacheDevuelveClonSeguro(t *testing.T) {
	resetWorkerSnapshotCacheForTest()
	prevTTL := workerSnapshotCacheTTL
	workerSnapshotCacheTTL = time.Hour
	defer func() {
		workerSnapshotCacheTTL = prevTTL
		resetWorkerSnapshotCacheForTest()
	}()

	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(manifestPath, WorkerManifest{Version: 1, Agent: "CodexClone", Driver: "tmux_cli_session"})
	write(statusPath, WorkerStatus{State: "ready", Alive: true})
	write(heartbeatPath, WorkerHeartbeat{Alive: true})

	metaRaw, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	first, err := LoadWorkerSnapshotFromMetadataJSON(string(metaRaw))
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	first.Status.State = "corrupto"
	first.Manifest.Driver = "mutado"

	second, err := LoadWorkerSnapshotFromMetadataJSON(string(metaRaw))
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if second == nil || second.Status == nil || second.Manifest == nil {
		t.Fatalf("snapshot inesperado: %+v", second)
	}
	if got := second.Status.State; got != "ready" {
		t.Fatalf("cache no deberia compartir punteros mutables, state=%q", got)
	}
	if got := second.Manifest.Driver; got != "tmux_cli_session" {
		t.Fatalf("manifest no deberia quedar mutado, driver=%q", got)
	}
}

func TestWorkerSnapshotHeartbeatStale(t *testing.T) {
	now := time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC)
	snap := &WorkerSnapshot{
		Heartbeat: &WorkerHeartbeat{
			Alive:       true,
			HeartbeatAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
	}
	if !snap.IsHeartbeatStale(now, time.Minute) {
		t.Fatalf("heartbeat deberia salir stale")
	}
	if snap.IsHeartbeatStale(now, 3*time.Minute) {
		t.Fatalf("heartbeat no deberia salir stale con umbral mayor")
	}
}

func TestWorkerSnapshotReadyForTextDispatch(t *testing.T) {
	now := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	base := &WorkerSnapshot{
		Manifest: &WorkerManifest{
			Driver:    "tmux_cli_session",
			Transport: "tmux",
		},
		Heartbeat: &WorkerHeartbeat{
			Alive:       true,
			HeartbeatAt: now.Format(time.RFC3339),
		},
		Status: &WorkerStatus{
			Alive:     true,
			UpdatedAt: now.Format(time.RFC3339),
		},
	}

	t.Run("ready", func(t *testing.T) {
		snap := *base
		snap.Status = &WorkerStatus{
			State:     "ready",
			Alive:     true,
			UpdatedAt: now.Format(time.RFC3339),
			ReadyAt:   now.Format(time.RFC3339),
		}
		ok, reason := snap.ReadyForTextDispatch(now, time.Minute)
		if !ok || reason != "" {
			t.Fatalf("dispatch ready inesperado: ok=%t reason=%q", ok, reason)
		}
	})

	t.Run("running_busy", func(t *testing.T) {
		snap := *base
		snap.Status = &WorkerStatus{
			State:          "running",
			Alive:          true,
			UpdatedAt:      now.Format(time.RFC3339),
			LastProgressAt: now.Format(time.RFC3339),
		}
		ok, reason := snap.ReadyForTextDispatch(now, time.Minute)
		if ok || reason != "worker_busy" {
			t.Fatalf("dispatch busy inesperado: ok=%t reason=%q", ok, reason)
		}
	})

	t.Run("blocked_quota", func(t *testing.T) {
		snap := *base
		snap.Status = &WorkerStatus{
			State:     "blocked_quota",
			Alive:     true,
			UpdatedAt: now.Format(time.RFC3339),
		}
		ok, reason := snap.ReadyForTextDispatch(now, time.Minute)
		if ok || reason != "worker_blocked_quota" {
			t.Fatalf("dispatch blocked_quota inesperado: ok=%t reason=%q", ok, reason)
		}
	})

	t.Run("blocked_trust", func(t *testing.T) {
		snap := *base
		snap.Status = &WorkerStatus{
			State:     "blocked_trust",
			Alive:     true,
			UpdatedAt: now.Format(time.RFC3339),
		}
		ok, reason := snap.ReadyForTextDispatch(now, time.Minute)
		if ok || reason != "worker_blocked_trust" {
			t.Fatalf("dispatch blocked_trust inesperado: ok=%t reason=%q", ok, reason)
		}
	})
}
