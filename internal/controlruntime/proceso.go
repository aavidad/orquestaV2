package controlruntime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ObjetivoProceso struct {
	PID          *int64
	HandleKind   string
	HandleRef    string
	MetadataJSON string
}

func ResolverPID(obj ObjetivoProceso) (int, bool, error) {
	if obj.PID != nil && *obj.PID > 0 {
		return int(*obj.PID), true, nil
	}
	if strings.TrimSpace(obj.HandleKind) != "process" {
		return 0, false, nil
	}
	ref := strings.TrimSpace(obj.HandleRef)
	if ref == "" {
		return 0, false, nil
	}
	pid, err := strconv.Atoi(ref)
	if err != nil || pid <= 0 {
		return 0, false, fmt.Errorf("handle_ref de proceso invalido: %s", ref)
	}
	return pid, true, nil
}

func ProcesoVivo(obj ObjetivoProceso) (bool, int, error) {
	if estado, observed, err := ConsultarEstadoLocal(obj); observed {
		if err != nil {
			if estado != nil {
				return false, estado.PID, err
			}
			return false, 0, err
		}
		if estado == nil {
			return false, 0, nil
		}
		return estado.Vivo, estado.PID, nil
	}
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
	}
	if ok {
		err := syscall.Kill(pid, 0)
		switch {
		case err == nil:
			return true, pid, nil
		case errors.Is(err, syscall.EPERM):
			return true, pid, nil
		case errors.Is(err, syscall.ESRCH):
			return false, pid, nil
		default:
			return false, pid, err
		}
	}
	return false, 0, nil
}

func PausarProceso(obj ObjetivoProceso) (bool, int, error) {
	return controlarProceso(obj, syscall.SIGSTOP, "pause")
}

func ContinuarProceso(obj ObjetivoProceso) (bool, int, error) {
	return controlarProceso(obj, syscall.SIGCONT, "continue")
}

func DetenerProceso(obj ObjetivoProceso) (bool, int, error) {
	return controlarProceso(obj, syscall.SIGTERM, "stop")
}

func DetenerSesionTMUXMetadata(raw string) (bool, error) {
	return detenerSesionTMUXDesdeMetadata(raw)
}

func TMUXSessionExistsMetadata(raw string) (bool, bool) {
	payload := metadataMap(raw)
	if payload == nil || !metadataLooksLikeTMUXRuntime(payload) {
		return false, false
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
		return false, false
	}
	return true, tmuxSessionExists(tmuxCommand, sessionName)
}

func controlarProceso(obj ObjetivoProceso, sig syscall.Signal, accion string) (bool, int, error) {
	if aplicado, pid, observed, err := controlarProcesoLocalSupervisado(obj, sig, accion); observed {
		if accion == "stop" && err == nil {
			if _, brokerErr := detenerBrokerAuxiliarDesdeMetadata(obj.MetadataJSON, pid); brokerErr != nil {
				return aplicado, pid, brokerErr
			}
		}
		return aplicado, pid, err
	}
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
	}
	if accion == "stop" {
		if detenido, err := detenerSesionTMUXDesdeMetadata(obj.MetadataJSON); detenido || err != nil {
			return detenido, pid, err
		}
	}
	if ok {
		return enviarSenalPID(pid, sig)
	}
	switch accion {
	case "pause":
		return controlRemoto(obj, remoteConfigFromMetadata(obj.MetadataJSON).PausePath)
	case "continue":
		return controlRemoto(obj, remoteConfigFromMetadata(obj.MetadataJSON).ContinuePath)
	case "stop":
		aplicado, gotPID, stopErr := controlRemoto(obj, remoteConfigFromMetadata(obj.MetadataJSON).StopPath)
		if stopErr == nil {
			if _, brokerErr := detenerBrokerAuxiliarDesdeMetadata(obj.MetadataJSON, gotPID); brokerErr != nil {
				return aplicado, gotPID, brokerErr
			}
		}
		return aplicado, gotPID, stopErr
	default:
		return false, 0, nil
	}
}

func detenerBrokerAuxiliarDesdeMetadata(raw string, childPID int) (bool, error) {
	payload := metadataMap(raw)
	if len(payload) == 0 {
		return false, nil
	}
	brokerPID := extractRemoteInt(payload, "broker_pid")
	if brokerPID <= 0 || brokerPID == childPID {
		return false, nil
	}
	terminated, err := esperarSalidaProcesoPID(brokerPID, 300*time.Millisecond)
	if err != nil || terminated {
		return terminated, err
	}
	aplicado, _, err := enviarSenalPID(brokerPID, syscall.SIGTERM)
	return aplicado, err
}

func enviarSenalPID(pid int, sig syscall.Signal) (bool, int, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return true, pid, err
	}
	if err := proc.Signal(sig); err != nil {
		return true, pid, err
	}
	if sig == syscall.SIGTERM || sig == syscall.SIGKILL {
		if terminado, err := esperarSalidaProcesoPID(pid, 500*time.Millisecond); err != nil {
			return true, pid, err
		} else if !terminado && sig == syscall.SIGTERM {
			if err := proc.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				return true, pid, err
			}
			if _, err := esperarSalidaProcesoPID(pid, 500*time.Millisecond); err != nil {
				return true, pid, err
			}
		}
	}
	return true, pid, nil
}

func esperarSalidaProcesoPID(pid int, timeout time.Duration) (bool, error) {
	if pid <= 0 {
		return true, nil
	}
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Kill(pid, 0)
		switch {
		case err == nil, errors.Is(err, syscall.EPERM):
			if time.Now().After(deadline) {
				return false, nil
			}
			time.Sleep(25 * time.Millisecond)
			continue
		case errors.Is(err, syscall.ESRCH):
			return true, nil
		default:
			return false, err
		}
	}
}
