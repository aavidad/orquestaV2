package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func appChangeTaskRequiredTestsV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	tests := []string{"validar criterios de aceptacion del cambio"}
	if appChangeWriteSetLooksLikeGoV0(request.AllowedWriteSet) {
		tests = append([]string{"go test ./..."}, tests...)
	}
	return tests
}

func appChangeWriteSetLooksLikeGoV0(writeSet []string) bool {
	for _, entry := range writeSet {
		normalized := strings.Trim(strings.ToLower(strings.TrimSpace(entry)), "/")
		if normalized == "go.mod" ||
			normalized == "main.go" ||
			strings.HasSuffix(normalized, ".go") ||
			strings.HasPrefix(normalized, "api/") ||
			strings.HasPrefix(normalized, "cmd/") ||
			strings.HasPrefix(normalized, "internal/") ||
			strings.HasPrefix(normalized, "pkg/") {
			return true
		}
	}
	return false
}
