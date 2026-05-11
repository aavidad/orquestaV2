package orquestaweb

import (
	"errors"
	"testing"
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
