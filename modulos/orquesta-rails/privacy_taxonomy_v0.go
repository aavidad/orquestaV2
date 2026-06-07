package orquestarails

import "strings"

const (
	OperationalPrivacyTaxonomyOwnerV0   = "orquesta-rails"
	OperationalPrivacyRedactionPolicyV0 = "privacy-redaction-policy-ref-orquesta-rails-v0"

	OperationalPrivacyRedactionNoneV0         = "none"
	OperationalPrivacyRedactionMetadataOnlyV0 = "metadata_only"
	OperationalPrivacyRedactionSummarizedV0   = "summarized"
	OperationalPrivacyRedactionRedactedV0     = "redacted"
)

type OperationalPrivacyFindingV0 struct {
	ContainsSecret           bool
	ContainsTranscript       bool
	ContainsPrompt           bool
	ContainsCompletion       bool
	ContainsConnectionDetail bool
	RedactionLevel           string
}

func OperationalPrivacyRedactionLevelsV0() []string {
	return []string{
		OperationalPrivacyRedactionNoneV0,
		OperationalPrivacyRedactionMetadataOnlyV0,
		OperationalPrivacyRedactionSummarizedV0,
		OperationalPrivacyRedactionRedactedV0,
	}
}

func OperationalPrivacyDefaultProjectionRedactionLevelV0() string {
	if !RailsEnforcedV0() {
		return OperationalPrivacyRedactionNoneV0
	}
	return OperationalPrivacyRedactionMetadataOnlyV0
}

func IsOperationalPrivacyRedactionLevelV0(value string) bool {
	trimmed := strings.TrimSpace(value)
	for _, allowed := range OperationalPrivacyRedactionLevelsV0() {
		if trimmed == allowed {
			return true
		}
	}
	return false
}

func IsOperationalPrivacyMetadataRefNameV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	if strings.HasSuffix(lower, "_ref") || strings.HasSuffix(lower, "_refs") {
		return true
	}
	if strings.HasSuffix(lower, "_policy_ref") || strings.HasSuffix(lower, "_redaction_ref") {
		return true
	}
	return strings.Contains(lower, "_policy_ref_") || strings.Contains(lower, "_redaction_ref_")
}

func ClassifyOperationalPrivacyTextV0(value string) OperationalPrivacyFindingV0 {
	if !RailsEnforcedV0() {
		return OperationalPrivacyFindingV0{RedactionLevel: OperationalPrivacyRedactionNoneV0}
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" || IsOperationalPrivacyMetadataRefNameV0(lower) {
		return OperationalPrivacyFindingV0{RedactionLevel: OperationalPrivacyRedactionNoneV0}
	}
	finding := OperationalPrivacyFindingV0{RedactionLevel: OperationalPrivacyRedactionNoneV0}
	if containsAnyPrivacyPartV0(lower, []string{
		"access_token", "refresh_token", "client_secret", "api_key", "apikey",
		"password", "passwd", "pwd", "credential", "credencial", "secret",
		"secreto", "oauth", "bearer",
	}) || lower == "token" || strings.HasSuffix(lower, "_token") {
		finding.ContainsSecret = true
		finding.RedactionLevel = OperationalPrivacyRedactionRedactedV0
		return finding
	}
	if containsAnyPrivacyPartV0(lower, []string{"transcript", "transcripcion"}) {
		finding.ContainsTranscript = true
		finding.RedactionLevel = OperationalPrivacyRedactionSummarizedV0
		return finding
	}
	if containsAnyPrivacyPartV0(lower, []string{"prompt", "raw_text", "full_text"}) {
		finding.ContainsPrompt = true
		finding.RedactionLevel = OperationalPrivacyRedactionSummarizedV0
		return finding
	}
	if strings.Contains(lower, "completion") {
		finding.ContainsCompletion = true
		finding.RedactionLevel = OperationalPrivacyRedactionSummarizedV0
		return finding
	}
	if lower == "home" || containsAnyPrivacyPartV0(lower, []string{
		"sql", "select", "insert", "update", "delete", "drop", "dsn",
		"connection", "conexion", "table", "tabla", "/home/", "home=",
		"$home", "~/", "home_path", "payload",
	}) {
		finding.ContainsConnectionDetail = true
		finding.RedactionLevel = OperationalPrivacyRedactionMetadataOnlyV0
		return finding
	}
	return finding
}

func containsAnyPrivacyPartV0(value string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
