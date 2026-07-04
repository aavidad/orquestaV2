package orquestamcp

import "strings"

const MCPConfigProjectionMismatchV0 = "config_projection_mismatch"
const MCPConfigProjectionVerifiedEvidenceRefV0 = "evidence-ref-config-projection-required-settings-verified"

type MCPRequiredSettingV0 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type MCPConfigProjectionSettingV0 struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

func ValidateMCPRequiredSettingsProjectionV0(
	required []MCPRequiredSettingV0,
	projection []MCPConfigProjectionSettingV0,
) []MCPValidationIssueV0 {
	if len(required) == 0 {
		return nil
	}
	byKey := map[string]string{}
	for _, setting := range projection {
		key := strings.TrimSpace(setting.Key)
		if key == "" {
			continue
		}
		byKey[key] = strings.TrimSpace(setting.Value)
	}
	issues := make([]MCPValidationIssueV0, 0)
	for _, expected := range required {
		key := strings.TrimSpace(expected.Key)
		field := "required_settings"
		if key != "" {
			field += "." + key
		}
		if key == "" {
			issues = append(issues, MCPValidationIssueV0{
				Code:    MCPConfigProjectionMismatchV0,
				Field:   field,
				Message: "required_settings requiere key no vacia",
			})
			continue
		}
		actual, ok := byKey[key]
		if !ok || actual != strings.TrimSpace(expected.Value) {
			issues = append(issues, MCPValidationIssueV0{
				Code:    MCPConfigProjectionMismatchV0,
				Field:   field,
				Message: "configuracion efectiva no coincide con required_settings para " + key + " (expected_configured=true actual_configured=" + configProjectionBoolTextV0(ok) + ")",
			})
			continue
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return issues
}

func WithMCPConfigProjectionVerifiedEvidenceV0(
	evidenceRefs []string,
	required []MCPRequiredSettingV0,
) []string {
	if len(required) == 0 {
		return compactStringsMCPV0(evidenceRefs)
	}
	return compactStringsMCPV0(append(evidenceRefs, MCPConfigProjectionVerifiedEvidenceRefV0))
}

func configProjectionBoolTextV0(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
