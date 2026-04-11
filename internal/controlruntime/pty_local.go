package controlruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"orquesta/runtimeagente"
)

var ttyInstructionASCIIReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
	"Á", "A", "À", "A", "Ä", "A", "Â", "A",
	"É", "E", "È", "E", "Ë", "E", "Ê", "E",
	"Í", "I", "Ì", "I", "Ï", "I", "Î", "I",
	"Ó", "O", "Ò", "O", "Ö", "O", "Ô", "O",
	"Ú", "U", "Ù", "U", "Ü", "U", "Û", "U",
	"Ñ", "N", "Ç", "C",
	"¿", "",
	"¡", "",
	"“", "\"",
	"”", "\"",
	"’", "'",
	"—", "-",
	"–", "-",
)

func arrancarPlanLocalPTY(req SolicitudArranque) (*ProcesoArrancado, error) {
	startedAt := time.Now().UTC()
	runDir := runtimeArtifactsRunDir(req, startedAt)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, err
	}

	stdinPath := filepath.Join(runDir, "pty.stdin")
	stdinRawPath := filepath.Join(runDir, "pty.stdin.raw")
	logPath := filepath.Join(runDir, "pty.log")
	manifestPath := filepath.Join(runDir, "runtime.json")
	workerPaths := workerArtifactsPaths(runDir)
	brokerSpecPath := filepath.Join(runDir, "pty-broker.json")
	brokerReadyPath := filepath.Join(runDir, "pty-ready.json")

	if err := syscall.Mkfifo(stdinPath, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(stdinRawPath, nil, 0o600); err != nil {
		_ = os.Remove(stdinPath)
		return nil, err
	}

	if req.Plan != nil {
		req.Plan.Comando = absolutizarComandoPlanLocal(strings.TrimSpace(req.Plan.Comando))
	}
	rendered := runtimeagente.RenderCommand(req.Plan)
	brokerCommand, brokerArgs, err := embeddedBrokerInvocation(brokerSpecPath)
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	spec := embeddedBrokerSpec{
		Command:             strings.TrimSpace(req.Plan.Comando),
		Args:                append([]string(nil), req.Plan.Args...),
		Env:                 map[string]string{},
		WorkingDir:          strings.TrimSpace(req.Plan.WorkingDir),
		StdinPath:           stdinPath,
		LogPath:             logPath,
		ReadyPath:           brokerReadyPath,
		Agent:               strings.TrimSpace(req.Agente),
		Project:             strings.TrimSpace(req.Proyecto),
		MailboxDeliveryMode: localPTYMailboxDeliveryMode(req.Plan, localPTYCanSendInput(req.Plan)),
		CanSendInput:        localPTYCanSendInput(req.Plan),
		StatusPath:          workerPaths.StatusPath,
		HeartbeatPath:       workerPaths.HeartbeatPath,
	}
	for k, v := range req.Plan.Env {
		spec.Env[k] = v
	}
	specData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	specData = append(specData, '\n')
	if err := os.WriteFile(brokerSpecPath, specData, 0o600); err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}

	wrapped := fmt.Sprintf("%s %s", shellQuoteSimple(brokerCommand), strings.TrimSpace(shellJoinQuoted(brokerArgs)))
	cmd := exec.Command(brokerCommand, brokerArgs...)
	cmd.Dir = strings.TrimSpace(req.Plan.WorkingDir)
	if cmd.Dir != "" {
		if err := os.MkdirAll(cmd.Dir, 0o700); err != nil {
			_ = os.Remove(stdinPath)
			_ = os.Remove(stdinRawPath)
			return nil, err
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	brokerStdoutPath := filepath.Join(runDir, "pty-broker.stdout")
	brokerStderrPath := filepath.Join(runDir, "pty-broker.stderr")
	brokerStdout, err := os.OpenFile(brokerStdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	defer brokerStdout.Close()
	brokerStderr, err := os.OpenFile(brokerStderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	defer brokerStderr.Close()
	cmd.Stdout = brokerStdout
	cmd.Stderr = brokerStderr

	env := append([]string(nil), os.Environ()...)
	if agente := strings.TrimSpace(req.Agente); agente != "" {
		env = append(env, "ORQUESTA_AGENTE="+agente)
	}
	if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
		env = append(env, "ORQUESTA_PROYECTO="+proyecto)
	}
	for k, v := range req.Plan.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	brokerPID := cmd.Process.Pid
	childPID, err := esperarBrokerReady(cmd, brokerReadyPath, workerPaths.StatusPath, brokerStderrPath, 15*time.Second)
	if err != nil {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: intPtr64(int64(cmd.Process.Pid))})
		return nil, err
	}

	canSendInput, canSendInputSource := localPTYInputPolicy(req.Plan)
	canRunSlashCommands := localPTYCanRunSlashCommands(req.Plan)
	mailboxDeliveryMode := localPTYMailboxDeliveryMode(req.Plan, canSendInput)
	profileWrapper, profileName, hasProfileStatus := codexProfileCommandFromCandidates(rendered, wrapped)
	externalSessionID := ""
	if renderedCommandLooksLikeCodexCLI(rendered) {
		externalSessionID, _ = detectCodexSessionID(rendered, cmd.Dir, startedAt, time.Now().UTC())
	}
	spec.ExternalSessionID = externalSessionID
	supervisorRef := runDir
	metaJSON, _ := json.Marshal(map[string]any{
		"agente":                 strings.TrimSpace(req.Agente),
		"proyecto":               strings.TrimSpace(req.Proyecto),
		"stdin_path":             stdinPath,
		"stdin_raw_path":         stdinRawPath,
		"log_path":               logPath,
		"trace_dir":              runDir,
		"trace_manifest":         manifestPath,
		"started_at":             startedAt.Format(time.RFC3339Nano),
		"working_dir":            cmd.Dir,
		"wrapped_command":        wrapped,
		"rendered_command":       rendered,
		"driver":                 "process_pty_cli",
		"broker_pid":             brokerPID,
		"supervisor_ref":         supervisorRef,
		"supervision_mode":       supervisionModoResidente,
		"supervisor_owner_pid":   os.Getpid(),
		"broker_spec_path":       brokerSpecPath,
		"broker_ready_path":      brokerReadyPath,
		"broker_stdout_path":     brokerStdoutPath,
		"broker_stderr_path":     brokerStderrPath,
		"worker_manifest_path":   workerPaths.ManifestPath,
		"worker_status_path":     workerPaths.StatusPath,
		"worker_heartbeat_path":  workerPaths.HeartbeatPath,
		"can_send_input":         canSendInput,
		"can_run_slash_commands": canRunSlashCommands,
		"can_send_input_source":  canSendInputSource,
		"mailbox_delivery_mode":  mailboxDeliveryMode,
		"external_session_id":    externalSessionID,
		"profile_name":           strings.TrimSpace(profileName),
		"profile_status_wrapper": func() string {
			if !hasProfileStatus {
				return ""
			}
			return strings.TrimSpace(profileWrapper)
		}(),
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":         canSendInput,
		"can_run_slash_commands": canRunSlashCommands,
		"can_checkpoint":         true,
		"can_resume":             true,
		"can_capture_pid":        true,
		"can_track_continuity":   true,
		"can_pause":              true,
		"can_stop":               true,
		"mailbox_delivery_mode":  mailboxDeliveryMode,
	})
	if err := writeRuntimeTraceManifest(manifestPath, req, startedAt, childPID, brokerPID, stdinPath, stdinRawPath, logPath, rendered, wrapped, canSendInput, canRunSlashCommands, canSendInputSource, mailboxDeliveryMode, externalSessionID, supervisorRef, supervisionModoResidente); err != nil {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: intPtr64(int64(childPID))})
		return nil, err
	}
	if err := writeWorkerManifestFile(workerPaths.ManifestPath, workerManifest{
		Agent:                strings.TrimSpace(req.Agente),
		Project:              strings.TrimSpace(req.Proyecto),
		Driver:               "process_pty_cli",
		Transport:            "pty_broker",
		Profile:              strings.TrimSpace(profileName),
		ProfileStatusWrapper: strings.TrimSpace(profileWrapper),
		CreatedAt:            startedAt.Format(time.RFC3339Nano),
		StartedAt:            startedAt.Format(time.RFC3339Nano),
		ChildPID:             childPID,
		WorkingDir:           strings.TrimSpace(cmd.Dir),
		Command:              rendered,
		RuntimeManifestPath:  manifestPath,
		StatusPath:           workerPaths.StatusPath,
		HeartbeatPath:        workerPaths.HeartbeatPath,
		LogPath:              logPath,
		StdinPath:            stdinPath,
		ExternalSessionID:    strings.TrimSpace(externalSessionID),
		MailboxDeliveryMode:  strings.TrimSpace(mailboxDeliveryMode),
		CanSendInput:         canSendInput,
	}); err != nil {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: intPtr64(int64(childPID))})
		return nil, err
	}
	_ = publicarLogAgente(req, logPath)

	return &ProcesoArrancado{
		PID:               childPID,
		HandleKind:        "process",
		HandleRef:         fmt.Sprintf("%d", childPID),
		StdinPath:         stdinPath,
		StdinRawPath:      stdinRawPath,
		LogPath:           logPath,
		WorkingDir:        cmd.Dir,
		WrappedCommand:    wrapped,
		RenderedCommand:   rendered,
		ExternalSessionID: externalSessionID,
		MetadataJSON:      string(metaJSON),
		CapabilitiesJSON:  string(capsJSON),
	}, nil
}

