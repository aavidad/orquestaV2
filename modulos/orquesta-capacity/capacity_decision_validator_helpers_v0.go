package orquestacapacity

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func (v *capacityDecisionValidatorV0) requireOpaque(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrReferenciaNoOpacaV0, field)
		return
	}
	v.optionalOpaque(field, value)
}

func (v *capacityDecisionValidatorV0) optionalOpaque(field, value string) {
	if value == "" {
		return
	}
	switch {
	case looksLikeSecretV0(value):
		v.add(ErrSecretoDetectadoV0, field)
	case looksLikeHomePathV0(value):
		v.add(ErrRutaHomeRealDetectadaV0, field)
	case !opaqueRefPatternV0.MatchString(value), containsForbiddenBrandV0(value):
		v.add(ErrReferenciaNoOpacaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) optionalPhaseRef(field, value string) {
	if value == "" {
		return
	}
	if !phaseRefPatternV0.MatchString(value) || containsForbiddenBrandV0(value) || looksLikeSecretV0(value) {
		v.add(ErrReferenciaNoOpacaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) requireSafeText(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrCapacityDecisionInvalidaV0, field)
		return
	}
	v.optionalSafeText(field, value)
}

func (v *capacityDecisionValidatorV0) optionalSafeText(field, value string) {
	if value == "" {
		return
	}
	if utf8.RuneCountInString(value) > 320 ||
		(orquestarails.DetailProhibitedRailsEnabledV0() && forbiddenSafeTextPatternV0.MatchString(value)) {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) requireConst(field, got, want string, code CapacityDecisionErrorCodeV0) {
	if got != want {
		v.add(code, field)
	}
}

func (v *capacityDecisionValidatorV0) requireCapacityLevel(field, value string) {
	if !isOneOfV0(value, "low", "medium", "high", "xhigh") {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) requireReasoningEffort(field, value string) {
	if !isOneOfV0(value, "none", "low", "medium", "high", "xhigh") {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) requireFreshness(field, value string) {
	if !isOneOfV0(value, "fresh", "stale", "obsolete", "unknown") {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) requireQuotaScope(field, value string) {
	if !isOneOfV0(value, "pool", "account", "home", "model", "provider", "task", "unknown") {
		v.add(ErrCuotaNoDisponibleV0, field)
	}
}

func (v *capacityDecisionValidatorV0) validateStringSet(field string, values []string, allowed ...string) {
	seen := map[string]bool{}
	for i, value := range values {
		if !isOneOfV0(value, allowed...) || seen[value] {
			v.add(ErrCapacityDecisionInvalidaV0, fmt.Sprintf("%s[%d]", field, i))
		}
		seen[value] = true
	}
}

func (v *capacityDecisionValidatorV0) optionalNonNegativeInt(field string, value *int) {
	if value != nil && *value < 0 {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) optionalNonNegativeFloat(field string, value *float64) {
	if value != nil && *value < 0 {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) optionalConfidence(field string, value *float64) {
	if value != nil && (*value < 0 || *value > 1) {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) optionalUTCTime(field, value string) {
	if value == "" {
		return
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", value); err != nil || !utcInstantPatternV0.MatchString(value) {
		v.add(ErrCapacityDecisionInvalidaV0, field)
	}
}

func (v *capacityDecisionValidatorV0) add(code CapacityDecisionErrorCodeV0, field string) {
	v.issues = append(v.issues, CapacityDecisionIssueV0{Code: code, Field: field})
}

func poolContainsModelV0(pool PoolCapacidadV0, modelRef string) bool {
	for _, model := range pool.Modelos {
		if model.ModelRef == modelRef {
			return true
		}
	}
	return false
}

func hasFreshEscalationGateV0(bundle EvidenceBundleV0) bool {
	for _, signals := range [][]EvidenceSignalV0{bundle.QualitySignals, bundle.FailureSignals, bundle.HumanSignals} {
		for _, signal := range signals {
			if signal.Freshness == "fresh" && isOneOfV0(signal.Weight, "high", "critical") && signal.Source != "unavailable" {
				return true
			}
		}
	}
	return false
}

func isForcedDegradationOrHandoffV0(res CapacityDecisionResponseV0) bool {
	return isOneOfV0(res.Handoff, "preventivo", "requerido") ||
		isOneOfV0(res.Degradacion.Action, "bajar_nivel", "cambiar_modelo", "cambiar_home", "cambiar_pool", "handoff_preventivo", "bloqueado")
}

func windowInsufficientV0(ctx ContextoEstimadoV0, quota QuotaSnapshotV0) bool {
	if ctx.Status != "known" {
		return false
	}
	if ctx.EstimatedSeconds != nil && quota.RemainingSeconds != nil && *quota.RemainingSeconds < *ctx.EstimatedSeconds {
		return true
	}
	if ctx.EstimatedMessages != nil && quota.RemainingMessages != nil && *quota.RemainingMessages < *ctx.EstimatedMessages {
		return true
	}
	if ctx.EstimatedTokens != nil && quota.RemainingTokens != nil && *quota.RemainingTokens < *ctx.EstimatedTokens {
		return true
	}
	return false
}

func looksLikeHomePathV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	trimmed := strings.TrimSpace(value)
	low := strings.ToLower(trimmed)
	return strings.HasPrefix(trimmed, "/") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.Contains(trimmed, "/") ||
		strings.Contains(trimmed, "\\") ||
		strings.Contains(low, "$home") ||
		(len(trimmed) >= 2 && isASCIIAlphaV0(trimmed[0]) && trimmed[1] == ':')
}

func looksLikeSecretV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(low, "bearer") ||
		strings.HasPrefix(low, "sk-") ||
		strings.HasPrefix(low, "ghp_") ||
		strings.HasPrefix(low, "xoxb-") ||
		strings.HasPrefix(low, "xoxa-") ||
		strings.HasPrefix(low, "xoxp-") ||
		strings.HasPrefix(low, "ya29") ||
		strings.Contains(low, "access_token") ||
		strings.Contains(low, "refresh_token") ||
		strings.Contains(low, "api_key") ||
		strings.Contains(low, "secret=") ||
		jwtLikePatternV0.MatchString(value)
}

func containsForbiddenBrandV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if forbiddenBrandTokensV0[token] {
			return true
		}
	}
	return false
}

func containsV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func isOneOfV0(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func isASCIIAlphaV0(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

var (
	opaqueRefPatternV0         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
	phaseRefPatternV0          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$`)
	utcInstantPatternV0        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)
	jwtLikePatternV0           = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
	forbiddenSafeTextPatternV0 = regexp.MustCompile(`(?i)([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}|(^|\s)(/home/|/Users/|[A-Za-z]:\\|~|\$HOME)|(^|\s)(sk-[A-Za-z0-9]|ghp_[A-Za-z0-9]|xox[baprs]-|ya29\.)|[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+|(^|[^a-z0-9])(openai|anthropic|google|gemini|claude|gpt|openrouter|mistral|deepseek|ollama|azure|bedrock|vertex|groq|cohere)([^a-z0-9]|$))`)
	forbiddenBrandTokensV0     = map[string]bool{
		"openai": true, "anthropic": true, "google": true, "gemini": true, "claude": true,
		"gpt": true, "openrouter": true, "mistral": true, "deepseek": true, "ollama": true,
		"azure": true, "bedrock": true, "vertex": true, "groq": true, "cohere": true,
	}
)
