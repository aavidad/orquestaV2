package orquestaobservability

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func forbiddenOperationalTextCodeV0(value string) string {
	lower := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), `\/`, "/"))
	if lower == "" {
		return ""
	}
	if containsEffectiveSecretTextV0(lower) {
		return ErrSecretoDetectadoV0
	}
	if containsAssignedOperationalFragmentV0(lower, transcriptValueFragmentsV0) {
		return ErrTranscriptNoPermitidoV0
	}
	if containsAssignedOperationalFragmentV0(lower, promptCompletionValueFragmentsV0) ||
		containsCredentialDSNV0(lower) ||
		containsRealLocalPathV0(lower) {
		return ErrOperationalStatusQueryInvalidaV0
	}
	return ""
}

func forbiddenOperationalKeyCodeV0(key string) string {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" ||
		orquestarails.IsOperationalPrivacyMetadataRefNameV0(lower) ||
		operationalKeyIsPolicyMetadataV0(lower) {
		return ""
	}
	tokens := operationalKeyTokensV0(lower)
	if operationalKeyHasAnySequenceV0(tokens, secretKeySequencesV0) {
		return ErrSecretoDetectadoV0
	}
	if operationalKeyHasAnySequenceV0(tokens, transcriptKeySequencesV0) {
		return ErrTranscriptNoPermitidoV0
	}
	if operationalKeyHasAnySequenceV0(tokens, promptCompletionKeySequencesV0) ||
		operationalKeyHasAnySequenceV0(tokens, connectionKeySequencesV0) {
		return ErrOperationalStatusQueryInvalidaV0
	}
	return ""
}

var (
	secretValueFragmentsV0 = []string{
		"api_key=", "api-key=", "api_key:", "api-key:",
		"access_token=", "access-token=", "access_token:", "access-token:",
		"refresh_token=", "refresh-token=", "refresh_token:", "refresh-token:",
		"client_secret=", "client-secret=", "client_secret:", "client-secret:",
		"authorization:", "bearer ",
		"password=", "password:", "passwd=", "passwd:", "pwd=", "pwd:",
		"secret=", "secret:", "secreto=", "secreto:",
		"credential=", "credential:", "credencial=", "credencial:",
		"-----begin ",
	}
	transcriptValueFragmentsV0 = []string{
		"transcript=", "transcript:",
		"raw_transcript=", "raw_transcript:",
		"transcripcion=", "transcripcion:",
	}
	promptCompletionValueFragmentsV0 = []string{
		"prompt=", "prompt:",
		"raw_prompt=", "raw_prompt:",
		"completion=", "completion:",
		"raw_completion=", "raw_completion:",
		"raw_text=", "raw_text:",
		"full_text=", "full_text:",
	}
	secretKeySequencesV0 = [][]string{
		{"api", "key"},
		{"apikey"},
		{"access", "token"},
		{"refresh", "token"},
		{"client", "secret"},
		{"password"},
		{"passwd"},
		{"pwd"},
		{"credential"},
		{"credencial"},
		{"secret"},
		{"secreto"},
		{"oauth"},
		{"bearer"},
	}
	transcriptKeySequencesV0 = [][]string{
		{"transcript"},
		{"transcripcion"},
		{"raw", "transcript"},
	}
	promptCompletionKeySequencesV0 = [][]string{
		{"prompt"},
		{"raw", "prompt"},
		{"completion"},
		{"raw", "completion"},
		{"raw", "text"},
		{"full", "text"},
	}
	connectionKeySequencesV0 = [][]string{
		{"dsn"},
		{"database", "url"},
		{"connection", "string"},
		{"sql", "query"},
		{"raw", "sql"},
		{"table", "name"},
		{"tabla", "name"},
		{"home", "path"},
	}
)

func containsEffectiveSecretTextV0(lower string) bool {
	if containsAssignedOperationalFragmentV0(lower, secretValueFragmentsV0) {
		return true
	}
	for start := 0; ; {
		index := strings.Index(lower[start:], "sk-")
		if index < 0 {
			return false
		}
		absolute := start + index
		if hasOperationalFragmentBoundaryV0(lower, "sk-", absolute, absolute+3) &&
			hasLongSecretTokenAfterV0(lower[absolute+3:]) {
			return true
		}
		start = absolute + 3
	}
}

