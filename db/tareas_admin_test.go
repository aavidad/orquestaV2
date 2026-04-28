package db

import (
	"database/sql"
	"testing"
)

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
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
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

func TestGetTareaActivaIDPorAgenteProyectoAcotaPorProyecto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	proyectoA, err := UpsertProyecto(&Proyecto{Slug: "a", Nombre: "A", RutaAbs: t.TempDir(), Tipo: ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("UpsertProyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{Slug: "b", Nombre: "B", RutaAbs: t.TempDir(), Tipo: ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("UpsertProyecto B: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{Titulo: "A", Descripcion: "A", ProyectoID: &proyectoA, Modulo: "core", Prioridad: PrioridadAlta, CreadoPor: "test"})
	if err != nil {
		t.Fatalf("CrearTarea A: %v", err)
	}
	tareaB, err := CrearTarea(&Tarea{Titulo: "B", Descripcion: "B", ProyectoID: &proyectoB, Modulo: "core", Prioridad: PrioridadAlta, CreadoPor: "test"})
	if err != nil {
		t.Fatalf("CrearTarea B: %v", err)
	}
	if err := TomarTarea(tareaA, "Gemini1"); err != nil {
		t.Fatalf("TomarTarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Gemini1"); err != nil {
		t.Fatalf("IniciarTarea A: %v", err)
	}
	if err := TomarTarea(tareaB, "Gemini1"); err != nil {
		t.Fatalf("TomarTarea B: %v", err)
	}

	gotA, err := GetTareaActivaIDPorAgenteProyecto("Gemini1", &proyectoA)
	if err != nil {
		t.Fatalf("GetTareaActivaIDPorAgenteProyecto A: %v", err)
	}
	if gotA != tareaA {
		t.Fatalf("tarea activa proyecto A inesperada: got=%d want=%d", gotA, tareaA)
	}
	gotB, err := GetTareaActivaIDPorAgenteProyecto("Gemini1", &proyectoB)
	if err != nil {
		t.Fatalf("GetTareaActivaIDPorAgenteProyecto B: %v", err)
	}
	if gotB != tareaB {
		t.Fatalf("tarea activa proyecto B inesperada: got=%d want=%d", gotB, tareaB)
	}
}

func TestGetTareaMantieneErrNoRowsSiNoExiste(t *testing.T) {
	abrirDBTemporalMemoria(t)

	tarea, err := GetTarea(99999)
	if err != sql.ErrNoRows {
		t.Fatalf("GetTarea deberia devolver sql.ErrNoRows, got tarea=%+v err=%v", tarea, err)
	}
}
