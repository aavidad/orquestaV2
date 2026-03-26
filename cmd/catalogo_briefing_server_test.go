/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetCatalogoFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = cmd.Flags().Set(f.Name, f.DefValue)
		f.Changed = false
	})
}

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
	mux.HandleFunc("/api/reglas/31", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiReglaResponse{Regla: &db.Regla{ID: 31, TipoAgente: "programador", Categoria: "calidad", Titulo: "No romper", Descripcion: "verde", Activa: true}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 31})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/reglas/31/activa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 31})
	})
	mux.HandleFunc("/api/reglas/31/versiones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(apiReglaVersionesResponse{Versiones: []*db.ReglaVersion{{VersionNum: 1, Activa: true, Actor: "Codex1", Accion: "crear", TipoAgente: "programador", Titulo: "No romper"}}})
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
	mux.HandleFunc("/api/skills/32", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiSkillResponse{Skill: &db.Skill{ID: 32, TipoAgente: "programador", Nombre: "rg", Descripcion: "buscar", CuandoUsar: "texto", Activa: true}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 32})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/skills/32/activa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 32})
	})
	mux.HandleFunc("/api/skills/32/versiones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(apiSkillVersionesResponse{Versiones: []*db.SkillVersion{{VersionNum: 2, Activa: true, Actor: "Codex1", Accion: "actualizar", TipoAgente: "programador", Nombre: "rg"}}})
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
	mux.HandleFunc("/api/workflows/33", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiWorkflowResponse{Workflow: &db.Workflow{ID: 33, TipoAgente: "programador", Nombre: "inicio-sesion", Descripcion: "flujo base", Pasos: "[\"leer\",\"votar\"]", Activo: true}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiCatalogoMutationResponse{ID: 33})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows/33/activa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 33})
	})
	mux.HandleFunc("/api/workflows/33/versiones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(apiWorkflowVersionesResponse{Versiones: []*db.WorkflowVersion{{VersionNum: 3, Activo: true, Actor: "Codex1", Accion: "actualizar", TipoAgente: "programador", Nombre: "inicio-sesion"}}})
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

	resetCatalogoFlags(reglasCrearCmd)
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

	resetCatalogoFlags(reglasEditarCmd)
	if err := reglasEditarCmd.Flags().Set("titulo", "No romper nunca"); err != nil {
		t.Fatalf("set titulo regla editar: %v", err)
	}
	outReglaEditar := capturarStdout(t, func() {
		if err := reglasEditarCmd.RunE(reglasEditarCmd, []string{"31"}); err != nil {
			t.Fatalf("reglas editar via api: %v", err)
		}
	})
	if !strings.Contains(outReglaEditar, "#31") {
		t.Fatalf("salida regla editar sin id remoto:\n%s", outReglaEditar)
	}

	resetCatalogoFlags(reglasActivarCmd)
	outReglaActivar := capturarStdout(t, func() {
		if err := reglasActivarCmd.RunE(reglasActivarCmd, []string{"31"}); err != nil {
			t.Fatalf("reglas activar via api: %v", err)
		}
	})
	if !strings.Contains(outReglaActivar, "activado") {
		t.Fatalf("salida regla activar inesperada:\n%s", outReglaActivar)
	}

	outReglaVersiones := capturarStdout(t, func() {
		if err := reglasVersionesCmd.RunE(reglasVersionesCmd, []string{"31"}); err != nil {
			t.Fatalf("reglas versiones via api: %v", err)
		}
	})
	if !strings.Contains(outReglaVersiones, "No romper") {
		t.Fatalf("salida regla versiones inesperada:\n%s", outReglaVersiones)
	}

	resetCatalogoFlags(skillsCrearCmd)
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

	resetCatalogoFlags(skillsEditarCmd)
	if err := skillsEditarCmd.Flags().Set("descripcion", "buscar rapido"); err != nil {
		t.Fatalf("set descripcion skill editar: %v", err)
	}
	outSkillEditar := capturarStdout(t, func() {
		if err := skillsEditarCmd.RunE(skillsEditarCmd, []string{"32"}); err != nil {
			t.Fatalf("skills editar via api: %v", err)
		}
	})
	if !strings.Contains(outSkillEditar, "#32") {
		t.Fatalf("salida skill editar sin id remoto:\n%s", outSkillEditar)
	}

	resetCatalogoFlags(skillsActivarCmd)
	outSkillActivar := capturarStdout(t, func() {
		if err := skillsActivarCmd.RunE(skillsActivarCmd, []string{"32"}); err != nil {
			t.Fatalf("skills activar via api: %v", err)
		}
	})
	if !strings.Contains(outSkillActivar, "activado") {
		t.Fatalf("salida skill activar inesperada:\n%s", outSkillActivar)
	}

	outSkillVersiones := capturarStdout(t, func() {
		if err := skillsVersionesCmd.RunE(skillsVersionesCmd, []string{"32"}); err != nil {
			t.Fatalf("skills versiones via api: %v", err)
		}
	})
	if !strings.Contains(outSkillVersiones, "rg") {
		t.Fatalf("salida skill versiones inesperada:\n%s", outSkillVersiones)
	}

	resetCatalogoFlags(workflowsCrearCmd)
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

	outWorkflowVer := capturarStdout(t, func() {
		if err := workflowsVerCmd.RunE(workflowsVerCmd, []string{"33"}); err != nil {
			t.Fatalf("workflows ver via api: %v", err)
		}
	})
	if !strings.Contains(outWorkflowVer, "Workflow #33") {
		t.Fatalf("salida workflow ver inesperada:\n%s", outWorkflowVer)
	}

	resetCatalogoFlags(workflowsEditarCmd)
	if err := workflowsEditarCmd.Flags().Set("descripcion", "flujo refinado"); err != nil {
		t.Fatalf("set descripcion workflow editar: %v", err)
	}
	outWorkflowEditar := capturarStdout(t, func() {
		if err := workflowsEditarCmd.RunE(workflowsEditarCmd, []string{"33"}); err != nil {
			t.Fatalf("workflows editar via api: %v", err)
		}
	})
	if !strings.Contains(outWorkflowEditar, "#33") {
		t.Fatalf("salida workflow editar sin id remoto:\n%s", outWorkflowEditar)
	}

	resetCatalogoFlags(workflowsActivarCmd)
	outWorkflowActivar := capturarStdout(t, func() {
		if err := workflowsActivarCmd.RunE(workflowsActivarCmd, []string{"33"}); err != nil {
			t.Fatalf("workflows activar via api: %v", err)
		}
	})
	if !strings.Contains(outWorkflowActivar, "activado") {
		t.Fatalf("salida workflow activar inesperada:\n%s", outWorkflowActivar)
	}

	outWorkflowVersiones := capturarStdout(t, func() {
		if err := workflowsVersionesCmd.RunE(workflowsVersionesCmd, []string{"33"}); err != nil {
			t.Fatalf("workflows versiones via api: %v", err)
		}
	})
	if !strings.Contains(outWorkflowVersiones, "inicio-sesion") {
		t.Fatalf("salida workflow versiones inesperada:\n%s", outWorkflowVersiones)
	}

	resetCatalogoFlags(permisosFijarCmd)
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

func TestCatalogoRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	cases := []struct {
		nombre string
		cmd    *cobra.Command
		args   []string
		setup  func(t *testing.T)
	}{
		{nombre: "reglas listar", cmd: reglasListarCmd},
		{nombre: "reglas crear", cmd: reglasCrearCmd},
		{nombre: "reglas editar", cmd: reglasEditarCmd, args: []string{"1"}},
		{nombre: "reglas activar", cmd: reglasActivarCmd, args: []string{"1"}},
		{nombre: "reglas versiones", cmd: reglasVersionesCmd, args: []string{"1"}},
		{nombre: "skills listar", cmd: skillsListarCmd},
		{nombre: "skills crear", cmd: skillsCrearCmd},
		{nombre: "skills editar", cmd: skillsEditarCmd, args: []string{"1"}},
		{nombre: "skills activar", cmd: skillsActivarCmd, args: []string{"1"}},
		{nombre: "skills versiones", cmd: skillsVersionesCmd, args: []string{"1"}},
		{nombre: "workflows listar", cmd: workflowsListarCmd},
		{nombre: "workflows ver", cmd: workflowsVerCmd, args: []string{"1"}},
		{nombre: "workflows crear", cmd: workflowsCrearCmd},
		{nombre: "workflows editar", cmd: workflowsEditarCmd, args: []string{"1"}},
		{nombre: "workflows activar", cmd: workflowsActivarCmd, args: []string{"1"}},
		{nombre: "workflows versiones", cmd: workflowsVersionesCmd, args: []string{"1"}},
		{nombre: "permisos listar", cmd: permisosListarCmd},
		{
			nombre: "permisos fijar",
			cmd:    permisosFijarCmd,
			setup: func(t *testing.T) {
				resetCatalogoFlags(permisosFijarCmd)
				if err := permisosFijarCmd.Flags().Set("entidad", "reglas"); err != nil {
					t.Fatalf("set entidad: %v", err)
				}
				if err := permisosFijarCmd.Flags().Set("rol", "programador"); err != nil {
					t.Fatalf("set rol: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			resetCatalogoFlags(tc.cmd)
			if tc.setup != nil {
				tc.setup(t)
			}
			err := tc.cmd.RunE(tc.cmd, tc.args)
			if err == nil {
				t.Fatalf("se esperaba error sin servidor")
			}
			if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}
