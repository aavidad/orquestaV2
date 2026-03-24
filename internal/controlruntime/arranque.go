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

type SolicitudArranque struct {
	Agente   string
	Proyecto string
	Plan     *runtimeagente.LaunchPlan
}

type ProcesoArrancado struct {
	PID             int
	StdinPath       string
	LogPath         string
	WorkingDir      string
	WrappedCommand  string
	RenderedCommand string
}

func ArrancarPlan(req SolicitudArranque) (*ProcesoArrancado, error) {
	if req.Plan == nil {
		return nil, fmt.Errorf("plan obligatorio")
	}
	if strings.TrimSpace(req.Plan.Comando) == "" {
		return nil, fmt.Errorf("comando obligatorio")
	}
	if _, err := exec.LookPath("script"); err != nil {
		return nil, fmt.Errorf("script no disponible: %w", err)
	}

	baseDir := filepath.Join(os.TempDir(), "orquesta-runtime", sanitizePathFragment(req.Agente))
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, err
	}

	token := fmt.Sprintf("%d", time.Now().UnixNano())
	stdinPath := filepath.Join(baseDir, token+".stdin")
	logPath := filepath.Join(baseDir, token+".log")

	if err := syscall.Mkfifo(stdinPath, 0o600); err != nil {
		return nil, err
	}

	stdin, err := os.OpenFile(stdinPath, os.O_RDWR, 0o600)
	if err != nil {
		_ = os.Remove(stdinPath)
		return nil, err
	}
	defer stdin.Close()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = os.Remove(stdinPath)
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
		return nil, err
	}

	return &ProcesoArrancado{
		PID:             cmd.Process.Pid,
		StdinPath:       stdinPath,
		LogPath:         logPath,
		WorkingDir:      cmd.Dir,
		WrappedCommand:  wrapped,
		RenderedCommand: rendered,
	}, nil
}

func EnviarInstruccionProceso(obj ObjetivoProceso, instruccion string) (bool, int, error) {
	pid, ok, err := ResolverPID(obj)
	if err != nil || !ok {
		return ok, pid, err
	}
	stdinPath := stdinPathFromMetadata(obj.MetadataJSON)
	if stdinPath == "" {
		return false, pid, nil
	}
	f, err := os.OpenFile(stdinPath, os.O_WRONLY, 0)
	if err != nil {
		return true, pid, err
	}
	defer f.Close()

	texto := strings.TrimRight(instruccion, "\n")
	if _, err := fmt.Fprintln(f, texto); err != nil {
		return true, pid, err
	}
	return true, pid, nil
}

func stdinPathFromMetadata(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	if v, ok := payload["stdin_path"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func sanitizePathFragment(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "runtime"
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
