package cmd

import (
	"bytes"
	"strings"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestWebTemplateOpenClawMuestraTrabajoYOrdenOperativos(t *testing.T) {
	tmpl, err := compiledWebTemplate("es", webTplLayout+webTplOpenClaw)
	if err != nil {
		t.Fatalf("compiledWebTemplate: %v", err)
	}

	var body bytes.Buffer
	err = tmpl.ExecuteTemplate(&body, "layout", webOpenClawData{
		Status: apiOpenClawStatusLite{
			AgentesActivos: []apiOpenClawAgentLite{{
				Nombre:            "Codex4",
				Rol:               "programador",
				OperationalState:  "trabajando",
				OperationalDetail: "worker ready",
				CurrentTask: &agentesapp.TaskFocus{
					TaskID: 41,
					Title:  "runtime mailbox",
					State:  db.TareaEnProgreso,
					Module: "cmd",
				},
				DominantOrder: &agentesapp.OrderFocus{
					OrderID: 91,
					Type:    "send_instruction",
					State:   "ejecutando",
					TaskID:  41,
					Action:  "continuar_trabajo",
				},
			}},
		},
		Integration: buildWebOpenClawIntegrationInfo(),
		Generado:    "2026-04-28 12:00:00",
	})
	if err != nil {
		t.Fatalf("render openclaw: %v", err)
	}

	html := body.String()
	for _, token := range []string{
		"Estado real",
		"Trabajo actual",
		"Orden dominante",
		"trabajando",
		"worker ready",
		"#41 [en_progreso] runtime mailbox modulo=cmd",
		"#91 send_instruction [ejecutando] tarea=#41 accion=continuar_trabajo",
	} {
		if !strings.Contains(html, token) {
			t.Fatalf("template openclaw sin %q:\n%s", token, html)
		}
	}
}
