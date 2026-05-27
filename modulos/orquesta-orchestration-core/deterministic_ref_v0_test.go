package orquestacionnucleoapp

import "testing"

func TestDeterministicRefDigestV0UsaPayloadCanonicoNoAmbiguo(t *testing.T) {
	digest := deterministicRefDigestV0("test_namespace", "run", "task", "agent")
	if len(digest) != 64 {
		t.Fatalf("digest len=%d value=%q", len(digest), digest)
	}

	joinedA := deterministicRefDigestV0("test_namespace", "ab", "c")
	joinedB := deterministicRefDigestV0("test_namespace", "a", "bc")
	if joinedA == joinedB {
		t.Fatalf("digest no distingue campos concatenados: %s", joinedA)
	}

	separatorA := deterministicRefDigestV0("test_namespace", "a|b", "c")
	separatorB := deterministicRefDigestV0("test_namespace", "a", "b|c")
	if separatorA == separatorB {
		t.Fatalf("digest no distingue separadores embebidos: %s", separatorA)
	}
}

func TestDeterministicRefDigestV0DistingueNamespace(t *testing.T) {
	left := deterministicRefDigestV0("namespace-a", "same")
	right := deterministicRefDigestV0("namespace-b", "same")
	if left == right {
		t.Fatalf("digest no distingue namespace: %s", left)
	}
}
