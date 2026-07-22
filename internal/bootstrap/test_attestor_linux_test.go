//go:build linux

package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestBubblewrapTestAttestorWiringUsesObjectStreamAndCanonicalLimits(t *testing.T) {
	content, err := os.ReadFile("runtime.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{
		"bubblewrap.New(config)",
		"TestAttestorTimeout()",
		"TestAttestorMaxSubjectBytes()",
		"TestAttestorMaxConcurrentRuns()",
		"RuntimeMaxOutputBytes()",
		"closer: attestor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("production wiring missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"WorkspacePathResolver: workspace",
		"MaterializeVerifiedTestWorkspace(",
		"bubblewrap.Prepare(",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("unsafe worktree authority remains: %q", forbidden)
		}
	}
}
