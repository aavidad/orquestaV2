package controlruntime

import (
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

func PausarProceso(obj ObjetivoProceso) (bool, int, error) {
	return enviarSenal(obj, syscall.SIGSTOP)
}

func ContinuarProceso(obj ObjetivoProceso) (bool, int, error) {
	return enviarSenal(obj, syscall.SIGCONT)
}

func DetenerProceso(obj ObjetivoProceso) (bool, int, error) {
	return enviarSenal(obj, syscall.SIGTERM)
}

func enviarSenal(obj ObjetivoProceso, sig syscall.Signal) (bool, int, error) {
	pid, ok, err := ResolverPID(obj)
	if err != nil || !ok {
		return ok, pid, err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return true, pid, err
	}
	if err := proc.Signal(sig); err != nil {
		return true, pid, err
	}
	return true, pid, nil
}
