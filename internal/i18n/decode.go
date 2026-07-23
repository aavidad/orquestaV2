package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/feature/plural"
)

type messageKind uint8

const (
	textMessage messageKind = iota + 1
	pluralMessage
)

type catalogMessage struct {
	kind             messageKind
	text             string
	placeholders     []string
	forms            map[plural.Form]string
	formPlaceholders map[plural.Form][]string
	labels           []string
}

var pluralForms = map[string]plural.Form{
	"zero":  plural.Zero,
	"one":   plural.One,
	"two":   plural.Two,
	"few":   plural.Few,
	"many":  plural.Many,
	"other": plural.Other,
}

func decodeCatalog(locale string, content []byte) (map[string]catalogMessage, error) {
	if err := validateStrictJSON(content); err != nil {
		return nil, fmt.Errorf("%w: locale=%s: %v", ErrCatalogInvalid, locale, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: locale=%s: %v", ErrCatalogInvalid, locale, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: locale=%s: empty", ErrCatalogInvalid, locale)
	}
	messages := make(map[string]catalogMessage, len(raw))
	for key, encoded := range raw {
		if !validKey(key) {
			return nil, fmt.Errorf("%w: locale=%s key=%q", ErrCatalogInvalid, locale, key)
		}
		message, err := decodeMessage(encoded)
		if err != nil {
			return nil, fmt.Errorf("%w: locale=%s key=%s: %v", ErrCatalogInvalid, locale, key, err)
		}
		messages[key] = message
	}
	return messages, nil
}

func decodeMessage(encoded []byte) (catalogMessage, error) {
	if len(encoded) == 0 {
		return catalogMessage{}, fmt.Errorf("empty value")
	}
	if encoded[0] == '"' {
		var value string
		if err := json.Unmarshal(encoded, &value); err != nil {
			return catalogMessage{}, err
		}
		if strings.TrimSpace(value) == "" {
			return catalogMessage{}, fmt.Errorf("empty text")
		}
		placeholders, err := extractPlaceholders(value)
		if err != nil {
			return catalogMessage{}, err
		}
		return catalogMessage{kind: textMessage, text: value, placeholders: placeholders}, nil
	}
	var raw map[string]string
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return catalogMessage{}, err
	}
	if len(raw) == 0 {
		return catalogMessage{}, fmt.Errorf("empty plural")
	}
	forms := make(map[plural.Form]string, len(raw))
	formPlaceholders := make(map[plural.Form][]string, len(raw))
	labels := make([]string, 0, len(raw))
	var referencePlaceholders []string
	for label, value := range raw {
		form, ok := pluralForms[label]
		if !ok {
			return catalogMessage{}, fmt.Errorf("unknown plural form %q", label)
		}
		if strings.TrimSpace(value) == "" || !validCountTemplate(value) {
			return catalogMessage{}, fmt.Errorf("invalid plural form %q", label)
		}
		placeholders, err := extractPlaceholders(value)
		if err != nil {
			return catalogMessage{}, fmt.Errorf("plural form %q: %w", label, err)
		}
		if referencePlaceholders == nil {
			referencePlaceholders = placeholders
		} else if !equalStrings(referencePlaceholders, placeholders) {
			return catalogMessage{}, fmt.Errorf("plural placeholder mismatch in %q", label)
		}
		forms[form] = value
		formPlaceholders[form] = placeholders
		labels = append(labels, label)
	}
	if _, ok := forms[plural.Other]; !ok {
		return catalogMessage{}, fmt.Errorf("plural form other missing")
	}
	return catalogMessage{
		kind: pluralMessage, forms: forms, formPlaceholders: formPlaceholders,
		labels: sortedCopy(labels),
	}, nil
}

func validateStrictJSON(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing value")
		}
		return fmt.Errorf("trailing data: %w", err)
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delimiter {
	case '{':
		names := make(map[string]struct{})
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return fmt.Errorf("object name is not a string")
			}
			if _, duplicate := names[name]; duplicate {
				return fmt.Errorf("duplicate name %q", name)
			}
			names[name] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array not closed")
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delimiter)
	}
	return nil
}

func validKey(key string) bool {
	if key == "" || strings.TrimSpace(key) != key {
		return false
	}
	for _, segment := range strings.Split(key, ".") {
		if segment == "" || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for index := 1; index < len(segment); index++ {
			character := segment[index]
			if (character < 'a' || character > 'z') &&
				(character < '0' || character > '9') && character != '_' {
				return false
			}
		}
	}
	return true
}

func validCountTemplate(template string) bool {
	placeholders := 0
	for index := 0; index < len(template); index++ {
		if template[index] != '%' {
			continue
		}
		if index+1 < len(template) && template[index+1] == '%' {
			index++
			continue
		}
		switch {
		case index+1 < len(template) && template[index+1] == 'd':
			index++
		case strings.HasPrefix(template[index:], "%[1]d"):
			index += len("%[1]d") - 1
		default:
			return false
		}
		placeholders++
	}
	return placeholders == 1
}

func extractPlaceholders(text string) ([]string, error) {
	names := make(map[string]struct{})
	for index := 0; index < len(text); index++ {
		switch text[index] {
		case '{':
			end := strings.IndexByte(text[index+1:], '}')
			if end < 0 {
				return nil, fmt.Errorf("placeholder is not closed")
			}
			end += index + 1
			name := text[index+1 : end]
			if !validPlaceholderName(name) {
				return nil, fmt.Errorf("invalid placeholder %q", name)
			}
			names[name] = struct{}{}
			index = end
		case '}':
			return nil, fmt.Errorf("placeholder closing brace without opening")
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return sortedCopy(result), nil
}

func validPlaceholderName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for index := 1; index < len(name); index++ {
		character := name[index]
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}
