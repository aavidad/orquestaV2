package orquestacoreworkflow

import (
	"strings"
	"unicode"
)

func normalizeLooseEnumTokenV0(value string) string {
	var builder strings.Builder
	lastSeparator := false
	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		mapped := normalizeLooseEnumRuneV0(current)
		if mapped == 0 {
			continue
		}
		if mapped == '_' {
			if builder.Len() == 0 || lastSeparator {
				continue
			}
			builder.WriteByte('_')
			lastSeparator = true
			continue
		}
		builder.WriteRune(mapped)
		lastSeparator = false
	}
	return strings.Trim(builder.String(), "_")
}

func normalizeLooseEnumRuneV0(value rune) rune {
	switch value {
	case '\u00e1', '\u00e0', '\u00e4', '\u00e2':
		return 'a'
	case '\u00e9', '\u00e8', '\u00eb', '\u00ea':
		return 'e'
	case '\u00ed', '\u00ec', '\u00ef', '\u00ee':
		return 'i'
	case '\u00f3', '\u00f2', '\u00f6', '\u00f4':
		return 'o'
	case '\u00fa', '\u00f9', '\u00fc', '\u00fb':
		return 'u'
	case '\u00f1':
		return 'n'
	case '-', '_', ' ', '\t', '\r', '\n', '.', '/', '\\', ':', ';', ',', '(', ')', '[', ']', '{', '}':
		return '_'
	default:
		if unicode.IsLetter(value) || unicode.IsDigit(value) {
			return value
		}
		return 0
	}
}
