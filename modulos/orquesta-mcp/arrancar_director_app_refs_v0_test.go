package orquestamcp

import (
	"strings"
	"testing"
)

func TestNeutralAppDirectorHashV0UsaPayloadCanonicoNoAmbiguo(t *testing.T) {
	ref := neutralAppDirectorRequestRefV0("req", "Mi App", "run", "task")
	if !strings.HasPrefix(ref, "req-mi-app-") || len(strings.TrimPrefix(ref, "req-mi-app-")) != 32 {
		t.Fatalf("ref=%q", ref)
	}

	joinedA := neutralAppDirectorHashV0("ab", "c")
	joinedB := neutralAppDirectorHashV0("a", "bc")
	if joinedA == joinedB {
		t.Fatalf("hash no distingue campos concatenados: %s", joinedA)
	}

	separatorA := neutralAppDirectorHashV0("a|b", "c")
	separatorB := neutralAppDirectorHashV0("a", "b|c")
	if separatorA == separatorB {
		t.Fatalf("hash no distingue separadores embebidos: %s", separatorA)
	}
}