func esperarBrokerReady(cmd *exec.Cmd, readyPath, statusPath, stderrPath string, timeout time.Duration) (int, error) {
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(readyPath); err == nil {
			var ready embeddedBrokerReady
			if json.Unmarshal(data, &ready) == nil && ready.ChildPID > 0 {
				return ready.ChildPID, nil
			}
		}
		select {
		case err := <-waitCh:
			if detalle := leerExitErrorWorkerStatus(statusPath); detalle != "" {
				return 0, fmt.Errorf("broker pty finalizo antes de publicar child_pid: %s", detalle)
			}
			if detalle := leerErrorBroker(stderrPath); detalle != "" {
				return 0, fmt.Errorf("broker pty finalizo antes de publicar child_pid: %s", detalle)
			}
			if err != nil {
				return 0, fmt.Errorf("broker pty finalizo antes de publicar child_pid: %w", err)
			}
			return 0, fmt.Errorf("broker pty finalizo antes de publicar child_pid")
		default:
		}
		time.Sleep(150 * time.Millisecond)
	}
	if detalle := leerExitErrorWorkerStatus(statusPath); detalle != "" {
		return 0, fmt.Errorf("timeout esperando handshake del broker pty: %s", detalle)
	}
	if detalle := leerErrorBroker(stderrPath); detalle != "" {
		return 0, fmt.Errorf("timeout esperando handshake del broker pty: %s", detalle)
	}
	return 0, fmt.Errorf("timeout esperando handshake del broker pty")
}