func containsAssignedOperationalFragmentV0(lower string, fragments []string) bool {
	for _, fragment := range fragments {
		start := 0
		for {
			index := strings.Index(lower[start:], fragment)
			if index < 0 {
				break
			}
			absolute := start + index
			end := absolute + len(fragment)
			if hasOperationalFragmentBoundaryV0(lower, fragment, absolute, end) &&
				assignedOperationalValuePresentV0(lower[end:]) {
				return true
			}
			start = end
		}
	}
	return false
}

func hasOperationalFragmentBoundaryV0(value string, fragment string, start int, end int) bool {
	before := true
	if len(fragment) > 0 && isASCIIAlnumObservabilityV0(fragment[0]) {
		before = start == 0 || !isASCIIAlnumObservabilityV0(value[start-1])
	}
	after := true
	if len(fragment) > 0 && isASCIIAlnumObservabilityV0(fragment[len(fragment)-1]) {
		after = end >= len(value) || !isASCIIAlnumObservabilityV0(value[end])
	}
	return before && after
}

func assignedOperationalValuePresentV0(value string) bool {
	trimmed := strings.TrimLeft(value, " \t\r\n\"'")
	if trimmed == "" {
		return false
	}
	redactedPrefixes := []string{"<redacted", "redacted", "[redacted", "***", "xxxxx"}
	for _, prefix := range redactedPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}
	return true
}

func hasLongSecretTokenAfterV0(value string) bool {
	count := 0
	for index := 0; index < len(value); index++ {
		ch := value[index]
		if !isASCIIAlnumObservabilityV0(ch) && ch != '_' && ch != '-' {
			break
		}
		count++
	}
	return count >= 12
}

func containsCredentialDSNV0(lower string) bool {
	for _, scheme := range []string{
		"postgres://",
		"postgresql://",
		"mysql://",
		"mongodb://",
		"redis://",
	} {
		start := 0
		for {
			index := strings.Index(lower[start:], scheme)
			if index < 0 {
				break
			}
			absolute := start + index + len(scheme)
			end := strings.IndexAny(lower[absolute:], "/?# \t\r\n\"'")
			authority := lower[absolute:]
			if end >= 0 {
				authority = lower[absolute : absolute+end]
			}
			if at := strings.LastIndex(authority, "@"); at > 0 && strings.Contains(authority[:at], ":") {
				return true
			}
			start = absolute
		}
	}
	return false
}

func containsRealLocalPathV0(lower string) bool {
	return containsLocalPathWithUserSegmentV0(lower, "/home/") ||
		containsLocalPathWithUserSegmentV0(lower, "/users/") ||
		containsLocalPathWithUserSegmentV0(lower, `c:\users\`)
}

func containsLocalPathWithUserSegmentV0(lower string, prefix string) bool {
	start := 0
	for {
		index := strings.Index(lower[start:], prefix)
		if index < 0 {
			return false
		}
		absolute := start + index + len(prefix)
		segmentEnd := strings.IndexAny(lower[absolute:], "/\\\t \"'")
		segment := lower[absolute:]
		if segmentEnd >= 0 {
			segment = lower[absolute : absolute+segmentEnd]
		}
		if segment != "" {
			return true
		}
		start = absolute
	}
}

func operationalKeyTokensV0(key string) []string {
	return strings.FieldsFunc(key, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ':'
	})
}

func operationalKeyIsPolicyMetadataV0(key string) bool {
	return strings.HasSuffix(key, "_policy") ||
		strings.HasSuffix(key, "_policies") ||
		strings.HasSuffix(key, "_redaction") ||
		strings.HasSuffix(key, "_classification") ||
		strings.HasSuffix(key, "_metadata")
}

func operationalKeyHasAnySequenceV0(tokens []string, sequences [][]string) bool {
	for _, sequence := range sequences {
		if operationalKeyHasSequenceV0(tokens, sequence) {
			return true
		}
	}
	return false
}

func operationalKeyHasSequenceV0(tokens []string, sequence []string) bool {
	if len(sequence) == 0 || len(sequence) > len(tokens) {
		return false
	}
	for index := 0; index <= len(tokens)-len(sequence); index++ {
		matches := true
		for offset, want := range sequence {
			if tokens[index+offset] != want {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func isASCIIAlnumObservabilityV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
