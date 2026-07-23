package i18n

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

//go:embed catalogs/*.json manifest.json
var bundledFiles embed.FS

type Catalog struct {
	manifest Manifest
	locales  []string
	keys     []string
	tags     map[string]language.Tag
	messages map[string]map[string]catalogMessage
}

func LoadBundled() (*Catalog, error) {
	catalog, err := loadCatalog(bundledFiles)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCatalogUnavailable, err)
	}
	return catalog, nil
}

func loadCatalog(source fs.FS) (*Catalog, error) {
	manifest, err := loadManifest(source)
	if err != nil {
		return nil, err
	}
	declared := make(map[string]string, len(manifest.Catalogs))
	for _, item := range manifest.Catalogs {
		declared[item.Locale] = item.Path
	}
	if err := validateCatalogFiles(source, declared); err != nil {
		return nil, err
	}

	messages := make(map[string]map[string]catalogMessage, len(manifest.Catalogs))
	tags := make(map[string]language.Tag, len(manifest.Catalogs))
	for _, locale := range manifest.EnabledLocales {
		catalogPath := declared[locale]
		content, readErr := fs.ReadFile(source, catalogPath)
		if readErr != nil {
			return nil, fmt.Errorf("%w: locale=%s: %v", ErrCatalogInvalid, locale, readErr)
		}
		values, decodeErr := decodeCatalog(locale, content)
		if decodeErr != nil {
			return nil, decodeErr
		}
		tag, _ := language.Parse(locale)
		messages[locale] = values
		tags[locale] = tag
	}
	if err := validateParity(messages, manifest.DefaultLocale); err != nil {
		return nil, err
	}
	keys := messageKeys(messages[manifest.DefaultLocale])
	if err := validateClaims(manifest, keys); err != nil {
		return nil, err
	}
	return &Catalog{
		manifest: cloneManifest(manifest),
		locales:  append([]string(nil), manifest.EnabledLocales...),
		keys:     keys, tags: tags, messages: messages,
	}, nil
}

func (catalog *Catalog) Resolve(locale string) (Resolution, error) {
	if catalog == nil {
		return Resolution{}, ErrCatalogUnavailable
	}
	if locale == "" {
		return Resolution{Locale: catalog.manifest.FallbackLocale, Fallback: true}, nil
	}
	if strings.TrimSpace(locale) != locale || strings.ContainsRune(locale, '_') {
		return Resolution{}, fmt.Errorf("%w: %q", ErrLocaleInvalid, locale)
	}
	tag, err := language.Parse(locale)
	if err != nil {
		return Resolution{}, fmt.Errorf("%w: %q", ErrLocaleInvalid, locale)
	}
	canonical := tag.String()
	if _, ok := catalog.tags[canonical]; ok {
		return Resolution{Requested: canonical, Locale: canonical}, nil
	}
	base, _ := tag.Base()
	for _, supported := range catalog.locales {
		supportedBase, _ := catalog.tags[supported].Base()
		if base == supportedBase {
			return Resolution{Requested: canonical, Locale: supported}, nil
		}
	}
	return Resolution{
		Requested: canonical, Locale: catalog.manifest.FallbackLocale, Fallback: true,
	}, nil
}

func (catalog *Catalog) Text(locale, key string) (string, error) {
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return "", err
	}
	message, err := catalog.message(resolution.Locale, key)
	if err != nil {
		return "", err
	}
	if message.kind != textMessage {
		return "", fmt.Errorf("%w: key=%s", ErrMessageKind, key)
	}
	return message.text, nil
}

func (catalog *Catalog) Keys() []string {
	if catalog == nil {
		return nil
	}
	return append([]string(nil), catalog.keys...)
}

func (catalog *Catalog) Locales() []string {
	if catalog == nil {
		return nil
	}
	return append([]string(nil), catalog.locales...)
}

func (catalog *Catalog) Manifest() Manifest {
	if catalog == nil {
		return Manifest{}
	}
	return cloneManifest(catalog.manifest)
}

func (catalog *Catalog) message(locale, key string) (catalogMessage, error) {
	if !validKey(key) {
		return catalogMessage{}, fmt.Errorf("%w: key=%q", ErrKeyMissing, key)
	}
	message, ok := catalog.messages[locale][key]
	if !ok {
		return catalogMessage{}, fmt.Errorf("%w: key=%s", ErrKeyMissing, key)
	}
	return message, nil
}

func validateCatalogFiles(source fs.FS, declared map[string]string) error {
	entries, err := fs.ReadDir(source, "catalogs")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCatalogInvalid, err)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("%w: unexpected catalog entry %s", ErrCatalogInvalid, entry.Name())
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		tag, parseErr := language.Parse(locale)
		if parseErr != nil || tag.String() != locale {
			return fmt.Errorf("%w: noncanonical filename locale=%s", ErrLocaleInvalid, locale)
		}
		catalogPath, ok := declared[locale]
		if !ok || catalogPath != "catalogs/"+entry.Name() {
			return fmt.Errorf("%w: undeclared catalog %s", ErrCatalogInvalid, entry.Name())
		}
		seen[locale] = struct{}{}
	}
	if len(seen) != len(declared) {
		return fmt.Errorf("%w: declared catalog missing", ErrCatalogInvalid)
	}
	return nil
}

func validateParity(messages map[string]map[string]catalogMessage, referenceLocale string) error {
	reference := messages[referenceLocale]
	if len(reference) == 0 {
		return fmt.Errorf("%w: default locale missing", ErrCatalogInvalid)
	}
	for locale, values := range messages {
		for key, expected := range reference {
			actual, ok := values[key]
			if !ok {
				return fmt.Errorf("%w: locale=%s key=%s", ErrKeyMissing, locale, key)
			}
			if actual.kind != expected.kind || !equalStrings(actual.labels, expected.labels) ||
				!equalStrings(actual.placeholders, expected.placeholders) ||
				!equalPluralPlaceholders(actual, expected) {
				return fmt.Errorf("%w: locale=%s key=%s kind mismatch", ErrCatalogInvalid, locale, key)
			}
		}
		for key := range values {
			if _, ok := reference[key]; !ok {
				return fmt.Errorf("%w: locale=%s key=%s extra", ErrCatalogInvalid, locale, key)
			}
		}
	}
	return nil
}

func equalPluralPlaceholders(left, right catalogMessage) bool {
	if left.kind != pluralMessage && right.kind != pluralMessage {
		return true
	}
	if left.kind != pluralMessage || right.kind != pluralMessage {
		return false
	}
	for _, label := range left.labels {
		form := pluralForms[label]
		if !equalStrings(left.formPlaceholders[form], right.formPlaceholders[form]) {
			return false
		}
	}
	return true
}

func messageKeys(messages map[string]catalogMessage) []string {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
