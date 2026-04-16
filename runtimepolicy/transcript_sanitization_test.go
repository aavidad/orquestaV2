package runtimepolicy

import "testing"

func TestCompactPendingTranscript(t *testing.T) {
	if got := CompactPendingTranscript(""); got != "" {
		t.Fatalf("unexpected non-empty compacted value: %q", got)
	}
	if got := CompactPendingTranscript("\x00\t\r\n"); got != "" {
		t.Fatalf("unexpected control-only compacted value: %q", got)
	}
	if got := CompactPendingTranscript("   keep.me   "); got != "keep.me" {
		t.Fatalf("unexpected trim: %q", got)
	}
	if got := CompactPendingTranscript("abc\x1b[0mdef"); got != "abcdef" {
		t.Fatalf("unexpected ansi strip: %q", got)
	}
	raw := make([]byte, 530)
	for i := 0; i < 530; i++ {
		raw[i] = 'a'
	}
	if got := CompactPendingTranscript(string(raw)); len(got) != 512 {
		t.Fatalf("unexpected compacted length: %d", len(got))
	}
}
