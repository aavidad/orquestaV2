package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestTareaMutacionesBasicasUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiTareaCrearRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode tarea nueva: %v", err)
		}
		if req.Titulo != "Cerrar hueco API" || req.Agente != "Codex1" {
			t.Fatalf("payload tarea nueva inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"tarea": &db.Tarea{ID: 12, Titulo: req.Titulo},
		})
	})
	mux.HandleFunc("/api/tareas/12/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiTareaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode tarea accion: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "accion": req.Accion})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(tareaNuevaCmd)
	_ = tareaNuevaCmd.Flags().Set("titulo", "Cerrar hueco API")
	_ = tareaNuevaCmd.Flags().Set("descripcion", "cliente fino")
	_ = tareaNuevaCmd.Flags().Set("agente", "Codex1")
	outNueva := capturarStdout(t, func() {
		if err := tareaNuevaCmd.RunE(tareaNuevaCmd, nil); err != nil {
			t.Fatalf("tarea nueva via api: %v", err)
		}
	})
	for _, token := range []string{"Tarea #12 creada", "Asignada a Codex1"} {
		if !strings.Contains(outNueva, token) {
			t.Fatalf("salida nueva sin %q:\n%s", token, outNueva)
		}
	}

	outTomar := capturarStdout(t, func() {
		if err := tareaTomar.RunE(tareaTomar, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea tomar via api: %v", err)
		}
	})
	if !strings.Contains(outTomar, "Tarea #12 asignada a Codex1") {
		t.Fatalf("salida tomar inesperada:\n%s", outTomar)
	}

	outIniciar := capturarStdout(t, func() {
		if err := tareaIniciarCmd.RunE(tareaIniciarCmd, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea iniciar via api: %v", err)
		}
	})
	if !strings.Contains(outIniciar, "Tarea #12 en progreso") {
		t.Fatalf("salida iniciar inesperada:\n%s", outIniciar)
	}

	resetCommandFlags(tareaCompletarCmd)
	_ = tareaCompletarCmd.Flags().Set("commit", "abc123")
	outCompletar := capturarStdout(t, func() {
		if err := tareaCompletarCmd.RunE(tareaCompletarCmd, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea completar via api: %v", err)
		}
	})
	if !strings.Contains(outCompletar, "Tarea #12 completada") {
		t.Fatalf("salida completar inesperada:\n%s", outCompletar)
	}

	resetCommandFlags(tareaBloquearCmd)
	_ = tareaBloquearCmd.Flags().Set("motivo", "pendiente de terceros")
	outBloquear := capturarStdout(t, func() {
		if err := tareaBloquearCmd.RunE(tareaBloquearCmd, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea bloquear via api: %v", err)
		}
	})
	if !strings.Contains(outBloquear, "bloqueada") {
		t.Fatalf("salida bloquear inesperada:\n%s", outBloquear)
	}

	outNota := capturarStdout(t, func() {
		if err := tareaNotaCmd.RunE(tareaNotaCmd, []string{"12", "Codex1", "nota", "api"}); err != nil {
			t.Fatalf("tarea nota via api: %v", err)
		}
	})
	if !strings.Contains(outNota, "Nota añadida") {
		t.Fatalf("salida nota inesperada:\n%s", outNota)
	}

	outReasignar := capturarStdout(t, func() {
		if err := tareaReasignarCmd.RunE(tareaReasignarCmd, []string{"12", "Codex2"}); err != nil {
			t.Fatalf("tarea reasignar via api: %v", err)
		}
	})
	if !strings.Contains(outReasignar, "reasignada a Codex2") {
		t.Fatalf("salida reasignar inesperada:\n%s", outReasignar)
	}

	resetCommandFlags(tareaDesbloquearCmd)
	_ = tareaDesbloquearCmd.Flags().Set("resolucion", "desbloqueada por api")
	outDesbloquear := capturarStdout(t, func() {
		if err := tareaDesbloquearCmd.RunE(tareaDesbloquearCmd, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea desbloquear via api: %v", err)
		}
	})
	if !strings.Contains(outDesbloquear, "desbloqueada") {
		t.Fatalf("salida desbloquear inesperada:\n%s", outDesbloquear)
	}

	outBacklog := capturarStdout(t, func() {
		if err := tareaBacklogCmd.RunE(tareaBacklogCmd, []string{"12"}); err != nil {
			t.Fatalf("tarea backlog via api: %v", err)
		}
	})
	if !strings.Contains(outBacklog, "movida a backlog") {
		t.Fatalf("salida backlog inesperada:\n%s", outBacklog)
	}

	outContrato := capturarStdout(t, func() {
		if err := tareaContratoCmd.RunE(tareaContratoCmd, []string{"12", "Codex1"}); err != nil {
			t.Fatalf("tarea contrato via api: %v", err)
		}
	})
	if !strings.Contains(outContrato, "Contrato definido") {
		t.Fatalf("salida contrato inesperada:\n%s", outContrato)
	}
}
