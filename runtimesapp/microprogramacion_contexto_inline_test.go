package runtimesapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstruirMensajeMicroprogramacionInlineOmiteTestsLateralesSiWriteSetYaIncluyeTest(t *testing.T) {
	dir := t.TempDir()
	escribirArchivoPruebaInline(t, dir, "capacidadapp/pool_local_compartido.go", "package capacidadapp\n\nfunc Demo() {}\n")
	escribirArchivoPruebaInline(t, dir, "capacidadapp/service_test.go", "package capacidadapp\n\nfunc TestDemo(t *testing.T) {}\n")
	escribirArchivoPruebaInline(t, dir, "capacidadapp/smoke_hexagonal_test.go", "package capacidadapp\n\nfunc TestHex(t *testing.T) {}\n")

	mensaje, err := construirMensajeMicroprogramacionInline(dir, MicroprogramacionDispatchRequest{
		ArchivoObjetivo: "capacidadapp/pool_local_compartido.go",
		WriteSet:        []string{"capacidadapp/pool_local_compartido.go", "capacidadapp/service_test.go"},
		FormatoSalida:   "patch+evidencia",
	}, "MICROTAREA CERRADA")
	if err != nil {
		t.Fatalf("construirMensajeMicroprogramacionInline: %v", err)
	}
	if !strings.Contains(mensaje, "=== WRITE_SET: capacidadapp/service_test.go ===") {
		t.Fatalf("faltaba el test explicito en el contexto:\n%s", mensaje)
	}
	if strings.Contains(mensaje, "smoke_hexagonal_test.go") {
		t.Fatalf("no deberia incluir tests laterales cuando el write_set ya trae test explicito:\n%s", mensaje)
	}
}

func TestConstruirMensajeMicroprogramacionInlineRecortaArchivoObjetivoAlSimbolo(t *testing.T) {
	dir := t.TempDir()
	escribirArchivoPruebaInline(t, dir, "microprogramacionapp/extractor_entrega.go", `package microprogramacionapp

import (
	"path/filepath"
	"strings"
)

func helperInterno() string { return "helper" }

func recortarSeccionEvidencia(contenido string) string {
	lineas := strings.Split(contenido, "\n")
	for idx, linea := range lineas {
		if strings.EqualFold(strings.TrimSpace(linea), "evidencia:") {
			return strings.TrimRight(strings.Join(lineas[:idx], "\n"), "\n")
		}
	}
	return strings.TrimRight(contenido, "\n")
}
`)

	mensaje, err := construirMensajeMicroprogramacionInline(dir, MicroprogramacionDispatchRequest{
		ArchivoObjetivo: "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo: "recortarSeccionEvidencia",
		WriteSet:        []string{"microprogramacionapp/extractor_entrega.go"},
		FormatoSalida:   "ficheros+evidencia",
	}, "MICROTAREA CERRADA")
	if err != nil {
		t.Fatalf("construirMensajeMicroprogramacionInline: %v", err)
	}
	if !strings.Contains(mensaje, "func recortarSeccionEvidencia(") {
		t.Fatalf("faltaba el simbolo objetivo en el contexto:\n%s", mensaje)
	}
	if strings.Contains(mensaje, "helperInterno") {
		t.Fatalf("no deberia incluir helpers ajenos al simbolo objetivo:\n%s", mensaje)
	}
}

func TestConstruirMensajeMicroprogramacionInlineRecortaTestObjetivoPorRun(t *testing.T) {
	dir := t.TempDir()
	escribirArchivoPruebaInline(t, dir, "microprogramacionapp/extractor_entrega.go", "package microprogramacionapp\n\nfunc recortarSeccionEvidencia(contenido string) string { return contenido }\n")
	escribirArchivoPruebaInline(t, dir, "microprogramacionapp/extractor_entrega_test.go", `package microprogramacionapp

import "testing"

func TestExtraerArchivosEntregaBasico(t *testing.T) {}

func TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios(t *testing.T) {}

func TestExtraerArchivosEntregaRecortaFenceMarkdown(t *testing.T) {}
`)

	mensaje, err := construirMensajeMicroprogramacionInline(dir, MicroprogramacionDispatchRequest{
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	}, "MICROTAREA CERRADA")
	if err != nil {
		t.Fatalf("construirMensajeMicroprogramacionInline: %v", err)
	}
	if !strings.Contains(mensaje, "TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios") {
		t.Fatalf("faltaba el test objetivo en el contexto:\n%s", mensaje)
	}
	if strings.Contains(mensaje, "TestExtraerArchivosEntregaBasico") || strings.Contains(mensaje, "TestExtraerArchivosEntregaRecortaFenceMarkdown") {
		t.Fatalf("no deberia incluir tests ajenos al -run objetivo:\n%s", mensaje)
	}
}

func escribirArchivoPruebaInline(t *testing.T, raiz, ruta, contenido string) {
	t.Helper()
	destino := filepath.Join(raiz, filepath.FromSlash(ruta))
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", destino, err)
	}
	if err := os.WriteFile(destino, []byte(contenido), 0o644); err != nil {
		t.Fatalf("write %s: %v", destino, err)
	}
}
