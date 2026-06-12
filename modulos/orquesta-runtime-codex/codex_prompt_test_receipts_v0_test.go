package orquestaruntimecodex

import (
	"encoding/json"
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

func TestBuildCodexAgentPromptV0ConservaComandosObligatoriosExactosEnACKV0(t *testing.T) {
	packet := codexPacketForTestV0()

	prompt := BuildCodexAgentPromptV0(packet, nil)

	for _, want := range []string{
		"cada test obligatorio pasado debe aparecer exactamente como aparece en el paquete",
		"conserva el comando obligatorio exacto del paquete",
		"sin prefijos de entorno, wrappers ni sufijos explicativos",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene disciplina de tests %q:\n%s", want, prompt)
		}
	}
}

func TestBuildCodexAgentPromptV0ACKEsperadoIncluyeReceiptPorCadaTestObligatorioV0(t *testing.T) {
	packet := codexPacketForTestV0()
	packet.Task.RequiredTests = []string{
		"go test ./...",
		"node --check web/app.js",
		"git diff --check",
	}

	prompt := BuildCodexAgentPromptV0(packet, nil)
	expectedACK := promptExpectedACKForTestV0(t, prompt)

	if len(expectedACK.TestReceipts) != len(packet.Task.RequiredTests) {
		t.Fatalf("test_receipts=%+v want %d", expectedACK.TestReceipts, len(packet.Task.RequiredTests))
	}
	for index, command := range packet.Task.RequiredTests {
		receipt := expectedACK.TestReceipts[index]
		if receipt.Command != command ||
			receipt.SchemaVersion != CodexRequiredTestReceiptSchemaVersionV0 ||
			receipt.Status != "passed" ||
			receipt.ExitCode != 0 ||
			receipt.Sequence != index+1 ||
			!receipt.OutputRedacted {
			t.Fatalf("receipt[%d]=+%v command=%q", index, receipt, command)
		}
	}
}

func promptExpectedACKForTestV0(t *testing.T, prompt string) struct {
	TestReceipts []struct {
		SchemaVersion  string `json:"schema_version"`
		Command        string `json:"command"`
		Status         string `json:"status"`
		ExitCode       int    `json:"exit_code"`
		Sequence       int    `json:"sequence"`
		OutputRedacted bool   `json:"output_redacted"`
	} `json:"test_receipts"`
} {
	t.Helper()
	_, after, ok := strings.Cut(prompt, "ACK esperado:\n")
	if !ok {
		t.Fatalf("prompt sin ACK esperado:\n%s", prompt)
	}
	raw, _, ok := strings.Cut(after, "\n\nTitulo:")
	if !ok {
		t.Fatalf("prompt sin cierre de ACK esperado:\n%s", prompt)
	}
	var out struct {
		TestReceipts []struct {
			SchemaVersion  string `json:"schema_version"`
			Command        string `json:"command"`
			Status         string `json:"status"`
			ExitCode       int    `json:"exit_code"`
			Sequence       int    `json:"sequence"`
			OutputRedacted bool   `json:"output_redacted"`
		} `json:"test_receipts"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		t.Fatalf("ACK esperado no parseable: %v\n%s", err, raw)
	}
	return out
}
