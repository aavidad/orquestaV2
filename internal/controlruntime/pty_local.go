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
	runDir := runtimeArtifactsRunDir(req, time.Now())
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, err
	}

	stdinPath := filepath.Join(runDir, "pty.stdin")
	stdinRawPath := filepath.Join(runDir, "pty.stdin.raw")
	logPath := filepath.Join(runDir, "pty.log")
	manifestPath := filepath.Join(runDir, "runtime.json")

	if err := syscall.Mkfifo(stdinPath, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(stdinRawPath, nil, 0o600); err != nil {
		_ = os.Remove(stdinPath)
		return nil, err
	}

	stdin, err := os.OpenFile(stdinPath, os.O_RDWR, 0o600)
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	defer stdin.Close()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = os.Remove(stdinPath)
		_ = os.Remove(stdinRawPath)
		return nil, err
	}
	defer devNull.Close()

	rendered := runtimeagente.RenderCommand(req.Plan)
	wrapped := fmt.Sprintf("script -q -e -f -c %q %q", rendered, logPath)
	cmd := exec.Command("script", "-q", "-e", "-f", "-c", rendered, logPath)
	cmd.Dir = strings.TrimSpace(req.Plan.WorkingDir)
	if cmd.Dir != "" {
		if err := os.MkdirAll(cmd.Dir, 0o700); err != nil {
			_ = os.Remove(stdinPath)
			_ = os.Remove(stdinRawPath)
			return nil, err
		}
	}
	cmd.Stdin = stdin
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

	canSendInput := !renderedCommandLooksLikeCodexCLI(rendered)
	metaJSON, _ := json.Marshal(map[string]any{
		"stdin_path":       stdinPath,
		"stdin_raw_path":   stdinRawPath,
		"log_path":         logPath,
		"trace_dir":        runDir,
		"trace_manifest":   manifestPath,
		"working_dir":      cmd.Dir,
		"wrapped_command":  wrapped,
		"rendered_command": rendered,
		"driver":           "process_pty_cli",
		"can_send_input":   canSendInput,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":       canSendInput,
		"can_checkpoint":       true,
		"can_resume":           true,
		"can_capture_pid":      true,
		"can_track_continuity": true,
		"can_pause":            true,
		"can_stop":             true,
	})
	if err := writeRuntimeTraceManifest(manifestPath, req, cmd.Process.Pid, stdinPath, stdinRawPath, logPath, rendered, wrapped, canSendInput); err != nil {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: intPtr64(int64(cmd.Process.Pid))})
		return nil, err
	}

	return &ProcesoArrancado{
		PID:              cmd.Process.Pid,
		HandleKind:       "process",
		HandleRef:        fmt.Sprintf("%d", cmd.Process.Pid),
		StdinPath:        stdinPath,
		StdinRawPath:     stdinRawPath,
		LogPath:          logPath,
		WorkingDir:       cmd.Dir,
		WrappedCommand:   wrapped,
		RenderedCommand:  rendered,
		MetadataJSON:     string(metaJSON),
		CapabilitiesJSON: string(capsJSON),
	}, nil
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

func writeRuntimeTraceManifest(path string, req SolicitudArranque, pid int, stdinPath, stdinRawPath, logPath, rendered, wrapped string, canSendInput bool) error {
	payload := map[string]any{
		"created_at":       time.Now().UTC().Format(time.RFC3339Nano),
		"agente":           strings.TrimSpace(req.Agente),
		"proyecto":         strings.TrimSpace(req.Proyecto),
		"working_dir":      strings.TrimSpace(renderedWorkingDir(req)),
		"driver":           "process_pty_cli",
		"pid":              pid,
		"stdin_path":       stdinPath,
		"stdin_raw_path":   stdinRawPath,
		"log_path":         logPath,
		"rendered_command": rendered,
		"wrapped_command":  wrapped,
		"can_send_input":   canSendInput,
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
	return strings.Contains(lower, "codex-perfil") ||
		strings.Contains(lower, "/codex") ||
		strings.HasPrefix(lower, "codex ")
}

func NormalizarInstruccionProceso(obj ObjetivoProceso, instruccion string) string {
	texto := strings.TrimSpace(instruccion)
	if texto == "" {
		return ""
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

func esRuntimeCodexLocal(obj ObjetivoProceso) bool {
	meta := strings.ToLower(strings.TrimSpace(obj.MetadataJSON))
	if meta == "" {
		return false
	}
	return strings.Contains(meta, "\"driver\":\"process_pty_cli\"") && strings.Contains(meta, "codex")
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
		{match: []string{"retoma el trabajo", "prompt de continuidad", "resume payload"}, out: "retoma trabajo previo y sigue"},
		{match: []string{"supervisor autonomo", "supervisar_proyecto", "supervision_transcript_signal"}, out: "supervisa proyecto actual y sigue"},
		{match: []string{"ready for review", "listo para review", "listo para revision"}, out: "prepara review y sigue"},
		{match: []string{"approval_request", "waiting_human", "esperando aprobacion", "esperando aprobacion", "me dejas", "can i ", "should i"}, out: "no esperes; decide y sigue"},
		{match: []string{"blocked", "bloqueado", "bloqueo"}, out: "reevalua bloqueo y sigue"},
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
	const maxChars = 48
	if len(texto) <= maxChars {
		return texto
	}
	truncado := strings.TrimSpace(texto[:maxChars])
	if idx := strings.LastIndex(truncado, " "); idx > maxChars/2 {
		truncado = strings.TrimSpace(truncado[:idx])
	}
	if truncado == "" {
		return "continua trabajo actual"
	}
	return truncado
}

func EnviarInstruccionProceso(obj ObjetivoProceso, instruccion string) (bool, int, error) {
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
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

		texto := strings.TrimRight(NormalizarInstruccionProceso(obj, instruccion), "\n")
		if texto == "" {
			return false, pid, nil
		}
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

func stdinPathFromMetadata(raw string) string {
	payload := metadataMap(raw)
	if v, ok := payload["stdin_path"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
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
