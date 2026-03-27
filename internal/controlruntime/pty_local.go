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

func arrancarPlanLocalPTY(req SolicitudArranque) (*ProcesoArrancado, error) {
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

	metaJSON, _ := json.Marshal(map[string]any{
		"stdin_path":       stdinPath,
		"log_path":         logPath,
		"working_dir":      cmd.Dir,
		"wrapped_command":  wrapped,
		"rendered_command": rendered,
		"driver":           "process_pty_cli",
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":       true,
		"can_checkpoint":       true,
		"can_resume":           true,
		"can_capture_pid":      true,
		"can_track_continuity": true,
		"can_pause":            true,
		"can_stop":             true,
	})

	return &ProcesoArrancado{
		PID:              cmd.Process.Pid,
		HandleKind:       "process",
		HandleRef:        fmt.Sprintf("%d", cmd.Process.Pid),
		StdinPath:        stdinPath,
		LogPath:          logPath,
		WorkingDir:       cmd.Dir,
		WrappedCommand:   wrapped,
		RenderedCommand:  rendered,
		MetadataJSON:     string(metaJSON),
		CapabilitiesJSON: string(capsJSON),
	}, nil
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

		texto := strings.TrimRight(instruccion, "\n")
		if _, err := fmt.Fprintln(f, texto); err != nil {
			if err == syscall.EPIPE {
				return false, pid, nil
			}
			return true, pid, err
		}
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
