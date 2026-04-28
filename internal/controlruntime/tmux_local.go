package controlruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

func arrancarPlanLocalTMUX(req SolicitudArranque) (*ProcesoArrancado, error) {
	startedAt := time.Now().UTC()
	runDir := runtimeArtifactsRunDir(req, startedAt)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, err
	}

	tmuxCommand, err := tmuxCommandPath(req)
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(runDir, "tmux.log")
	manifestPath := filepath.Join(runDir, "runtime.json")
	workerPaths := workerArtifactsPaths(runDir)

	if req.Plan != nil {
		req.Plan.Comando = absolutizarComandoPlanLocal(strings.TrimSpace(req.Plan.Comando))
	}
	rendered := runtimeagente.RenderCommand(req.Plan)
	commandLine := "exec " + shellQuoteSimple(strings.TrimSpace(req.Plan.Comando))
	if joined := strings.TrimSpace(shellJoinQuoted(req.Plan.Args)); joined != "" {
		commandLine += " " + joined
	}
	profileWrapper, profileName, hasProfileStatus := codexProfileCommandFromCandidates(rendered, commandLine)

	sessionName := buildTmuxSessionName(req, startedAt)
	windowName := buildTmuxWindowName(req)
	paneInfo, err := startTmuxSession(tmuxCommand, sessionName, windowName, strings.TrimSpace(req.Plan.WorkingDir), commandLine, req.Plan.Env)
	if err != nil {
		_ = writeWorkerStatusFile(workerPaths.StatusPath, workerStatus{
			State:     workerStatusFailed,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Alive:     false,
			Agent:     strings.TrimSpace(req.Agente),
			Project:   strings.TrimSpace(req.Proyecto),
			LogPath:   logPath,
			ExitError: strings.TrimSpace(err.Error()),
		})
		return nil, err
	}

	if err := tmuxPipePane(tmuxCommand, paneInfo.PaneID, logPath); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}
	if err := tmuxSeedLogFromPaneIfEmpty(tmuxCommand, paneInfo.PaneID, logPath); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}

	canSendInput, canSendInputSource := localPTYInputPolicy(req.Plan)
	mailboxDeliveryMode := localPTYMailboxDeliveryMode(req.Plan, canSendInput)
	externalSessionID := ""
	if renderedCommandLooksLikeCodexCLI(rendered) {
		externalSessionID, _ = detectCodexSessionID(rendered, strings.TrimSpace(req.Plan.WorkingDir), startedAt, time.Now().UTC())
	}
	if mailboxDeliveryMode == runtimeagente.MailboxDeliverySessionResume && strings.TrimSpace(externalSessionID) == "" {
		mailboxDeliveryMode = runtimeagente.MailboxDeliveryBootstrapOnly
	}

	supervisorRef := runDir
	if err := writeRuntimeTraceManifestTMUX(manifestPath, req, startedAt, paneInfo.PanePID, logPath, rendered, commandLine, canSendInput, canSendInputSource, mailboxDeliveryMode, externalSessionID, supervisorRef, tmuxCommand, paneInfo); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}
	if err := writeWorkerManifestFile(workerPaths.ManifestPath, workerManifest{
		Agent:                strings.TrimSpace(req.Agente),
		Project:              strings.TrimSpace(req.Proyecto),
		Driver:               "tmux_cli_session",
		Transport:            "tmux",
		Profile:              strings.TrimSpace(profileName),
		ExecutionProfile:     strings.TrimSpace(req.Plan.PerfilOperativo),
		ProfileStatusWrapper: strings.TrimSpace(profileWrapper),
		TmuxSession:          strings.TrimSpace(paneInfo.SessionName),
		TmuxWindow:           strings.TrimSpace(paneInfo.WindowName),
		TmuxPaneID:           strings.TrimSpace(paneInfo.PaneID),
		CreatedAt:            startedAt.Format(time.RFC3339Nano),
		StartedAt:            startedAt.Format(time.RFC3339Nano),
		ChildPID:             paneInfo.PanePID,
		WorkingDir:           strings.TrimSpace(req.Plan.WorkingDir),
		Command:              rendered,
		RuntimeManifestPath:  manifestPath,
		StatusPath:           workerPaths.StatusPath,
		HeartbeatPath:        workerPaths.HeartbeatPath,
		LogPath:              logPath,
		ExternalSessionID:    strings.TrimSpace(externalSessionID),
		MailboxDeliveryMode:  strings.TrimSpace(mailboxDeliveryMode),
		CanSendInput:         canSendInput,
	}); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}
	if err := writeWorkerStatusFile(workerPaths.StatusPath, workerStatus{
		State:               workerStatusStarting,
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339Nano),
		Alive:               true,
		ChildPID:            paneInfo.PanePID,
		Agent:               strings.TrimSpace(req.Agente),
		Project:             strings.TrimSpace(req.Proyecto),
		WorkingDir:          strings.TrimSpace(req.Plan.WorkingDir),
		LogPath:             logPath,
		ExternalSessionID:   strings.TrimSpace(externalSessionID),
		MailboxDeliveryMode: strings.TrimSpace(mailboxDeliveryMode),
	}); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}
	if err := writeWorkerHeartbeatFile(workerPaths.HeartbeatPath, workerHeartbeat{
		Alive:             true,
		HeartbeatAt:       time.Now().UTC().Format(time.RFC3339Nano),
		StartedAt:         startedAt.Format(time.RFC3339Nano),
		ChildPID:          paneInfo.PanePID,
		Agent:             strings.TrimSpace(req.Agente),
		Project:           strings.TrimSpace(req.Proyecto),
		ExternalSessionID: strings.TrimSpace(externalSessionID),
	}); err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}

	monitorSpec := embeddedTmuxMonitorSpec{
		TmuxCommand:         tmuxCommand,
		SessionName:         paneInfo.SessionName,
		WindowName:          paneInfo.WindowName,
		PaneID:              paneInfo.PaneID,
		ChildPID:            paneInfo.PanePID,
		Agent:               strings.TrimSpace(req.Agente),
		Project:             strings.TrimSpace(req.Proyecto),
		WorkingDir:          strings.TrimSpace(req.Plan.WorkingDir),
		StartedAt:           startedAt.Format(time.RFC3339Nano),
		Command:             rendered,
		LogPath:             logPath,
		RuntimeManifestPath: manifestPath,
		StatusPath:          workerPaths.StatusPath,
		HeartbeatPath:       workerPaths.HeartbeatPath,
		ExternalSessionID:   strings.TrimSpace(externalSessionID),
		MailboxDeliveryMode: strings.TrimSpace(mailboxDeliveryMode),
		OwnerPID:            os.Getpid(),
	}
	monitorPID, err := launchEmbeddedTmuxMonitor(monitorSpec, runDir)
	if err != nil {
		_ = tmuxKillSession(tmuxCommand, sessionName)
		return nil, err
	}
	if tmuxStartShouldWaitForReady(rendered) {
		timeout := req.TimeoutReady
		if timeout <= 0 {
			timeout = tmuxStartReadyTimeout()
		}
		ready, readyErr := waitForTMUXPaneReady(tmuxCommand, paneInfo.PaneID, timeout)
		if readyErr != nil {
			_ = tmuxKillSession(tmuxCommand, sessionName)
			return nil, readyErr
		}
		if ready {
			_ = writeEmbeddedTmuxWorkerSnapshot(monitorSpec, workerStatusReady, true, "", nil, paneInfo.PanePID, time.Now().UTC(), time.Time{}, time.Time{}, strings.TrimSpace(monitorSpec.WorkingDir))
		}
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"worker_schema_version": runtimeagente.WorkerMetadataSchemaVersion,
		"agente":                strings.TrimSpace(req.Agente),
		"proyecto":              strings.TrimSpace(req.Proyecto),
		"log_path":              logPath,
		"trace_dir":             runDir,
		"trace_manifest":        manifestPath,
		"started_at":            startedAt.Format(time.RFC3339Nano),
		"working_dir":           strings.TrimSpace(req.Plan.WorkingDir),
		"wrapped_command":       commandLine,
		"rendered_command":      rendered,
		"driver":                "tmux_cli_session",
		"transporte":            "tmux",
		"supervisor_ref":        supervisorRef,
		"supervisor_driver":     "tmux_runtime_monitor",
		"supervision_mode":      supervisionModoAdjunto,
		"supervisor_owner_pid":  os.Getpid(),
		"tmux_command":          tmuxCommand,
		"tmux_session":          paneInfo.SessionName,
		"tmux_window":           paneInfo.WindowName,
		"tmux_pane_id":          paneInfo.PaneID,
		"tmux_monitor_pid":      monitorPID,
		"worker_manifest_path":  workerPaths.ManifestPath,
		"worker_status_path":    workerPaths.StatusPath,
		"worker_heartbeat_path": workerPaths.HeartbeatPath,
		"can_send_input":        canSendInput,
		"can_send_input_source": canSendInputSource,
		"mailbox_delivery_mode": mailboxDeliveryMode,
		"external_session_id":   externalSessionID,
		"execution_profile":     strings.TrimSpace(req.Plan.PerfilOperativo),
		"perfil_operativo":      strings.TrimSpace(req.Plan.PerfilOperativo),
		"profile_name":          strings.TrimSpace(profileName),
		"profile_status_wrapper": func() string {
			if !hasProfileStatus {
				return ""
			}
			return strings.TrimSpace(profileWrapper)
		}(),
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        canSendInput,
		"can_checkpoint":        true,
		"can_resume":            true,
		"can_capture_pid":       paneInfo.PanePID > 0,
		"can_track_continuity":  true,
		"can_pause":             paneInfo.PanePID > 0,
		"can_stop":              true,
		"mailbox_delivery_mode": mailboxDeliveryMode,
	})
	_ = publicarLogAgente(req, logPath)

	return &ProcesoArrancado{
		PID:               paneInfo.PanePID,
		HandleKind:        "session",
		HandleRef:         tmuxSessionHandleRef(paneInfo.SessionName, paneInfo.PaneID),
		ExternalSessionID: externalSessionID,
		LogPath:           logPath,
		WorkingDir:        strings.TrimSpace(req.Plan.WorkingDir),
		WrappedCommand:    commandLine,
		RenderedCommand:   rendered,
		MetadataJSON:      string(metaJSON),
		CapabilitiesJSON:  string(capsJSON),
	}, nil
}

