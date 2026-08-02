package ports

import (
	"os"
	"testing"
)

func TestAgentMicroVMNetworkAuthorityLivesOnlyInSignedSiblingContract(t *testing.T) {
	for _, retired := range []string{
		"agent_microvm_network.go",
		"agent_microvm_launch_auth.go",
		"agent_microvm_launch_auth_test.go",
	} {
		if _, err := os.Lstat(retired); !os.IsNotExist(err) {
			t.Fatalf("retired duplicate authority remains at %q: %v", retired, err)
		}
	}
}