func absolutizarComandoPlanLocal(command string) string {
	command = strings.TrimSpace(command)
	if command == "" || filepath.IsAbs(command) {
		return command
	}
	if !strings.Contains(command, "/") && !strings.Contains(command, string(os.PathSeparator)) {
		return command
	}
	if abs, err := filepath.Abs(command); err == nil {
		return abs
	}
	return command
}

func leerErrorBroker(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	for i := 0; i < 10; i++ {
		data, err := os.ReadFile(path)
		if err == nil {
			if texto := strings.TrimSpace(string(data)); texto != "" {
				return texto
			}
		}
		if i < 9 {
			time.Sleep(25 * time.Millisecond)
		}
	}
	return ""
}

func leerExitErrorWorkerStatus(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	for i := 0; i < 10; i++ {
		data, err := os.ReadFile(path)
		if err == nil {
			var payload struct {
				ExitError string `json:"exit_error"`
			}
			if json.Unmarshal(data, &payload) == nil {
				if detalle := strings.TrimSpace(payload.ExitError); detalle != "" {
					return detalle
				}
			}
		}
		if i < 9 {
			time.Sleep(25 * time.Millisecond)
		}
	}
	return ""
}

func shellQuoteSimple(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

func shellJoinQuoted(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, shellQuoteSimple(part))
	}
	return strings.Join(out, " ")
}

// publicarLogAgente crea (o actualiza) un symlink en
// <workDir>/logs/<agente>/agent.log apuntando al pty.log de esta sesión,
// de forma que los logs queden organizados por nombre de agente.
func publicarLogAgente(req SolicitudArranque, logPath string) error {
	logPath = strings.TrimSpace(logPath)
	if logPath == "" {
		return nil
	}
	agente := sanitizePathFragment(strings.TrimSpace(req.Agente))
	if agente == "" {
		return nil
	}
	var baseDir string
	if req.Plan != nil {
		if wd := strings.TrimSpace(req.Plan.WorkingDir); wd != "" {
			baseDir = filepath.Join(wd, "logs", agente)
		}
	}
	if baseDir == "" {
		return nil
	}
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return err
	}
	linkPath := filepath.Join(baseDir, "agent.log")
	// Eliminar symlink anterior si existe (puede apuntar a sesión anterior).
	_ = os.Remove(linkPath)
	return os.Symlink(logPath, linkPath)
}

func runtimeArtifactsBaseDir(req SolicitudArranque) string {
	if req.Plan != nil {
		if workDir := strings.TrimSpace(req.Plan.WorkingDir); workDir != "" {
			return filepath.Join(workDir, ".orquesta-runtime", sanitizePathFragment(req.Agente))
		}
	}
	return filepath.Join(os.TempDir(), "orquesta-runtime", sanitizePathFragment(req.Agente))
}

