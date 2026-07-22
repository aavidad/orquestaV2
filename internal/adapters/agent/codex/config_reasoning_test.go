package codex

import "testing"

func TestReasoningEffortAcceptsXHighAndUltraOnlyAsDeclared(t *testing.T) {
	for _, value := range []string{"low", "medium", "high", "xhigh", "ultra"} {
		if !validReasoningEffort(value) {
			t.Fatalf("declared reasoning effort rejected: %q", value)
		}
	}
	for _, value := range []string{"", "max", "XHIGH", " ultra", "ultra "} {
		if validReasoningEffort(value) {
			t.Fatalf("undeclared reasoning effort accepted: %q", value)
		}
	}
}