func tmuxStartShouldWaitForReady(rendered string) bool {
	lower := strings.ToLower(strings.TrimSpace(rendered))
	if lower == "" {
		return true
	}
	if renderedCommandLooksLikeCodexCLI(rendered) {
		return false
	}
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = strings.Trim(fields[0], "'\"")
	}
	base := filepath.Base(first)
	switch {
	case strings.Contains(lower, "claude-perfil"),
		strings.Contains(lower, "claude-code"),
		lower == "claude",
		strings.HasPrefix(lower, "claude "),
		base == "claude",
		base == "claude-code",
		strings.Contains(lower, "gemini-perfil"),
		lower == "gemini",
		strings.HasPrefix(lower, "gemini "),
		base == "gemini":
		return false
	default:
		return true
	}
}

func tmuxSessionHandleRef(sessionName, paneID string) string {
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	switch {
	case sessionName != "" && paneID != "":
		return sessionName + "/" + paneID
	case sessionName != "":
		return sessionName
	default:
		return paneID
	}
}

type tmuxPaneInfo struct {
	SessionName string
	WindowName  string
	PaneID      string
	PanePID     int
}

func tmuxCommandPath(req SolicitudArranque) (string, error) {
	if req.Plan != nil && req.Plan.Env != nil {
		if path := strings.TrimSpace(req.Plan.Env["ORQUESTA_TMUX_BIN"]); path != "" {
			return path, nil
		}
	}
	if path := strings.TrimSpace(os.Getenv("ORQUESTA_TMUX_BIN")); path != "" {
		return path, nil
	}
	return exec.LookPath("tmux")
}

