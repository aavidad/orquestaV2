// Este contrato impide que una aplicación auxiliar del inventario nazca sin
// manifiesto de responsabilidad ni cabeceras cercanas al código.
package orquesta_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestLegacyInventoryApplicationsDeclareTheirContract(t *testing.T) {
	applications, err := filepath.Glob("scripts/legacy_*")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(applications)

	checked := 0
	for _, application := range applications {
		if !regularFileExists(filepath.Join(application, "main.go")) {
			continue
		}
		checked++
		t.Run(filepath.Base(application), func(t *testing.T) {
			requireApplicationManifest(t, application)
			requireGoFileHeaders(t, application)
		})
	}
	if checked == 0 {
		t.Fatal("no se encontró ninguna aplicación auxiliar para comprobar")
	}
}

func requireApplicationManifest(t *testing.T, application string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(application, "README.md"))
	if err != nil {
		t.Fatalf("manifiesto ausente: %v", err)
	}
	text := string(content)
	required := []string{
		"## Propósito y usuarios",
		"## Alcance y exclusiones",
		"## Entradas y salidas",
		"## Arquitectura y módulos",
		"## Autoridad, datos, permisos, secretos y efectos",
		"## Arranque, diagnóstico, recuperación y parada",
		"## Contratos y pruebas",
	}
	for _, heading := range required {
		if strings.Count(text, heading) != 1 {
			t.Errorf("el manifiesto %s debe contener una vez %q", application, heading)
		}
	}
}

func requireGoFileHeaders(t *testing.T, application string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(application, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("%s no contiene ficheros Go", application)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		first := strings.TrimSpace(strings.SplitN(string(content), "\n", 2)[0])
		if !strings.HasPrefix(first, "//") {
			t.Errorf("%s no empieza con una cabecera de responsabilidad", file)
		}
	}
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
