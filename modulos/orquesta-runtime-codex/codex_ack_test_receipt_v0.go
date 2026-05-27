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
	required := codexStrictTestReceiptRequiredCommandsV0(ack, packet)
	if len(required) == 0 {
		return
	}
	receipts := normalizeCodexRequiredTestReceiptsV0(ack.TestReceipts)
	if len(receipts) < len(required) {
		v.add(CodexConnectorAckArtifactV0, "test_receipts", "missing_required_test_receipt")
		return
	}
	if !codexRequiredTestReceiptsCoverRequiredCommandsV0(receipts, required) {
		v.add(CodexConnectorAckArtifactV0, "test_receipts", "required_test_receipt_mismatch")
		return
	}
	receiptsByCommand := map[string]CodexRequiredTestReceiptV0{}
	for _, receipt := range receipts {
		if issue := codexRequiredTestReceiptIssueV0(receipt); issue != "" {
			v.add(CodexConnectorAckArtifactV0, "test_receipts", issue)
			return
		}
		if _, exists := receiptsByCommand[receipt.Command]; !exists {
			receiptsByCommand[receipt.Command] = receipt
		}
	}
	for _, command := range required {
		if _, exists := receiptsByCommand[command]; !exists {
			v.add(CodexConnectorAckArtifactV0, "test_receipts", "missing_required_test_receipt")
			return
		}
	}
}

func codexStrictTestReceiptRequiredCommandsV0(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) []string {
	required := compactCodexAckStringsV0(packet.Task.RequiredTests)
	out := make([]string, 0, len(required))
	for _, command := range required {
		if codexStrictTestReceiptContextualRefOnlyResolvedV0(ack, packet, command) {
			continue
		}
		out = append(out, command)
	}
	return out
}

func codexStrictTestReceiptContextualRefOnlyResolvedV0(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
	command string,
) bool {
	trimmed := strings.TrimSpace(command)
	lower := strings.ToLower(trimmed)
	return trimmed != "" &&
		strings.Contains(lower, "ref_only") &&
		codexPacketRequiresRefOnlyAckEvidenceV0(packet) &&
		codexAckContainsTrimmedV0(ack.Tests, trimmed) &&
		codexAckContainsNotePrefixV0(ack.Notes, "contexto_ref_only_resuelto")
}

func codexRequiredTestReceiptsCoverRequiredCommandsV0(
	receipts []CodexRequiredTestReceiptV0,
	required []string,
) bool {
	seen := map[string]bool{}
	for _, receipt := range receipts {
		seen[strings.TrimSpace(receipt.Command)] = true
	}
	for _, command := range required {
		if !seen[strings.TrimSpace(command)] {
			return false
		}
	}
	return true
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
