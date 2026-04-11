package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/microprogramacionapp"
)

func TestMicroprogramacionEspecificacionCrearUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/microprogramacion/especificaciones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiEspecificacionFuncionCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Proyecto != "orquestador" || req.ArchivoObjetivo != "runtimeagente/driver.go" || req.SimboloObjetivo != "BuildSpec" {
			t.Fatalf("request inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiEspecificacionFuncionCreateResponse{
			OK: true,
			ID: 31,
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(microprogramacionEspecificacionCrearCmd)
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("proyecto", "orquestador")
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("archivo", "runtimeagente/driver.go")
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("simbolo", "BuildSpec")
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("descripcion", "Construir la especificacion de driver")
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("test", "go test ./runtimeagente -run TestBuildSpec")
	_ = microprogramacionEspecificacionCrearCmd.Flags().Set("write-set", "runtimeagente/driver.go")

	out := capturarStdout(t, func() {
		if err := microprogramacionEspecificacionCrearCmd.RunE(microprogramacionEspecificacionCrearCmd, nil); err != nil {
			t.Fatalf("crear via API: %v", err)
		}
	})
	if !strings.Contains(out, "Especificacion creada (id: 31)") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestMicroprogramacionEspecificacionListarYVerUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/microprogramacion/especificaciones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
			t.Fatalf("proyecto inesperado: %q", got)
		}
		_ = json.NewEncoder(w).Encode(apiEspecificacionesFuncionResponse{
			Especificaciones: []*microprogramacionapp.EspecificacionFuncion{
				{ID: 44, Estado: microprogramacionapp.EstadoEspecificacionActiva, ArchivoObjetivo: "runtimeagente/driver.go", SimboloObjetivo: "BuildSpec", TestsObligatorios: []string{"go test ./runtimeagente -run TestBuildSpec"}},
			},
		})
	})
	mux.HandleFunc("/api/microprogramacion/especificaciones/44", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiEspecificacionFuncionResponse{
			Especificacion: &microprogramacionapp.EspecificacionFuncion{
				ID:                44,
				Titulo:            "runtimeagente/driver.go::BuildSpec",
				ArchivoObjetivo:   "runtimeagente/driver.go",
				SimboloObjetivo:   "BuildSpec",
				Descripcion:       "Construye la especificacion",
				FormatoSalida:     "patch+evidencia",
				CreadoPor:         "alberto",
				TestsObligatorios: []string{"go test ./runtimeagente -run TestBuildSpec"},
				WriteSet:          []string{"runtimeagente/driver.go"},
				Estado:            microprogramacionapp.EstadoEspecificacionActiva,
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(microprogramacionEspecificacionListarCmd)
	_ = microprogramacionEspecificacionListarCmd.Flags().Set("proyecto", "orquestador")
	out := capturarStdout(t, func() {
		if err := microprogramacionEspecificacionListarCmd.RunE(microprogramacionEspecificacionListarCmd, nil); err != nil {
			t.Fatalf("listar via API: %v", err)
		}
	})
	if !strings.Contains(out, "runtimeagente/driver.go") {
		t.Fatalf("salida listar inesperada:\n%s", out)
	}

	out = capturarStdout(t, func() {
		if err := microprogramacionEspecificacionVerCmd.RunE(microprogramacionEspecificacionVerCmd, []string{"44"}); err != nil {
			t.Fatalf("ver via API: %v", err)
		}
	})
	for _, token := range []string{"Especificación #44", "BuildSpec", "patch+evidencia"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida ver sin %q:\n%s", token, out)
		}
	}
}
