package db

import "testing"

func TestReasignarTarea(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := CrearTarea(&Tarea{
		Titulo:      "Reasignable",
		Descripcion: "prueba",
		Modulo:      "orquestador",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(id, "claude"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}

	if err := ReasignarTarea(id, "codex1"); err != nil {
		t.Fatalf("ReasignarTarea: %v", err)
	}

	tarea, err := GetTarea(id)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "codex1" {
		t.Fatalf("agente inesperado: %+v", tarea)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("estado inesperado: %s", tarea.Estado)
	}
}

func TestMoverTareaABacklog(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := CrearTarea(&Tarea{
		Titulo:      "Backlogeable",
		Descripcion: "prueba",
		Modulo:      "orquestador",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(id, "claude"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}

	if err := MoverTareaABacklog(id); err != nil {
		t.Fatalf("MoverTareaABacklog: %v", err)
	}

	tarea, err := GetTarea(id)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente != nil {
		t.Fatalf("se esperaba tarea sin agente: %+v", tarea)
	}
	if tarea.Estado != TareaBacklog {
		t.Fatalf("estado inesperado: %s", tarea.Estado)
	}
}
