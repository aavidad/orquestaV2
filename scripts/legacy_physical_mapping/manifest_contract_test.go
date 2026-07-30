// Estas pruebas impiden perder el manifiesto o añadir capacidades laterales.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADMEDeclaresAuxiliaryApplicationContract(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, required := range []string{
		"## Manifiesto de la aplicación", "### Propósito y usuarios",
		"### Alcance y exclusiones", "### Entradas y salidas",
		"### Arquitectura y módulos", "### Autoridad, datos, permisos, secretos y efectos",
		"### Arranque, diagnóstico, recuperación y parada", "### Contratos y pruebas",
		"Solo demuestra «candidato bien formado, ligado por bytes y coherente internamente»",
		"No emite un recibo productivo", "no acredita `mapeo_1_1_y_1_n_acreditado`",
		"queda prohibido encadenar estas órdenes mediante subprocesos",
		"Esta utilidad no define puerto, adaptador ni",
		"presupuestos ni límites de ejecución", "`semantic_integrity_sha256` vacío",
		"no un registro durable de idempotencia", "El JSON portable es UTF-8 sin BOM",
		"APP-13 productivo sigue pendiente",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("falta declaración %q", required)
		}
	}
	if strings.Contains(text, "detrás de un puerto") {
		t.Error("el manifiesto promete un puerto no autorizado")
	}
}

func TestGoFilesHaveSpanishResponsibilityHeaderAndNoLateralEffects(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		first, _, _ := strings.Cut(string(raw), "\n")
		if !strings.HasPrefix(first, "// Este fichero ") &&
			!strings.HasPrefix(first, "// Estas pruebas ") {
			t.Errorf("%s carece de cabecera castellana", path)
		}
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		for _, forbidden := range []string{
			"os.Getenv", "os.LookupEnv", "os.Environ", `"os/exec"`, `"net`,
			"exec.Command", "os.Create", "os.WriteFile", "os.Mkdir", "os.Remove",
			"unix.Unlink", "receipt_ref", "closed",
		} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("%s contiene capacidad o afirmación prohibida %q", path, forbidden)
			}
		}
	}
}
