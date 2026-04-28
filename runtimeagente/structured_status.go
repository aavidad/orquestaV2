package runtimeagente

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type WorkerManifest struct {
	Version              int    `json:"version"`
	Agent                string `json:"agent"`
	Project              string `json:"project,omitempty"`
	Driver               string `json:"driver"`
	Transport            string `json:"transport"`
	Profile              string `json:"profile,omitempty"`
	ExecutionProfile     string `json:"execution_profile,omitempty"`
	ProfileStatusWrapper string `json:"profile_status_wrapper,omitempty"`
	TmuxSession          string `json:"tmux_session,omitempty"`
	TmuxWindow           string `json:"tmux_window,omitempty"`
	TmuxPaneID           string `json:"tmux_pane_id,omitempty"`
	CreatedAt            string `json:"created_at"`
	StartedAt            string `json:"started_at"`
	ChildPID             int    `json:"child_pid"`
	WorkingDir           string `json:"working_dir,omitempty"`
	Command              string `json:"command,omitempty"`
	RuntimeManifestPath  string `json:"runtime_manifest_path,omitempty"`
	StatusPath           string `json:"status_path,omitempty"`
	HeartbeatPath        string `json:"heartbeat_path,omitempty"`
	LogPath              string `json:"log_path,omitempty"`
	StdinPath            string `json:"stdin_path,omitempty"`
	ExternalSessionID    string `json:"external_session_id,omitempty"`
	MailboxDeliveryMode  string `json:"mailbox_delivery_mode,omitempty"`
	CanSendInput         bool   `json:"can_send_input"`
}

type WorkerStatus struct {
	State               string `json:"state"`
	UpdatedAt           string `json:"updated_at"`
	Alive               bool   `json:"alive"`
	ChildPID            int    `json:"child_pid"`
	Agent               string `json:"agent,omitempty"`
	Project             string `json:"project,omitempty"`
	WorkingDir          string `json:"working_dir,omitempty"`
	CurrentPath         string `json:"current_path,omitempty"`
	LogPath             string `json:"log_path,omitempty"`
	ExternalSessionID   string `json:"external_session_id,omitempty"`
	MailboxDeliveryMode string `json:"mailbox_delivery_mode,omitempty"`
	ReadyAt             string `json:"ready_at,omitempty"`
	LastOutputAt        string `json:"last_output_at,omitempty"`
	LastProgressAt      string `json:"last_progress_at,omitempty"`
	ExitCode            *int   `json:"exit_code,omitempty"`
	ExitError           string `json:"exit_error,omitempty"`
}

type WorkerHeartbeat struct {
	Alive             bool   `json:"alive"`
	HeartbeatAt       string `json:"heartbeat_at"`
	StartedAt         string `json:"started_at,omitempty"`
	ChildPID          int    `json:"child_pid"`
	Agent             string `json:"agent,omitempty"`
	Project           string `json:"project,omitempty"`
	ExternalSessionID string `json:"external_session_id,omitempty"`
	ReadyAt           string `json:"ready_at,omitempty"`
	LastOutputAt      string `json:"last_output_at,omitempty"`
	LastProgressAt    string `json:"last_progress_at,omitempty"`
	ExitCode          *int   `json:"exit_code,omitempty"`
	ExitError         string `json:"exit_error,omitempty"`
}

type WorkerSnapshot struct {
	Manifest      *WorkerManifest
	Status        *WorkerStatus
	Heartbeat     *WorkerHeartbeat
	WorkQueue     *WorkerWorkQueue
	ManifestPath  string
	StatusPath    string
	HeartbeatPath string
}

type WorkerWorkQueue struct {
	Version   int                   `json:"version"`
	UpdatedAt string                `json:"updated_at"`
	Current   *WorkerWorkQueueEntry `json:"current,omitempty"`
}

