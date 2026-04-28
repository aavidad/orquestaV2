package controlruntime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

const tmuxMonitorRepairHeartbeatThreshold = 45 * time.Second

func EnsureTMUXMonitorFromMetadataJSON(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, false, nil
	}
	meta := metadataMap(raw)
	if !metadataLooksLikeTMUXRuntime(meta) {
		return raw, false, nil
	}

	tmuxCommand := strings.TrimSpace(stringValueFromMetadata(meta, "tmux_command"))
	if tmuxCommand == "" {
		path, err := exec.LookPath("tmux")
		if err != nil {
			return raw, false, nil
		}
		tmuxCommand = path
		meta["tmux_command"] = tmuxCommand
	}

	metaRaw, err := json.Marshal(meta)
	if err != nil {
		return raw, false, err
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(string(metaRaw))
	if err != nil {
		return raw, false, err
	}

	currentOwner := extractRemoteInt(meta, "supervisor_owner_pid") == os.Getpid()
	monitorPID := extractRemoteInt(meta, "tmux_monitor_pid")
	monitorKnown := monitorPID > 0
	monitorAlive := false
	if monitorKnown {
		if alive, err := procesoVivoPID(monitorPID); err == nil {
			monitorAlive = alive
		}
	}
	if monitorAlive {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}

	if !monitorKnown && currentOwner && snap != nil && snap.Alive() && !snap.IsHeartbeatStale(time.Now().UTC(), tmuxMonitorRepairHeartbeatThreshold) {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}

	sessionName := strings.TrimSpace(stringValueFromMetadata(meta, "tmux_session"))
	windowName := strings.TrimSpace(stringValueFromMetadata(meta, "tmux_window"))
	paneID := strings.TrimSpace(stringValueFromMetadata(meta, "tmux_pane_id"))
	if snap != nil && snap.Manifest != nil {
		if sessionName == "" {
			sessionName = strings.TrimSpace(snap.Manifest.TmuxSession)
		}
		if windowName == "" {
			windowName = strings.TrimSpace(snap.Manifest.TmuxWindow)
		}
		if paneID == "" {
			paneID = strings.TrimSpace(snap.Manifest.TmuxPaneID)
		}
	}
	if sessionName == "" && paneID == "" {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}

	var paneSnap *tmuxPaneSnapshot
	if paneID != "" {
		paneSnap, err = consultarTmuxPane(embeddedTmuxMonitorSpec{
			TmuxCommand: tmuxCommand,
			PaneID:      paneID,
		})
		if err != nil && sessionName != "" && !tmuxSessionExists(tmuxCommand, sessionName) {
			if string(metaRaw) == raw {
				return raw, false, nil
			}
			return string(metaRaw), false, nil
		}
	} else if sessionName != "" && !tmuxSessionExists(tmuxCommand, sessionName) {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}
	if paneSnap == nil {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}
	if paneSnap.PaneDead || paneSnap.PanePID <= 0 {
		if string(metaRaw) == raw {
			return raw, false, nil
		}
		return string(metaRaw), false, nil
	}
	if sessionName == "" {
		sessionName = strings.TrimSpace(paneSnap.SessionName)
	}
	if windowName == "" {
		windowName = strings.TrimSpace(paneSnap.WindowName)
	}
	if paneID == "" {
		paneID = strings.TrimSpace(paneSnap.PaneID)
	}

	runDir := tmuxMonitorRunDir(meta, snap)
	if runDir == "" {
		return raw, false, nil
	}
	runtimeManifestPath := strings.TrimSpace(stringValueFromMetadata(meta, "trace_manifest"))
	if runtimeManifestPath == "" && snap != nil && snap.Manifest != nil {
		runtimeManifestPath = strings.TrimSpace(snap.Manifest.RuntimeManifestPath)
	}
	if runtimeManifestPath == "" {
		runtimeManifestPath = filepath.Join(runDir, "runtime.json")
	}
	statusPath, heartbeatPath, manifestPath := tmuxMonitorWorkerPaths(meta, snap)
	if statusPath == "" || heartbeatPath == "" || manifestPath == "" {
		return raw, false, nil
	}

	command := strings.TrimSpace(stringValueFromMetadata(meta, "rendered_command"))
	workingDir := strings.TrimSpace(stringValueFromMetadata(meta, "working_dir"))
	logPath := strings.TrimSpace(stringValueFromMetadata(meta, "log_path"))
	externalSessionID := strings.TrimSpace(stringValueFromMetadata(meta, "external_session_id"))
	mailboxDeliveryMode := strings.TrimSpace(stringValueFromMetadata(meta, "mailbox_delivery_mode"))
	if snap != nil && snap.Manifest != nil {
		if command == "" {
			command = strings.TrimSpace(snap.Manifest.Command)
		}
		if workingDir == "" {
			workingDir = strings.TrimSpace(snap.Manifest.WorkingDir)
		}
		if logPath == "" {
			logPath = strings.TrimSpace(snap.Manifest.LogPath)
		}
		if externalSessionID == "" {
			externalSessionID = strings.TrimSpace(snap.Manifest.ExternalSessionID)
		}
		if mailboxDeliveryMode == "" {
			mailboxDeliveryMode = strings.TrimSpace(snap.Manifest.MailboxDeliveryMode)
		}
	}

	monitorPID, err = launchEmbeddedTmuxMonitor(embeddedTmuxMonitorSpec{
		TmuxCommand:         tmuxCommand,
		SessionName:         sessionName,
		WindowName:          windowName,
		PaneID:              paneID,
		ChildPID:            paneSnap.PanePID,
		Agent:               strings.TrimSpace(stringValueFromMetadata(meta, "agente")),
		Project:             strings.TrimSpace(stringValueFromMetadata(meta, "proyecto")),
		WorkingDir:          workingDir,
		Command:             command,
		LogPath:             logPath,
		RuntimeManifestPath: runtimeManifestPath,
		StatusPath:          statusPath,
		HeartbeatPath:       heartbeatPath,
		ExternalSessionID:   externalSessionID,
		MailboxDeliveryMode: mailboxDeliveryMode,
		OwnerPID:            os.Getpid(),
	}, runDir)
	if err != nil {
		return raw, false, err
	}

	meta["tmux_command"] = tmuxCommand
	meta["tmux_session"] = sessionName
	meta["tmux_window"] = windowName
	meta["tmux_pane_id"] = paneID
	meta["tmux_monitor_pid"] = monitorPID
	meta["supervisor_owner_pid"] = os.Getpid()
	meta["worker_schema_version"] = runtimeagente.WorkerMetadataSchemaVersion
	meta["worker_manifest_path"] = manifestPath
	meta["worker_status_path"] = statusPath
	meta["worker_heartbeat_path"] = heartbeatPath
	meta["tmux_monitor_restarted_at"] = time.Now().UTC().Format(time.RFC3339Nano)

	out, err := json.Marshal(meta)
	if err != nil {
		return raw, false, err
	}
	return string(out), true, nil
}

