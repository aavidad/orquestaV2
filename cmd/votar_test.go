package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestVotarExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	db.Close()
	defer db.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	_ = votarCmd.Flags().Set("agente", "Codex1")
	defer votarCmd.Flags().Set("agente", "")

	err := votarCmd.RunE(votarCmd, []string{"OP-200", "acuerdo"})
	if err == nil {
		t.Fatalf("votar deberia exigir servidor o recuperacion local explicita")
	}
	if !strings.Contains(err.Error(), "ORQUESTA_FORCE_LOCAL_DB=1") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestVotarUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas/OP-200/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiPropuestaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode votar: %v", err)
		}
		if req.Accion != "votar" || req.Agente != "Codex1" || req.Posicion != "acuerdo" || req.Comentario != "ok" {
			t.Fatalf("payload inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas/OP-200", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestaDetalleResponse{
			Propuesta: &db.Propuesta{Codigo: "OP-200", Estado: db.PropuestaAbierta},
			ConteoVotos: apiConteoVotosResponse{
				Acuerdo: 2, Desacuerdo: 0, Abstencion: 0, Pendiente: 1,
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	_ = votarCmd.Flags().Set("agente", "Codex1")
	_ = votarCmd.Flags().Set("comentario", "ok")
	defer votarCmd.Flags().Set("agente", "")
	defer votarCmd.Flags().Set("comentario", "")

	out := capturarStdout(t, func() {
		if err := votarCmd.RunE(votarCmd, []string{"OP-200", "acuerdo"}); err != nil {
			t.Fatalf("votar via api: %v", err)
		}
	})
	for _, token := range []string{"Voto registrado: Codex1", "Estado votos: ✓2"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida votar sin %q:\n%s", token, out)
		}
	}
}
