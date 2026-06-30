package orquestaweb

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	if err := catalog.ValidateRequired(); err != nil {
		t.Fatalf("catalogo incompleto: %v", err)
	}

	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		for _, key := range NuevaAppI18nRequiredKeysV0() {
			text, err := catalog.Lookup(locale, key)
			if err != nil {
				t.Fatalf("lookup %s/%s: %v", locale, key, err)
			}
			if text == "" {
				t.Fatalf("texto vacio para %s/%s", locale, key)
			}
		}
	}
}

func TestNuevaAppI18nRequiredKeysV0DevuelveCopia(t *testing.T) {
	keys := NuevaAppI18nRequiredKeysV0()
	if len(keys) == 0 {
		t.Fatalf("claves requeridas vacias")
	}
	keys[0] = "nueva_app.clave_mutada"

	freshKeys := NuevaAppI18nRequiredKeysV0()
	if freshKeys[0] == "nueva_app.clave_mutada" {
		t.Fatalf("NuevaAppI18nRequiredKeysV0 no debe exponer slice mutable")
	}
}

func TestNuevaAppI18nCatalogV0LookupPorLocaleYFallback(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	esText, err := catalog.Lookup("es-ES", "nueva_app.titulo")
	if err != nil {
		t.Fatalf("lookup es: %v", err)
	}
	enText, err := catalog.Lookup("en-US", "nueva_app.titulo")
	if err != nil {
		t.Fatalf("lookup en: %v", err)
	}
	if esText == enText {
		t.Fatalf("se esperaban textos localizados distintos: es=%q en=%q", esText, enText)
	}

	aliasText, err := catalog.Lookup("en", "nueva_app.titulo")
	if err != nil {
		t.Fatalf("lookup alias en: %v", err)
	}
	if aliasText != enText {
		t.Fatalf("fallback de alias en inesperado: %q != %q", aliasText, enText)
	}
}

func TestNuevaAppI18nCatalogV0LocaleDesconocidoCaeAlDefault(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	defaultText, err := catalog.Lookup(NuevaAppI18nDefaultLocaleV0, "nueva_app.accion.submit")
	if err != nil {
		t.Fatalf("lookup default: %v", err)
	}
	unknownText, err := catalog.Lookup("fr-FR", "nueva_app.accion.submit")
	if err != nil {
		t.Fatalf("lookup locale desconocido: %v", err)
	}
	if unknownText != defaultText {
		t.Fatalf("locale desconocido no uso default: %q != %q", unknownText, defaultText)
	}
}

func TestNuevaAppI18nCatalogV0ClaveInexistenteDevuelveErrorPublico(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	text, err := catalog.Lookup("en-US", "nueva_app.no_existe")
	if err == nil {
		t.Fatalf("se esperaba error publico para clave inexistente")
	}
	if text == "" {
		t.Fatalf("clave inexistente debe devolver fallback publico")
	}

	var i18nErr NuevaAppI18nErrorV0
	if !errors.As(err, &i18nErr) {
		t.Fatalf("error no tipado: %T", err)
	}
	if i18nErr.Code != NuevaAppI18nErrClaveNoEncontradaV0 || i18nErr.Locale != NuevaAppI18nEnglishLocaleV0 {
		t.Fatalf("error publico inesperado: %+v", i18nErr)
	}
	if i18nErr.Key != "nueva_app.no_existe" {
		t.Fatalf("key del error=%q", i18nErr.Key)
	}
}

func TestNuevaAppI18nCatalogV0GoalBackendUnavailableTieneTextoDedicado(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		text, err := catalog.Lookup(locale, nuevaAppErrKeyGoalBackendUnavailableV0)
		if err != nil {
			t.Fatalf("lookup %s/%s: %v", locale, nuevaAppErrKeyGoalBackendUnavailableV0, err)
		}
		if text == "" || text == "Texto no disponible." || text == "Text unavailable." {
			t.Fatalf("texto goal backend generico para %s: %q", locale, text)
		}
	}
}

