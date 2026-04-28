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
		case r.URL.Path == "/api/agente/investigar" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"investigacion": map[string]any{
					"query":         r.URL.Query().Get("q"),
					"total_matches": 1,
					"results": []map[string]any{{
						"agente":     map[string]any{"nombre": "Codex2", "rol": "programador"},
						"open_tasks": 1,
						"matches": []map[string]any{{
							"trace_dir": "/repo/.orquesta-runtime/codex2/20260329-120000-000000001",
							"log_path":  "/repo/.orquesta-runtime/codex2/20260329-120000-000000001/pty.log",
							"transcript": map[string]any{
								"id":         90,
								"agente":     "Codex2",
								"stream":     "pty_out",
								"text":       "He implementado el refactor del router",
								"created_at": "2026-03-29T12:00:00Z",
							},
						}},
					}},
				},
			})
		case r.URL.Path == "/api/agentes/Codex1/overview" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": map[string]any{
					"row": map[string]any{
						"estado_operativo":  "trabajando",
						"detalle_operativo": "worker ready",
						"asignacion": map[string]any{
							"proyecto_slug": "orquestador",
							"estado":        "activa",
							"nota":          "microciclo_exclusivo",
						},
						"sesion": map[string]any{
							"id":          1809,
							"estado":      "activa",
							"herramienta": "codex-cli",
						},
						"runtime": map[string]any{
							"id":            1708,
							"logical_state": "activo",
						},
						"handle": map[string]any{
							"id":         1702,
							"estado":     "activo",
							"transporte": "tmux",
						},
					},
					"entity": map[string]any{
						"name": "Codex1",
						"role": "programador",
						"current_task": map[string]any{
							"task_id": 558,
							"title":   "frente activo",
							"state":   "en_progreso",
							"module":  "cmd",
						},
						"dominant_order": map[string]any{
							"order_id": 41,
							"type":     "send_instruction",
							"state":    "ejecutando",
							"task_id":  558,
							"action":   "continuar_trabajo",
						},
						"leases": []map[string]any{{
							"task_id": 558,
							"title":   "frente activo",
							"state":   "en_progreso",
						}},
					},
					"mailbox":                   []map[string]any{{"id": 99, "estado": "pendiente"}},
					"mailbox_pending_visible":   0,
					"mailbox_covered_bootstrap": 1,
				},
			})
		case r.URL.Path == "/api/agente/pausar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agente": "Codex1"})
		case r.URL.Path == "/api/agente/handoff" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": 33})
		case r.URL.Path == "/api/agente/adoptar-contexto" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sesion":             map[string]any{"id": 44, "agente": "Codex1"},
				"checkpoint":         map[string]any{"id": 55, "checkpoint_kind": "contexto_adoptado"},
				"runtime_order_id":   66,
				"rol":                "programador",
				"governance_catalog": map[string]any{"resolucion_actual": "rol+proyecto"},
			})
		case r.URL.Path == "/api/agentes" && r.Method == http.MethodPost:
			var req apiAgenteRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode agente request: %v", err)
			}
			nombre := strings.TrimSpace(req.Nombre)
			if nombre == "" {
				nombre = "Ollama1"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "nombre": nombre, "rol": req.Rol})
		case r.URL.Path == "/api/agentes/Codex1/rehabilitar" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agente": "Codex1"})
		case r.URL.Path == "/api/agentes/Codex1/retirar" && r.Method == http.MethodPost:
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

	outInvestigar := capturarStdout(t, func() {
		if err := agenteInvestigarCmd.RunE(agenteInvestigarCmd, []string{"refactor", "router"}); err != nil {
			t.Fatalf("agente investigar via api: %v", err)
		}
	})
	if !strings.Contains(outInvestigar, "Codex2") || !strings.Contains(outInvestigar, "refactor del router") {
		t.Fatalf("salida investigar inesperada:\n%s", outInvestigar)
	}

	outHandoff := capturarStdout(t, func() {
		if err := agenteHandoffCmd.RunE(agenteHandoffCmd, []string{"Codex1", "Codex2"}); err != nil {
			t.Fatalf("agente handoff via api: %v", err)
		}
	})
	if !strings.Contains(outHandoff, "runtime_order: 33") {
		t.Fatalf("salida handoff inesperada:\n%s", outHandoff)
	}

	resetCommandFlags(agenteAdoptarContextoCmd)
	if err := agenteAdoptarContextoCmd.Flags().Set("proyecto", "demo"); err != nil {
		t.Fatalf("set proyecto adoptar-contexto: %v", err)
	}
	if err := agenteAdoptarContextoCmd.Flags().Set("cwd", "/tmp/demo"); err != nil {
		t.Fatalf("set cwd adoptar-contexto: %v", err)
	}
	if err := agenteAdoptarContextoCmd.Flags().Set("branch", "feature/demo"); err != nil {
		t.Fatalf("set branch adoptar-contexto: %v", err)
	}
	outAdoptar := capturarStdout(t, func() {
		if err := agenteAdoptarContextoCmd.RunE(agenteAdoptarContextoCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente adoptar-contexto via api: %v", err)
		}
	})
	if !strings.Contains(outAdoptar, "Contexto adoptado para Codex1 en demo") || !strings.Contains(outAdoptar, "Runtime order: 66") {
		t.Fatalf("salida adoptar-contexto inesperada:\n%s", outAdoptar)
	}

	outReset := capturarStdout(t, func() {
		if err := agenteRehabilitarCmd.RunE(agenteRehabilitarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente rehabilitar via api: %v", err)
		}
	})
	if !strings.Contains(outReset, "rehabilitado correctamente") {
		t.Fatalf("salida rehabilitar inesperada:\n%s", outReset)
	}

	outRetirar := capturarStdout(t, func() {
		if err := agenteRetirarCmd.RunE(agenteRetirarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente retirar via api: %v", err)
		}
	})
	if !strings.Contains(outRetirar, "retirado correctamente") {
		t.Fatalf("salida retirar inesperada:\n%s", outRetirar)
	}

	outEliminar := capturarStdout(t, func() {
		if err := agentePurgarCmd.RunE(agentePurgarCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente eliminar via api: %v", err)
		}
	})
	if !strings.Contains(outEliminar, "eliminado correctamente") {
		t.Fatalf("salida eliminar inesperada:\n%s", outEliminar)
	}

	outRegistrar := capturarStdout(t, func() {
		if err := agenteRegistrarCmd.RunE(agenteRegistrarCmd, []string{"Qwen1"}); err != nil {
			t.Fatalf("agente registrar via api: %v", err)
		}
	})
	if !strings.Contains(outRegistrar, "Qwen1") {
		t.Fatalf("salida registrar inesperada:\n%s", outRegistrar)
	}

	resetCommandFlags(agenteRegistrarCmd)
	if err := agenteRegistrarCmd.Flags().Set("proveedor", "ollama"); err != nil {
		t.Fatalf("set proveedor registrar: %v", err)
	}
	outRegistrarAuto := capturarStdout(t, func() {
		if err := agenteRegistrarCmd.RunE(agenteRegistrarCmd, nil); err != nil {
			t.Fatalf("agente registrar auto via api: %v", err)
		}
	})
	if !strings.Contains(outRegistrarAuto, "Ollama1") {
		t.Fatalf("salida registrar auto inesperada:\n%s", outRegistrarAuto)
	}

	outOverview := capturarStdout(t, func() {
		if err := agenteOverviewCmd.RunE(agenteOverviewCmd, []string{"Codex1"}); err != nil {
			t.Fatalf("agente overview via api: %v", err)
		}
	})
	if !strings.Contains(outOverview, "Mailbox:   0 pendiente(s) · 1 cubierta(s) / 1 total") {
		t.Fatalf("salida overview inesperada:\n%s", outOverview)
	}
	if !strings.Contains(outOverview, "Tareas:    1 lease(s) · abiertas=1 bloqueadas=0") {
		t.Fatalf("salida overview sin resumen de leases:\n%s", outOverview)
	}
	if !strings.Contains(outOverview, "Trabajo:   #558 [en_progreso] frente activo modulo=cmd") ||
		!strings.Contains(outOverview, "Orden:     #41 send_instruction [ejecutando] tarea=#558 accion=continuar_trabajo") {
		t.Fatalf("salida overview sin foco canonico:\n%s", outOverview)
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

func TestAgenteOverviewMarcaContinuidadEmbebidaEnMailboxCompacta(t *testing.T) {
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/agentes/Codex3/overview" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail": map[string]any{
					"row": map[string]any{
						"EstadoOperativo":    "trabajando",
						"DetalleOperativo":   "worker ready",
						"LastAutonomySource": "resume_payload_mailbox",
						"LastAutonomyAction": "continuar_trabajo",
						"LastAutonomyState":  "embedded",
						"MailboxPending":     1,
						"MailboxTotal":       1,
					},
					"entity": map[string]any{
						"Name":   "Codex3",
						"Role":   "programador",
						"Leases": []map[string]any{},
					},
					"mailbox_pending_visible":   1,
					"mailbox_covered_bootstrap": 0,
					"mailbox_total_count":       1,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_ADDR", strings.TrimPrefix(srv.URL, "http://"))()

	outOverview := capturarStdout(t, func() {
		if err := agenteOverviewCmd.RunE(agenteOverviewCmd, []string{"Codex3"}); err != nil {
			t.Fatalf("agente overview via api: %v", err)
		}
	})
	if !strings.Contains(outOverview, "Mailbox:   1 pendiente(s) · continuidad embebida / 1 total") {
		t.Fatalf("salida overview sin marca de continuidad embebida:\n%s", outOverview)
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
		{nombre: "investigar", cmd: func() *cobra.Command { return agenteInvestigarCmd }, args: []string{"refactor"}},
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
		{
			nombre: "adoptar-contexto",
			cmd:    func() *cobra.Command { return agenteAdoptarContextoCmd },
			args:   []string{"Codex1"},
			setup: func(t *testing.T, cmd *cobra.Command) {
				if err := cmd.Flags().Set("proyecto", "demo"); err != nil {
					t.Fatalf("set proyecto adoptar-contexto: %v", err)
				}
				if err := cmd.Flags().Set("cwd", "/tmp/demo"); err != nil {
					t.Fatalf("set cwd adoptar-contexto: %v", err)
				}
				if err := cmd.Flags().Set("branch", "feature/demo"); err != nil {
					t.Fatalf("set branch adoptar-contexto: %v", err)
				}
			},
		},
		{nombre: "handoff", cmd: func() *cobra.Command { return agenteHandoffCmd }, args: []string{"Codex1", "Codex2"}},
		{nombre: "reasignar-vivo", cmd: func() *cobra.Command { return agenteReasignarVivoCmd }, args: []string{"1", "Codex1", "Codex2"}},
		{nombre: "registrar", cmd: func() *cobra.Command { return agenteRegistrarCmd }, args: []string{"Qwen1"}},
		{nombre: "pausar", cmd: func() *cobra.Command { return agentePausarCmd }, args: []string{"Codex1", "15", "rate", "limit"}},
		{nombre: "retirar", cmd: func() *cobra.Command { return agenteRetirarCmd }, args: []string{"Codex1"}},
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

	if err := agenteOverviewCmd.RunE(agenteOverviewCmd, []string{"Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente overview deberia exigir servidor, err=%v", err)
	}
	if err := agenteReanimacionesCmd.RunE(agenteReanimacionesCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente reanimaciones deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(agenteAdoptarContextoCmd)
	agenteAdoptarContextoCmd.Flags().Set("proyecto", "orquestador")
	agenteAdoptarContextoCmd.Flags().Set("cwd", "/tmp/orquestador")
	agenteAdoptarContextoCmd.Flags().Set("branch", "main")
	defer agenteAdoptarContextoCmd.Flags().Set("proyecto", "")
	defer agenteAdoptarContextoCmd.Flags().Set("cwd", "")
	defer agenteAdoptarContextoCmd.Flags().Set("branch", "")
	if err := agenteAdoptarContextoCmd.RunE(agenteAdoptarContextoCmd, []string{"Codex1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente adoptar-contexto deberia exigir servidor, err=%v", err)
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

	if err := agenteRegistrarCmd.RunE(agenteRegistrarCmd, []string{"Qwen1"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("agente registrar deberia exigir servidor, err=%v", err)
	}
}
