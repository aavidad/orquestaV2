package orquestaappcodexstack

import "testing"

func TestCodexStackDeterministicRefV0UsaDigestCompletoYCamposNoAmbiguos(t *testing.T) {
	ref := codexStackDeterministicRefV0("evidence-ref-test-", "run", "task", "delivery")
	const prefix = "evidence-ref-test-"
	if len(ref) != len(prefix)+64 {
		t.Fatalf("ref=%q len=%d, want prefix+64", ref, len(ref))
	}

	joinedA := codexStackDeterministicDigestV0("ab", "c")
	joinedB := codexStackDeterministicDigestV0("a", "bc")
	if joinedA == joinedB {
		t.Fatalf("digest no distingue campos con misma concatenacion: %s", joinedA)
	}

	withSeparatorA := codexStackDeterministicDigestV0("a\x00b", "c")
	withSeparatorB := codexStackDeterministicDigestV0("a", "b\x00c")
	if withSeparatorA == withSeparatorB {
		t.Fatalf("digest no distingue separadores embebidos: %s", withSeparatorA)
	}
}
