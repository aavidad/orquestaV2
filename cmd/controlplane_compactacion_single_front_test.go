package cmd

import (
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestCompactarFrentesExclusividadPremiumAgenteProyectoConservaFrenteUnico(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente unico recuperado",
		Descripcion: "Trabajo recuperado tras cuota",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	n, err := compactarFrentesExclusividadPremiumAgenteProyecto("Gemini1", proyectoID)
	if err != nil {
		t.Fatalf("compactar frentes: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia compactar el unico frente activo, got=%d", n)
	}

	actual, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if actual == nil || actual.Estado != db.TareaEnProgreso {
		t.Fatalf("el frente unico debe seguir en progreso: %+v", actual)
	}
}
