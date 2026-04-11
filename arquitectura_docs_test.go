package main

import (
	"os"
	"strings"
	"testing"
)

func TestDocumentacionCanonicaNoRecuperaRutasLegacy(t *testing.T) {
	t.Parallel()

	files := []string{
		"ARQUITECTURA.md",
		"docs/orquesta_v1_vision.md",
		"docs/BIBLIA_APP_ORQUESTA.md",
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("leyendo %s: %v", path, err)
		}
		text := string(data)
		lower := strings.ToLower(text)
		if strings.Contains(lower, "el objetivo fisico final es `~/trabajo/orquestador`") {
			t.Fatalf("%s vuelve a presentar ~/Trabajo/orquestador como objetivo vivo", path)
		}
		if strings.Contains(lower, "sigue ubicado en `~/trabajo/plataformamunicipal/orquestador`") {
			t.Fatalf("%s vuelve a presentar la ruta legacy de PlataformaMunicipal como estado actual", path)
		}
	}
}

func TestArbolNoRecuperaDirectoriosMuertos(t *testing.T) {
	t.Parallel()

	paths := []string{
		"controlruntimes",
		"configapp",
		"internal/localrpc",
		"internal/runtimeobs",
	}
	for _, path := range paths {
		_, err := os.Stat(path)
		if err == nil {
			t.Fatalf("%s no debería existir; es deuda estructural retirada", path)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