func runtimeArtifactsRunDir(req SolicitudArranque, now time.Time) string {
	return filepath.Join(runtimeArtifactsBaseDir(req), runtimeArtifactsToken(now))
}

func runtimeArtifactsToken(now time.Time) string {
	now = now.UTC()
	return fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
}

func writeRuntimeTraceManifest(path string, req SolicitudArranque, startedAt time.Time, pid, brokerPID int, stdinPath, stdinRawPath, logPath, rendered, wrapped string, canSendInput, canRunSlashCommands bool, canSendInputSource, mailboxDeliveryMode, externalSessionID, supervisorRef, supervisionMode string) error {
	payload := map[string]any{
		"created_at":             startedAt.Format(time.RFC3339Nano),
		"agente":                 strings.TrimSpace(req.Agente),
		"proyecto":               strings.TrimSpace(req.Proyecto),
		"working_dir":            strings.TrimSpace(renderedWorkingDir(req)),
		"driver":                 "process_pty_cli",
		"pid":                    pid,
		"broker_pid":             brokerPID,
		"stdin_path":             stdinPath,
		"stdin_raw_path":         stdinRawPath,
		"log_path":               logPath,
		"rendered_command":       rendered,
		"wrapped_command":        wrapped,
		"can_send_input":         canSendInput,
		"can_run_slash_commands": canRunSlashCommands,
		"can_send_input_source":  strings.TrimSpace(canSendInputSource),
		"mailbox_delivery_mode":  strings.TrimSpace(mailboxDeliveryMode),
		"external_session_id":    strings.TrimSpace(externalSessionID),
		"supervisor_ref":         strings.TrimSpace(supervisorRef),
		"supervision_mode":       strings.TrimSpace(supervisionMode),
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

func renderedWorkingDir(req SolicitudArranque) string {
	if req.Plan == nil {
		return ""
	}
	return req.Plan.WorkingDir
}

func intPtr64(v int64) *int64 {
	return &v
}

func renderedCommandLooksLikeCodexCLI(rendered string) bool {
	lower := strings.ToLower(strings.TrimSpace(rendered))
	if lower == "" {
		return false
	}
	if lower == "codex" || strings.HasPrefix(lower, "codex ") || strings.Contains(lower, "codex-perfil") {
		return true
	}
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = fields[0]
	}
	first = strings.Trim(first, "'\"")
	base := filepath.Base(first)
	return base == "codex" || base == "codex-cli" || strings.HasPrefix(base, "codex-perfil")
}

func localPTYCanSendInput(plan *runtimeagente.LaunchPlan) bool {
	value, _ := localPTYInputPolicy(plan)
	return value
}

func localPTYCanRunSlashCommands(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return true
	}
	if plan.Env != nil {
		if raw, ok := plan.Env["ORQUESTA_DISABLE_SLASH_COMMANDS"]; ok && strings.EqualFold(strings.TrimSpace(raw), "1") {
			return false
		}
	}
	return true
}

func localPTYInputPolicy(plan *runtimeagente.LaunchPlan) (bool, string) {
	if plan == nil || plan.CanSendInput == nil {
		return true, "default"
	}
	return *plan.CanSendInput, "plan_override"
}

func localPTYMailboxDeliveryMode(plan *runtimeagente.LaunchPlan, canSendInput bool) string {
	if plan != nil {
		if mode := runtimeagente.NormalizeMailboxDeliveryMode(plan.MailboxDeliveryMode); mode != "" {
			return mode
		}
	}
	if canSendInput {
		return runtimeagente.MailboxDeliveryInteractive
	}
	return runtimeagente.MailboxDeliveryBootstrapOnly
}

func NormalizarInstruccionProceso(obj ObjetivoProceso, instruccion string) string {
	texto := strings.TrimSpace(instruccion)
	if texto == "" {
		return ""
	}
	// Para transporte TMUX: preservar UTF-8. En Ollama, el REPL interactivo
	// trata peor las instrucciones multilínea y puede dejarlas en composición
	// ("Press Enter to send"), así que ahí se compactan a una sola línea.
	// En el resto de runtimes TMUX se preserva la estructura multilínea.
	if metadataLooksLikeTMUXRuntime(metadataMap(obj.MetadataJSON)) {
		texto = sanitizarInstruccionTMUX(texto)
		if esRuntimeOllamaLocal(obj) {
			return compactarInstruccionTMUXOllama(texto)
		}
		return texto
	}
	if _, ok, _ := ResolverPID(obj); ok {
		texto = sanitizarInstruccionTTY(texto)
		if esRuntimeCodexLocal(obj) {
			return compactarInstruccionCodexTTY(texto)
		}
		return texto
	}
	return texto
}

