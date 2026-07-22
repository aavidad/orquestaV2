package application

import "errors"

// TestAttestationPolicy is composition-owned sandbox policy identity. The
// application binds it into every subject without knowing adapter mechanics.
type TestAttestationPolicy struct {
	Ref    string
	Digest string
}

func ValidateTestAttestationPolicy(policy TestAttestationPolicy) error {
	if !validApplicationRef(policy.Ref) || !validEffectDigest(policy.Digest) {
		return errors.New("application.test_attestation_policy_invalid")
	}
	return nil
}