func buildTmuxSessionName(req SolicitudArranque, startedAt time.Time) string {
	return fmt.Sprintf("orq-%s-%s", sanitizePathFragment(req.Agente), strings.ToLower(startedAt.UTC().Format("150405")))
}

func buildTmuxWindowName(req SolicitudArranque) string {
	if proyecto := sanitizePathFragment(strings.TrimSpace(req.Proyecto)); proyecto != "" {
		return proyecto
	}
	return "worker"
}

func startTmuxSession(tmuxCommand, sessionName, windowName, workingDir, command string, env map[string]string) (*tmuxPaneInfo, error) {
	args := []string{"new-session", "-d", "-P", "-F", "#{session_name}|#{window_name}|#{pane_id}|#{pane_pid}", "-s", sessionName, "-n", windowName}
	if strings.TrimSpace(workingDir) != "" {
		args = append(args, "-c", workingDir)
	}
	args = append(args, "bash", "-lc", command)
	ctx, cancel := context.WithTimeout(context.Background(), tmuxStartCommandTimeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, tmuxCommand, args...)
	cmd.Env = append([]string(nil), os.Environ()...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			detail := strings.TrimSpace(string(out))
			if detail != "" {
				return nil, fmt.Errorf("timeout arrancando tmux: %s", detail)
			}
			return nil, fmt.Errorf("timeout arrancando tmux")
		}
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("salida tmux invalida: %q", strings.TrimSpace(string(out)))
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	return &tmuxPaneInfo{
		SessionName: strings.TrimSpace(parts[0]),
		WindowName:  strings.TrimSpace(parts[1]),
		PaneID:      strings.TrimSpace(parts[2]),
		PanePID:     pid,
	}, nil
}

func tmuxStartCommandTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("ORQUESTA_TMUX_START_TIMEOUT_MS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return 15 * time.Second
}

func tmuxPipePane(tmuxCommand, paneID, logPath string) error {
	command := fmt.Sprintf("cat >> %s", shellQuoteSimple(strings.TrimSpace(logPath)))
	return tmuxRunCommand(tmuxCommand, "pipe-pane", "-O", "-t", strings.TrimSpace(paneID), command)
}

func tmuxSeedLogFromPaneIfEmpty(tmuxCommand, paneID, logPath string) error {
	info, err := os.Stat(strings.TrimSpace(logPath))
	if err == nil && info != nil && info.Size() > 0 {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	out, err := tmuxOutputCommand(strings.TrimSpace(tmuxCommand), "capture-pane", "-t", strings.TrimSpace(paneID), "-p")
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return nil
	}
	fd, err := os.OpenFile(strings.TrimSpace(logPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer fd.Close()
	if _, err := fd.Write(out); err != nil {
		return err
	}
	if out[len(out)-1] != '\n' {
		if _, err := fd.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func tmuxKillSession(tmuxCommand, sessionName string) error {
	if strings.TrimSpace(sessionName) == "" {
		return nil
	}
	return tmuxRunCommand(tmuxCommand, "kill-session", "-t", strings.TrimSpace(sessionName))
}

func writeRuntimeTraceManifestTMUX(path string, req SolicitudArranque, startedAt time.Time, pid int, logPath, rendered, wrapped string, canSendInput bool, canSendInputSource, mailboxDeliveryMode, externalSessionID, supervisorRef, tmuxCommand string, paneInfo *tmuxPaneInfo) error {
	payload := map[string]any{
		"created_at":            startedAt.Format(time.RFC3339Nano),
		"agente":                strings.TrimSpace(req.Agente),
		"proyecto":              strings.TrimSpace(req.Proyecto),
		"working_dir":           strings.TrimSpace(renderedWorkingDir(req)),
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"pid":                   pid,
		"log_path":              logPath,
		"rendered_command":      rendered,
		"wrapped_command":       wrapped,
		"can_send_input":        canSendInput,
		"can_send_input_source": strings.TrimSpace(canSendInputSource),
		"mailbox_delivery_mode": strings.TrimSpace(mailboxDeliveryMode),
		"external_session_id":   strings.TrimSpace(externalSessionID),
		"execution_profile":     strings.TrimSpace(req.Plan.PerfilOperativo),
		"perfil_operativo":      strings.TrimSpace(req.Plan.PerfilOperativo),
		"supervisor_ref":        strings.TrimSpace(supervisorRef),
		"supervisor_driver":     "tmux_runtime_monitor",
		"supervision_mode":      supervisionModoAdjunto,
		"tmux_command":          strings.TrimSpace(tmuxCommand),
	}
	if paneInfo != nil {
		payload["tmux_session"] = strings.TrimSpace(paneInfo.SessionName)
		payload["tmux_window"] = strings.TrimSpace(paneInfo.WindowName)
		payload["tmux_pane_id"] = strings.TrimSpace(paneInfo.PaneID)
	}
	if paths := workerArtifactsPaths(filepath.Dir(path)); strings.TrimSpace(paths.ManifestPath) != "" {
		payload["worker_manifest_path"] = paths.ManifestPath
		payload["worker_status_path"] = paths.StatusPath
		payload["worker_heartbeat_path"] = paths.HeartbeatPath
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
