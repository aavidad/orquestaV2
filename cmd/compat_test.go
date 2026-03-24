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

func TestCatalogoBriefingCommandsExposeAgenteAlias(t *testing.T) {
	t.Parallel()

	cmds := []*cobra.Command{
		reglasCrearCmd,
		reglasEditarCmd,
		reglasActivarCmd,
		reglasDesactivarCmd,
		skillsCrearCmd,
		skillsEditarCmd,
		skillsActivarCmd,
		skillsDesactivarCmd,
		workflowsCrearCmd,
		workflowsEditarCmd,
		workflowsActivarCmd,
		workflowsDesactivarCmd,
	}

	for _, cmd := range cmds {
		if cmd.Flags().Lookup("agente") == nil {
			t.Fatalf("el comando %s no expone alias --agente", cmd.Use)
		}
	}
}

func TestValidateServeSecurityRechazaRemotoSinMTLS(t *testing.T) {
	t.Parallel()

	err := validateServeSecurity("0.0.0.0", "", "", "")
	if err == nil {
		t.Fatalf("debería exigir tls-client-ca para exposición remota")
	}
}

func TestValidateServeSecurityAceptaLocalSinTLS(t *testing.T) {
	t.Parallel()

	if err := validateServeSecurity("127.0.0.1", "", "", ""); err != nil {
		t.Fatalf("local sin tls no debería fallar: %v", err)
	}
}

func TestValidateServeSecurityExigeParCertKey(t *testing.T) {
	t.Parallel()

	err := validateServeSecurity("127.0.0.1", "/tmp/server.crt", "", "")
	if err == nil {
		t.Fatalf("debería exigir tls-key junto a tls-cert")
	}
}