// sanitizarInstruccionTMUX prepara texto para entrega vía tmux send-keys -l.
// A diferencia de sanitizarInstruccionTTY, preserva UTF-8 y la estructura
// multilínea, eliminando solo secuencias ANSI y caracteres de control reales
// que podrían interferir con la sesión tmux.
func sanitizarInstruccionTMUX(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	// Eliminar secuencias de escape ANSI.
	raw = ansiEscapePattern.ReplaceAllString(raw, "")
	// Eliminar caracteres de control (0x00-0x1F excepto \n y \t) y DEL (0x7F).
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7F:
			// descarta caracteres de control
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func esRuntimeCodexLocal(obj ObjetivoProceso) bool {
	meta := metadataMap(obj.MetadataJSON)
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(obj.MetadataJSON); err == nil && snap != nil && snap.Manifest != nil {
		if strings.TrimSpace(snap.Manifest.ProfileStatusWrapper) != "" && strings.TrimSpace(snap.Manifest.Profile) != "" {
			return true
		}
	}
	if strings.TrimSpace(stringValueFromMetadata(meta, "profile_status_wrapper")) != "" &&
		strings.TrimSpace(stringValueFromMetadata(meta, "profile_name")) != "" {
		return true
	}
	for _, candidate := range []string{
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
	} {
		if renderedCommandLooksLikeCodexCLI(candidate) {
			return true
		}
	}
	for _, candidate := range []string{
		stringValueFromMetadata(meta, "herramienta"),
		stringValueFromMetadata(meta, "conector"),
	} {
		text := strings.ToLower(strings.TrimSpace(candidate))
		if text == "" {
			continue
		}
		first := text
		if fields := strings.Fields(text); len(fields) > 0 {
			first = fields[0]
		}
		base := filepath.Base(first)
		if strings.Contains(text, "codex") || base == "codex" || base == "codex-cli" {
			return true
		}
	}
	return false
}

func esRuntimeOllamaLocal(obj ObjetivoProceso) bool {
	meta := metadataMap(obj.MetadataJSON)
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(obj.MetadataJSON); err == nil && snap != nil && snap.Manifest != nil {
		for _, candidate := range []string{
			strings.TrimSpace(snap.Manifest.Profile),
			strings.TrimSpace(snap.Manifest.Command),
			strings.TrimSpace(snap.Manifest.Driver),
		} {
			if runtimeTextLooksLikeOllama(candidate) {
				return true
			}
		}
	}
	for _, candidate := range []string{
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
		stringValueFromMetadata(meta, "herramienta"),
		stringValueFromMetadata(meta, "conector"),
		stringValueFromMetadata(meta, "profile_name"),
		stringValueFromMetadata(meta, "driver"),
	} {
		if runtimeTextLooksLikeOllama(candidate) {
			return true
		}
	}
	return false
}

func runtimeTextLooksLikeOllama(raw string) bool {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text == "" {
		return false
	}
	for _, token := range strings.Fields(text) {
		base := filepath.Base(strings.TrimSpace(token))
		if base == "ollama" || base == "ollama-cli" || strings.HasPrefix(base, "ollama-") {
			return true
		}
		if strings.HasPrefix(base, "gemma") || strings.HasPrefix(base, "qwen") || strings.HasPrefix(base, "llama") {
			return true
		}
	}
	return strings.Contains(text, "ollama")
}

func compactarInstruccionTMUXOllama(texto string) string {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return ""
	}
	texto = strings.Join(strings.Fields(texto), " ")
	const maxChars = 720
	if len(texto) <= maxChars {
		return texto
	}
	truncado := strings.TrimSpace(texto[:maxChars])
	if idx := strings.LastIndex(truncado, " "); idx > maxChars/2 {
		truncado = strings.TrimSpace(truncado[:idx])
	}
	return truncado
}

func sanitizarInstruccionTTY(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	raw = ttyInstructionASCIIReplacer.Replace(raw)

	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Map(func(r rune) rune {
			switch {
			case r == '\t' || r == ' ':
				return ' '
			case r >= 32 && r <= 126:
				return r
			default:
				return ' '
			}
		}, line)
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}
	texto := strings.TrimSpace(strings.Join(out, " "))
	if texto == "" {
		return ""
	}
	const maxChars = 96
	if len(texto) <= maxChars {
		return texto
	}
	truncado := strings.TrimSpace(texto[:maxChars])
	if idx := strings.LastIndex(truncado, " "); idx > maxChars/2 {
		truncado = strings.TrimSpace(truncado[:idx])
	}
	if truncado == "" {
		truncado = strings.TrimSpace(texto[:maxChars])
	}
	return truncado + " ..."
}

