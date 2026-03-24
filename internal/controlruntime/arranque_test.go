package controlruntime

import (
	"os/exec"
	"os"
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
	pid64 := int64(arranque.PID)

	t.Cleanup(func() {
		_, _, _ = DetenerProceso(ObjetivoProceso{PID: &pid64})
	})

	time.Sleep(150 * time.Millisecond)
	meta := `{"stdin_path":"` + arranque.StdinPath + `"}`
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
}
