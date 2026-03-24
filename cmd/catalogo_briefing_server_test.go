package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCatalogoMutacionesUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/reglas", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiReglasResponse{})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 21})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiSkillsResponse{})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 22})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiWorkflowsResponse{})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 23})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/permisos-catalogo", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiPermisosCatalogoResponse{})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := reglasCrearCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol regla: %v", err)
	}
	if err := reglasCrearCmd.Flags().Set("categoria", "calidad"); err != nil {
		t.Fatalf("set categoria regla: %v", err)
	}
	if err := reglasCrearCmd.Flags().Set("titulo", "No romper tests"); err != nil {
		t.Fatalf("set titulo regla: %v", err)
	}
	outRegla := capturarStdout(t, func() {
		if err := reglasCrearCmd.RunE(reglasCrearCmd, nil); err != nil {
			t.Fatalf("reglas crear via api: %v", err)
		}
	})
	if !strings.Contains(outRegla, "#21") {
		t.Fatalf("salida regla crear sin id remoto:\n%s", outRegla)
	}

	if err := skillsCrearCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol skill: %v", err)
	}
	if err := skillsCrearCmd.Flags().Set("nombre", "rg"); err != nil {
		t.Fatalf("set nombre skill: %v", err)
	}
	outSkill := capturarStdout(t, func() {
		if err := skillsCrearCmd.RunE(skillsCrearCmd, nil); err != nil {
			t.Fatalf("skills crear via api: %v", err)
		}
	})
	if !strings.Contains(outSkill, "#22") {
		t.Fatalf("salida skill crear sin id remoto:\n%s", outSkill)
	}

	if err := workflowsCrearCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol workflow: %v", err)
	}
	if err := workflowsCrearCmd.Flags().Set("nombre", "inicio-sesion"); err != nil {
		t.Fatalf("set nombre workflow: %v", err)
	}
	if err := workflowsCrearCmd.Flags().Set("paso", "leer"); err != nil {
		t.Fatalf("set paso workflow: %v", err)
	}
	outWorkflow := capturarStdout(t, func() {
		if err := workflowsCrearCmd.RunE(workflowsCrearCmd, nil); err != nil {
			t.Fatalf("workflows crear via api: %v", err)
		}
	})
	if !strings.Contains(outWorkflow, "#23") {
		t.Fatalf("salida workflow crear sin id remoto:\n%s", outWorkflow)
	}

	if err := permisosFijarCmd.Flags().Set("entidad", "reglas"); err != nil {
		t.Fatalf("set entidad permiso: %v", err)
	}
	if err := permisosFijarCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol permiso: %v", err)
	}
	if err := permisosFijarCmd.Flags().Set("crear", "true"); err != nil {
		t.Fatalf("set crear permiso: %v", err)
	}
	outPermiso := capturarStdout(t, func() {
		if err := permisosFijarCmd.RunE(permisosFijarCmd, nil); err != nil {
			t.Fatalf("permisos fijar via api: %v", err)
		}
	})
	if !strings.Contains(outPermiso, "reglas/programador") {
		t.Fatalf("salida permiso fijar inesperada:\n%s", outPermiso)
	}
}
