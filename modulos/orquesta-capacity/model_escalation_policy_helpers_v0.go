package orquestacapacity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

func detectForbiddenModelEscalationShapeV0(data []byte) []ModelEscalationPolicyIssueV0 {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	var issues []ModelEscalationPolicyIssueV0
	scanForbiddenModelEscalationKeysV0("", value, &issues)
	return issues
}

func scanForbiddenModelEscalationKeysV0(path string, value any, issues *[]ModelEscalationPolicyIssueV0) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			field := joinModelEscalationPathV0(path, key)
			if isForbiddenModelEscalationKeyV0(key) {
				*issues = append(*issues, ModelEscalationPolicyIssueV0{
					Code:  ErrModelEscalationPolicyTablaRolModeloV0,
					Field: field,
				})
			}
			scanForbiddenModelEscalationKeysV0(field, child, issues)
		}
	case []any:
		for i, child := range node {
			scanForbiddenModelEscalationKeysV0(fmt.Sprintf("%s[%d]", path, i), child, issues)
		}
	}
}

func joinModelEscalationPathV0(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func isForbiddenModelEscalationKeyV0(key string) bool {
	switch strings.ToLower(key) {
	case "role_model", "role_model_table", "role_models", "role_to_model", "roles_to_models":
		return true
	default:
		return false
	}
}

func (v *modelEscalationPolicyValidatorV0) requireOpaque(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
		return
	}
	v.optionalOpaque(field, value)
}

func (v *modelEscalationPolicyValidatorV0) optionalOpaque(field, value string) {
	if value == "" {
		return
	}
	switch {
	case looksLikeSecretV0(value):
		v.add(ErrModelEscalationPolicySecretoDetectadoV0, field)
	case looksLikeHomePathV0(value):
		v.add(ErrModelEscalationPolicyRutaHomeRealV0, field)
	case !opaqueRefPatternV0.MatchString(value), containsForbiddenBrandV0(value):
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireVersionedPolicyRef(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
		return
	}
	switch {
	case looksLikeSecretV0(value):
		v.add(ErrModelEscalationPolicySecretoDetectadoV0, field)
	case looksLikeHomePathV0(value):
		v.add(ErrModelEscalationPolicyRutaHomeRealV0, field)
	case !versionedPolicyRefPatternV0.MatchString(value), containsForbiddenBrandV0(value):
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requirePhaseRef(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
		return
	}
	if !phaseRefPatternV0.MatchString(value) || containsForbiddenBrandV0(value) || looksLikeSecretV0(value) || looksLikeHomePathV0(value) {
		v.add(ErrModelEscalationPolicyReferenciaNoOpacaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireSafeText(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
		return
	}
	v.optionalSafeText(field, value)
}

func (v *modelEscalationPolicyValidatorV0) optionalSafeText(field, value string) {
	if value == "" {
		return
	}
	if utf8.RuneCountInString(value) > 320 || forbiddenSafeTextPatternV0.MatchString(value) {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireConst(field, got, want string, code ModelEscalationPolicyErrorCodeV0) {
	if got != want {
		v.add(code, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireCapacityLevel(field, value string) {
	if !isOneOfV0(value, "low", "medium", "high", "xhigh") {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireFreshness(field, value string) {
	if !isOneOfV0(value, "fresh", "stale", "obsolete", "unknown") {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireQuotaScope(field, value string) {
	if !isOneOfV0(value, "pool", "account", "home", "model", "provider", "unknown") {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) requireCount(field string, got, min, max int) {
	if got < min || got > max {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateLevelSet(field string, values []string) {
	v.validateStringSet(field, values, 1, 4, "low", "medium", "high", "xhigh")
}

func (v *modelEscalationPolicyValidatorV0) validateActionSet(field string, values []string, min, max int) {
	v.validateStringSet(field, values, min, max, "bajar_effort", "cambiar_modelo", "cambiar_home", "cambiar_pool", "handoff_preventivo", "bloquear")
}

func (v *modelEscalationPolicyValidatorV0) validateStringSet(field string, values []string, min, max int, allowed ...string) {
	code := ErrModelEscalationPolicyInvalidaV0
	if strings.HasPrefix(field, "xhigh_gate") {
		code = ErrModelEscalationPolicyXHighGateV0
	}
	if len(values) < min || len(values) > max {
		v.add(code, field)
	}
	seen := map[string]bool{}
	for i, value := range values {
		if !isOneOfV0(value, allowed...) || seen[value] {
			v.add(code, fmt.Sprintf("%s[%d]", field, i))
		}
		seen[value] = true
	}
}

func (v *modelEscalationPolicyValidatorV0) validateTriggers(field string, values []string) {
	v.validateStringSet(field, values, 0, 12, "ambiguedad_material", "riesgo_alto", "calidad_critica", "fallo_pruebas", "revision_fallida", "impacto_contractual", "ventana_insuficiente", "solicitud_humana")
}

func (v *modelEscalationPolicyValidatorV0) optionalConfidence(field string, value *float64) {
	if value != nil && (*value < 0 || *value > 1) {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) optionalNonNegativeInt(field string, value *int) {
	if value != nil && *value < 0 {
		v.add(ErrModelEscalationPolicyInvalidaV0, field)
	}
}

func (v *modelEscalationPolicyValidatorV0) validateLocalityInvariant(field string, rule ModelEscalationLocalityRuleV0) {
	if rule.Locality == "local" && isOneOfV0(rule.Effect, "eligible", "preferred") {
		if !rule.RequiresRuntimeAvailable {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".requires_runtime_available")
		}
		if rule.MinimumConfidence == nil || *rule.MinimumConfidence < 0.6 {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".minimum_confidence")
		}
		if rule.MinimumSamples == nil || *rule.MinimumSamples < 1 {
			v.add(ErrModelEscalationPolicyInvalidaV0, field+".minimum_samples")
		}
	}
	if isOneOfV0(rule.Locality, "remote", "hybrid") && rule.RequiresQuotaFreshness == "not_applicable" {
		v.add(ErrModelEscalationPolicyInvalidaV0, field+".requires_quota_freshness")
	}
}

func (v *modelEscalationPolicyValidatorV0) add(code ModelEscalationPolicyErrorCodeV0, field string) {
	v.issues = append(v.issues, ModelEscalationPolicyIssueV0{Code: code, Field: field})
}

func policyAllowsXHighV0(policy ModelEscalationPolicyV0) bool {
	for _, rule := range policy.PhaseRules {
		if containsV0(rule.AllowedLevels, "xhigh") {
			return true
		}
	}
	for _, rule := range policy.RiskQualityRules {
		if containsV0(rule.AllowedLevels, "xhigh") || rule.Adjustment == "allow_xhigh_with_gate" {
			return true
		}
	}
	return false
}

func policyHasFreshRaiseEvidenceV0(rules []ModelEscalationEvidenceRuleV0) bool {
	for _, rule := range rules {
		if rule.Effect == "raise" && rule.Freshness == "fresh" && isOneOfV0(rule.Weight, "high", "critical") &&
			isOneOfV0(rule.SignalKind, "quality", "failure", "human") && rule.Source != "unavailable" {
			return true
		}
	}
	return false
}

var versionedPolicyRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*:v[0-9]+$`)