type WorkerWorkQueueEntry struct {
	MailboxID       int64  `json:"mailbox_id,omitempty"`
	RuntimeOrderID  int64  `json:"runtime_order_id,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Action          string `json:"action,omitempty"`
	TaskID          int64  `json:"task_id,omitempty"`
	VerificationKey string `json:"verification_key,omitempty"`
	State           string `json:"state,omitempty"`
	Title           string `json:"title,omitempty"`
	Reason          string `json:"reason,omitempty"`
	RecordedAt      string `json:"recorded_at"`
}

type WorkerStatusView struct {
	Driver                   string     `json:"driver,omitempty"`
	Transport                string     `json:"transport,omitempty"`
	ExecutionProfile         string     `json:"execution_profile,omitempty"`
	State                    string     `json:"state,omitempty"`
	Alive                    bool       `json:"alive"`
	HeartbeatStale           bool       `json:"heartbeat_stale"`
	ChildPID                 int        `json:"child_pid,omitempty"`
	RuntimeRef               string     `json:"runtime_ref,omitempty"`
	SessionRef               string     `json:"session_ref,omitempty"`
	ExternalSessionID        string     `json:"external_session_id,omitempty"`
	MailboxDeliveryMode      string     `json:"mailbox_delivery_mode,omitempty"`
	ExitError                string     `json:"exit_error,omitempty"`
	ManifestPath             string     `json:"manifest_path,omitempty"`
	StatusPath               string     `json:"status_path,omitempty"`
	HeartbeatPath            string     `json:"heartbeat_path,omitempty"`
	TmuxSession              string     `json:"tmux_session,omitempty"`
	TmuxWindow               string     `json:"tmux_window,omitempty"`
	TmuxPaneID               string     `json:"tmux_pane_id,omitempty"`
	CurrentPath              string     `json:"current_path,omitempty"`
	WorkQueueMailboxID       int64      `json:"work_queue_mailbox_id,omitempty"`
	WorkQueueKind            string     `json:"work_queue_kind,omitempty"`
	WorkQueueAction          string     `json:"work_queue_action,omitempty"`
	WorkQueueTaskID          int64      `json:"work_queue_task_id,omitempty"`
	WorkQueueVerificationKey string     `json:"work_queue_verification_key,omitempty"`
	WorkQueueState           string     `json:"work_queue_state,omitempty"`
	WorkQueueTitle           string     `json:"work_queue_title,omitempty"`
	WorkQueueReason          string     `json:"work_queue_reason,omitempty"`
	WorkQueueUpdatedAt       *time.Time `json:"work_queue_updated_at,omitempty"`
	CanSendInput             bool       `json:"can_send_input"`
	StartedAt                *time.Time `json:"started_at,omitempty"`
	UpdatedAt                *time.Time `json:"updated_at,omitempty"`
	HeartbeatAt              *time.Time `json:"heartbeat_at,omitempty"`
	ReadyAt                  *time.Time `json:"ready_at,omitempty"`
	LastOutputAt             *time.Time `json:"last_output_at,omitempty"`
	LastProgressAt           *time.Time `json:"last_progress_at,omitempty"`
}

type runtimeManifestOverlay struct {
	Driver                 string `json:"driver"`
	Transport              string `json:"transport"`
	ExecutionProfile       string `json:"execution_profile"`
	LegacyExecutionProfile string `json:"perfil_operativo"`
	TmuxSession            string `json:"tmux_session"`
	TmuxWindow             string `json:"tmux_window"`
	TmuxPaneID             string `json:"tmux_pane_id"`
}

const WorkerMetadataSchemaVersion = 1

type workerMetadataPaths struct {
	SchemaVersion    int    `json:"worker_schema_version,omitempty"`
	ManifestPath     string `json:"worker_manifest_path"`
	StatusPath       string `json:"worker_status_path"`
	HeartbeatPath    string `json:"worker_heartbeat_path"`
	TraceDir         string `json:"trace_dir"`
	ExecutionProfile string `json:"perfil_operativo,omitempty"`
}

type cachedWorkerSnapshot struct {
	snapshot *WorkerSnapshot
	expires  time.Time
}

var workerSnapshotCacheTTL = 1500 * time.Millisecond

var workerSnapshotCache = struct {
	mu    sync.Mutex
	items map[string]cachedWorkerSnapshot
}{
	items: map[string]cachedWorkerSnapshot{},
}

func LoadWorkerSnapshotFromMetadataJSON(raw string) (*WorkerSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	meta, err := workerMetadataPathsFromJSON(raw)
	if err != nil {
		return nil, err
	}
	if workerMetadataPathsEmpty(meta) {
		return nil, nil
	}
	if cached := cachedWorkerSnapshotByPaths(meta); cached != nil {
		return cached, nil
	}
	snap := &WorkerSnapshot{
		ManifestPath:  strings.TrimSpace(meta.ManifestPath),
		StatusPath:    strings.TrimSpace(meta.StatusPath),
		HeartbeatPath: strings.TrimSpace(meta.HeartbeatPath),
	}
	if data, err := os.ReadFile(snap.ManifestPath); err == nil {
		var manifest WorkerManifest
		if json.Unmarshal(data, &manifest) == nil {
			snap.Manifest = &manifest
		}
	}
	if data, err := os.ReadFile(snap.StatusPath); err == nil {
		var status WorkerStatus
		if json.Unmarshal(data, &status) == nil {
			snap.Status = &status
		}
	}
	if data, err := os.ReadFile(snap.HeartbeatPath); err == nil {
		var heartbeat WorkerHeartbeat
		if json.Unmarshal(data, &heartbeat) == nil {
			snap.Heartbeat = &heartbeat
		}
	}
	if workQueuePath := workerWorkQueuePath(meta); workQueuePath != "" {
		if data, err := os.ReadFile(workQueuePath); err == nil {
			var queue WorkerWorkQueue
			if json.Unmarshal(data, &queue) == nil {
				snap.WorkQueue = &queue
			}
		}
	}
	if snap.Manifest != nil && strings.TrimSpace(snap.Manifest.RuntimeManifestPath) != "" {
		if data, err := os.ReadFile(strings.TrimSpace(snap.Manifest.RuntimeManifestPath)); err == nil {
			var overlay runtimeManifestOverlay
			if json.Unmarshal(data, &overlay) == nil {
				if strings.TrimSpace(snap.Manifest.Driver) == "" {
					snap.Manifest.Driver = strings.TrimSpace(overlay.Driver)
				}
				if strings.TrimSpace(snap.Manifest.Transport) == "" {
					snap.Manifest.Transport = strings.TrimSpace(overlay.Transport)
				}
				if strings.TrimSpace(snap.Manifest.ExecutionProfile) == "" {
					snap.Manifest.ExecutionProfile = overlay.executionProfile()
				}
				if strings.TrimSpace(snap.Manifest.TmuxSession) == "" {
					snap.Manifest.TmuxSession = strings.TrimSpace(overlay.TmuxSession)
				}
				if strings.TrimSpace(snap.Manifest.TmuxWindow) == "" {
					snap.Manifest.TmuxWindow = strings.TrimSpace(overlay.TmuxWindow)
				}
				if strings.TrimSpace(snap.Manifest.TmuxPaneID) == "" {
					snap.Manifest.TmuxPaneID = strings.TrimSpace(overlay.TmuxPaneID)
				}
			}
		}
	}
	if snap.Manifest != nil && strings.TrimSpace(snap.Manifest.ExecutionProfile) == "" {
		snap.Manifest.ExecutionProfile = strings.TrimSpace(meta.ExecutionProfile)
	}
	cacheWorkerSnapshotByPaths(meta, snap)
	return snap, nil
}

func (o runtimeManifestOverlay) executionProfile() string {
	if value := strings.TrimSpace(o.ExecutionProfile); value != "" {
		return value
	}
	return strings.TrimSpace(o.LegacyExecutionProfile)
}

func workerMetadataPathsFromJSON(raw string) (workerMetadataPaths, error) {
	var meta workerMetadataPaths
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return workerMetadataPaths{}, err
	}
	if meta.SchemaVersion <= 0 {
		meta.SchemaVersion = WorkerMetadataSchemaVersion
	}
	meta.ManifestPath = strings.TrimSpace(meta.ManifestPath)
	meta.StatusPath = strings.TrimSpace(meta.StatusPath)
	meta.HeartbeatPath = strings.TrimSpace(meta.HeartbeatPath)
	meta.TraceDir = strings.TrimSpace(meta.TraceDir)
	meta.ExecutionProfile = strings.TrimSpace(meta.ExecutionProfile)
	return meta, nil
}

func workerMetadataPathsEmpty(meta workerMetadataPaths) bool {
	return meta.ManifestPath == "" && meta.StatusPath == "" && meta.HeartbeatPath == ""
}

func workerSnapshotCacheKey(meta workerMetadataPaths) string {
	return strings.Join([]string{
		meta.ManifestPath,
		meta.StatusPath,
		meta.HeartbeatPath,
		meta.TraceDir,
	}, "|")
}

func cachedWorkerSnapshotByPaths(meta workerMetadataPaths) *WorkerSnapshot {
	ttl := workerSnapshotCacheTTL
	if ttl <= 0 || workerMetadataPathsEmpty(meta) {
		return nil
	}
	now := time.Now()
	key := workerSnapshotCacheKey(meta)
	workerSnapshotCache.mu.Lock()
	defer workerSnapshotCache.mu.Unlock()
	item, ok := workerSnapshotCache.items[key]
	if !ok {
		return nil
	}
	if now.After(item.expires) {
		delete(workerSnapshotCache.items, key)
		return nil
	}
	return cloneWorkerSnapshot(item.snapshot)
}

func cacheWorkerSnapshotByPaths(meta workerMetadataPaths, snap *WorkerSnapshot) {
	ttl := workerSnapshotCacheTTL
	if ttl <= 0 || workerMetadataPathsEmpty(meta) || snap == nil {
		return
	}
	key := workerSnapshotCacheKey(meta)
	workerSnapshotCache.mu.Lock()
	defer workerSnapshotCache.mu.Unlock()
	workerSnapshotCache.items[key] = cachedWorkerSnapshot{
		snapshot: cloneWorkerSnapshot(snap),
		expires:  time.Now().Add(ttl),
	}
}

func cloneWorkerSnapshot(src *WorkerSnapshot) *WorkerSnapshot {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Manifest != nil {
		manifest := *src.Manifest
		dst.Manifest = &manifest
	}
	if src.Status != nil {
		status := *src.Status
		if src.Status.ExitCode != nil {
			exitCode := *src.Status.ExitCode
			status.ExitCode = &exitCode
		}
		dst.Status = &status
	}
	if src.Heartbeat != nil {
		heartbeat := *src.Heartbeat
		if src.Heartbeat.ExitCode != nil {
			exitCode := *src.Heartbeat.ExitCode
			heartbeat.ExitCode = &exitCode
		}
		dst.Heartbeat = &heartbeat
	}
	if src.WorkQueue != nil {
		queue := *src.WorkQueue
		if src.WorkQueue.Current != nil {
			current := *src.WorkQueue.Current
			queue.Current = &current
		}
		dst.WorkQueue = &queue
	}
	return &dst
}

func workerWorkQueuePath(meta workerMetadataPaths) string {
	if strings.TrimSpace(meta.TraceDir) != "" {
		return filepath.Join(strings.TrimSpace(meta.TraceDir), "work-queue.json")
	}
	for _, path := range []string{meta.ManifestPath, meta.StatusPath, meta.HeartbeatPath} {
		if strings.TrimSpace(path) != "" {
			return filepath.Join(filepath.Dir(strings.TrimSpace(path)), "work-queue.json")
		}
	}
	return ""
}

func (s *WorkerSnapshot) EffectiveState() string {
	if s == nil {
		return ""
	}
	if s.Status != nil && strings.TrimSpace(s.Status.State) != "" {
		return strings.TrimSpace(s.Status.State)
	}
	if s.Heartbeat != nil {
		if s.Heartbeat.Alive {
			return "running"
		}
		if strings.TrimSpace(s.Heartbeat.ExitError) != "" {
			return "failed"
		}
	}
	return ""
}

func (s *WorkerSnapshot) Alive() bool {
	if s == nil {
		return false
	}
	if s.Status != nil {
		return s.Status.Alive
	}
	if s.Heartbeat != nil {
		return s.Heartbeat.Alive
	}
	return false
}

func (s *WorkerSnapshot) HeartbeatTime() *time.Time {
	if s == nil || s.Heartbeat == nil {
		return nil
	}
	return parseWorkerTimestamp(s.Heartbeat.HeartbeatAt)
}

func (s *WorkerSnapshot) UpdatedTime() *time.Time {
	if s == nil || s.Status == nil {
		return nil
	}
	return parseWorkerTimestamp(s.Status.UpdatedAt)
}

func (s *WorkerSnapshot) LastOutputTime() *time.Time {
	if s == nil {
		return nil
	}
	if s.Status != nil && strings.TrimSpace(s.Status.LastOutputAt) != "" {
		if ts := parseWorkerTimestamp(s.Status.LastOutputAt); ts != nil {
			return ts
		}
	}
	if s.Heartbeat != nil && strings.TrimSpace(s.Heartbeat.LastOutputAt) != "" {
		return parseWorkerTimestamp(s.Heartbeat.LastOutputAt)
	}
	return nil
}

func (s *WorkerSnapshot) ReadyTime() *time.Time {
	if s == nil {
		return nil
	}
	if s.Status != nil && strings.TrimSpace(s.Status.ReadyAt) != "" {
		if ts := parseWorkerTimestamp(s.Status.ReadyAt); ts != nil {
			return ts
		}
	}
	if s.Heartbeat != nil && strings.TrimSpace(s.Heartbeat.ReadyAt) != "" {
		return parseWorkerTimestamp(s.Heartbeat.ReadyAt)
	}
	return nil
}

func (s *WorkerSnapshot) LastProgressTime() *time.Time {
	if s == nil {
		return nil
	}
	if s.Status != nil && strings.TrimSpace(s.Status.LastProgressAt) != "" {
		if ts := parseWorkerTimestamp(s.Status.LastProgressAt); ts != nil {
			return ts
		}
	}
	if s.Heartbeat != nil && strings.TrimSpace(s.Heartbeat.LastProgressAt) != "" {
		return parseWorkerTimestamp(s.Heartbeat.LastProgressAt)
	}
	return nil
}

func (s *WorkerSnapshot) ExternalSessionID() string {
	if s == nil {
		return ""
	}
	if s.Status != nil && strings.TrimSpace(s.Status.ExternalSessionID) != "" {
		return strings.TrimSpace(s.Status.ExternalSessionID)
	}
	if s.Heartbeat != nil && strings.TrimSpace(s.Heartbeat.ExternalSessionID) != "" {
		return strings.TrimSpace(s.Heartbeat.ExternalSessionID)
	}
	if s.Manifest != nil {
		return strings.TrimSpace(s.Manifest.ExternalSessionID)
	}
	return ""
}

func (s *WorkerSnapshot) SessionRef() string {
	if s == nil {
		return ""
	}
	if external := s.ExternalSessionID(); external != "" {
		return external
	}
	if s.Manifest != nil && strings.EqualFold(strings.TrimSpace(s.Manifest.Transport), "tmux") {
		session := strings.TrimSpace(s.Manifest.TmuxSession)
		pane := strings.TrimSpace(s.Manifest.TmuxPaneID)
		switch {
		case session != "" && pane != "":
			return session + "/" + pane
		case session != "":
			return session
		case pane != "":
			return pane
		}
	}
	return ""
}

func (s *WorkerSnapshot) MailboxDeliveryMode() string {
	if s == nil {
		return ""
	}
	if s.Status != nil && strings.TrimSpace(s.Status.MailboxDeliveryMode) != "" {
		return strings.TrimSpace(s.Status.MailboxDeliveryMode)
	}
	if s.Manifest != nil {
		return strings.TrimSpace(s.Manifest.MailboxDeliveryMode)
	}
	return ""
}

func (s *WorkerSnapshot) ExitError() string {
	if s == nil {
		return ""
	}
	if s.Status != nil && strings.TrimSpace(s.Status.ExitError) != "" {
		return strings.TrimSpace(s.Status.ExitError)
	}
	if s.Heartbeat != nil {
		return strings.TrimSpace(s.Heartbeat.ExitError)
	}
	return ""
}

func (s *WorkerSnapshot) Driver() string {
	if s == nil || s.Manifest == nil {
		return ""
	}
	return strings.TrimSpace(s.Manifest.Driver)
}

func (s *WorkerSnapshot) Transport() string {
	if s == nil || s.Manifest == nil {
		return ""
	}
	return strings.TrimSpace(s.Manifest.Transport)
}

func (s *WorkerSnapshot) ExecutionProfile() string {
	if s == nil || s.Manifest == nil {
		return ""
	}
	return strings.TrimSpace(s.Manifest.ExecutionProfile)
}

func (s *WorkerSnapshot) ChildPID() int {
	if s == nil {
		return 0
	}
	if s.Status != nil && s.Status.ChildPID > 0 {
		return s.Status.ChildPID
	}
	if s.Heartbeat != nil && s.Heartbeat.ChildPID > 0 {
		return s.Heartbeat.ChildPID
	}
	if s.Manifest != nil && s.Manifest.ChildPID > 0 {
		return s.Manifest.ChildPID
	}
	return 0
}

func (s *WorkerSnapshot) RuntimeRef() string {
	if s == nil {
		return ""
	}
	if s.Manifest != nil && strings.EqualFold(strings.TrimSpace(s.Manifest.Transport), "tmux") {
		session := strings.TrimSpace(s.Manifest.TmuxSession)
		pane := strings.TrimSpace(s.Manifest.TmuxPaneID)
		switch {
		case session != "" && pane != "":
			return session + "/" + pane
		case session != "":
			return session
		case pane != "":
			return pane
		}
	}
	if transport := s.Transport(); transport != "" {
		return transport
	}
	return ""
}

func (s *WorkerSnapshot) IsHeartbeatStale(now time.Time, threshold time.Duration) bool {
	if s == nil {
		return false
	}
	if threshold <= 0 {
		threshold = time.Minute
	}
	var ts *time.Time
	if hb := s.HeartbeatTime(); hb != nil {
		ts = hb
	} else if updated := s.UpdatedTime(); updated != nil {
		ts = updated
	}
	if ts == nil || ts.IsZero() {
		return false
	}
	now = now.UTC()
	return now.Sub(ts.UTC()) > threshold
}

func (s *WorkerSnapshot) StartedTime() *time.Time {
	if s == nil {
		return nil
	}
	if s.Heartbeat != nil && strings.TrimSpace(s.Heartbeat.StartedAt) != "" {
		if ts := parseWorkerTimestamp(s.Heartbeat.StartedAt); ts != nil {
			return ts
		}
	}
	if s.Manifest != nil {
		if ts := parseWorkerTimestamp(s.Manifest.StartedAt); ts != nil {
			return ts
		}
		if ts := parseWorkerTimestamp(s.Manifest.CreatedAt); ts != nil {
			return ts
		}
	}
	return nil
}

func (s *WorkerSnapshot) View(now time.Time, heartbeatThreshold time.Duration) *WorkerStatusView {
	if s == nil {
		return nil
	}
	view := &WorkerStatusView{
		Driver:              s.Driver(),
		Transport:           s.Transport(),
		ExecutionProfile:    s.ExecutionProfile(),
		State:               s.EffectiveState(),
		Alive:               s.Alive(),
		HeartbeatStale:      s.IsHeartbeatStale(now, heartbeatThreshold),
		ChildPID:            s.ChildPID(),
		RuntimeRef:          s.RuntimeRef(),
		SessionRef:          s.SessionRef(),
		ExternalSessionID:   s.ExternalSessionID(),
		MailboxDeliveryMode: s.MailboxDeliveryMode(),
		ExitError:           s.ExitError(),
		ManifestPath:        strings.TrimSpace(s.ManifestPath),
		StatusPath:          strings.TrimSpace(s.StatusPath),
		HeartbeatPath:       strings.TrimSpace(s.HeartbeatPath),
		StartedAt:           s.StartedTime(),
		UpdatedAt:           s.UpdatedTime(),
		HeartbeatAt:         s.HeartbeatTime(),
		ReadyAt:             s.ReadyTime(),
		LastOutputAt:        s.LastOutputTime(),
		LastProgressAt:      s.LastProgressTime(),
	}
	if s.Manifest != nil {
		view.TmuxSession = strings.TrimSpace(s.Manifest.TmuxSession)
		view.TmuxWindow = strings.TrimSpace(s.Manifest.TmuxWindow)
		view.TmuxPaneID = strings.TrimSpace(s.Manifest.TmuxPaneID)
		view.CanSendInput = s.Manifest.CanSendInput
	}
	if s.Status != nil {
		view.CurrentPath = strings.TrimSpace(s.Status.CurrentPath)
	}
	if s.WorkQueue != nil {
		if current := s.WorkQueue.Current; current != nil {
			view.WorkQueueMailboxID = current.MailboxID
			view.WorkQueueKind = strings.TrimSpace(current.Kind)
			view.WorkQueueAction = strings.TrimSpace(current.Action)
			view.WorkQueueTaskID = current.TaskID
			view.WorkQueueVerificationKey = strings.TrimSpace(current.VerificationKey)
			view.WorkQueueState = strings.TrimSpace(current.State)
			view.WorkQueueTitle = strings.TrimSpace(current.Title)
			view.WorkQueueReason = strings.TrimSpace(current.Reason)
			view.WorkQueueUpdatedAt = parseWorkerTimestamp(current.RecordedAt)
		} else {
			view.WorkQueueUpdatedAt = parseWorkerTimestamp(s.WorkQueue.UpdatedAt)
		}
	}
	return view
}

func (s *WorkerSnapshot) ReadyForTextDispatch(now time.Time, heartbeatThreshold time.Duration) (bool, string) {
	if s == nil {
		return false, "worker_snapshot_missing"
	}
	view := s.View(now, heartbeatThreshold)
	if view == nil {
		return false, "worker_view_missing"
	}
	if view.HeartbeatStale {
		return false, "worker_heartbeat_stale"
	}
	if !view.Alive {
		return false, "worker_not_alive"
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "idle":
		return true, ""
	case "blocked_trust":
		return false, "worker_blocked_trust"
	case "blocked_auth":
		return false, "worker_blocked_auth"
	case "blocked_quota":
		return false, "worker_blocked_quota"
	case "failed", "stopped", "exited", "closed", "stale":
		return false, "worker_" + strings.ToLower(strings.TrimSpace(view.State))
	case "running":
		return false, "worker_busy"
	case "starting", "":
		return false, "worker_starting"
	default:
		return false, "worker_not_ready"
	}
}

func parseWorkerTimestamp(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			utc := ts.UTC()
			return &utc
		}
	}
	return nil
}
