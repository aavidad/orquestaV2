package orquestamcp

import (
	"fmt"
	"hash/fnv"
	"strings"
)

func neutralAppDirectorRequestRefV0(prefix string, appName string, externalRefs ...string) string {
	slug := neutralAppDirectorSlugV0(appName)
	hash := neutralAppDirectorHashV0(strings.Join(externalRefs, "|"))
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

func neutralAppDirectorHashV0(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = "empty"
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(trimmed))
	return fmt.Sprintf("%08x", hasher.Sum32())
}

func isMCPRefLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
