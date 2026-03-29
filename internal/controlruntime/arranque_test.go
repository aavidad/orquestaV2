package controlruntime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

func TestArrancarPlanYEnviarInstruccionProceso(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script no disponible en el entorno")
	}

	dir := t.TempDir()
	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "cat",
			WorkingDir: dir,
			Env:        map[string]string{},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan: %v", err)
	}
	traceDir := filepath.Dir(arranque.LogPath)
	tracePrefix := filepath.Join(dir, ".orquesta-runtime", "codex1") + string(os.PathSeparator)
	if !strings.HasPrefix(traceDir, tracePrefix) {
		t.Fatalf("trace dir fuera del subdirectorio de trabajo: got=%s want_prefix=%s", traceDir, tracePrefix)
	}
	if filepath.Base(arranque.LogPath) != "pty.log" || filepath.Base(arranque.StdinPath) != "pty.stdin" || filepath.Base(arranque.StdinRawPath) != "pty.stdin.raw" {
		t.Fatalf("nombres de artefactos PTY inesperados: log=%s stdin=%s stdin_raw=%s", arranque.LogPath, arranque.StdinPath, arranque.StdinRawPath)
	}
	pid64 := int64(arranque.PID)

	t.Cleanup(func() {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: &pid64})
	})

	time.Sleep(150 * time.Millisecond)
	meta := `{"stdin_path":"` + arranque.StdinPath + `","stdin_raw_path":"` + arranque.StdinRawPath + `"}`
	aplicado, pid, err := EnviarInstruccionProceso(ObjetivoProceso{
		PID:          &pid64,
		HandleKind:   "process",
		HandleRef:    arranque.StdinPath,
		MetadataJSON: meta,
	}, "hola runtime")
	if err != nil {
		t.Fatalf("enviar instruccion: %v", err)
	}
	if !aplicado || pid != arranque.PID {
		t.Fatalf("entrega directa inesperada: aplicado=%t pid=%d", aplicado, pid)
	}

	time.Sleep(250 * time.Millisecond)
	logData, err := os.ReadFile(arranque.LogPath)
	if err != nil {
		t.Fatalf("leer log: %v", err)
	}
	if !strings.Contains(string(logData), "hola runtime") {
		t.Fatalf("el log no contiene la instruccion: %s", string(logData))
	}
	rawData, err := os.ReadFile(arranque.StdinRawPath)
	if err != nil {
		t.Fatalf("leer stdin raw: %v", err)
	}
	if string(rawData) != "hola runtime\n" {
		t.Fatalf("stdin raw inesperado: %q", string(rawData))
	}

	manifestPath := filepath.Join(traceDir, "runtime.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("leer manifest runtime: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsear manifest runtime: %v", err)
	}
	if got, _ := manifest["agente"].(string); got != "Codex1" {
		t.Fatalf("agente inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["proyecto"].(string); got != "orquestador" {
		t.Fatalf("proyecto inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["log_path"].(string); got != arranque.LogPath {
		t.Fatalf("log_path inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["stdin_path"].(string); got != arranque.StdinPath {
		t.Fatalf("stdin_path inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["stdin_raw_path"].(string); got != arranque.StdinRawPath {
		t.Fatalf("stdin_raw_path inesperado en manifest: %+v", manifest)
	}
}

func TestRuntimeArtifactsRunDirOrdenaPorAgenteYFecha(t *testing.T) {
	dir := t.TempDir()
	req := SolicitudArranque{
		Agente:   "Codex 2",
		Proyecto: "demo",
		Plan: &runtimeagente.LaunchPlan{
			WorkingDir: dir,
		},
	}
	t1 := time.Date(2026, 3, 29, 12, 30, 1, 123, time.UTC)
	t2 := t1.Add(time.Second)

	dir1 := runtimeArtifactsRunDir(req, t1)
	dir2 := runtimeArtifactsRunDir(req, t2)
	prefix := filepath.Join(dir, ".orquesta-runtime", "codex-2") + string(os.PathSeparator)
	if !strings.HasPrefix(dir1, prefix) || !strings.HasPrefix(dir2, prefix) {
		t.Fatalf("las trazas deberian agruparse por agente bajo el working dir: dir1=%s dir2=%s prefix=%s", dir1, dir2, prefix)
	}
	if !(dir1 < dir2) {
		t.Fatalf("las rutas deberian quedar ordenables cronologicamente: dir1=%s dir2=%s", dir1, dir2)
	}
}
