/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"strings"
	"testing"

	"orquesta/db"
)

func TestPropuestaActualizarCLIAnexaDescripcionYSoportaAliasAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.CrearPropuesta(&db.Propuesta{
		Codigo:       "OP-920",
		Titulo:       "Base",
		Descripcion:  "Inicio",
		Tipo:         "implementacion",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	if err := propuestaActualizarCmd.Flags().Set("agente", "Codex4"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := propuestaActualizarCmd.Flags().Set("anexar-descripcion", " + detalle operativo"); err != nil {
		t.Fatalf("set anexar-descripcion: %v", err)
	}
	t.Cleanup(func() {
		_ = propuestaActualizarCmd.Flags().Set("agente", "")
		_ = propuestaActualizarCmd.Flags().Set("anexar-descripcion", "")
	})

	out := capturarStdout(t, func() {
		if err := propuestaActualizarCmd.RunE(propuestaActualizarCmd, []string{"OP-920"}); err != nil {
			t.Fatalf("propuesta actualizar: %v", err)
		}
	})
	if !strings.Contains(out, "Propuesta OP-920 actualizada") {
		t.Fatalf("salida inesperada:\n%s", out)
	}

	propuesta, err := db.GetPropuesta("OP-920")
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if propuesta.Descripcion != "Inicio + detalle operativo" {
		t.Fatalf("descripcion inesperada: %q", propuesta.Descripcion)
	}
}

func TestPropuestaVotosCLIFiltraPorAgenteSinImportarMayusculas(t *testing.T) {
	prepararDBTemporalCmd(t)

	propuestaID, err := db.CrearPropuesta(&db.Propuesta{
		Codigo:       "OP-921",
		Titulo:       "Filtro de votos",
		Descripcion:  "Comprobar filtro por agente",
		Tipo:         "implementacion",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	})
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	if _, err := db.Votar(propuestaID, "Codex2", db.VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("votar codex2: %v", err)
	}
	if _, err := db.Votar(propuestaID, "Claude1", db.VotoDesacuerdo, "no"); err != nil {
		t.Fatalf("votar claude1: %v", err)
	}

	if err := propuestaVotosCmd.Flags().Set("agente", "codex2"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	t.Cleanup(func() {
		_ = propuestaVotosCmd.Flags().Set("agente", "")
	})

	out := capturarStdout(t, func() {
		if err := propuestaVotosCmd.RunE(propuestaVotosCmd, []string{"OP-921"}); err != nil {
			t.Fatalf("propuesta votos: %v", err)
		}
	})

	if !strings.Contains(out, "Codex2") {
		t.Fatalf("faltó el voto filtrado:\n%s", out)
	}
	if strings.Contains(out, "Claude1") {
		t.Fatalf("salieron votos de otros agentes:\n%s", out)
	}
}
