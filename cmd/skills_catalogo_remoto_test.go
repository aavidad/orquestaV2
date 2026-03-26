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
)

func TestSkillsRemotasYBorrarUsanAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/skills/remotas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{
				"skill":        "openai-docs",
				"repo":         "openai/skills",
				"url_canonica": "https://skills.sh/openai/skills/openai-docs",
			}},
		})
	})
	mux.HandleFunc("/api/skills/7/borrar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(skillsRemotasCmd)
	_ = skillsRemotasCmd.Flags().Set("q", "openai")
	outListar := capturarStdout(t, func() {
		if err := skillsRemotasCmd.RunE(skillsRemotasCmd, nil); err != nil {
			t.Fatalf("skills remotas via api: %v", err)
		}
	})
	for _, token := range []string{"openai-docs", "openai/skills", "skills.sh"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida skills remotas sin %q:\n%s", token, outListar)
		}
	}

	resetCommandFlags(skillsBorrarCmd)
	_ = skillsBorrarCmd.Flags().Set("agente", "Codex1")
	outBorrar := capturarStdout(t, func() {
		if err := skillsBorrarCmd.RunE(skillsBorrarCmd, []string{"7"}); err != nil {
			t.Fatalf("skills borrar via api: %v", err)
		}
	})
	if !strings.Contains(outBorrar, "Skill #7 borrada") {
		t.Fatalf("salida skills borrar inesperada:\n%s", outBorrar)
	}
}

func TestSkillsRemotasYBorrarExigenServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(skillsRemotasCmd)
	if err := skillsRemotasCmd.RunE(skillsRemotasCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("skills remotas deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(skillsBorrarCmd)
	_ = skillsBorrarCmd.Flags().Set("agente", "Codex1")
	if err := skillsBorrarCmd.RunE(skillsBorrarCmd, []string{"7"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("skills borrar deberia exigir servidor, err=%v", err)
	}
}
