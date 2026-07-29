package config

import (
	"testing"
	"time"
)

func TestSelfWatchdogPolicyValidatesEnabledProjection(t *testing.T) {
	valid := SelfWatchdogPolicy{
		Enabled:             true,
		ObservationInterval: time.Second,
		CPUHighPercent:      80,
		SustainedFor:        3 * time.Second,
		NoProgressFor:       5 * time.Second,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid policy: %v", err)
	}

	tests := map[string]func(*SelfWatchdogPolicy){
		"observation interval":      func(policy *SelfWatchdogPolicy) { policy.ObservationInterval = 0 },
		"CPU threshold lower bound": func(policy *SelfWatchdogPolicy) { policy.CPUHighPercent = 0 },
		"CPU threshold upper bound": func(policy *SelfWatchdogPolicy) { policy.CPUHighPercent = 101 },
		"sustained window":          func(policy *SelfWatchdogPolicy) { policy.SustainedFor = time.Millisecond },
		"no-progress window":        func(policy *SelfWatchdogPolicy) { policy.NoProgressFor = time.Millisecond },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := candidate.Validate(); !HasErrorCode(err, ErrorSelfWatchdogInvalid) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestSelfWatchdogDisabledPolicyNeedsNoShadowDefaults(t *testing.T) {
	if err := (SelfWatchdogPolicy{}).Validate(); err != nil {
		t.Fatalf("disabled empty policy: %v", err)
	}
}
