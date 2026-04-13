package microprogramacionapp

import "testing"

func TestExtraerArchivosEntregaBasico(t *testing.T) {
	respuesta := "// FILE: pkg/hola/hola.go\npackage hola\n\nfunc Hola() string { return \"hola\" }\n"
	items := ExtraerArchivosEntrega(respuesta)
	if len(items) != 1 {
		t.Fatalf("se esperaba 1 fichero, got=%d", len(items))
	}
	if items[0].RutaRelativa != "pkg/hola/hola.go" {
		t.Fatalf("ruta inesperada: %+v", items[0])
	}
}

func TestExtraerArchivosEntregaRechazaRutaPeligrosa(t *testing.T) {
	respuesta := "// FILE: ../cmd/api.go\npackage cmd\n"
	items := ExtraerArchivosEntrega(respuesta)
	if len(items) != 0 {
		t.Fatalf("no deberia extraer ruta peligrosa: %+v", items)
	}
}

func TestExtraerArchivosEntregaRecortaSeccionEvidencia(t *testing.T) {
	respuesta := "// FILE: pkg/hola/hola.go\npackage hola\n\nfunc Hola() string { return \"hola\" }\n\nEVIDENCIA:\nLinea 1\n"
	items := ExtraerArchivosEntrega(respuesta)
	if len(items) != 1 {
		t.Fatalf("se esperaba 1 fichero, got=%d", len(items))
	}
	if items[0].Contenido != "package hola\n\nfunc Hola() string { return \"hola\" }" {
		t.Fatalf("contenido inesperado: %q", items[0].Contenido)
	}
}

func TestExtraerArchivosEntregaRecortaFenceMarkdown(t *testing.T) {
	respuesta := "// FILE: pkg/hola/hola.go\n```go\npackage hola\n\nfunc Hola() string { return \"hola\" }\n```\n"
	items := ExtraerArchivosEntrega(respuesta)
	if len(items) != 1 {
		t.Fatalf("se esperaba 1 fichero, got=%d", len(items))
	}
	if items[0].Contenido != "package hola\n\nfunc Hola() string { return \"hola\" }" {
		t.Fatalf("contenido inesperado: %q", items[0].Contenido)
	}
}
