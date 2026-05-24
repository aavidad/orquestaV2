package orquestaautoprogramming

import (
	"path/filepath"
	"regexp"
	"strings"
)

func autoprogrammingReviewGatePathAllowedByWriteSetV0(path string, writeSet []string) bool {
	path, ok := autoprogrammingReviewGateCleanPathV0(path, false)
	if !ok {
		return false
	}
	for _, raw := range writeSet {
		allowed, ok := autoprogrammingReviewGateCleanPathV0(raw, true)
		if !ok {
			continue
		}
		if allowed == "." || path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
		if autoprogrammingReviewGatePathGlobMatchesV0(allowed, path) {
			return true
		}
	}
	return false
}

func autoprogrammingReviewGateCleanPathV0(value string, allowDot bool) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" ||
		strings.Contains(value, "://") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$HOME") ||
		strings.ContainsAny(value, "\x00\r\n") ||
		filepath.IsAbs(value) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." {
		return cleaned, allowDot
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func autoprogrammingReviewGatePathGlobMatchesV0(pattern string, path string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		return false
	}
	re, err := autoprogrammingReviewGatePathGlobRegexpV0(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

func autoprogrammingReviewGatePathGlobRegexpV0(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++
			continue
		}
		switch pattern[i] {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
