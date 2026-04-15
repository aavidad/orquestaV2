package cmd

import (
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestRuntimeOrderAutonomiaRecienteBloqueaRecuperacionIgnoraStartRecienteSiElWorkerYaActualizo(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      "claude1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start"}`,
	}); err != nil {
		t.Fatalf("enqueue start: %v", err)
	}
	updated := time.Now().UTC().Add(2 * time.Second)
	row := agentesapp.Row{
		Agente:          &db.Agente{Nombre: "claude1"},
		EstadoOperativo: "atascado",
		WorkerUpdatedAt: &updated,
	}

	bloquea, err := runtimeOrderAutonomiaRecienteBloqueaRecuperacion(row, &proyectoID, "start", "start", 30*time.Minute)
	if err != nil {
		t.Fatalf("runtimeOrderAutonomiaRecienteBloqueaRecuperacion: %v", err)
	}
	if bloquea {
		t.Fatalf("no deberia bloquear si el worker ya actualizo tras la orden reciente")
	}
}

func TestRuntimeOrderAutonomiaRecienteBloqueaRecuperacionMantieneCooldownSinUpdatePosterior(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      "claude1",
		ProyectoID:  &proyectoID,
		Tipo:        "stop",
		PayloadJSON: `{"accion":"stop"}`,
	}); err != nil {
		t.Fatalf("enqueue stop: %v", err)
	}
	row := agentesapp.Row{
		Agente:          &db.Agente{Nombre: "claude1"},
		EstadoOperativo: "atascado",
	}

	bloquea, err := runtimeOrderAutonomiaRecienteBloqueaRecuperacion(row, &proyectoID, "stop", "stop", 30*time.Minute)
	if err != nil {
		t.Fatalf("runtimeOrderAutonomiaRecienteBloqueaRecuperacion: %v", err)
	}
	if !bloquea {
		t.Fatalf("deberia mantener cooldown si no hay update posterior del worker")
	}
}