func compactarInstruccionCodexTTY(texto string) string {
	lower := strings.ToLower(strings.TrimSpace(texto))
	if lower == "" {
		return ""
	}
	cases := []struct {
		match []string
		out   string
	}{
		{match: []string{"se te ha asignado automaticamente la tarea", "se te ha asignado la tarea"}, out: "toma tarea asignada y sigue"},
		{match: []string{"has sido arrancado como programador", "has sido arrancado como agente", "has sido arrancado como"}, out: "continua trabajo actual"},
		{match: []string{"retoma el trabajo", "prompt de continuidad", "resume payload"}, out: "retoma trabajo previo y sigue"},
		{match: []string{"supervisor autonomo", "orquestador autonomo", "arrancado como orquestador", "supervisar_proyecto", "supervision_transcript_signal"}, out: "supervisa proyecto actual y sigue"},
		{match: []string{"ready for review", "listo para review", "listo para revision"}, out: "prepara review y sigue"},
		{match: []string{"approval_request", "waiting_human", "esperando aprobacion", "esperando aprobacion", "me dejas", "can i ", "should i"}, out: "no esperes; decide y sigue"},
		{match: []string{"blocked", "bloqueado", "bloqueo"}, out: "reevalua bloqueo y sigue"},
		{match: []string{"riesgo principal", "risk principal", "riesgo"}, out: "resume riesgo y siguiente paso"},
		{match: []string{"comprobacion server-first", "server-first", "siguiente test", "siguiente prueba", "que vas a correr"}, out: "corre prueba server-first y resume"},
		{match: []string{"resume en dos lineas", "resume en dos lineas", "resumelo en dos lineas", "resume en dos lineas"}, out: "resume hallazgo y sigue"},
		{match: []string{"prueba grande", "prueba real", "stress test"}, out: "corre prueba grande y resume"},
		{match: []string{"needs_replan", "siguiente paso", "backlog"}, out: "elige siguiente frente util"},
		{match: []string{"continua de forma autonoma", "sigue trabajando", "continuar_trabajo"}, out: "continua trabajo actual"},
	}
	for _, tc := range cases {
		for _, needle := range tc.match {
			if strings.Contains(lower, needle) {
				return tc.out
			}
		}
	}
	if strings.Contains(lower, "checkpoint") {
		return "haz checkpoint si hace falta"
	}
	if strings.HasPrefix(lower, "orquesta:") {
		return "continua trabajo actual"
	}
	// Codex TTY and session_resume are fragile under arbitrary free-form text.
	// If no canonical pattern matches, fall back to a safe generic guidance
	// instead of passing truncated user text that may still crash the TUI.
	return "continua trabajo actual"
}

func EnviarInstruccionProceso(obj ObjetivoProceso, instruccion string) (bool, int, error) {
	if aplicado, pid, observed, err := enviarInstruccionProcesoLocalSupervisado(obj, instruccion); observed {
		return aplicado, pid, err
	}
	return enviarInstruccionProcesoBase(obj, instruccion)
}

func enviarInstruccionProcesoLocalSupervisado(obj ObjetivoProceso, instruccion string) (bool, int, bool, error) {
	estado, observed, err := ConsultarEstadoLocal(obj)
	if err != nil || !observed {
		return false, 0, observed, err
	}
	if estado == nil {
		return false, 0, true, nil
	}
	if !estado.Vivo || estado.PID <= 0 {
		return false, estado.PID, true, nil
	}
	aplicado, pid, err := enviarInstruccionProcesoBase(obj, instruccion)
	return aplicado, pid, true, err
}

