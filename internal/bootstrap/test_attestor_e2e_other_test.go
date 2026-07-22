//go:build !linux && v17_real_e2e

package bootstrap

import "testing"

func TestRealGitSQLiteCASBubblewrapAttestationEndToEnd(t *testing.T) {
	t.Fatal("V17_GATE_REAL_E2E_UNSUPPORTED: production bubblewrap attestation requires Linux")
}
