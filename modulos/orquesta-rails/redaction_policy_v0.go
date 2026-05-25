package orquestarails

import (
	"regexp"
	"strings"
)

var operationalRedactionPatternsV0 = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|passwd|pwd|secret|secreto|credential|credencial)\s*[:=]\s*[^,\s"'\\]+`), `$1=<redacted>`},
	{regexp.MustCompile(`(?i)\b(authorization)\s*:\s*bearer\s+[^,\s"'\\]+`), `$1: Bearer <redacted>`},
	{regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]{8,}`), `Bearer <redacted>`},
	{regexp.MustCompile(`(?i)\b(prompt|transcript|completion|raw_text|payload)\s*[:=]\s*[^,\n]+`), `$1=<redacted>`},
	{regexp.MustCompile(`(?i)"(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|secret|credential|authorization|prompt|transcript|completion|raw_text|payload)"\s*:\s*"[^"]*"`), `"$1":"<redacted>"`},
	{regexp.MustCompile(`(?i)'(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|secret|credential|authorization|prompt|transcript|completion|raw_text|payload)'\s*:\s*'[^']*'`), `'$1':'<redacted>'`},
	{regexp.MustCompile(`(?i)/home/[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`(?i)/users/[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`(?i)[a-z]:\\users\\[^ \t\r\n"']+`), `<home-path-redacted>`},
	{regexp.MustCompile(`~[/\\][^ \t\r\n"']+`), `<home-path-redacted>`},
}

// RedactOperationalTextForFieldV0 removes values that should not cross an
// operational boundary while preserving enough shape for local diagnosis.
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
		if strings.Contains(lower, "-----begin ") {
			inBlock = true
			changed = true
			lines[i] = "<private-material-redacted>\n"
			continue
		}
		if inBlock {
			changed = true
			lines[i] = ""
			if strings.Contains(lower, "-----end ") {
				inBlock = false
			}
		}
	}
	return strings.Join(lines, ""), changed
}
