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
