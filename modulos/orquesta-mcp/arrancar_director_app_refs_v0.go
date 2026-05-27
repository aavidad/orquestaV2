package orquestamcp

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func neutralAppDirectorRequestRefV0(prefix string, appName string, externalRefs ...string) string {
	slug := neutralAppDirectorSlugV0(appName)
	hash := neutralAppDirectorHashV0(externalRefs...)
	return prefix + "-" + slug + "-" + hash
}

func neutralAppDirectorSlugV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if isMCPRefLetterOrDigitV0(ch) {
			builder.WriteByte(ch)
			lastDash = false
			continue
		}
		if !lastDash {
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

func neutralAppDirectorHashV0(fields ...string) string {
	if len(fields) == 0 {
		fields = []string{"empty"}
	}
	sum := sha256.Sum256(neutralAppDirectorCanonicalPayloadV0(fields...))
	return hex.EncodeToString(sum[:])[:32]
}

func neutralAppDirectorCanonicalPayloadV0(fields ...string) []byte {
	var builder strings.Builder
	builder.WriteString("orquesta-mcp.app-director-ref.v0|")
	builder.WriteString(strconv.Itoa(len(fields)))
	builder.WriteByte('|')
	for _, field := range fields {
		value := strings.TrimSpace(field)
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
		builder.WriteByte(';')
	}
	return []byte(builder.String())
}

func isMCPRefLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
