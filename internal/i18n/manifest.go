package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

const catalogSourcePrefix = "catalog:"

func loadManifest(source fs.FS) (Manifest, error) {
	content, err := fs.ReadFile(source, "manifest.json")
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
	}
	if err := validateStrictJSON(content); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Manifest{}, fmt.Errorf("%w: trailing data", ErrManifestInvalid)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 || manifest.DefaultLocale != DefaultLocale ||
		manifest.FallbackLocale != DefaultLocale ||
		!equalStrings(manifest.EnabledLocales, []string{"es", "en"}) {
		return fmt.Errorf("%w: identity/locales", ErrManifestInvalid)
	}
	enabled := make(map[string]struct{}, len(manifest.EnabledLocales))
	for _, locale := range manifest.EnabledLocales {
		tag, err := language.Parse(locale)
		if err != nil || tag.String() != locale {
			return fmt.Errorf("%w: locale=%s", ErrManifestInvalid, locale)
		}
		enabled[locale] = struct{}{}
	}
	catalogs := make(map[string]struct{}, len(manifest.Catalogs))
	for _, item := range manifest.Catalogs {
		if _, ok := enabled[item.Locale]; !ok || item.Path != "catalogs/"+item.Locale+".json" {
			return fmt.Errorf("%w: catalog=%s", ErrManifestInvalid, item.Locale)
		}
		if _, duplicate := catalogs[item.Locale]; duplicate {
			return fmt.Errorf("%w: duplicate catalog=%s", ErrManifestInvalid, item.Locale)
		}
		catalogs[item.Locale] = struct{}{}
	}
	if len(catalogs) != len(enabled) {
		return fmt.Errorf("%w: catalog parity", ErrManifestInvalid)
	}
	requiredSurfaces := map[string][2]string{
		"cli": {"active", "catalog_only"}, "command_registry": {"active", "registry_keys_catalog_values"},
		"http": {"active", "machine_envelope_catalog_presenter"}, "mcp": {"active", "catalog_only"},
		"public_docs": {"active", "localized_document_bundle"}, "notifications": {"future", "catalog_required_before_activation"},
		"prompts": {"active", "catalog_only"}, "web": {"future", "catalog_required_before_activation"},
		"wizard": {"future", "catalog_required_before_activation"},
	}
	surfaces := make(map[string]Surface, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		expected, ok := requiredSurfaces[surface.ID]
		if !ok || surface.State != expected[0] || surface.LiteralPolicy != expected[1] {
			return fmt.Errorf("%w: surface=%s", ErrManifestInvalid, surface.ID)
		}
		if _, duplicate := surfaces[surface.ID]; duplicate {
			return fmt.Errorf("%w: duplicate surface=%s", ErrManifestInvalid, surface.ID)
		}
		if surface.State == "future" && len(surface.KeySources) != 0 {
			return fmt.Errorf("%w: future surface has sources=%s", ErrManifestInvalid, surface.ID)
		}
		if surface.State == "active" && len(surface.KeySources) == 0 {
			return fmt.Errorf("%w: active surface has no sources=%s", ErrManifestInvalid, surface.ID)
		}
		if !sort.StringsAreSorted(surface.KeySources) || hasDuplicateStrings(surface.KeySources) {
			return fmt.Errorf("%w: unsorted or duplicate key sources=%s", ErrManifestInvalid, surface.ID)
		}
		surfaces[surface.ID] = surface
	}
	if len(surfaces) != len(requiredSurfaces) {
		return fmt.Errorf("%w: surface set", ErrManifestInvalid)
	}
	if err := validatePublicDocuments(manifest.PublicDocuments, enabled); err != nil {
		return err
	}
	return nil
}

func hasDuplicateStrings(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func validatePublicDocuments(documents []PublicDocument, enabled map[string]struct{}) error {
	if len(documents) == 0 {
		return fmt.Errorf("%w: public documents missing", ErrManifestInvalid)
	}
	seen := make(map[string]struct{}, len(documents))
	for _, document := range documents {
		if !validKey(document.ID) || len(document.Locales) != len(enabled) {
			return fmt.Errorf("%w: public document=%s", ErrManifestInvalid, document.ID)
		}
		if _, duplicate := seen[document.ID]; duplicate {
			return fmt.Errorf("%w: duplicate public document=%s", ErrManifestInvalid, document.ID)
		}
		seen[document.ID] = struct{}{}
		for locale := range enabled {
			documentPath, ok := document.Locales[locale]
			if !ok || path.Clean(documentPath) != documentPath ||
				!strings.HasPrefix(documentPath, "docs/public/"+locale+"/") {
				return fmt.Errorf("%w: public document=%s locale=%s", ErrManifestInvalid, document.ID, locale)
			}
		}
	}
	return nil
}

func validateClaims(manifest Manifest, catalogKeys []string) error {
	claims := make(map[string]string, len(catalogKeys))
	surfaceIDs := make(map[string]struct{}, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		surfaceIDs[surface.ID] = struct{}{}
	}
	for _, surface := range manifest.Surfaces {
		for _, source := range surface.KeySources {
			switch {
			case strings.HasPrefix(source, catalogSourcePrefix):
				key := strings.TrimPrefix(source, catalogSourcePrefix)
				if !validKey(key) {
					return fmt.Errorf("%w: invalid claimed key=%s", ErrManifestInvalid, key)
				}
				if owner, duplicate := claims[key]; duplicate {
					return fmt.Errorf("%w: key=%s owners=%s,%s", ErrManifestInvalid, key, owner, surface.ID)
				}
				claims[key] = surface.ID
			case strings.HasPrefix(source, "surface:"):
				target := strings.TrimPrefix(source, "surface:")
				if _, ok := surfaceIDs[target]; !ok || target == surface.ID {
					return fmt.Errorf("%w: invalid surface source=%s", ErrManifestInvalid, source)
				}
			case source == "public_documents":
				if surface.ID != "public_docs" {
					return fmt.Errorf("%w: invalid public document source", ErrManifestInvalid)
				}
			default:
				return fmt.Errorf("%w: unknown key source=%s", ErrManifestInvalid, source)
			}
		}
	}
	for _, key := range catalogKeys {
		if _, ok := claims[key]; !ok {
			return fmt.Errorf("%w: unclaimed key=%s", ErrManifestInvalid, key)
		}
		delete(claims, key)
	}
	for key := range claims {
		return fmt.Errorf("%w: claimed key missing=%s", ErrManifestInvalid, key)
	}
	return nil
}

func cloneManifest(source Manifest) Manifest {
	result := source
	result.EnabledLocales = append([]string(nil), source.EnabledLocales...)
	result.Catalogs = append([]CatalogSource(nil), source.Catalogs...)
	result.Surfaces = make([]Surface, len(source.Surfaces))
	for index, surface := range source.Surfaces {
		result.Surfaces[index] = surface
		result.Surfaces[index].KeySources = append([]string(nil), surface.KeySources...)
	}
	result.PublicDocuments = make([]PublicDocument, len(source.PublicDocuments))
	for index, document := range source.PublicDocuments {
		locales := make(map[string]string, len(document.Locales))
		for locale, documentPath := range document.Locales {
			locales[locale] = documentPath
		}
		result.PublicDocuments[index] = PublicDocument{ID: document.ID, Locales: locales}
	}
	return result
}
