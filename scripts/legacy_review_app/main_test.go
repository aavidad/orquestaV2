// Este fichero prueba la composición segura antes de abrir el puerto local.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationAcceptsOnlyNumericLoopback(t *testing.T) {
	for _, accepted := range []string{"127.0.0.1:0", "127.0.0.1:8787", "[::1]:8787"} {
		if err := validateLoopbackAddress(accepted); err != nil {
			t.Errorf("se rechazó loopback %s: %v", accepted, err)
		}
	}
	for _, rejected := range []string{"0.0.0.0:8787", "localhost:8787", "192.168.1.2:8787", "127.0.0.1:70000"} {
		if err := validateLoopbackAddress(rejected); err == nil {
			t.Errorf("se aceptó una escucha no permitida: %s", rejected)
		}
	}
}

func TestAuthorizationRequiresPrivateRegularFile(t *testing.T) {
	directory := t.TempDir()
	private := filepath.Join(directory, "autorizacion")
	if err := os.WriteFile(private, []byte(testAuthorization+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, err := readAuthorization(private)
	if err != nil || string(content) != testAuthorization {
		t.Fatalf("autorización privada rechazada: %q %v", content, err)
	}
	if err := os.Chmod(private, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readAuthorization(private); err == nil {
		t.Fatal("se aceptó una autorización legible por terceros")
	}
	link := filepath.Join(directory, "enlace")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readAuthorization(link); err == nil {
		t.Fatal("se aceptó una autorización mediante enlace simbólico")
	}
}

func TestConfigurationRequiresExplicitScopeAndShowsCataloguedHelp(t *testing.T) {
	catalog := newCatalog("es")
	var help bytes.Buffer
	_, shown, err := parseConfiguration([]string{"--help"}, catalog, &help)
	if err != nil || !shown || !bytes.Contains(help.Bytes(), []byte(catalog.text("cli_usage"))) {
		t.Fatalf("ayuda no catalogada: shown=%v err=%v texto=%q", shown, err, help.String())
	}
	arguments := []string{
		"--inventory", "/tmp/inventario.jsonl",
		"--proposals", "/tmp/propuestas.jsonl",
		"--authorization-file", "/tmp/autorizacion",
		"--actor-ref", "actor:uno",
		"--project-ref", "project:uno",
		"--listen", "127.0.0.1:0",
	}
	config, shown, err := parseConfiguration(arguments, catalog, &help)
	if err != nil || shown || config.actorRef != "actor:uno" {
		t.Fatalf("configuración válida rechazada: %#v shown=%v err=%v", config, shown, err)
	}
	arguments[3] = "/tmp/inventario.jsonl"
	if _, _, err := parseConfiguration(arguments, catalog, &help); err == nil {
		t.Fatal("se aceptó el mismo fichero para inventario y propuestas")
	}
}
