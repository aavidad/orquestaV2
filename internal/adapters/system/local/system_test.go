package local

import (
	"context"
	"strings"
	"testing"
)

func TestIDGeneratorCreatesOpaqueUniqueIDs(t *testing.T) {
	generator := IDGenerator{}
	first, err := generator.NewID(context.Background(), "goal")
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}
	second, err := generator.NewID(context.Background(), "goal")
	if err != nil {
		t.Fatalf("NewID() second error = %v", err)
	}
	if first == second || !strings.HasPrefix(first, "goal:") || len(first) != len("goal:")+32 {
		t.Fatalf("generated IDs = %q, %q", first, second)
	}
}

func TestIDGeneratorRejectsSemanticOrUnsafePrefix(t *testing.T) {
	generator := IDGenerator{}
	for _, prefix := range []string{"", "Goal", "goal/ref", " goal"} {
		if _, err := generator.NewID(context.Background(), prefix); err == nil {
			t.Fatalf("prefix %q accepted", prefix)
		}
	}
}
