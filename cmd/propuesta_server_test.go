package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestPropuestaUsaAPI(t *testing.T) {
	createdAt := time.Date(2026, 3, 24, 18, 0, 0, 0, time.UTC)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiPropuestasResponse{
				Propuestas: []*db.Propuesta{
					{
						ID:           200,
						Codigo:       "OP-200",
						Titulo:       "Servidor unico",
						Tipo:         "arquitectura",
						Estado:       db.PropuestaAbierta,
						PropuestoPor: "Codex1",
						CreatedAt:    createdAt,
					},
				},
			})
		case http.MethodPost:
			var req apiPropuestaCrearRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode crear propuesta: %v", err)
			}
			if req.Titulo != "Nueva propuesta" {
				t.Fatalf("titulo inesperado: %+v", req)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"id": 201,
				"propuesta": &db.Propuesta{
					ID:           201,
					Codigo:       "OP-201",
					Titulo:       req.Titulo,
					Tipo:         req.Tipo,
					Estado:       db.PropuestaAbierta,
					PropuestoPor: req.PropuestoPor,
					CreatedAt:    createdAt,
				},
			})
		default:
			http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/propuestas/OP-200", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestaDetalleResponse{
			Propuesta: &db.Propuesta{
				ID:           200,
				Codigo:       "OP-200",
				Titulo:       "Servidor unico",
				Tipo:         "arquitectura",
				Estado:       db.PropuestaAbierta,
				PropuestoPor: "Codex1",
				CreatedAt:    createdAt,
				Votos: []*db.Voto{
					{Agente: "Codex1", Posicion: db.VotoAcuerdo, Comentario: "ok"},
				},
			},
		})
	})
	mux.HandleFunc("/api/propuestas/OP-200/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiPropuestaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode accion propuesta: %v", err)
		}
		if req.Accion != "cerrar" || req.EstadoCierre != "consenso" {
			t.Fatalf("accion propuesta inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(propuestaListarCmd)
	resetCommandFlags(propuestaVerCmd)
	resetCommandFlags(propuestaNuevaCmd)
	resetCommandFlags(propuestaCerrarCmd)

	out := capturarStdout(t, func() {
		if err := propuestaListarCmd.RunE(propuestaListarCmd, nil); err != nil {
			t.Fatalf("propuesta listar: %v", err)
		}
	})
	if !strings.Contains(out, "OP-200") {
		t.Fatalf("propuesta listar no uso API:\n%s", out)
	}

	out = capturarStdout(t, func() {
		if err := propuestaVerCmd.RunE(propuestaVerCmd, []string{"OP-200"}); err != nil {
			t.Fatalf("propuesta ver: %v", err)
		}
	})
	if !strings.Contains(out, "Servidor unico") || !strings.Contains(out, "Codex1") {
		t.Fatalf("propuesta ver no uso API:\n%s", out)
	}

	if err := propuestaNuevaCmd.Flags().Set("titulo", "Nueva propuesta"); err != nil {
		t.Fatalf("set titulo propuesta nueva: %v", err)
	}
	if err := propuestaNuevaCmd.Flags().Set("tipo", "implementacion"); err != nil {
		t.Fatalf("set tipo propuesta nueva: %v", err)
	}
	if err := propuestaNuevaCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por propuesta nueva: %v", err)
	}
	out = capturarStdout(t, func() {
		if err := propuestaNuevaCmd.RunE(propuestaNuevaCmd, nil); err != nil {
			t.Fatalf("propuesta nueva: %v", err)
		}
	})
	if !strings.Contains(out, "OP-201") {
		t.Fatalf("propuesta nueva no uso API:\n%s", out)
	}

	if err := propuestaCerrarCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por propuesta cerrar: %v", err)
	}
	out = capturarStdout(t, func() {
		if err := propuestaCerrarCmd.RunE(propuestaCerrarCmd, []string{"OP-200", "consenso"}); err != nil {
			t.Fatalf("propuesta cerrar: %v", err)
		}
	})
	if !strings.Contains(out, "cerrada como 'consenso'") {
		t.Fatalf("propuesta cerrar no uso API:\n%s", out)
	}
}

