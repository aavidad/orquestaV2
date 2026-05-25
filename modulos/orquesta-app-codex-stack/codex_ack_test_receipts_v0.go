package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexStackRequiredTestReceiptsV0(
	commands []string,
) []orquestaruntimecodex.CodexRequiredTestReceiptV0 {
	commands = compactStringsV0(commands)
	receipts := make([]orquestaruntimecodex.CodexRequiredTestReceiptV0, 0, len(commands))
	for index, command := range commands {
		receipts = append(receipts, orquestaruntimecodex.CodexRequiredTestReceiptV0{
			SchemaVersion:  orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
			Command:        strings.TrimSpace(command),
			Status:         "passed",
			ExitCode:       codexStackIntPtrV0(0),
			EvidenceRefs:   []string{fmt.Sprintf("required-test-receipt-ref-codex-stack-%03d", index+1)},
			OccurredAt:     "2026-05-24T10:00:00Z",
			Sequence:       index + 1,
			OutputRedacted: codexStackBoolPtrV0(true),
		})
	}
	return receipts
}

func codexStackIntPtrV0(value int) *int {
	return &value
}

func codexStackBoolPtrV0(value bool) *bool {
	return &value
}
