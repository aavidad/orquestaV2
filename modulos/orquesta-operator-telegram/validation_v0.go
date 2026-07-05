package orquestaoperatortelegram

import (
	"strings"
	"unicode"
)

func ValidateConfigV0(config ConfigV0) []IssueV0 {
	if !config.Enabled {
		return nil
	}
	var issues []IssueV0
	if strings.TrimSpace(config.BotLinkRef) == "" {
		issues = append(issues, IssueV0{Code: ErrTelegramLinkMissingV0, Field: "bot_link_ref"})
	}
	if !config.TokenConfigured {
		issues = append(issues, IssueV0{Code: ErrTelegramLinkMissingV0, Field: "token"})
	}
	if len(normalizeChatRefsV0(config.AuthorizedChatRefs)) == 0 {
		issues = append(issues, IssueV0{Code: ErrTelegramLinkMissingV0, Field: "authorized_chat_refs"})
	}
	return issues
}

func ChatAuthorizedV0(config ConfigV0, chatRef string) bool {
	chatRef = normalizeRefV0(chatRef)
	if chatRef == "" {
		return false
	}
	for _, allowed := range normalizeChatRefsV0(config.AuthorizedChatRefs) {
		if allowed == chatRef {
			return true
		}
	}
	return false
}

func RedactTelegramSecretV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "telegram-secret-configured"
}

func normalizeChatRefsV0(refs []string) []string {
	out := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		value := normalizeRefV0(ref)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeRefV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r)
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
