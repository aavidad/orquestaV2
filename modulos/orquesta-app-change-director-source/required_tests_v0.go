package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const (
	maxAppChangeTaskRequiredTestsV0        = 24
	maxAppChangeTaskRequiredTestCharsV0    = 260
	appChangeExternalRequiredTestMarkerV0  = "validar required_tests externos declarados en paquete de dominio"
	appChangeCompactedRequiredTestSuffixV0 = " (detalle completo en paquete externo)"
)

func appChangeTaskRequiredTestsV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	tests := append([]string(nil), request.RequiredTests...)
	tests = append(tests, "validar criterios de aceptacion del cambio")
	if appChangeHasExternalWorkV0(request) {
		tests = append(tests, "validar contrato externo de dominio")
		if appChangeIsDraftContentBlockWorkV0(request) {
			tests = append(tests,
				"validar paquete de bloque documental",
				"validar granularidad editorial coherente",
				"validar fuentes, criterios y longitud si llegan",
			)
		} else if appChangeIsVisualExternalWorkV0(request) {
			tests = append(tests, appChangeVisualRequiredTestsV0(request)...)
		} else if appChangeIsAudioExternalWorkV0(request) {
			tests = append(tests, appChangeAudioRequiredTestsV0(request)...)
		} else if appChangeIsFinalPackageExternalWorkV0(request) {
			tests = append(tests,
				"validar final_domain_package",
				"validar matriz de evidencias de cierre",
				"validar bloqueos causales antes de publicar",
				"validar que no hay subida a produccion sin confirmacion",
			)
		} else if appChangeIsSummaryExternalWorkV0(request) {
			tests = append(tests,
				"validar trazabilidad del resumen",
				"validar conservacion de matices criticos",
			)
		} else if appChangeIsExpansionExternalWorkV0(request) {
			tests = append(tests,
				"validar topic_expansion_package",
				"validar longitud declarada del tema grande",
				"validar variantes documentales requeridas",
			)
		} else if appChangeIsDocumentPlanWorkV0(request) {
			tests = append(tests,
				"validar document_plan",
				"validar secciones y entregables del plan",
				"validar criterios de calidad documentales",
			)
		} else if appChangeIsDocumentaryExternalWorkV0(request) {
			tests = append(tests,
				"validar paquete documental de dominio",
				"validar fuentes y criterios documentales",
			)
		}
	}
	if appChangeWriteSetLooksLikeGoV0(request.AllowedWriteSet) {
		if !appChangeRequiredTestsContainGoTestV0(tests) {
			tests = append([]string{"go test ./..."}, tests...)
		}
	}
	return compactAppChangeTaskRequiredTestsV0(tests)
}

func appChangeRequiredTestsContainGoTestV0(tests []string) bool {
	for _, test := range tests {
		if strings.HasPrefix(strings.TrimSpace(test), "go test") {
			return true
		}
	}
	return false
}

func compactAppChangeTaskRequiredTestsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if trimmed == "" {
			continue
		}
		trimmed = compactAppChangeTaskRequiredTestForDirectorV0(trimmed)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
		if len(out) >= maxAppChangeTaskRequiredTestsV0 {
			break
		}
	}
	return out
}

func compactAppChangeTaskRequiredTestForDirectorV0(value string) string {
	if len(value) <= maxAppChangeTaskRequiredTestCharsV0 {
		return value
	}
	if appChangeRequiredTestLooksInlineCommandV0(value) {
		return appChangeExternalRequiredTestMarkerV0
	}
	suffix := appChangeCompactedRequiredTestSuffixV0
	limit := maxAppChangeTaskRequiredTestCharsV0 - len(suffix)
	if limit < 1 {
		return strings.TrimSpace(suffix)
	}
	prefix := strings.TrimSpace(value[:limit])
	if cut := strings.LastIndex(prefix, " "); cut > 80 {
		prefix = strings.TrimSpace(prefix[:cut])
	}
	return prefix + suffix
}

func appChangeRequiredTestLooksInlineCommandV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	commandPrefixes := []string{
		"python ",
		"python3 ",
		"go test ",
		"npm ",
		"pnpm ",
		"yarn ",
		"node ",
		"bash ",
		"sh ",
	}
	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return strings.Contains(lower, " -c ") ||
		strings.Contains(lower, " && ") ||
		strings.Contains(lower, " | ")
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
