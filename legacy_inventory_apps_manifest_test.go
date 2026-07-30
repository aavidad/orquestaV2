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

func TestApplicationManifestSectionParsingRejectsLookalikes(t *testing.T) {
	for _, test := range []struct {
		name         string
		text         string
		title        string
		wantContent  string
		wantSections int
	}{
		{name: "sección de segundo nivel", text: "## Propósito\ncontenido\n## Fin", title: "Propósito", wantContent: "contenido", wantSections: 1},
		{name: "sección anidada", text: "## Manifiesto\n### Propósito\ncontenido\n### Fin", title: "Propósito", wantContent: "contenido", wantSections: 1},
		{name: "nivel no permitido", text: "#### Propósito\ncontenido", title: "Propósito"},
		{name: "título parecido", text: "## Propósito adicional\ncontenido", title: "Propósito"},
		{name: "duplicada", text: "## Propósito\nuno\n## Propósito\ndos", title: "Propósito", wantContent: "dos", wantSections: 2},
		{name: "vacía", text: "## Propósito\n\n## Fin", title: "Propósito", wantSections: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			content, sections := findManifestSection(test.text, test.title)
			if strings.TrimSpace(content) != test.wantContent || sections != test.wantSections {
				t.Fatalf("contenido/secciones = %q/%d; esperado %q/%d", content, sections, test.wantContent, test.wantSections)
			}
		})
	}
}

func requireApplicationManifest(t *testing.T, application string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(application, "README.md"))
	if err != nil {
		t.Fatalf("manifiesto ausente: %v", err)
	}
	required := []string{
		"Propósito y usuarios",
		"Alcance y exclusiones",
		"Entradas y salidas",
		"Arquitectura y módulos",
		"Autoridad, datos, permisos, secretos y efectos",
		"Arranque, diagnóstico, recuperación y parada",
		"Contratos y pruebas",
	}
	for _, heading := range required {
		section, count := findManifestSection(string(content), heading)
		if count != 1 {
			t.Errorf("el manifiesto %s debe contener una sección exacta %q", application, heading)
		} else if strings.TrimSpace(section) == "" {
			t.Errorf("la sección %q de %s está vacía", heading, application)
		}
	}
}

func findManifestSection(text, title string) (string, int) {
	lines := strings.Split(text, "\n")
	found, content := 0, ""
	for index, line := range lines {
		level, heading := markdownHeading(line)
		if (level != 2 && level != 3) || heading != title {
			continue
		}
		found++
		end := len(lines)
		for candidate := index + 1; candidate < len(lines); candidate++ {
			nextLevel, _ := markdownHeading(lines[candidate])
			if nextLevel > 0 && nextLevel <= level {
				end = candidate
				break
			}
		}
		content = strings.Join(lines[index+1:end], "\n")
	}
	return content, found
}

func markdownHeading(line string) (int, string) {
	line = strings.TrimSpace(line)
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level < 1 || level > 6 || level >= len(line) || line[level] != ' ' {
		return 0, ""
	}
	return level, strings.TrimSpace(line[level+1:])
}

func requireGoFileHeaders(t *testing.T, application string) {
	t.Helper()
	var files []string
	err := filepath.WalkDir(application, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && filepath.Ext(path) == ".go" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("%s no contiene ficheros Go", application)
	}
	sort.Strings(files)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		first := strings.TrimSpace(strings.SplitN(string(content), "\n", 2)[0])
		description := strings.TrimSpace(strings.TrimPrefix(first, "//"))
		if !strings.HasPrefix(first, "//") || len([]rune(description)) < 20 {
			t.Errorf("%s no empieza con una cabecera de responsabilidad", file)
		}
	}
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
