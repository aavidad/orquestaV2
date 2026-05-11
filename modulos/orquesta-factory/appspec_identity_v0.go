package orquestafactory

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func buildSpecIDV0(req AppSpecRequestV0) string {
	id := "spec-" + slugV0(req.Nombre) + "-" + cleanIDPartV0(req.RequestID)
	if len(id) <= 120 {
		return id
	}
	return strings.TrimRight(id[:120], "-._:")
}

func slugV0(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range norm.NFD.String(strings.ToLower(strings.TrimSpace(value))) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "app"
	}
	return slug
}

func cleanIDPartV0(value string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == ':' || r == '-' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	return strings.Trim(builder.String(), "-")
}
