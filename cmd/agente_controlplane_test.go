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

	"github.com/spf13/cobra"
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

func TestAgenteControlRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	casos := []struct {
		nombre string
		cmd    func() *cobra.Command
		args   []string
		setup  func(t *testing.T, cmd *cobra.Command)
	}{
		{nombre: "control", cmd: func() *cobra.Command { return agenteControlCmd }, args: []string{"arrancar", "Codex1"}},
		{
			nombre: "preparar",
			cmd:    func() *cobra.Command { return agentePrepararCmd },
			args:   []string{"Codex1"},
			setup: func(t *testing.T, cmd *cobra.Command) {
				if err := cmd.Flags().Set("proyecto", "demo"); err != nil {
					t.Fatalf("set proyecto preparar: %v", err)
				}
			},
		},
		{
			nombre: "tick",
			cmd:    func() *cobra.Command { return agenteTickCmd },
			args:   []string{"Codex1"},
			setup: func(t *testing.T, cmd *cobra.Command) {
				if err := cmd.Flags().Set("proyecto", "demo"); err != nil {
					t.Fatalf("set proyecto tick: %v", err)
				}
			},
		},
		{nombre: "handoff", cmd: func() *cobra.Command { return agenteHandoffCmd }, args: []string{"Codex1", "Codex2"}},
		{nombre: "reasignar-vivo", cmd: func() *cobra.Command { return agenteReasignarVivoCmd }, args: []string{"1", "Codex1", "Codex2"}},
		{nombre: "pausar", cmd: func() *cobra.Command { return agentePausarCmd }, args: []string{"Codex1", "15", "rate", "limit"}},
		{nombre: "eliminar", cmd: func() *cobra.Command { return agentePurgarCmd }, args: []string{"Codex1"}},
		{nombre: "rehabilitar", cmd: func() *cobra.Command { return agenteRehabilitarCmd }, args: []string{"Codex1"}},
		{nombre: "fusionar", cmd: func() *cobra.Command { return agenteFusionarCmd }, args: []string{"Codex1", "Codex2"}},
	}

	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			cmd := tc.cmd()
			resetCommandFlags(cmd)
			if tc.setup != nil {
				tc.setup(t, cmd)
			}
			err := cmd.RunE(cmd, tc.args)
			if err == nil {
				t.Fatalf("se esperaba error sin servidor")
			}
			if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}

func TestAgenteComandosExigenServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	agentePrepararCmd.Flags().Set("proyecto", "orquestador")
	defer agentePrepararCmd.Flags().Set("proyecto", "")
	if err := agentePrepararCmd.RunE(agentePrepararCmd, []string{"Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente preparar deberia exigir servidor, err=%v", err)
	}

	agenteTickCmd.Flags().Set("proyecto", "orquestador")
	defer agenteTickCmd.Flags().Set("proyecto", "")
	if err := agenteTickCmd.RunE(agenteTickCmd, []string{"Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente tick deberia exigir servidor, err=%v", err)
	}

	if err := agenteControlCmd.RunE(agenteControlCmd, []string{"arrancar", "Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente control deberia exigir servidor, err=%v", err)
	}

	if err := agentePausarCmd.RunE(agentePausarCmd, []string{"Codex1", "15", "rate", "limit"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente pausar deberia exigir servidor, err=%v", err)
	}

	if err := agenteHandoffCmd.RunE(agenteHandoffCmd, []string{"Codex1", "Codex2"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente handoff deberia exigir servidor, err=%v", err)
	}

	if err := agenteReasignarVivoCmd.RunE(agenteReasignarVivoCmd, []string{"1", "Codex1", "Codex2"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente reasignar-vivo deberia exigir servidor, err=%v", err)
	}
}
