package application

import "testing"

func TestSuccessfulAgentEnvironmentEffectsNeverAliasControlStop(t *testing.T) {
	tests := []struct {
		effect EffectKind
		action ActionKind
	}{
		{EffectKindAgentQuiesce, ActionQuiesceAgent},
		{EffectKindAgentPreserve, ActionPreserveAgentEnvironment},
		{EffectKindAgentClose, ActionCloseAgentEnvironment},
	}
	for _, testCase := range tests {
		if !validEffectKind(testCase.effect) || !effectActionKindMatches(testCase.effect, testCase.action) {
			t.Fatalf("successful lifecycle pair rejected: %s/%s", testCase.effect, testCase.action)
		}
		if effectActionKindMatches(testCase.effect, ActionStopAgent) ||
			effectActionKindMatches(EffectKindAgentStop, testCase.action) {
			t.Fatalf("successful lifecycle aliases control stop: %s/%s", testCase.effect, testCase.action)
		}
	}
}
