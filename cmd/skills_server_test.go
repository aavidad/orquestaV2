package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/skillsapp"
)

func TestSkillsComandosUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/skills/remotas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSkillsRemotasResponse{
			Items: []*skillsapp.SkillRemota{{
				Nombre:      "openai-docs",
				Repo:        "openai/skills",
				Skill:       "openai-docs",
				URLCanonica: "https://skills.sh/openai/skills/openai-docs",
			}},
		})
	})
	mux.HandleFunc("/api/skills/importar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSkillImportarResponse{
			ID:        41,
			Existente: false,
			Skill: &db.Skill{
				ID:         41,
				Nombre:     "openai-docs",
				Origen:     "third_party",
				Activa:     false,
				TipoAgente: "programador",
			},
		})
	})
	mux.HandleFunc("/api/skills/detectar-carencia", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSkillDeteccionResponse{
			Resultado: &db.ResultadoDeteccionSkill{
				Falta:             true,
				Motivo:            "no existe skill equivalente",
				InvocacionCreador: "orquesta skills crear --rol programador --nombre goimports",
			},
		})
	})
	mux.HandleFunc("/api/skills/42/borrar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(skillsRemotasCmd)
	if err := skillsRemotasCmd.Flags().Set("q", "openai"); err != nil {
		t.Fatalf("set q: %v", err)
	}
	outRemotas := capturarStdout(t, func() {
		if err := skillsRemotasCmd.RunE(skillsRemotasCmd, nil); err != nil {
			t.Fatalf("skills remotas via api: %v", err)
		}
	})
	if !strings.Contains(outRemotas, "openai-docs") {
		t.Fatalf("salida remotas inesperada:\n%s", outRemotas)
	}

	resetCommandFlags(skillsImportarCmd)
	if err := skillsImportarCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol importar: %v", err)
	}
	if err := skillsImportarCmd.Flags().Set("url", "https://skills.sh/openai/skills/openai-docs"); err != nil {
		t.Fatalf("set url importar: %v", err)
	}
	outImportar := capturarStdout(t, func() {
		if err := skillsImportarCmd.RunE(skillsImportarCmd, nil); err != nil {
			t.Fatalf("skills importar via api: %v", err)
		}
	})
	if !strings.Contains(outImportar, "#41") {
		t.Fatalf("salida importar inesperada:\n%s", outImportar)
	}

	resetCommandFlags(skillsDetectarCarenciaCmd)
	if err := skillsDetectarCarenciaCmd.Flags().Set("rol", "programador"); err != nil {
		t.Fatalf("set rol detectar: %v", err)
	}
	if err := skillsDetectarCarenciaCmd.Flags().Set("nombre", "goimports"); err != nil {
		t.Fatalf("set nombre detectar: %v", err)
	}
	var bufDetectar bytes.Buffer
	prevOut := skillsDetectarCarenciaCmd.OutOrStdout()
	prevErr := skillsDetectarCarenciaCmd.ErrOrStderr()
	t.Cleanup(func() {
		skillsDetectarCarenciaCmd.SetOut(prevOut)
		skillsDetectarCarenciaCmd.SetErr(prevErr)
	})
	skillsDetectarCarenciaCmd.SetOut(&bufDetectar)
	skillsDetectarCarenciaCmd.SetErr(&bufDetectar)
	if err := skillsDetectarCarenciaCmd.RunE(skillsDetectarCarenciaCmd, nil); err != nil {
		t.Fatalf("skills detectar via api: %v", err)
	}
	outDetectar := bufDetectar.String()
	if !strings.Contains(outDetectar, "\"falta\": true") || !strings.Contains(outDetectar, "no existe skill equivalente") {
		t.Fatalf("salida detectar inesperada:\n%s", outDetectar)
	}

	resetCommandFlags(skillsBorrarCmd)
	outBorrar := capturarStdout(t, func() {
		if err := skillsBorrarCmd.RunE(skillsBorrarCmd, []string{"42"}); err != nil {
			t.Fatalf("skills borrar via api: %v", err)
		}
	})
	if !strings.Contains(outBorrar, "#42") {
		t.Fatalf("salida borrar inesperada:\n%s", outBorrar)
	}
}

func TestSkillsComandosRequierenServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	casos := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{name: "remotas", cmd: skillsRemotasCmd},
		{name: "importar", cmd: skillsImportarCmd},
		{name: "detectar", cmd: skillsDetectarCarenciaCmd},
		{name: "borrar", cmd: skillsBorrarCmd, args: []string{"42"}},
	}

	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			resetCommandFlags(tc.cmd)
			if tc.cmd == skillsImportarCmd {
				_ = tc.cmd.Flags().Set("rol", "programador")
				_ = tc.cmd.Flags().Set("url", "https://skills.sh/openai/skills/openai-docs")
			}
			if tc.cmd == skillsDetectarCarenciaCmd {
				_ = tc.cmd.Flags().Set("rol", "programador")
				_ = tc.cmd.Flags().Set("nombre", "goimports")
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
