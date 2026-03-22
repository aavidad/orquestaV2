/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestCalcularResumenProgresoProyecto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	faseDisenoID, err := RegistrarFaseProyecto(&FaseProyecto{
		Proyecto: "orquestador",
		Nombre:   "Diseno",
		Orden:    10,
		Peso:     1,
		Estado:   "activa",
	})
	if err != nil {
		t.Fatalf("RegistrarFaseProyecto diseno: %v", err)
	}
	faseImplID, err := RegistrarFaseProyecto(&FaseProyecto{
		Proyecto: "orquestador",
		Nombre:   "Implementacion",
		Orden:    20,
		Peso:     2,
		Estado:   "pendiente",
	})
	if err != nil {
		t.Fatalf("RegistrarFaseProyecto implementacion: %v", err)
	}

	tarea1ID, err := CrearTarea(&Tarea{
		Titulo:    "Modelar progreso",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea 1: %v", err)
	}
	tarea2ID, err := CrearTarea(&Tarea{
		Titulo:    "Exponer progreso",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea 2: %v", err)
	}
	if err := TomarTarea(tarea2ID, "codex1"); err != nil {
		t.Fatalf("TomarTarea 2: %v", err)
	}
	if err := IniciarTarea(tarea2ID, "codex1"); err != nil {
		t.Fatalf("IniciarTarea 2: %v", err)
	}

	if err := RegistrarAvanceTarea(&AvanceTarea{
		TareaID:        tarea1ID,
		Proyecto:       "orquestador",
		FaseID:         &faseDisenoID,
		ProgresoPct:    100,
		ActualizadoPor: "Codex3",
	}); err != nil {
		t.Fatalf("RegistrarAvanceTarea 1: %v", err)
	}
	if err := RegistrarAvanceTarea(&AvanceTarea{
		TareaID:        tarea2ID,
		Proyecto:       "orquestador",
		FaseID:         &faseImplID,
		ProgresoPct:    40,
		ActualizadoPor: "Codex3",
	}); err != nil {
		t.Fatalf("RegistrarAvanceTarea 2: %v", err)
	}

	resumen, err := CalcularResumenProgresoProyecto("orquestador")
	if err != nil {
		t.Fatalf("CalcularResumenProgresoProyecto: %v", err)
	}
	if resumen.TareasTotales != 2 || resumen.TareasCompletadas != 1 {
		t.Fatalf("conteos inesperados: %+v", resumen)
	}
	if resumen.ProgresoPct != 60 {
		t.Fatalf("progreso esperado 60.0, obtenido %.1f", resumen.ProgresoPct)
	}
}

func TestCalcularResumenProgresoProyectoConFallbackDeEstado(t *testing.T) {
	abrirDBTemporalMemoria(t)

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Tarea en progreso sin avance manual",
		Modulo:    "orquestador",
		Prioridad: PrioridadMedia,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	if err := RegistrarAvanceTarea(&AvanceTarea{
		TareaID:        tareaID,
		Proyecto:       "orquestador",
		ActualizadoPor: "Codex3",
		ProgresoPct:    50,
	}); err != nil {
		t.Fatalf("RegistrarAvanceTarea: %v", err)
	}

	resumen, err := CalcularResumenProgresoProyecto("orquestador")
	if err != nil {
		t.Fatalf("CalcularResumenProgresoProyecto: %v", err)
	}
	if resumen.ProgresoPct != 50 {
		t.Fatalf("progreso esperado 50.0, obtenido %.1f", resumen.ProgresoPct)
	}
	if len(resumen.TareasSinFase) != 1 {
		t.Fatalf("se esperaba una tarea sin fase: %+v", resumen.TareasSinFase)
	}
}
