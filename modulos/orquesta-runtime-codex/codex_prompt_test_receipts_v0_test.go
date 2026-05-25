package orquestaruntimecodex

import (
	"strings"
	"testing"
)

func TestBuildCodexAgentPromptV0PideRecibosEstructuradosDeTestsV0(t *testing.T) {
	packet := codexPacketForTestV0()

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"test_receipts",
		"exit_code 0",
		"output_redacted=true",
		"codex_required_test_receipt.v0",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
}
