package orquestaappcodexstack

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const domainWorkStructuredNonTerminalArtifactIssueCodeV0 = "domain-work-artifact-structured-non-terminal"

func domainWorkStructuredNonTerminalArtifactIssueRefV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) string {
	if domainWorkStructuredNonTerminalFieldsV0(fields) {
		return domainWorkStructuredNonTerminalArtifactIssueCodeV0
	}
	return ""
}

func domainWorkStructuredNonTerminalFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) bool {
	for _, field := range fields {
		name := normalizeDomainWorkDeliveryAliasV0(field.Name)
		if name == "" {
			continue
		}
		if domainWorkStructuredBlockerFieldV0(name, field) ||
			domainWorkStructuredFalseTerminalBoolFieldV0(name, field) ||
			domainWorkStructuredNonTerminalStatusFieldV0(name, field) ||
			domainWorkStructuredNonTerminalJSONFieldV0(name, field.ValueJSON) {
			return true
		}
	}
	return false
}

func domainWorkStructuredBlockerFieldV0(
	name string,
	field orquestadomainwork.DomainWorkFieldV0,
) bool {
	switch name {
	case "public_blocker", "blocker", "blocking_issue", "blocking_issues",
		"pending", "pending_actions", "required_external_state_change":
		return domainWorkFieldHasStructuredValueV0(field)
	default:
		return false
	}
}

func domainWorkStructuredFalseTerminalBoolFieldV0(
	name string,
	field orquestadomainwork.DomainWorkFieldV0,
) bool {
	switch name {
	case "ready", "artifact_ready", "delivery_ready", "package_ready",
		"registry_update_applied", "validation_passed", "valid":
		value, ok := domainWorkStructuredBoolValueV0(field)
		return ok && !value
	default:
		return false
	}
}

func domainWorkStructuredNonTerminalStatusFieldV0(
	name string,
	field orquestadomainwork.DomainWorkFieldV0,
) bool {
	switch name {
	case "status", "validation_status", "result", "resultado", "registry_update_status":
		return domainWorkStructuredNonTerminalStatusV0(domainWorkStructuredStringValueV0(field))
	default:
		return false
	}
}

func domainWorkStructuredNonTerminalStatusV0(value string) bool {
	switch normalizeDomainWorkDeliveryAliasV0(value) {
	case "blocked", "blocked_public_pending_official_connector", "blocked_pending_external_state",
		"failed", "failure", "invalid", "incomplete", "needs_rework", "pending",
		"pending_external_state", "pending_official_connector", "rework":
		return true
	default:
		return false
	}
}

func domainWorkStructuredNonTerminalJSONFieldV0(
	name string,
	raw json.RawMessage,
) bool {
	if len(raw) == 0 || !json.Valid(raw) {
		return false
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	return domainWorkStructuredNonTerminalJSONValueV0(name, decoded)
}

func domainWorkStructuredNonTerminalJSONValueV0(
	name string,
	value any,
) bool {
	switch typed := value.(type) {
	case map[string]any:
		if domainWorkStructuredBlockerJSONFieldV0(name, typed) {
			return true
		}
		for key, item := range typed {
			if domainWorkStructuredNonTerminalJSONValueV0(normalizeDomainWorkDeliveryAliasV0(key), item) {
				return true
			}
		}
	case []any:
		if domainWorkStructuredBlockerJSONListV0(name, typed) {
			return true
		}
		for _, item := range typed {
			if domainWorkStructuredNonTerminalJSONValueV0("", item) {
				return true
			}
		}
	case bool:
		return domainWorkStructuredFalseTerminalBoolNameV0(name) && !typed
	case string:
		return domainWorkStructuredNonTerminalStatusNameV0(name) &&
			domainWorkStructuredNonTerminalStatusV0(typed)
	}
	return false
}

func domainWorkStructuredBlockerJSONFieldV0(
	name string,
	value map[string]any,
) bool {
	if !domainWorkStructuredBlockerNameV0(name) {
		return false
	}
	return len(value) > 0
}

func domainWorkStructuredBlockerJSONListV0(
	name string,
	value []any,
) bool {
	if !domainWorkStructuredBlockerNameV0(name) {
		return false
	}
	return len(value) > 0
}

func domainWorkStructuredBlockerNameV0(name string) bool {
	switch name {
	case "public_blocker", "blocker", "blocking_issue", "blocking_issues",
		"pending", "pending_actions", "required_external_state_change":
		return true
	default:
		return false
	}
}

func domainWorkStructuredFalseTerminalBoolNameV0(name string) bool {
	switch name {
	case "ready", "artifact_ready", "delivery_ready", "package_ready",
		"registry_update_applied", "validation_passed", "valid":
		return true
	default:
		return false
	}
}

func domainWorkStructuredNonTerminalStatusNameV0(name string) bool {
	switch name {
	case "status", "validation_status", "result", "resultado", "registry_update_status":
		return true
	default:
		return false
	}
}

func domainWorkStructuredBoolValueV0(
	field orquestadomainwork.DomainWorkFieldV0,
) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(field.Value)) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	if len(field.ValueJSON) > 0 && json.Valid(field.ValueJSON) {
		var value bool
		if err := json.Unmarshal(field.ValueJSON, &value); err == nil {
			return value, true
		}
	}
	return false, false
}

func domainWorkStructuredStringValueV0(
	field orquestadomainwork.DomainWorkFieldV0,
) string {
	if value := strings.TrimSpace(field.Value); value != "" {
		return value
	}
	if len(field.ValueJSON) > 0 && json.Valid(field.ValueJSON) {
		var value string
		if err := json.Unmarshal(field.ValueJSON, &value); err == nil {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func domainWorkFieldHasStructuredValueV0(
	field orquestadomainwork.DomainWorkFieldV0,
) bool {
	if strings.TrimSpace(field.Value) != "" || len(field.Values) > 0 {
		return true
	}
	if len(field.ValueJSON) == 0 || !json.Valid(field.ValueJSON) {
		return false
	}
	var decoded any
	if err := json.Unmarshal(field.ValueJSON, &decoded); err != nil {
		return false
	}
	switch value := decoded.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return true
	}
}
