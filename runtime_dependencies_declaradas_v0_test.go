package orquesta_test

import (
	"os"
	"strings"
	"testing"
)

// La capacidad de extraccion de PDF invoca `pdftotext` (poppler). Si la imagen
// donde corre el servidor no lo trae, la tool esta viva en mi maquina y MUERTA en
// produccion: exactamente el fallo que Codex me caza aqui. Un test que solo
// corriera en el host no lo detectaria nunca.
//
// Regla: toda dependencia de binario externo se declara en TODAS las imagenes.
func TestImagenesDeclaranPopplerParaExtraccionDePDFV0(t *testing.T) {
	for _, dockerfile := range []string{"Dockerfile", "Dockerfile.dev", "Dockerfile.self-programming"} {
		contenido, err := os.ReadFile(dockerfile)
		if err != nil {
			t.Fatalf("leyendo %s: %v", dockerfile, err)
		}
		if !strings.Contains(string(contenido), "poppler-utils") {
			t.Fatalf(
				"%s no instala poppler-utils: orquesta.document.text.extract.v0 quedaria muerta dentro del contenedor",
				dockerfile,
			)
		}
	}
}
