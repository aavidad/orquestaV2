/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolverValorFlagPriorizaPrimerValorNoVacio(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("agente", "", "")
	cmd.Flags().String("por", "alberto", "")
	if err := cmd.Flags().Set("agente", "codex2"); err != nil {
		t.Fatalf("set agente: %v", err)
	}

	valor, err := resolverValorFlag(cmd, "agente", "por")
	if err != nil {
		t.Fatalf("resolverValorFlag: %v", err)
	}
	if valor != "codex2" {
		t.Fatalf("valor inesperado: %s", valor)
	}
}

func TestResolverTextoFlagOPosicionalAceptaFlag(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("motivo", "", "")
	if err := cmd.Flags().Set("motivo", "esperando validación externa"); err != nil {
		t.Fatalf("set motivo: %v", err)
	}

	valor, err := resolverTextoFlagOPosicional(cmd, []string{"15", "codex2"}, 2, "motivo")
	if err != nil {
		t.Fatalf("resolverTextoFlagOPosicional: %v", err)
	}
	if valor != "esperando validación externa" {
		t.Fatalf("valor inesperado: %s", valor)
	}
}

func TestResolverTextoFlagOPosicionalAceptaTextoPosicional(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("resolucion", "", "")

	valor, err := resolverTextoFlagOPosicional(cmd, []string{"15", "codex2", "se", "corrigió", "la", "migración"}, 2, "resolucion")
	if err != nil {
		t.Fatalf("resolverTextoFlagOPosicional: %v", err)
	}
	if valor != "se corrigió la migración" {
		t.Fatalf("valor inesperado: %s", valor)
	}
}
