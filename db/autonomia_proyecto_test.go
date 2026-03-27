package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProyectoAutonomiaUpsertYListarActivos(t *testing.T) {
	tmp := prepararDBTemporal(t)
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

	item, err := GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get default autonomia: %v", err)
	}
	if item == nil || item.Enabled {
		t.Fatalf("default autonomia inesperada: %+v", item)
	}

	now := time.Now().UTC().Truncate(time.Second)
	item.Enabled = true
	item.ObjetivoGeneral = "Terminar la app sin intervención humana normal"
	item.DefinitionOfDoneJSON = `{"tests":"green","api":"server-first"}`
	item.MaxWorkers = 4
	item.ReserveReviewer = true
	item.ReserveSupervisor = true
	item.LastSupervisionAt = &now
	if _, err := UpsertProyectoAutonomia(item); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}

	got, err := GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get autonomia: %v", err)
	}
	if got == nil || !got.Enabled || got.MaxWorkers != 4 || got.ObjetivoGeneral == "" {
		t.Fatalf("autonomia persistida inesperada: %+v", got)
	}

	enabled := true
	lista, err := ListarProyectosAutonomia(&enabled)
	if err != nil {
		t.Fatalf("listar autonomias activas: %v", err)
	}
	if len(lista) != 1 || lista[0].ProyectoID != proyectoID {
		t.Fatalf("autonomias activas inesperadas: %+v", lista)
	}
}

func TestAutonomiaCyclesRegistrarYFiltrar(t *testing.T) {
	tmp := prepararDBTemporal(t)
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

	if _, err := RegistrarAutonomiaCiclo(&AutonomiaCiclo{
		ProyectoID:   proyectoID,
		Kind:         "supervision",
		Agente:       "Codex3",
		InputJSON:    `{"snapshot":"ok"}`,
		DecisionJSON: `{"accion":"crear_tarea"}`,
		Resultado:    "ok",
	}); err != nil {
		t.Fatalf("registrar ciclo: %v", err)
	}

	kind := "supervision"
	ciclos, err := ListarAutonomiaCiclos(FiltroAutonomiaCiclos{
		ProyectoID: &proyectoID,
		Kind:       &kind,
	})
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(ciclos) != 1 || ciclos[0].Agente != "Codex3" {
		t.Fatalf("ciclos inesperados: %+v", ciclos)
	}
}
