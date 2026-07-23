//go:build !linux && v18_real_e2e

package bootstrap

import "testing"

func TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd(t *testing.T) {
	t.Fatal("V18_GATE_REAL_E2E_UNSUPPORTED: production bubblewrap attestation requires Linux")
}
