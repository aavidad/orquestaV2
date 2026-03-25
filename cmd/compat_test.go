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
		t.Fatalf("debería exigir tls-cert, tls-key y tls-client-ca para exposición remota")
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

func TestValidateServeSecurityExigeCertKeySiHayClientCA(t *testing.T) {
	t.Parallel()

	err := validateServeSecurity("127.0.0.1", "", "", "/tmp/ca.crt")
	if err == nil {
		t.Fatalf("debería exigir tls-cert y tls-key si se indica tls-client-ca")
	}
}

func TestValidateServeSecurityAceptaRemotoConMTLSCompleto(t *testing.T) {
	t.Parallel()

	if err := validateServeSecurity("0.0.0.0", "/tmp/server.crt", "/tmp/server.key", "/tmp/ca.crt"); err != nil {
		t.Fatalf("remoto con mTLS completo no debería fallar: %v", err)
	}
}

func TestServeCommandExponeFlagsDeSeguridad(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"host", "tls-cert", "tls-key", "tls-client-ca"} {
		if serveCmd.Flags().Lookup(name) == nil {
			t.Fatalf("serve debería exponer flag %s", name)
		}
	}
}

func TestPublicServerURLUsaHTTPSConTLS(t *testing.T) {
	t.Parallel()

	got := publicServerURL("0.0.0.0:8443", serverSecurityOptions{
		TLSCert: "/tmp/server.crt",
		TLSKey:  "/tmp/server.key",
	})
	if got != "https://127.0.0.1:8443" {
		t.Fatalf("publicServerURL inesperada: %s", got)
	}
}

func TestShouldExposeLocalRPC(t *testing.T) {
	t.Parallel()

	if !shouldExposeLocalRPC("server", "127.0.0.1:16543", serverSecurityOptions{}) {
		t.Fatalf("server local debería exponer rpclocal")
	}
	if !shouldExposeLocalRPC("serve", "127.0.0.1:16543", serverSecurityOptions{}) {
		t.Fatalf("serve local sin TLS debería exponer rpclocal")
	}
	if shouldExposeLocalRPC("serve", "0.0.0.0:8443", serverSecurityOptions{
		TLSCert: "/tmp/server.crt",
		TLSKey:  "/tmp/server.key",
	}) {
		t.Fatalf("serve remoto con TLS no debería publicar rpclocal")
	}
	if shouldExposeLocalRPC("serve", "0.0.0.0:16543", serverSecurityOptions{}) {
		t.Fatalf("serve remoto sin TLS no debería publicar rpclocal")
	}
}

func TestServerRunRechazaTLS(t *testing.T) {
	t.Parallel()

	cmd := *serverRunCmd
	flags := *serverRunCmd.Flags()
	cmd.Flags().AddFlagSet(&flags)
	if err := cmd.Flags().Set("addr", "127.0.0.1:16543"); err != nil {
		t.Fatalf("set addr: %v", err)
	}
	if err := cmd.Flags().Set("tls-cert", "/tmp/server.crt"); err != nil {
		t.Fatalf("set tls-cert: %v", err)
	}
	if err := cmd.Flags().Set("tls-key", "/tmp/server.key"); err != nil {
		t.Fatalf("set tls-key: %v", err)
	}

	err := cmd.RunE(&cmd, nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "no soporta tls") {
		t.Fatalf("error inesperado: %v", err)
	}
}