func TestPropuestaMutacionesYVotosUsanAPI(t *testing.T) {
	createdAt := time.Date(2026, 3, 24, 18, 30, 0, 0, time.UTC)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas/OP-220", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestaDetalleResponse{
			Propuesta: &db.Propuesta{
				ID:           220,
				Codigo:       "OP-220",
				Titulo:       "Propuesta mutable",
				Tipo:         "implementacion",
				Estado:       db.PropuestaAbierta,
				PropuestoPor: "Codex1",
				CreatedAt:    createdAt,
				Votos: []*db.Voto{
					{Agente: "Codex1", Posicion: db.VotoAcuerdo, Comentario: "adelante"},
					{Agente: "Codex2", Posicion: db.VotoDesacuerdo, Comentario: "falta cierre"},
				},
			},
		})
	})
	mux.HandleFunc("/api/propuestas/OP-220/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiPropuestaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode accion propuesta: %v", err)
		}
		switch req.Accion {
		case "actualizar":
			if req.Titulo == nil || *req.Titulo != "Titulo revisado" {
				t.Fatalf("actualizar sin titulo esperado: %+v", req)
			}
		case "reabrir", "reparar_votos":
		default:
			t.Fatalf("accion inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(propuestaActualizarCmd)
	resetCommandFlags(propuestaReabrirCmd)
	resetCommandFlags(propuestaRepararVotosCmd)
	resetCommandFlags(propuestaVotosCmd)

	if err := propuestaActualizarCmd.Flags().Set("titulo", "Titulo revisado"); err != nil {
		t.Fatalf("set titulo propuesta actualizar: %v", err)
	}
	if err := propuestaActualizarCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente propuesta actualizar: %v", err)
	}
	out := capturarStdout(t, func() {
		if err := propuestaActualizarCmd.RunE(propuestaActualizarCmd, []string{"OP-220"}); err != nil {
			t.Fatalf("propuesta actualizar: %v", err)
		}
	})
	if !strings.Contains(out, "actualizada") {
		t.Fatalf("propuesta actualizar no uso API:\n%s", out)
	}

	if err := propuestaVotosCmd.Flags().Set("agente", "codex2"); err != nil {
		t.Fatalf("set agente propuesta votos: %v", err)
	}
	out = capturarStdout(t, func() {
		if err := propuestaVotosCmd.RunE(propuestaVotosCmd, []string{"OP-220"}); err != nil {
			t.Fatalf("propuesta votos: %v", err)
		}
	})
	if !strings.Contains(out, "Codex2") || strings.Contains(out, "Codex1") {
		t.Fatalf("propuesta votos no aplico filtro via API:\n%s", out)
	}

	if err := propuestaReabrirCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente propuesta reabrir: %v", err)
	}
	out = capturarStdout(t, func() {
		if err := propuestaReabrirCmd.RunE(propuestaReabrirCmd, []string{"OP-220"}); err != nil {
			t.Fatalf("propuesta reabrir: %v", err)
		}
	})
	if !strings.Contains(out, "reabierta") {
		t.Fatalf("propuesta reabrir no uso API:\n%s", out)
	}

	if err := propuestaRepararVotosCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente propuesta reparar-votos: %v", err)
	}
	out = capturarStdout(t, func() {
		if err := propuestaRepararVotosCmd.RunE(propuestaRepararVotosCmd, []string{"OP-220"}); err != nil {
			t.Fatalf("propuesta reparar-votos: %v", err)
		}
	})
	if !strings.Contains(out, "reparados") {
		t.Fatalf("propuesta reparar-votos no uso API:\n%s", out)
	}
}

func TestPropuestaRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(propuestaListarCmd)
	err := propuestaListarCmd.RunE(propuestaListarCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
