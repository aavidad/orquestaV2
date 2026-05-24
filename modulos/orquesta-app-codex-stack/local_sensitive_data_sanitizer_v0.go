package orquestaappcodexstack

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

type LocalSensitiveDataSanitizerV0 struct {
	SanitizerRef string
}

func sanitizeCodexStackAgentContextV0(
	bundle orquestacontext.ContextMaterializedBundleV0,
	sanitizer orquestacontext.ContextSanitizerPortV0,
) orquestacontext.ContextMaterializedBundleV0 {
	return orquestacontext.SanitizeMaterializedContextBundleV0(bundle, sanitizer)
}

func (sanitizer LocalSensitiveDataSanitizerV0) SanitizeContextEntryV0(
	request orquestacontext.ContextSanitizationRequestV0,
) orquestacontext.ContextSanitizationResultV0 {
	content := strings.TrimSpace(request.Content)
	categories := map[string]bool{}
	replacements := 0
	needsReview := localSensitiveDataSanitizerNeedsReviewV0(content)
	content = sanitizer.replaceAssignmentsV0(request, content, categories, &replacements)
	content = sanitizer.replacePatternV0(request, content, categories, &replacements, "credential_value", sensitiveNamePatternV0)
	content = sanitizer.replacePatternV0(request, content, categories, &replacements, "credential_value", bearerTokenPatternV0)
	content = sanitizer.replacePatternV0(request, content, categories, &replacements, "credential_value", secretPrefixPatternV0)
	content = sanitizer.replacePatternV0(request, content, categories, &replacements, "private_path", privatePathPatternV0)
	content = sanitizer.replacePatternV0(request, content, categories, &replacements, "url", urlPatternV0)
	status := orquestacontext.ContextSanitizationStatusCleanV0
	if replacements > 0 {
		status = orquestacontext.ContextSanitizationStatusSanitizedV0
	}
	reviewRequired := needsReview || localSensitiveDataSanitizerNeedsReviewV0(content)
	if reviewRequired {
		status = orquestacontext.ContextSanitizationStatusReviewRequiredV0
		categories["non_public_context"] = true
		content = ""
	}
	return orquestacontext.ContextSanitizationResultV0{
		Status:  status,
		Content: content,
		Evidence: sanitizer.evidenceV0(
			request,
			status,
			categories,
			replacements,
			reviewRequired,
		),
	}
}

func (sanitizer LocalSensitiveDataSanitizerV0) replaceAssignmentsV0(
	request orquestacontext.ContextSanitizationRequestV0,
	content string,
	categories map[string]bool,
	replacements *int,
) string {
	matches := sensitiveAssignmentPatternV0.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}
	var out strings.Builder
	cursor := 0
	for _, match := range matches {
		if match[0] < cursor {
			continue
		}
		valueEnd := sensitiveAssignmentValueEndV0(content, match[1])
		if valueEnd <= match[1] {
			continue
		}
		out.WriteString(content[cursor:match[0]])
		prefix := content[match[0]:match[1]]
		*replacements = *replacements + 1
		categories["credential_value"] = true
		ref := sanitizer.refV0(request, "value", *replacements)
		out.WriteString(sensitiveAssignmentReplacementV0(prefix, ref))
		cursor = valueEnd
	}
	out.WriteString(content[cursor:])
	return out.String()
}

func (sanitizer LocalSensitiveDataSanitizerV0) replacePatternV0(
	request orquestacontext.ContextSanitizationRequestV0,
	content string,
	categories map[string]bool,
	replacements *int,
	category string,
	pattern *regexp.Regexp,
) string {
	return pattern.ReplaceAllStringFunc(content, func(string) string {
		*replacements = *replacements + 1
		categories[category] = true
		return sanitizer.refV0(request, localSensitiveDataRefCategoryV0(category), *replacements)
	})
}

func (sanitizer LocalSensitiveDataSanitizerV0) evidenceV0(
	request orquestacontext.ContextSanitizationRequestV0,
	status orquestacontext.ContextSanitizationStatusV0,
	categories map[string]bool,
	replacements int,
	reviewRequired bool,
) orquestacontext.ContextSanitizationEvidenceV0 {
	return orquestacontext.ContextSanitizationEvidenceV0{
		SchemaVersion:    orquestacontext.ContextSanitizationEvidenceSchemaVersionV0,
		EvidenceRef:      "context-sanitization-evidence-" + safeContextSanitizerPartV0(request.EntryRef),
		BundleRef:        request.BundleRef,
		WorkOrderRef:     request.WorkOrderRef,
		TargetModule:     request.TargetModule,
		EntryRef:         localSensitiveDataSafeRefValueV0(request.EntryRef, "entry-ref"),
		SourceRef:        localSensitiveDataSafeRefValueV0(request.SourceRef, "source-ref"),
		SanitizerRef:     firstCodexStackStringV0(sanitizer.SanitizerRef, "sanitizer-ref-local-sensitive-data-v0"),
		Status:           status,
		Categories:       sortedSensitiveDataCategoriesV0(categories),
		ReplacementCount: replacements,
		ReviewRequired:   reviewRequired,
	}
}

