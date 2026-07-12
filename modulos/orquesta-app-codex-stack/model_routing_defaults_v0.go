package orquestaappcodexstack

import (
	"reflect"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

const (
	codexModelRefLunaV0  = "codex-model-ref-luna-v0"
	codexModelRefTerraV0 = "codex-model-ref-terra-v0"
	codexModelRefSolV0   = "codex-model-ref-sol-v0"

	claudeModelRefHaikuV0  = "claude-model-ref-haiku-v0"
	claudeModelRefSonnetV0 = "claude-model-ref-sonnet-v0"
	claudeModelRefFableV0  = "claude-model-ref-fable-v0"
)

// DefaultCodexModelRoutingConfigV0 materializes the strict composition policy
// for legacy callers that did not yet declare ModelRouting.
func DefaultCodexModelRoutingConfigV0() CodexModelRoutingConfigV0 {
	return CodexModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: "codex-model-routing-policy-v0", Strict: true,
			TrivialModelRef: codexModelRefLunaV0, NormalModelRef: codexModelRefTerraV0, CriticalModelRef: codexModelRefSolV0,
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: map[string]string{
			codexModelRefLunaV0: "gpt-5.6-luna", codexModelRefTerraV0: "gpt-5.6-terra", codexModelRefSolV0: "gpt-5.6-sol",
		},
		TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{},
	}
}

// DefaultClaudeModelRoutingConfigV0 materializes the strict composition policy
// for legacy callers that did not yet declare ModelRouting.
func DefaultClaudeModelRoutingConfigV0() ClaudeModelRoutingConfigV0 {
	return ClaudeModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: "claude-model-routing-policy-v0", Strict: true,
			TrivialModelRef: claudeModelRefHaikuV0, NormalModelRef: claudeModelRefSonnetV0, CriticalModelRef: claudeModelRefFableV0,
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: map[string]string{
			claudeModelRefHaikuV0: "haiku-4.5", claudeModelRefSonnetV0: "sonnet-5", claudeModelRefFableV0: "fable-5",
		},
		TaskRoutes: map[string]orquestacapacity.ModelRoutingRequestV0{},
	}
}

func normalizeCodexModelRoutingConfigV0(config CodexModelRoutingConfigV0) CodexModelRoutingConfigV0 {
	if reflect.ValueOf(config).IsZero() {
		return DefaultCodexModelRoutingConfigV0()
	}
	return config
}

func normalizeClaudeModelRoutingConfigV0(config ClaudeModelRoutingConfigV0) ClaudeModelRoutingConfigV0 {
	if reflect.ValueOf(config).IsZero() {
		return DefaultClaudeModelRoutingConfigV0()
	}
	return config
}

func normalizeModelRoutingConfigV0(config ConfigV0) ConfigV0 {
	config.Codex.ModelRouting = normalizeCodexModelRoutingConfigV0(config.Codex.ModelRouting)
	config.Claude.ModelRouting = normalizeClaudeModelRoutingConfigV0(config.Claude.ModelRouting)
	return config
}