func TestNuevaAppHTMLHelpV0TodasLasClavesTienenTextoDedicado(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		help := nuevaAppHTMLHelpV0(locale, catalog)
		for _, key := range nuevaAppHTMLHelpKeysV0 {
			text := strings.TrimSpace(help[key])
			if text == "" || text == "Texto no disponible." || text == "Text unavailable." {
				t.Fatalf("ayuda HTML generica o vacia para %s/%s: %q", locale, key, text)
			}
		}
	}
}

func TestNuevaAppHTMLHelpV0IntegracionesEnumeraCatalogoVisibleV0(t *testing.T) {
	catalog := NewNuevaAppI18nCatalogV0()

	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		help := nuevaAppHTMLHelpV0(locale, catalog)["integraciones.0.tipo"]
		for _, integrationType := range nuevaAppHTMLIntegrationTypesV0() {
			if !strings.Contains(help, integrationType) {
				t.Fatalf("ayuda %s no enumera integracion %q: %s", locale, integrationType, help)
			}
		}
	}
}

func TestNuevaAppHTMLOptionHelpV0CubreCatalogosAmpliosV0(t *testing.T) {
	groups := map[string][]string{
		"architecture": {
			"",
			"hexagonal",
			"clean_architecture",
			"onion",
			"modular_monolith",
			"layered",
			"event_driven",
			"microservices",
			"serverless",
			"plugin_based",
			"data_pipeline",
		},
		"storage":       append([]string{""}, orquestafactory.SupportedDataStorageTypesV0()...),
		"integration":   append([]string{""}, nuevaAppHTMLIntegrationTypesV0()...),
		"accessibility": append([]string{"basica", "no_aplica"}, nuevaAppHTMLAccessibilityOptionsV0()...),
	}

	for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
		for group, values := range groups {
			seen := map[string]bool{}
			for _, value := range values {
				if seen[value] {
					continue
				}
				seen[value] = true
				help := strings.TrimSpace(nuevaAppHTMLOptionHelpV0(locale, group, value))
				if help == "" ||
					strings.Contains(help, "valor seleccionable para este campo") ||
					strings.Contains(help, "selectable value for this field") {
					t.Fatalf("ayuda generica para %s/%s/%s: %q", locale, group, value, help)
				}
			}
		}
	}
}

func TestNuevaAppGuideV0IntegracionesEnumeraCatalogoVisibleV0(t *testing.T) {
	for _, integrationType := range nuevaAppHTMLIntegrationTypesV0() {
		if !strings.Contains(nuevaAppGuideMarkdownV0, "`"+integrationType+"`") {
			t.Fatalf("guia no documenta integracion %q", integrationType)
		}
	}
}

func TestNuevaAppGuideV0DocumentaProfundidadDeManualesV0(t *testing.T) {
	for _, want := range []string{
		"### `documentacion.profundidad`",
		"`basica`",
		"`normal`",
		"`profunda`",
		"manuales desglosados por rol",
		"criterios de aceptacion",
		"diagnostico",
	} {
		if !strings.Contains(nuevaAppGuideMarkdownV0, want) {
			t.Fatalf("guia no documenta profundidad de manuales: falta %q", want)
		}
	}
}

func TestNuevaAppHTMLHelpKeysV0CubrenClavesUsadasEnPlantilla(t *testing.T) {
	source, err := os.ReadFile("nueva_app_html_render_v0.go")
	if err != nil {
		t.Fatalf("leer render HTML: %v", err)
	}
	matches := regexp.MustCompile(`index \.Help "([^"]+)"`).FindAllStringSubmatch(string(source), -1)
	if len(matches) == 0 {
		t.Fatalf("no se encontraron usos de .Help en la plantilla")
	}
	declared := map[string]bool{}
	for _, key := range nuevaAppHTMLHelpKeysV0 {
		declared[key] = true
	}
	for _, match := range matches {
		if !declared[match[1]] {
			t.Fatalf("clave .Help usada en plantilla sin declarar: %s", match[1])
		}
	}
}
