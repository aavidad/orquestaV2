package db

import (
	"path/filepath"
	"testing"
)

func prepararDBTemporalDiagnostico(t *testing.T) string {
	t.Helper()
	return prepararDBTemporalConNombre(t, "orquesta-diagnostico-test.db")
}

func TestConstruirSnapshotDiagnostico(t *testing.T) {
	tmp := prepararDBTemporalDiagnostico(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-001",
		ResumenContinuidad: "continuar diagnostico",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Tarea diagnostico",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "Codex1",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	propuesta := &Propuesta{
		Titulo:       "Diagnostico",
		Descripcion:  "Propuesta abierta",
		Tipo:         "implementacion",
		PropuestoPor: "Codex1",
		Distribuidor: "Codex1",
		ProyectoID:   &proyectoID,
	}
	if _, err := CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	snapshot, err := ConstruirSnapshotDiagnostico(10)
	if err != nil {
		t.Fatalf("ConstruirSnapshotDiagnostico: %v", err)
	}
	if snapshot == nil {
		t.Fatalf("snapshot nil")
	}
	if len(snapshot.Agentes) == 0 {
		t.Fatalf("esperaba agentes")
	}
	if len(snapshot.SesionesActivas) == 0 {
		t.Fatalf("esperaba sesiones activas")
	}
	if got := snapshot.ConteoTareas[string(TareaEnProgreso)]; got == 0 {
		t.Fatalf("esperaba tareas en progreso, conteo=%v", snapshot.ConteoTareas)
	}
	if len(snapshot.PropuestasAbiertas) == 0 {
		t.Fatalf("esperaba propuestas abiertas")
	}
	if len(snapshot.Config) == 0 {
		t.Fatalf("esperaba configuración")
	}
	if len(snapshot.AuditoriaReciente) == 0 {
		t.Fatalf("esperaba auditoría reciente")
	}
}
