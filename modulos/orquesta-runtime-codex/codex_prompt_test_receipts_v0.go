package orquestaruntimecodex

import "encoding/json"

func promptACKTestReceiptsJSONV0(commands []string) string {
	type promptReceiptV0 struct {
		SchemaVersion  string   `json:"schema_version"`
		Command        string   `json:"command"`
		Status         string   `json:"status"`
		ExitCode       int      `json:"exit_code"`
		EvidenceRefs   []string `json:"evidence_refs"`
		OccurredAt     string   `json:"occurred_at"`
		Sequence       int      `json:"sequence"`
		OutputRedacted bool     `json:"output_redacted"`
	}
	compact := compactPromptValuesV0(commands)
	receipts := make([]promptReceiptV0, 0, len(compact))
	for index, command := range compact {
		receipts = append(receipts, promptReceiptV0{
			SchemaVersion:  OrquestaRequiredTestReceiptSchemaVersionV0,
			Command:        command,
			Status:         "passed",
			ExitCode:       0,
			EvidenceRefs:   []string{"required-test-receipt-ref-<compacta>"},
			OccurredAt:     "<RFC3339>",
			Sequence:       index + 1,
			OutputRedacted: true,
		})
	}
	data, err := json.Marshal(receipts)
	if err != nil {
		return "[]"
	}
	return string(data)
}
