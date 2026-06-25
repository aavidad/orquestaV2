package orquestaweb

import (
	"fmt"
	"sort"
	"strings"
)

const (
	NuevaAppI18nDefaultLocaleV0 = "es-ES"
	NuevaAppI18nEnglishLocaleV0 = "en-US"

	NuevaAppI18nErrClaveNoEncontradaV0  = "nueva_app_i18n_clave_no_encontrada"
	NuevaAppI18nErrCatalogoIncompletoV0 = "nueva_app_i18n_catalogo_incompleto"
)

type NuevaAppI18nCatalogV0 struct {
	defaultLocale string
	messages      map[string]map[string]string
}

type NuevaAppI18nErrorV0 struct {
	Code   string
	Locale string
	Key    string
	Text   string
}

func (err NuevaAppI18nErrorV0) Error() string {
	return err.Code
}

func NewNuevaAppI18nCatalogV0() NuevaAppI18nCatalogV0 {
	return NuevaAppI18nCatalogV0{
		defaultLocale: NuevaAppI18nDefaultLocaleV0,
		messages: map[string]map[string]string{
			NuevaAppI18nDefaultLocaleV0: nuevaAppI18nSpanishV0(),
			NuevaAppI18nEnglishLocaleV0: nuevaAppI18nEnglishV0(),
		},
	}
}

func NuevaAppI18nTextV0(locale, key string) (string, error) {
	return NewNuevaAppI18nCatalogV0().Lookup(locale, key)
}

func NuevaAppI18nRequiredKeysV0() []string {
	keys := append([]string{}, nuevaAppI18nRequiredKeysV0...)
	keys = append(keys, nuevaAppHTMLHelpI18nKeysV0()...)
	return keys
}

func (catalog NuevaAppI18nCatalogV0) Lookup(locale, key string) (string, error) {
	normalizedLocale := catalog.normalizeLocale(locale)
	normalizedKey := strings.TrimSpace(key)
	if text := catalog.lookupExact(normalizedLocale, normalizedKey); text != "" {
		return text, nil
	}
	if normalizedLocale != catalog.defaultLocale {
		if text := catalog.lookupExact(catalog.defaultLocale, normalizedKey); text != "" {
			return text, nil
		}
	}
	return catalog.publicMissingKeyError(normalizedLocale, normalizedKey)
}

func (catalog NuevaAppI18nCatalogV0) ValidateRequired() error {
	for _, locale := range catalog.SupportedLocales() {
		for _, key := range NuevaAppI18nRequiredKeysV0() {
			if catalog.lookupExact(locale, key) == "" {
				return NuevaAppI18nErrorV0{
					Code:   NuevaAppI18nErrCatalogoIncompletoV0,
					Locale: locale,
					Key:    key,
					Text:   NuevaAppI18nErrCatalogoIncompletoV0,
				}
			}
		}
	}
	return nil
}

func (catalog NuevaAppI18nCatalogV0) SupportedLocales() []string {
	locales := make([]string, 0, len(catalog.messages))
	for locale := range catalog.messages {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

func (catalog NuevaAppI18nCatalogV0) DefaultLocale() string {
	return catalog.defaultLocale
}

func (catalog NuevaAppI18nCatalogV0) lookupExact(locale, key string) string {
	return strings.TrimSpace(catalog.messages[locale][key])
}

func (catalog NuevaAppI18nCatalogV0) normalizeLocale(locale string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")) {
	case "es", "es-es":
		return NuevaAppI18nDefaultLocaleV0
	case "en", "en-us":
		return NuevaAppI18nEnglishLocaleV0
	default:
		return catalog.defaultLocale
	}
}

func (catalog NuevaAppI18nCatalogV0) publicMissingKeyError(locale, key string) (string, error) {
	text := catalog.lookupExact(locale, "nueva_app.error.clave_no_encontrada")
	if text == "" && locale != catalog.defaultLocale {
		text = catalog.lookupExact(catalog.defaultLocale, "nueva_app.error.clave_no_encontrada")
	}
	if text == "" {
		text = NuevaAppI18nErrClaveNoEncontradaV0
	}
	return text, NuevaAppI18nErrorV0{
		Code:   NuevaAppI18nErrClaveNoEncontradaV0,
		Locale: locale,
		Key:    key,
		Text:   fmt.Sprintf("%s:%s", text, key),
	}
}