func tmuxSessionExists(tmuxCommand, sessionName string) bool {
	tmuxCommand = strings.TrimSpace(tmuxCommand)
	sessionName = strings.TrimSpace(sessionName)
	if tmuxCommand == "" || sessionName == "" {
		return false
	}
	return tmuxRunCommand(tmuxCommand, "has-session", "-t", sessionName) == nil
}

func tmuxMonitorRunDir(meta map[string]any, snap *runtimeagente.WorkerSnapshot) string {
	if runDir := strings.TrimSpace(stringValueFromMetadata(meta, "trace_dir")); runDir != "" {
		return runDir
	}
	for _, candidate := range []string{
		strings.TrimSpace(stringValueFromMetadata(meta, "worker_manifest_path")),
		strings.TrimSpace(stringValueFromMetadata(meta, "worker_status_path")),
		strings.TrimSpace(stringValueFromMetadata(meta, "worker_heartbeat_path")),
	} {
		if candidate != "" {
			return filepath.Dir(candidate)
		}
	}
	if snap != nil {
		for _, candidate := range []string{
			strings.TrimSpace(snap.ManifestPath),
			strings.TrimSpace(snap.StatusPath),
			strings.TrimSpace(snap.HeartbeatPath),
		} {
			if candidate != "" {
				return filepath.Dir(candidate)
			}
		}
	}
	return ""
}

func tmuxMonitorWorkerPaths(meta map[string]any, snap *runtimeagente.WorkerSnapshot) (string, string, string) {
	manifestPath := strings.TrimSpace(stringValueFromMetadata(meta, "worker_manifest_path"))
	statusPath := strings.TrimSpace(stringValueFromMetadata(meta, "worker_status_path"))
	heartbeatPath := strings.TrimSpace(stringValueFromMetadata(meta, "worker_heartbeat_path"))
	if snap != nil {
		if manifestPath == "" {
			manifestPath = strings.TrimSpace(snap.ManifestPath)
		}
		if statusPath == "" {
			statusPath = strings.TrimSpace(snap.StatusPath)
		}
		if heartbeatPath == "" {
			heartbeatPath = strings.TrimSpace(snap.HeartbeatPath)
		}
		if snap.Manifest != nil {
			if statusPath == "" {
				statusPath = strings.TrimSpace(snap.Manifest.StatusPath)
			}
			if heartbeatPath == "" {
				heartbeatPath = strings.TrimSpace(snap.Manifest.HeartbeatPath)
			}
		}
	}
	return statusPath, heartbeatPath, manifestPath
}