func enviarInstruccionProcesoBase(obj ObjetivoProceso, instruccion string) (bool, int, error) {
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
	}
	if !stdinPermitidoDesdeMetadata(obj.MetadataJSON) {
		if ok {
			vivo, _, err := ProcesoVivo(obj)
			if err != nil {
				return false, pid, err
			}
			if !vivo {
				return false, pid, nil
			}
		}
		return false, pid, nil
	}
	texto := strings.TrimRight(NormalizarInstruccionProceso(obj, instruccion), "\n")
	if texto == "" {
		return false, pid, nil
	}
	if enviado, err := enviarInstruccionTMUXDesdeMetadata(obj.MetadataJSON, texto); enviado || err != nil {
		return enviado, pid, err
	}
	if ok {
		vivo, _, err := ProcesoVivo(obj)
		if err != nil {
			return false, pid, err
		}
		if !vivo {
			return false, pid, nil
		}
		stdinPath := stdinPathFromMetadata(obj.MetadataJSON)
		if stdinPath == "" {
			return false, pid, nil
		}
		fd, err := syscall.Open(stdinPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			if err == syscall.ENXIO || err == syscall.ENOENT {
				return false, pid, nil
			}
			return true, pid, err
		}
		f := os.NewFile(uintptr(fd), stdinPath)
		if f == nil {
			_ = syscall.Close(fd)
			return false, pid, nil
		}
		defer f.Close()

		if _, err := fmt.Fprintln(f, texto); err != nil {
			if err == syscall.EPIPE {
				return false, pid, nil
			}
			return true, pid, err
		}
		// La traza raw no debe invalidar una instruccion ya entregada al PTY.
		_ = appendRuntimeRawInput(stdinRawPathFromMetadata(obj.MetadataJSON), texto)
		return true, pid, nil
	}
	return enviarInstruccionRemota(obj, instruccion)
}

func stdinPermitidoDesdeMetadata(raw string) bool {
	payload := metadataMap(raw)
	if payload == nil {
		return true
	}
	if v, ok := payload["can_send_input"].(bool); ok {
		return v
	}
	return true
}

func stdinPathFromMetadata(raw string) string {
	payload := metadataMap(raw)
	if v, ok := payload["stdin_path"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

type TMUXDispatchResult struct {
	Attempted  bool
	Delivered  bool
	ActiveTask bool
	Reason     string
}

func EnviarInstruccionTMUXVerificada(obj ObjetivoProceso, instruccion string) (TMUXDispatchResult, error) {
	texto := strings.TrimSpace(NormalizarInstruccionProceso(obj, instruccion))
	if texto == "" {
		return TMUXDispatchResult{}, nil
	}
	return enviarInstruccionTMUXVerificadaDesdeMetadata(obj.MetadataJSON, texto)
}

func enviarInstruccionTMUXDesdeMetadata(raw, texto string) (bool, error) {
	payload := metadataMap(raw)
	if payload == nil {
		return false, nil
	}
	if !metadataLooksLikeTMUXRuntime(payload) {
		return false, nil
	}
	tmuxCommand := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_command"))
	paneID := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_pane_id"))
	if tmuxCommand == "" || paneID == "" {
		return false, nil
	}
	ready, err := waitForTMUXPaneReady(tmuxCommand, paneID, tmuxReadyTimeout)
	if err != nil {
		return true, err
	}
	if !ready {
		if captured, captureErr := captureTMUXPane(tmuxCommand, paneID); captureErr == nil {
			switch {
			case tmuxPaneHasCodexAuthPrompt(captured):
				return true, fmt.Errorf("tmux pane requiere autenticacion manual")
			case tmuxPaneHasUsageLimitPrompt(captured):
				return true, fmt.Errorf("tmux pane bloqueado por cuota")
			}
		}
		return true, fmt.Errorf("tmux pane no listo para send-keys")
	}
	cmd := exec.Command(tmuxCommand, "send-keys", "-t", paneID, "-l", texto)
	if out, err := cmd.CombinedOutput(); err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return true, fmt.Errorf("tmux send-keys literal: %s", detalle)
		}
		return true, err
	}
	enterCmd := exec.Command(tmuxCommand, "send-keys", "-t", paneID, "Enter")
	if out, err := enterCmd.CombinedOutput(); err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return true, fmt.Errorf("tmux send-keys enter: %s", detalle)
		}
		return true, err
	}
	return true, nil
}

