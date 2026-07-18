package governance_test

import (
	"testing"

	"orquesta/internal/governance"
)

func TestSecurityCriticalityAndReasoningEffortAreIndependent(t *testing.T) {
	criticalities := []governance.SecurityCriticality{
		governance.SecurityCriticalityNormal,
		governance.SecurityCriticalitySensitive,
		governance.SecurityCriticalityCritical,
	}
	efforts := []governance.ReasoningEffort{
		governance.ReasoningEffortLow,
		governance.ReasoningEffortMedium,
		governance.ReasoningEffortHigh,
		governance.ReasoningEffortXHigh,
	}
	for _, criticality := range criticalities {
		if err := governance.ValidateSecurityCriticality(criticality); err != nil {
			t.Fatal(err)
		}
		for _, effort := range efforts {
			if err := governance.ValidateReasoningEffort(effort); err != nil {
				t.Fatalf("criticality=%s effort=%s: %v", criticality, effort, err)
			}
		}
	}
	if governance.ValidateSecurityCriticality("provider-derived") == nil || governance.ValidateReasoningEffort("auto") == nil {
		t.Fatal("unknown policy value accepted")
	}
}
