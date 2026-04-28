package cmd

import (
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestCompactOpenClawAgentsPropagaFocoOperativoDesdePanel(t *testing.T) {
	task := &agentesapp.TaskFocus{
		TaskID: 41,
		Title:  "runtime mailbox/session_resume",
		State:  db.TareaEnProgreso,
		Module: "cmd",
	}
	order := &agentesapp.OrderFocus{
		OrderID: 91,
		Type:    "send_instruction",
		State:   "ejecutando",
		TaskID:  41,
		Action:  "continuar_trabajo",
	}
	items := compactOpenClawAgents([]*db.Agente{{
		Nombre: "Codex4",
		Rol:    "programador",
	}}, []tareaLite{{
		ID:     41,
		Agente: "Codex4",
		Estado: db.TareaEnProgreso,
	}}, []agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Codex4"},
		EstadoOperativo:  "trabajando",
		DetalleOperativo: "worker ready",
		CurrentTask:      task,
		DominantOrder:    order,
	}})

	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].OperationalState != "trabajando" || items[0].OperationalDetail != "worker ready" {
		t.Fatalf("estado operativo inesperado: %+v", items[0])
	}
	if items[0].CurrentTask == nil || items[0].CurrentTask.TaskID != 41 {
		t.Fatalf("current task inesperada: %+v", items[0].CurrentTask)
	}
	if items[0].DominantOrder == nil || items[0].DominantOrder.OrderID != 91 {
		t.Fatalf("dominant order inesperada: %+v", items[0].DominantOrder)
	}
}
