package controlruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

func TestEnviarInstruccionProcesoSinLectorNoBloquea(t *testing.T) {
	tmp := t.TempDir()
	stdinPath := filepath.Join(tmp, "agent.stdin")
	if err := syscall.Mkfifo(stdinPath, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	pid := int64(os.Getpid())
	start := time.Now()
	ok, gotPID, err := EnviarInstruccionProceso(ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"stdin_path":"` + stdinPath + `"}`,
	}, "continua")
	if err != nil {
		t.Fatalf("enviar instruccion sin lector: %v", err)
	}
	if ok {
		t.Fatalf("sin lector no debería aplicar control real: pid=%d", gotPID)
	}
	if gotPID != os.Getpid() {
		t.Fatalf("pid inesperado: %d", gotPID)
	}
	if time.Since(start) > time.Second {
		t.Fatal("la llamada no debería bloquearse")
	}
}

func TestNormalizarInstruccionProcesoHaceASCIIYCorta(t *testing.T) {
	pid := int64(os.Getpid())
	texto := "Orquesta: continúa de forma autónoma dentro de la gobernanza efectiva del proyecto. No necesitas aprobación humana salvo que falten credenciales, secretos o un recurso externo real. Sigue con el trabajo en curso y cierra el siguiente frente útil."
	got := NormalizarInstruccionProceso(ObjetivoProceso{PID: &pid}, texto)
	if got == "" {
		t.Fatal("la instruccion no deberia quedar vacia")
	}
	if len(got) > 100 {
		t.Fatalf("la instruccion deberia quedar acotada, got len=%d text=%q", len(got), got)
	}
	for _, r := range got {
		if r < 32 || r > 126 {
			t.Fatalf("la instruccion deberia quedar en ASCII seguro, got=%q", got)
		}
	}
	if strings.Contains(got, "continúa") || strings.Contains(got, "autónoma") {
		t.Fatalf("la instruccion no deberia conservar acentos en PTY: %q", got)
	}
}

func TestNormalizarInstruccionProcesoCompactaParaCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex"}`,
	}
	texto := "Orquesta: actúa como supervisor autónomo del proyecto y mantén el trabajo alineado con la planificación aprobada. Revisa el estado real del proyecto y empuja el siguiente frente útil sin detenerte."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "supervisa proyecto actual y sigue" {
		t.Fatalf("compactacion inesperada para codex local: %q", got)
	}
}

func TestNormalizarInstruccionProcesoCompactaBootstrapSupervisorCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1 exec"}`,
	}
	texto := "Orquesta: has sido arrancado como orquestador autonomo del proyecto. Coordina a los demas Codex desde dentro de Orquesta, reparte trabajo real, revisa pruebas, deduplica frentes y no pidas intervencion humana salvo que falten credenciales, secretos o un recurso externo real."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "continua trabajo actual" {
		t.Fatalf("compactacion inesperada para bootstrap supervisor: %q", got)
	}
}

func TestNormalizarInstruccionProcesoCompactaAsignacionTareaCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex4 exec"}`,
	}
	texto := "Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "toma tarea asignada y sigue" {
		t.Fatalf("compactacion inesperada para asignacion de tarea: %q", got)
	}
}

func TestNormalizarInstruccionProcesoCompactaOrquestaGenericaCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex4 exec"}`,
	}
	texto := "Orquesta: has sido arrancado como programador del proyecto. Revisa el estado vivo, comprueba la tarea activa y continua sin esperar aprobacion humana."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "continua trabajo actual" {
		t.Fatalf("compactacion inesperada para guidance generica de Orquesta: %q", got)
	}
}

func TestNormalizarInstruccionProcesoCompactaPruebaGrandeCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex3 exec"}`,
	}
	texto := "Prueba grande: resume en dos lineas el riesgo principal de #410 y el siguiente test que vas a correr."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "resume riesgo y siguiente paso" {
		t.Fatalf("compactacion inesperada para prueba grande: %q", got)
	}
}

func TestNormalizarInstruccionProcesoNoDejaFallbackLibreParaCodexLocal(t *testing.T) {
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex4 exec"}`,
	}
	texto := "Texto libre muy raro sin patron estable 1234567890 y detalles arbitrarios que antes se truncaban y aun asi podian romper la TUI."
	got := NormalizarInstruccionProceso(obj, texto)
	if got != "continua trabajo actual" {
		t.Fatalf("el fallback libre de Codex deberia colapsar a guidance segura, got=%q", got)
	}
}

func TestRenderedCommandLooksLikeCodexCLI(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{raw: "codex --cd /tmp", want: true},
		{raw: "/usr/local/bin/codex exec", want: true},
		{raw: "/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex2", want: true},
		{raw: "cat", want: false},
	}
	for _, tc := range cases {
		if got := renderedCommandLooksLikeCodexCLI(tc.raw); got != tc.want {
			t.Fatalf("renderedCommandLooksLikeCodexCLI(%q)=%v want=%v", tc.raw, got, tc.want)
		}
	}
}
