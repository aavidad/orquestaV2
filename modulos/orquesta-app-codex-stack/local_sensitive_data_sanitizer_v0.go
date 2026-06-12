package orquestaappcodexstack

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

type LocalSensitiveDataSanitizerV0 struct {
	SanitizerRef string
	Egress       EgressSanitizerConfigV0
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
	content = sanitizer.replaceURLsV0(request, content, categories, &replacements)
	categories = sanitizer.egressCategoriesV0(categories)
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
		if localSensitiveDataIndexInsideURLV0(content, match[0]) {
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
	matches := pattern.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}
	var out strings.Builder
	cursor := 0
	for _, match := range matches {
		if match[0] < cursor {
			continue
		}
		if localSensitiveDataIndexInsideURLV0(content, match[0]) {
			continue
		}
		out.WriteString(content[cursor:match[0]])
		*replacements = *replacements + 1
		categories[category] = true
		out.WriteString(sanitizer.refV0(request, localSensitiveDataRefCategoryV0(category), *replacements))
		cursor = match[1]
	}
	out.WriteString(content[cursor:])
	return out.String()
}

func (sanitizer LocalSensitiveDataSanitizerV0) replaceURLsV0(
	request orquestacontext.ContextSanitizationRequestV0,
	content string,
	categories map[string]bool,
	replacements *int,
) string {
	matches := urlPatternV0.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}
	var out strings.Builder
	cursor := 0
	for _, match := range matches {
		if match[0] < cursor {
			continue
		}
		out.WriteString(content[cursor:match[0]])
		out.WriteString(sanitizer.sanitizeURLV0(
			request,
			content[match[0]:match[1]],
			categories,
			replacements,
		))
		cursor = match[1]
	}
	out.WriteString(content[cursor:])
	return out.String()
}

func (sanitizer LocalSensitiveDataSanitizerV0) sanitizeURLV0(
	request orquestacontext.ContextSanitizationRequestV0,
	raw string,
	categories map[string]bool,
	replacements *int,
) string {
	candidate, suffix := splitURLTrailingPunctuationV0(raw)
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" {
		return raw
	}
	if localSensitiveDataURLRequiresRedactionV0(parsed) {
		*replacements = *replacements + 1
		categories["url"] = true
		return sanitizer.refV0(request, "url", *replacements) + suffix
	}
	publicURL := publicURLWithoutQueryOrCredentialsV0(parsed)
	if publicURL != candidate {
		*replacements = *replacements + 1
		categories["url"] = true
		return publicURL + suffix
	}
	return raw
}

func splitURLTrailingPunctuationV0(value string) (string, string) {
	candidate := value
	suffix := ""
	for len(candidate) > 0 {
		switch candidate[len(candidate)-1] {
		case '.', ';', '!', ')', ']':
			suffix = candidate[len(candidate)-1:] + suffix
			candidate = candidate[:len(candidate)-1]
		default:
			return candidate, suffix
		}
	}
	return value, ""
}

func localSensitiveDataURLRequiresRedactionV0(parsed *url.URL) bool {
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return true
	}
	if parsed.User != nil {
		return true
	}
	if localSensitiveDataURLHostIsPrivateV0(parsed.Hostname()) {
		return true
	}
	if localSensitiveDataURLQueryIsSensitiveV0(parsed.RawQuery) {
		return true
	}
	if localSensitiveDataURLValueLooksSensitiveV0(parsed.Fragment) {
		return true
	}
	return false
}

func localSensitiveDataURLHostIsPrivateV0(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	host = strings.TrimSuffix(host, ".")
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() ||
			ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified()
	}
	if !strings.Contains(host, ".") {
		return true
	}
	return false
}

func localSensitiveDataURLQueryIsSensitiveV0(rawQuery string) bool {
	if rawQuery == "" {
		return false
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return localSensitiveDataURLValueLooksSensitiveV0(rawQuery)
	}
	for key, vals := range values {
		if localSensitiveDataURLQueryKeyIsSensitiveV0(key) {
			return true
		}
		for _, value := range vals {
			if localSensitiveDataURLValueLooksSensitiveV0(value) {
				return true
			}
		}
	}
	return localSensitiveDataURLValueLooksSensitiveV0(rawQuery)
}

func localSensitiveDataURLQueryKeyIsSensitiveV0(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.NewReplacer("-", "_", ".", "_").Replace(key)
	for _, marker := range []string{
		"access_token",
		"refresh_token",
		"id_token",
		"api_key",
		"apikey",
		"client_secret",
		"secret",
		"password",
		"passwd",
		"credential",
		"authorization",
		"bearer",
		"token",
		"jwt",
		"session",
		"sessionid",
		"x_amz_signature",
		"x_amz_credential",
		"x_amz_security_token",
		"signature",
		"sig",
		"signed",
	} {
		if key == marker || strings.Contains(key, "_"+marker) || strings.Contains(key, marker+"_") {
			return true
		}
	}
	return false
}

func localSensitiveDataURLValueLooksSensitiveV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, marker := range []string{
		"bearer ",
		"sk-",
		"access_token",
		"refresh_token",
		"id_token",
		"api_key",
		"client_secret",
		"password=",
		"token=",
		"credential",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func publicURLWithoutQueryOrCredentialsV0(parsed *url.URL) string {
	publicURL := *parsed
	publicURL.User = nil
	publicURL.RawQuery = ""
	publicURL.ForceQuery = false
	publicURL.Fragment = ""
	publicURL.RawFragment = ""
	return publicURL.String()
}

func localSensitiveDataIndexInsideURLV0(content string, index int) bool {
	if index <= 0 || index > len(content) {
		return false
	}
	prefix := content[:index]
	schemeSeparator := strings.LastIndex(prefix, "://")
	if schemeSeparator < 1 {
		return false
	}
	schemeStart := schemeSeparator - 1
	for schemeStart >= 0 && localSensitiveDataURLSchemeByteV0(content[schemeStart]) {
		schemeStart--
	}
	schemeStart++
	if schemeStart >= schemeSeparator {
		return false
	}
	for cursor := schemeSeparator + len("://"); cursor < index; cursor++ {
		switch content[cursor] {
		case '"', '\'', '`', ' ', '\t', '\n', '\r', ',', '}':
			return false
		}
	}
	return true
}

func localSensitiveDataURLSchemeByteV0(value byte) bool {
	return (value >= 'A' && value <= 'Z') ||
		(value >= 'a' && value <= 'z') ||
		(value >= '0' && value <= '9') ||
		value == '+' ||
		value == '-' ||
		value == '.'
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
		strings.Contains(lower, "private key") ||
		strings.Contains(lower, "private_material") ||
		strings.Contains(lower, "private-material") ||
		strings.Contains(lower, "private-material-redacted") ||
		strings.Contains(lower, "non_public_context_review_required") ||
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
