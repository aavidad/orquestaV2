package orquestarails

import (
	"regexp"
	"strings"
)

var operationalRedactionPatternsV0 = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)"(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|passwd|pwd|secret|secreto|credential|credencial|authorization)"\s*:\s*"[^"]*"`), `"$1":"<redacted>"`},
	{regexp.MustCompile(`(?i)'(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|passwd|pwd|secret|secreto|credential|credencial|authorization)'\s*:\s*'[^']*'`), `'$1':'<redacted>'`},
	{regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|passwd|pwd|secret|secreto|credential|credencial)\s*[:=]\s*(?:"[^"]*"|'[^']*'|[^,\s"'\\]+)`), `$1=<redacted>`},
	{regexp.MustCompile(`(?i)\b(authorization)\s*:\s*bearer\s+[^,\s"'\\]+`), `$1: Bearer <redacted>`},
	{regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]{8,}`), `Bearer <redacted>`},
	{regexp.MustCompile(`(?i)\bsk-[a-z0-9][a-z0-9_-]{12,}\b`), `<secret-token-redacted>`},
	{regexp.MustCompile(`(?i)\b((?:postgres|postgresql|mysql|mongodb|redis)://[^:\s/@]+:)[^@\s"']+(@)`), `${1}<redacted>${2}`},
	{regexp.MustCompile(`(?i)\b(prompt|transcript|completion|raw_text|payload)\s*[:=]\s*[^,\n]+`), `$1=<redacted>`},
	{regexp.MustCompile(`(?i)/home/[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`(?i)/users/[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`(?i)[a-z]:\\users\\[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`~[/\\][^ \t\r\n"']+`), `<home-path-redacted>`},
}

// RedactOperationalTextForFieldV0 sanitizes effective secret values that should
// not cross an operational boundary. This is intentionally independent from
// soft rail enforcement: callers may keep agent flow alive while projecting a
// safer diagnostic/output value.
func RedactOperationalTextForFieldV0(boundary string, field string, value string) (string, bool) {
	redacted := value
	changed := false
	for _, item := range operationalRedactionPatternsV0 {
		next := item.pattern.ReplaceAllString(redacted, item.replacement)
		if next != redacted {
			changed = true
			redacted = next
		}
	}
	next, blockChanged := redactPrivateKeyBlocksV0(redacted)
	if blockChanged {
		changed = true
		redacted = next
	}
	return redacted, changed
}

func redactPrivateKeyBlocksV0(value string) (string, bool) {
	lines := strings.SplitAfter(value, "\n")
	changed := false
	inBlock := false
	for i, line := range lines {
		lower := strings.ToLower(line)
		if isPrivateMaterialBoundaryLineV0(lower, "-----begin ") {
			inBlock = true
			changed = true
			lines[i] = "<private-material-redacted>\n"
			continue
		}
		if inBlock {
			changed = true
			lines[i] = ""
			if isPrivateMaterialBoundaryLineV0(lower, "-----end ") {
				inBlock = false
			}
		}
	}
	return strings.Join(lines, ""), changed
}

func isPrivateMaterialBoundaryLineV0(lowerLine string, boundary string) bool {
	return strings.Contains(lowerLine, boundary) &&
		(strings.Contains(lowerLine, "private key") ||
			strings.Contains(lowerLine, "private material"))
}
