/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"path/filepath"
	"testing"
)

func TestClaimNextBootstrapRuntimeOrderPriorizaProyectoYTipo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto B: %v", err)
	}

	orders := []*RuntimeOrder{
		{Agente: "Codex1", Tipo: "start", PayloadJSON: `{"scope":"global-start"}`},
		{Agente: "Codex1", Tipo: "handoff", PayloadJSON: `{"scope":"global-handoff"}`},
		{Agente: "Codex1", ProyectoID: &proyectoA, Tipo: "resume", PayloadJSON: `{"scope":"resume-a"}`},
		{Agente: "Codex1", ProyectoID: &proyectoA, Tipo: "handoff", PayloadJSON: `{"scope":"handoff-a"}`},
		{Agente: "Codex1", ProyectoID: &proyectoB, Tipo: "handoff", PayloadJSON: `{"scope":"handoff-b"}`},
	}
	for _, order := range orders {
		if _, err := EncolarRuntimeOrder(order); err != nil {
			t.Fatalf("encolar runtime order %+v: %v", order, err)
		}
	}

	claimed, err := ClaimNextBootstrapRuntimeOrder("Codex1", &proyectoA)
	if err != nil {
		t.Fatalf("claim bootstrap: %v", err)
	}
	if claimed == nil {
		t.Fatalf("se esperaba runtime order bootstrap")
	}
	if claimed.ProyectoID == nil || *claimed.ProyectoID != proyectoA {
		t.Fatalf("orden bootstrap fuera de proyecto: %+v", claimed)
	}
	if claimed.Tipo != "handoff" {
		t.Fatalf("tipo bootstrap inesperado: %+v", claimed)
	}
	if claimed.PayloadJSON != `{"scope":"handoff-a"}` {
		t.Fatalf("payload bootstrap inesperado: %s", claimed.PayloadJSON)
	}
}
