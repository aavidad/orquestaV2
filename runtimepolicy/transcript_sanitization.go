package runtimepolicy

import (
	"regexp"
	"strings"
	"unicode"
)

var runtimepolicyOSCTranscriptRegexp = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
var runtimepolicyAnsiTranscriptRegexp = regexp.MustCompile(`\x1b\[[0-9;?<]*[ -/]*[@-~]`)

const maxPendingTranscriptRunes = 512

func CompactPendingTranscript(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = runtimepolicyOSCTranscriptRegexp.ReplaceAllString(raw, "")
	raw = runtimepolicyAnsiTranscriptRegexp.ReplaceAllString(raw, "")
	raw = strings.Map(cleansePendingTranscriptControlRune, raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if countSemanticRunes(raw) == 0 {
		return ""
	}
	if len(raw) > maxPendingTranscriptRunes {
		raw = raw[len(raw)-maxPendingTranscriptRunes:]
	}
	return raw
}

func countSemanticRunes(raw string) int {
	count := 0
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			count++
		}
	}
	return count
}

func cleansePendingTranscriptControlRune(r rune) rune {
	switch {
	case r == '\n' || r == '\r':
		return ' '
	case r == '\t':
		return ' '
	case unicode.IsControl(r):
		return -1
	default:
		return r
	}
}