func (sanitizer LocalSensitiveDataSanitizerV0) refV0(
	request orquestacontext.ContextSanitizationRequestV0,
	category string,
	index int,
) string {
	return fmt.Sprintf(
		"sanitized-ref-%s-%s-%02d",
		safeContextSanitizerPartV0(request.EntryRef),
		safeContextSanitizerPartV0(category),
		index,
	)
}

func localSensitiveDataSanitizerNeedsReviewV0(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "-----begin ") ||
		strings.Contains(lower, "private key-----") ||
		strings.Contains(lower, "prompt completo") ||
		strings.Contains(lower, "transcript completo")
}

func localSensitiveDataRefCategoryV0(category string) string {
	if category == "credential_value" {
		return "value"
	}
	return category
}

func sortedSensitiveDataCategoriesV0(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func localSensitiveDataSafeRefValueV0(value string, prefix string) string {
	value = strings.TrimSpace(value)
	if value == "" || !localSensitiveDataRefHasForbiddenDetailV0(value) {
		return value
	}
	return strings.Trim(prefix, "-") + "-" + safeContextSanitizerPartV0(value)
}

func safeContextSanitizerPartV0(value string) string {
	if localSensitiveDataRefHasForbiddenDetailV0(value) {
		return "context-" + localSensitiveDataStableHashV0(value)
	}
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "_", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "context"
	}
	return value
}

func localSensitiveDataRefHasForbiddenDetailV0(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"://", "/home/", "\\home\\", "/users/", "\\users\\", "c:\\users\\",
		"$home", "~/", "sk-", "bearer ", "access_token", "refresh_token",
		"access-token", "refresh-token", "api_key", "api-key", "secret",
		"password", "token=", "credential", "prompt", "completion", "transcript",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func localSensitiveDataStableHashV0(value string) string {
	var hash uint32 = 2166136261
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	const alphabet = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = alphabet[hash&0xf]
		hash >>= 4
	}
	return string(out)
}

func firstCodexStackStringV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sensitiveAssignmentValueEndV0(content string, start int) int {
	if start >= len(content) {
		return start
	}
	switch content[start] {
	case '"', '\'', '`':
		return sensitiveAssignmentQuotedValueEndV0(content, start)
	default:
		return sensitiveAssignmentBareValueEndV0(content, start)
	}
}

func sensitiveAssignmentQuotedValueEndV0(content string, start int) int {
	quote := content[start]
	escaped := false
	for index := start + 1; index < len(content); index++ {
		switch {
		case escaped:
			escaped = false
		case content[index] == '\\':
			escaped = true
		case content[index] == quote:
			return index + 1
		}
	}
	return len(content)
}

func sensitiveAssignmentBareValueEndV0(content string, start int) int {
	for index := start; index < len(content); index++ {
		switch content[index] {
		case '\n', '\r', ',', '}':
			return index
		}
	}
	return len(content)
}

func sensitiveAssignmentReplacementV0(prefix string, ref string) string {
	if strings.Contains(prefix, ":") {
		if strings.Contains(prefix, `"`) {
			return fmt.Sprintf(`"sanitized_ref":"%s"`, ref)
		}
		return "sanitized_ref: " + ref
	}
	return "sanitized_ref=" + ref
}

var (
	sensitiveAssignmentPatternV0 = regexp.MustCompile(`(?i)"?(access_token|refresh_token|api_key|secret|password|credential|token|prompt|completion|transcript)"?\s*[:=]\s*`)
	sensitiveNamePatternV0       = regexp.MustCompile(`(?i)access_token|refresh_token|api_key|secret|password|credential|prompt|transcript|completion`)
	bearerTokenPatternV0         = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._=-]{6,}`)
	secretPrefixPatternV0        = regexp.MustCompile(`sk-[A-Za-z0-9_-]{6,}`)
	privatePathPatternV0         = regexp.MustCompile(`(/home/|/Users/|/users/|[A-Za-z]:[\\/]+Users[\\/]+)[^"',\s}]+`)
	urlPatternV0                 = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^"',\s}]+`)
)
