package orquestaruntimecodex

import (
	"encoding/json"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (v *codexAckValidatorV0) validateStrictTestReceipts(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) {
	required := compactCodexAckStringsV0(packet.Task.RequiredTests)
	if len(required) == 0 {
		return
	}
	receipts := normalizeCodexRequiredTestReceiptsV0(ack.TestReceipts)
	if len(receipts) != len(required) {
		v.add(CodexConnectorAckArtifactV0, "test_receipts", "missing_required_test_receipt")
		return
	}
	for index, command := range required {
		receipt := receipts[index]
		if receipt.Command != command {
			v.add(CodexConnectorAckArtifactV0, "test_receipts", "required_test_receipt_mismatch")
			return
		}
		if issue := codexRequiredTestReceiptIssueV0(receipt); issue != "" {
			v.add(CodexConnectorAckArtifactV0, "test_receipts", issue)
			return
		}
	}
}

func normalizeCodexRequiredTestReceiptsV0(
	receipts []CodexRequiredTestReceiptV0,
) []CodexRequiredTestReceiptV0 {
	out := make([]CodexRequiredTestReceiptV0, 0, len(receipts))
	for _, receipt := range receipts {
		out = append(out, CodexRequiredTestReceiptV0{
			SchemaVersion:  strings.TrimSpace(receipt.SchemaVersion),
			Command:        strings.TrimSpace(receipt.Command),
			Status:         strings.TrimSpace(receipt.Status),
			ExitCode:       receipt.ExitCode,
			EvidenceRefs:   compactCodexAckStringsV0(receipt.EvidenceRefs),
			OccurredAt:     strings.TrimSpace(receipt.OccurredAt),
			Sequence:       receipt.Sequence,
			OutputRedacted: receipt.OutputRedacted,
		})
	}
	return out
}

func codexRequiredTestReceiptIssueV0(receipt CodexRequiredTestReceiptV0) string {
	if receipt.SchemaVersion != CodexRequiredTestReceiptSchemaVersionV0 {
		return "required_test_receipt_schema_invalid"
	}
	if receipt.Command == "" {
		return "required_test_receipt_command_required"
	}
	if receipt.Status != "passed" {
		return "required_test_receipt_not_passed"
	}
	if receipt.ExitCode == nil || *receipt.ExitCode != 0 {
		return "required_test_receipt_exit_code_invalid"
	}
	if len(receipt.EvidenceRefs) == 0 {
		return "required_test_receipt_evidence_required"
	}
	if receipt.OccurredAt == "" || receipt.Sequence <= 0 {
		return "required_test_receipt_order_required"
	}
	if receipt.OutputRedacted == nil || !*receipt.OutputRedacted {
		return "required_test_receipt_output_redaction_required"
	}
	return ""
}

func codexStrictCompletedAckRawTestReceiptIssuesV0(
	data []byte,
	correlationID string,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	var raw struct {
		TestReceipts []map[string]json.RawMessage `json:"test_receipts"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	for _, receipt := range raw.TestReceipts {
		for key, value := range receipt {
			if codexTestReceiptRawFieldForbiddenV0(key) && len(value) > 0 {
				return []orquestaruntime.ExternalAgentConnectorErrorV0{codexIssueV0(
					CodexConnectorAckForbiddenV0,
					"test_receipts",
					correlationID,
					"raw_test_output_forbidden",
				)}
			}
		}
	}
	return nil
}

func codexTestReceiptRawFieldForbiddenV0(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "stdout", "stderr", "output", "prompt", "completion", "transcript", "home":
		return true
	default:
		return false
	}
}

func CodexRequiredTestReceiptEvidenceRefsV0(receipts []CodexRequiredTestReceiptV0) []string {
	refs := []string{}
	for _, receipt := range receipts {
		refs = append(refs, receipt.EvidenceRefs...)
	}
	return compactCodexDeliveryObservationRefsV0(refs)
}

func codexRequiredTestReceiptSensitiveValuesV0(
	receipts []CodexRequiredTestReceiptV0,
) []string {
	values := []string{}
	for _, receipt := range receipts {
		values = append(values,
			receipt.Command,
			receipt.Status,
			receipt.OccurredAt,
		)
		values = append(values, receipt.EvidenceRefs...)
	}
	return values
}
