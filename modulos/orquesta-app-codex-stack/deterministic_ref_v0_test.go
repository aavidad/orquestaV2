package orquestaappcodexstack

import (
	"testing"
	"time"
)

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

func TestCodexStackReworkDigestsV0MantienenMargenSuficiente(t *testing.T) {
	digest := assessmentReplanDigestV0("run", "assessment", "agent", "task")
	if len(digest) != 32 {
		t.Fatalf("assessment digest=%q len=%d", digest, len(digest))
	}

	joinedA := assessmentReplanDigestV0("ab", "c")
	joinedB := assessmentReplanDigestV0("a", "bc")
	if joinedA == joinedB {
		t.Fatalf("digest no distingue campos concatenados: %s", joinedA)
	}
}

func TestAutoprogrammingPrepareRetryRefV0NoMezclaBaseYStamp(t *testing.T) {
	left := autoprogrammingPrepareRetryRefV0("request-ref-a|b", "c", time.Time{})
	right := autoprogrammingPrepareRetryRefV0("request-ref-a", "b|c", time.Time{})
	if left == right {
		t.Fatalf("retry ref no distingue base/stamp con separador embebido: %s", left)
	}
}
