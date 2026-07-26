package intake

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidAnswerText validates the reusable free-text value contract without
// inferring meaning from content. Wizard evaluator identities bind this source
// because their typed free-text validation delegates to this exact contract.
func ValidAnswerText(value string) bool {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) ||
		utf8.RuneCountInString(value) > MaxAnswerTextRunes {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			return false
		}
	}
	return true
}
