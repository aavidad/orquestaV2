package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReviewGateCRUDYResolucion(t *testing.T) {
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
	agente := "Codex3"
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Cerrar runner autónomo",
		Descripcion: "Completar el lazo supervisor/reviewer",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		Agente:      &agente,
		CreadoPor:   "server",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	id, err := CrearReviewGate(&ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "Codex4",
		Estado:         ReviewGatePendiente,
		SeverityMax:    "media",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	gate, err := GetReviewGate(id)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.ReviewerAgente != "Codex4" || gate.Estado != ReviewGatePendiente {
		t.Fatalf("review gate inesperado: %+v", gate)
	}

	estado := "pendiente"
	lista, err := ListarReviewGates(FiltroReviewGates{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar review gates: %v", err)
	}
	if len(lista) != 1 || lista[0].ID != id {
		t.Fatalf("lista review gates inesperada: %+v", lista)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := ResolverReviewGate(id, ReviewGateAprobado, `[{"severity":"baja","title":"ok"}]`, &now); err != nil {
		t.Fatalf("resolver review gate: %v", err)
	}
	gate, err = GetReviewGate(id)
	if err != nil {
		t.Fatalf("get review gate final: %v", err)
	}
	if gate == nil || gate.Estado != ReviewGateAprobado || gate.ResolvedAt == nil {
		t.Fatalf("review gate no resuelto correctamente: %+v", gate)
	}
}
