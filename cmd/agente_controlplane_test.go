package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestAgenteControlPlaneUsaAPICuandoHayServidor(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/agente/control" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 41, "agente": "Codex1", "accion": "start"})
		case r.URL.Path == "/api/agente/pausar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agente": "Codex1"})
		case r.URL.Path == "/api/agente/handoff" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": 33})
		case r.URL.Path == "/api/agentes/Codex1/reset-reanimacion" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agente": "Codex1"})
		case r.URL.Path == "/api/agentes/Codex1/eliminar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agente": "Codex1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	outPausar := capturarStdout(t, func() {
		if err := agentePausarCmd.RunE(agentePausarCmd, []string{"Codex1", "15", "rate", "limit"}); err != nil {
			t.Fatalf("agente pausar via api: %v", err)
		}
	})
	if !strings.Contains(outPausar, "Codex1 pausado") {
		t.Fatalf("salida pausar inesperada:\n%s", outPausar)
	}

	outHandoff := capturarStdout(t, func() {
		if err := agenteHandoffCmd.RunE(agenteHandoffCmd, []string{"Codex1", "Codex2"}); err != nil {
			t.Fatalf("agente handoff via api: %v", err)
		}
	})
	if !strings.Contains(outHandoff, "runtime_order: 33") {
		t.Fatalf("salida handoff inesperada:\n%s", outHandoff)
	}

	outReset := capturarStdout(t, func() {
		if err := agenteRehabilitarCmd.RunE(agenteRehabilitarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente rehabilitar via api: %v", err)
		}
	})
	if !strings.Contains(outReset, "rehabilitado correctamente") {
		t.Fatalf("salida rehabilitar inesperada:\n%s", outReset)
	}

	outEliminar := capturarStdout(t, func() {
		if err := agentePurgarCmd.RunE(agentePurgarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente eliminar via api: %v", err)
		}
	})
	if !strings.Contains(outEliminar, "eliminado correctamente") {
		t.Fatalf("salida eliminar inesperada:\n%s", outEliminar)
	}

	outControl := capturarStdout(t, func() {
		if err := agenteControlCmd.RunE(agenteControlCmd, []string{"arrancar", "Codex1"}); err != nil {
			t.Fatalf("agente control via api: %v", err)
		}
	})
	if !strings.Contains(outControl, "Orden start #") {
		t.Fatalf("salida control inesperada:\n%s", outControl)
	}
}

func TestAgentePurgarBloqueaTareasActivasEnModoLocal(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexLocal", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "No borrar agente con trabajo vivo",
		Modulo:    "controlplane",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexLocal"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	err = agentePurgarCmd.RunE(agentePurgarCmd, []string{"CodexLocal"})
	if err == nil {
		t.Fatalf("esperaba bloqueo al eliminar agente con tareas activas")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "tarea") {
		t.Fatalf("error inesperado: %v", err)
	}
	if _, err := db.GetAgente("CodexLocal"); err != nil {
		t.Fatalf("el agente no deberia haberse eliminado: %v", err)
	}
}
