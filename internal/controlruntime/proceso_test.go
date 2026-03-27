package controlruntime

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestResolverPIDUsaPIDExplicitoYHandleRef(t *testing.T) {
	pid := int64(4321)
	resolved, ok, err := ResolverPID(ObjetivoProceso{PID: &pid})
	if err != nil || !ok || resolved != 4321 {
		t.Fatalf("resolver pid explicito: pid=%d ok=%v err=%v", resolved, ok, err)
	}

	resolved, ok, err = ResolverPID(ObjetivoProceso{HandleKind: "process", HandleRef: "9876"})
	if err != nil || !ok || resolved != 9876 {
		t.Fatalf("resolver handle_ref: pid=%d ok=%v err=%v", resolved, ok, err)
	}
}

func TestControlProcesoPausaContinuaYDetiene(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := int64(cmd.Process.Pid)
	obj := ObjetivoProceso{PID: &pid}

	if ok, gotPID, err := PausarProceso(obj); err != nil || !ok || gotPID != cmd.Process.Pid {
		t.Fatalf("pausar: ok=%v pid=%d err=%v", ok, gotPID, err)
	}
	if ok, gotPID, err := ContinuarProceso(obj); err != nil || !ok || gotPID != cmd.Process.Pid {
		t.Fatalf("continuar: ok=%v pid=%d err=%v", ok, gotPID, err)
	}
	if ok, gotPID, err := DetenerProceso(obj); err != nil || !ok || gotPID != cmd.Process.Pid {
		t.Fatalf("detener: ok=%v pid=%d err=%v", ok, gotPID, err)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("el proceso no terminó tras SIGTERM")
	case err := <-waitDone:
		if err == nil {
			return
		}
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("wait inesperado: %v", err)
		}
	}
}

func TestProcesoVivoDetectaPIDActualEInexistente(t *testing.T) {
	pid := int64(os.Getpid())
	vivo, gotPID, err := ProcesoVivo(ObjetivoProceso{PID: &pid})
	if err != nil || !vivo || gotPID != os.Getpid() {
		t.Fatalf("proceso actual no detectado: vivo=%v pid=%d err=%v", vivo, gotPID, err)
	}

	muertoPID := int64(999999)
	vivo, gotPID, err = ProcesoVivo(ObjetivoProceso{PID: &muertoPID})
	if err != nil {
		t.Fatalf("proceso inexistente devolvió error: %v", err)
	}
	if vivo {
		t.Fatalf("un PID inexistente no debería aparecer vivo: pid=%d", gotPID)
	}
}
