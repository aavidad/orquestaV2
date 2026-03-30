package controlruntime

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
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

func controlarProceso(obj ObjetivoProceso, sig syscall.Signal, accion string) (bool, int, error) {
	if aplicado, pid, observed, err := controlarProcesoLocalSupervisado(obj, sig, accion); observed {
		return aplicado, pid, err
	}
	pid, ok, err := ResolverPID(obj)
	if err != nil {
		return false, pid, err
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
		return controlRemoto(obj, remoteConfigFromMetadata(obj.MetadataJSON).StopPath)
	default:
		return false, 0, nil
	}
}

func enviarSenalPID(pid int, sig syscall.Signal) (bool, int, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return true, pid, err
	}
	if err := proc.Signal(sig); err != nil {
		return true, pid, err
	}
	return true, pid, nil
}
