// Este fichero impide perder el manifiesto universal o las cabeceras explicativas.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADMEDeclaresUniversalApplicationManifest(t *testing.T) {
	content, err := os.ReadFile("README.md")
	mustSucceed(t, err)
	text := string(content)
	required := []string{
		"## Manifiesto de la aplicación", "### Propósito y usuarios",
		"### Alcance y exclusiones", "### Entradas y salidas",
		"### Arquitectura y módulos", "### Autoridad, datos, permisos, secretos y efectos",
		"### Arranque, diagnóstico, recuperación y parada",
		"### Contratos y pruebas",
		"**Autoridad:**",
		"Su promesa se limita a confirmar la publicación atómica",
		"No valida la verdad semántica",
		"La evidencia debe vivir en un dispositivo distinto de todas las raíces.",
		"La herramienta no borra automáticamente etapas, temporales, diarios ni finales.",
		"Un corte o cualquier fallo previo a la confirmación puede dejar",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Errorf("falta la sección de manifiesto %q", fragment)
		}
	}
	for _, forbidden := range []string{"etapas solo", "etapas solamente", "etapas únicamente"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("formulación de recuperación incompatible %q", forbidden)
		}
	}
}
func TestGoFilesExplainTheirResponsibilityInHeader(t *testing.T) {
	files, err := filepath.Glob("*.go")
	mustSucceed(t, err)
	for _, path := range files {
		content, err := os.ReadFile(path)
		mustSucceed(t, err)
		first, _, _ := strings.Cut(string(content), "\n")
		if !strings.HasPrefix(first, "// Este fichero ") &&
			!strings.HasPrefix(first, "// El paquete ") {
			t.Errorf("%s carece de cabecera explicativa: %q", path, first)
		}
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, forbidden := range []string{
			"os.Remove(", "os.RemoveAll(", "unix.Unlink(", "unix.Unlinkat(",
			"unix.Rmdir(", "syscall.Unlink(",
		} {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s contiene API de borrado prohibida %q", path, forbidden)
			}
		}
	}
}
