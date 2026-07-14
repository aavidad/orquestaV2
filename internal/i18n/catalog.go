package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

const DefaultLocale = "es"

//go:embed catalogs/*.json
var bundledCatalogs embed.FS

type Catalog struct {
	matcher  language.Matcher
	tags     []language.Tag
	messages map[string]map[string]string
}

func LoadBundled() (*Catalog, error) {
	entries, err := bundledCatalogs.ReadDir("catalogs")
	if err != nil {
		return nil, fmt.Errorf("i18n_catalog_read: %w", err)
	}

	messages := make(map[string]map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		body, readErr := bundledCatalogs.ReadFile("catalogs/" + entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("i18n_catalog_read: %w", readErr)
		}
		var values map[string]string
		if decodeErr := json.Unmarshal(body, &values); decodeErr != nil {
			return nil, fmt.Errorf("i18n_catalog_decode: %s: %w", locale, decodeErr)
		}
		messages[locale] = values
	}
	if len(messages) == 0 {
		return nil, errors.New("i18n_catalog_empty")
	}
	if _, ok := messages[DefaultLocale]; !ok {
		return nil, errors.New("i18n_default_locale_missing")
	}
	if err := validateParity(messages); err != nil {
		return nil, err
	}

	locales := make([]string, 0, len(messages))
	for locale := range messages {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	tags := make([]language.Tag, 0, len(locales))
	for _, locale := range locales {
		tag, parseErr := language.Parse(locale)
		if parseErr != nil {
			return nil, fmt.Errorf("i18n_locale_invalid: %s: %w", locale, parseErr)
		}
		tags = append(tags, tag)
	}

	return &Catalog{
		matcher:  language.NewMatcher(tags),
		tags:     tags,
		messages: messages,
	}, nil
}

func (c *Catalog) Text(locale, key string) string {
	if c == nil {
		return key
	}
	tag, _, _ := c.matcher.Match(language.Make(locale))
	matched := tag.String()
	if values, ok := c.messages[matched]; ok {
		if value, found := values[key]; found {
			return value
		}
	}
	if values, ok := c.messages[baseLocale(matched)]; ok {
		if value, found := values[key]; found {
			return value
		}
	}
	if value, ok := c.messages[DefaultLocale][key]; ok {
		return value
	}
	return key
}

func validateParity(messages map[string]map[string]string) error {
	reference := messages[DefaultLocale]
	for locale, values := range messages {
		for key := range reference {
			if _, ok := values[key]; !ok {
				return fmt.Errorf("i18n_key_missing: locale=%s key=%s", locale, key)
			}
		}
		for key := range values {
			if _, ok := reference[key]; !ok {
				return fmt.Errorf("i18n_key_extra: locale=%s key=%s", locale, key)
			}
		}
	}
	return nil
}

func baseLocale(locale string) string {
	if index := strings.IndexByte(locale, '-'); index >= 0 {
		return locale[:index]
	}
	return locale
}
