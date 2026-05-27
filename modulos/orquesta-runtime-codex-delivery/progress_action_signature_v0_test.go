package orquestaruntimecodexdelivery

import "testing"

func TestCodexProgressActionHashV0UsaPayloadCanonicoNoAmbiguo(t *testing.T) {
	hash := codexProgressActionHashV0([]string{"docs/a.md", "internal/app.go"})
	if len(hash) != 32 {
		t.Fatalf("hash len=%d value=%q", len(hash), hash)
	}

	joinedA := codexProgressActionHashV0([]string{"ab", "c"})
	joinedB := codexProgressActionHashV0([]string{"a", "bc"})
	if joinedA == joinedB {
		t.Fatalf("hash no distingue campos concatenados: %s", joinedA)
	}

	separatorA := codexProgressActionHashV0([]string{"a\x00b", "c"})
	separatorB := codexProgressActionHashV0([]string{"a", "b\x00c"})
	if separatorA == separatorB {
		t.Fatalf("hash no distingue separadores embebidos: %s", separatorA)
	}
}
