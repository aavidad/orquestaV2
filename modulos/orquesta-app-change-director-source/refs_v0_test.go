package orquestaappchangedirectorsource

import (
	"strings"
	"testing"
)

func TestAppChangeSuffixV0UsaDigestCanonicoNoAmbiguo(t *testing.T) {
	suffix := appChangeSuffixV0("change-ref-001")
	if !strings.HasPrefix(suffix, "appchange-") || len(strings.TrimPrefix(suffix, "appchange-")) != 32 {
		t.Fatalf("suffix=%q", suffix)
	}

	joinedA := appChangeCanonicalRefPayloadV0("ab", "c")
	joinedB := appChangeCanonicalRefPayloadV0("a", "bc")
	if string(joinedA) == string(joinedB) {
		t.Fatalf("payload no distingue campos concatenados: %q", joinedA)
	}

	separatorA := appChangeCanonicalRefPayloadV0("a|b", "c")
	separatorB := appChangeCanonicalRefPayloadV0("a", "b|c")
	if string(separatorA) == string(separatorB) {
		t.Fatalf("payload no distingue separadores embebidos: %q", separatorA)
	}
}
