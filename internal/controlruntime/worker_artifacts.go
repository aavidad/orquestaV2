package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	workerStatusStarting     = "starting"
	workerStatusReady        = "ready"
	workerStatusRunning      = "running"
	workerStatusBlockedTrust = "blocked_trust"
	workerStatusBlockedAuth  = "blocked_auth"
	workerStatusBlockedQuota = "blocked_quota"
	workerStatusStopped      = "stopped"
	workerStatusFailed       = "failed"
)

type workerArtifactPaths struct {
	ManifestPath  string
	StatusPath    string
	HeartbeatPath string
}

type workerManifest struct {
	Version              int    `json:"version"`
	Agent                string `json:"agent"`
	Project              string `json:"project,omitempty"`
	Driver               string `json:"driver"`
	Transport            string `json:"transport"`
	Profile              string `json:"profile,omitempty"`
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

type workerStatus struct {
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

type workerHeartbeat struct {
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

func workerArtifactsPaths(runDir string) workerArtifactPaths {
	runDir = strings.TrimSpace(runDir)
	return workerArtifactPaths{
		ManifestPath:  filepath.Join(runDir, "manifest.json"),
		StatusPath:    filepath.Join(runDir, "status.json"),
		HeartbeatPath: filepath.Join(runDir, "heartbeat.json"),
	}
}

func writeWorkerManifestFile(path string, payload workerManifest) error {
	payload.Version = 1
	return writeJSONAtomic(path, payload)
}

func writeWorkerStatusFile(path string, payload workerStatus) error {
	return writeJSONAtomic(path, payload)
}

func writeWorkerHeartbeatFile(path string, payload workerHeartbeat) error {
	return writeJSONAtomic(path, payload)
}

func writeJSONAtomic(path string, payload any) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmpPath := path + ".tmp-" + time.Now().UTC().Format("20060102150405.000000000")
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
