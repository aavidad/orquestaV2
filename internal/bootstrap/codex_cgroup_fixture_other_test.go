//go:build !linux

package bootstrap

import "testing"

func configureTestCodexRuntimeCgroup(t *testing.T, _ string) {
	t.Helper()
}