func enviarInstruccionTMUXVerificadaDesdeMetadata(raw, texto string) (TMUXDispatchResult, error) {
	payload := metadataMap(raw)
	if payload == nil {
		return TMUXDispatchResult{}, nil
	}
	if !metadataLooksLikeTMUXRuntime(payload) {
		return TMUXDispatchResult{}, nil
	}
	tmuxCommand := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_command"))
	paneID := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_pane_id"))
	if tmuxCommand == "" || paneID == "" {
		return TMUXDispatchResult{}, nil
	}
	result := TMUXDispatchResult{Attempted: true}
	ready, err := waitForTMUXPaneReady(tmuxCommand, paneID, tmuxReadyTimeout)
	if err != nil {
		return result, err
	}
	if !ready {
		if captured, captureErr := captureTMUXPane(tmuxCommand, paneID); captureErr == nil {
			switch {
			case tmuxPaneHasCodexAuthPrompt(captured):
				result.Reason = "auth_required"
			case tmuxPaneHasUsageLimitPrompt(captured):
				result.Reason = "quota_blocked"
			default:
				result.Reason = "pane_not_ready"
			}
		} else {
			result.Reason = "pane_not_ready"
		}
		return result, nil
	}
	beforeCapture, err := captureTMUXPane(tmuxCommand, paneID)
	if err != nil {
		return result, err
	}
	if err := sendTMUXLiteral(tmuxCommand, paneID, texto); err != nil {
		return result, err
	}
	if err := sendTMUXKey(tmuxCommand, paneID, "Enter"); err != nil {
		return result, err
	}
	for round := 0; round < 3; round++ {
		time.Sleep(250 * time.Millisecond)
		captured, err := captureTMUXPane(tmuxCommand, paneID)
		if err == nil {
			if tmuxPaneHasActiveTask(captured) {
				result.Delivered = true
				result.ActiveTask = true
				result.Reason = "active_task"
				return result, nil
			}
			if tmuxPaneDispatchConsumed(beforeCapture, captured, texto) {
				result.Delivered = true
				result.Reason = "consumed"
				return result, nil
			}
		}
		if err := sendTMUXKey(tmuxCommand, paneID, "Enter"); err != nil {
			return result, err
		}
	}
	result.Reason = "unconfirmed"
	return result, nil
}

func tmuxPaneDispatchConsumed(before, after, trigger string) bool {
	if !tmuxPaneLooksReady(after) {
		return false
	}
	if tmuxPaneContainsTriggerNearTail(after, trigger, 24) {
		return false
	}
	if tmuxNormalizeCapture(before) == tmuxNormalizeCapture(after) {
		return false
	}
	return true
}

func sendTMUXLiteral(tmuxCommand, paneID, texto string) error {
	cmd := exec.Command(tmuxCommand, "send-keys", "-t", paneID, "-l", texto)
	if out, err := cmd.CombinedOutput(); err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return fmt.Errorf("tmux send-keys literal: %s", detalle)
		}
		return err
	}
	return nil
}

func tmuxPaneContainsTriggerNearTail(captured, trigger string, nonEmptyTailLines int) bool {
	trigger = tmuxNormalizeCapture(trigger)
	if trigger == "" {
		return false
	}
	lines := tmuxNormalizePaneLines(captured)
	if len(lines) == 0 {
		return false
	}
	if nonEmptyTailLines <= 0 {
		nonEmptyTailLines = 24
	}
	if len(lines) > nonEmptyTailLines {
		lines = lines[len(lines)-nonEmptyTailLines:]
	}
	return strings.Contains(tmuxNormalizeCapture(strings.Join(lines, " ")), trigger)
}

func tmuxNormalizeCapture(raw string) string {
	raw = sanitizeTMUXPaneText(raw)
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(raw, "\r", " "))), " ")
}

func detenerSesionTMUXDesdeMetadata(raw string) (bool, error) {
	payload := metadataMap(raw)
	if payload == nil {
		return false, nil
	}
	if !metadataLooksLikeTMUXRuntime(payload) {
		return false, nil
	}
	tmuxCommand := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_command"))
	sessionName := strings.TrimSpace(stringValueFromMetadata(payload, "tmux_session"))
	if tmuxCommand == "" {
		path, err := exec.LookPath("tmux")
		if err == nil {
			tmuxCommand = path
		}
	}
	if tmuxCommand == "" || sessionName == "" {
		return false, nil
	}
	cmd := exec.Command(tmuxCommand, "kill-session", "-t", sessionName)
	if out, err := cmd.CombinedOutput(); err != nil {
		detalle := strings.TrimSpace(string(out))
		if detalle != "" {
			return true, fmt.Errorf("tmux kill-session: %s", detalle)
		}
		return true, err
	}
	return true, nil
}

func metadataLooksLikeTMUXRuntime(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	for _, key := range []string{"driver", "runtime_driver"} {
		if strings.EqualFold(strings.TrimSpace(stringValueFromMetadata(payload, key)), "tmux_cli_session") {
			return true
		}
	}
	return strings.TrimSpace(stringValueFromMetadata(payload, "tmux_session")) != "" ||
		strings.TrimSpace(stringValueFromMetadata(payload, "tmux_pane_id")) != ""
}

func stdinRawPathFromMetadata(raw string) string {
	payload := metadataMap(raw)
	if v, ok := payload["stdin_raw_path"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func appendRuntimeRawInput(path, texto string) error {
	path = strings.TrimSpace(path)
	texto = strings.TrimRight(texto, "\n")
	if path == "" || texto == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(texto + "\n")
	return err
}

func stringValueFromMetadata(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
