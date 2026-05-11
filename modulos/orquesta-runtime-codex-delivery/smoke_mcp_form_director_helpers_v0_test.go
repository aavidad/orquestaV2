package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaweb "orquesta/modulos/orquesta-web"
)

func mcpFormDirectorSmokeFormV0() orquestaweb.WebNuevaAppFormV0 {
	enabled := true
	return orquestaweb.WebNuevaAppFormV0{
		RequestID:        "req-form-director-real-001",
		Locale:           "es-ES",
		Nombre:           "Agenda",
		Objetivo:         "Gestionar contactos y citas con API REST en Go y una web de administracion.",
		Descripcion:      "Prueba real de Orquesta desde formulario: el director debe preparar arquitectura y microtareas.",
		TipoApp:          "mixed",
		UsuariosObjetivo: []string{"equipos pequenos", "usuarios internos"},
		Plataformas:      []string{"server", "web"},
		PreferenciasTecnicas: orquestaweb.WebNuevaAppPreferenciasFormV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
			Restricciones: []string{
				"persistencia solo mediante puerto y conector",
				"funciones pequenas",
				"sin archivos gigantes",
			},
		},
		Datos: orquestaweb.WebNuevaAppDatosFormV0{
			DBRequired:         true,
			NecesidadFuncional: "Guardar contactos, citas y cambios de estado mediante un puerto de persistencia.",
			TiposDatos:         []string{"contactos", "citas", "estado"},
			Sensibilidad:       "media",
			Retencion:          "configurable",
		},
		Calidad: orquestaweb.WebNuevaAppCalidadFormV0{
			Pruebas:        "alta",
			Accesibilidad:  "basica",
			Observabilidad: &enabled,
		},
		Documentacion: orquestaweb.WebNuevaAppDocumentacionFormV0{
			Usuario:    &enabled,
			Desarrollo: &enabled,
			Sistemas:   &enabled,
			Locales:    []string{"es-ES", "en-US"},
		},
		I18N: orquestaweb.WebNuevaAppI18NFormV0{
			Enabled:       &enabled,
			DefaultLocale: "es-ES",
			Locales:       []string{"es-ES", "en-US"},
		},
		Restricciones: []string{
			"API REST",
			"web de administracion",
			"preparada para pruebas de seguridad",
		},
	}
}

func mcpFormDirectorSmokeObjectiveV0(form orquestaweb.WebNuevaAppFormV0) string {
	return strings.Join([]string{
		"Actua como agente director de Orquesta para la app solicitada desde formulario.",
		"App: " + form.Nombre + ".",
		"Objetivo: " + form.Objetivo,
		"Tipo: API REST en Go y web de administracion.",
		"Reglas: arquitectura hexagonal, i18n es-ES/en-US, persistencia por puerto/conector, funciones pequenas, sin archivos gigantes.",
		"Entrega solo dos documentos profesionales y accionables: docs/arquitectura.md y docs/plan_microtareas.md.",
		"El plan debe separar brainstorming, documentacion, programacion, pruebas, seguridad y revision final.",
		"Si falta informacion no inferible, deja CONSULTA AL DIRECTOR en el documento.",
	}, " ")
}

func mcpFormDirectorSmokeDescriptorV0(
	t *testing.T,
	store *InMemoryCodexReceiptDescriptorStoreV0,
	runRef string,
	startedAgents []string,
) CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         runRef,
		StartedAgents: startedAgents,
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	return descriptors[0]
}

func mcpFormDirectorSmokeVerifyDocsV0(t *testing.T, projectDir string) {
	t.Helper()
	for _, path := range []string{"docs/arquitectura.md", "docs/plan_microtareas.md"} {
		data, err := os.ReadFile(filepath.Join(projectDir, path))
		if err != nil {
			t.Fatalf("doc missing %s: %v", path, err)
		}
		if len(strings.TrimSpace(string(data))) < 200 {
			t.Fatalf("doc demasiado pequeno %s", path)
		}
	}
}
