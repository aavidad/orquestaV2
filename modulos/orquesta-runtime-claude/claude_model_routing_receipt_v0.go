package orquestaruntimeclaude

import "strings"

const (
	ClaudeModelRoutingReceiptSchemaVersionV0 = "claude_model_routing_receipt.v0"
	ClaudeModelRoutingReceiptFileNameV0      = "model_routing_receipt.json"
)

// ClaudeModelRoutingReceiptV0 is launcher-owned evidence with opaque refs.
type ClaudeModelRoutingReceiptV0 struct {
	SchemaVersion         string   `json:"schema_version"`
	RequestID             string   `json:"request_id"`
	CorrelationID         string   `json:"correlation_id"`
	TaskRef               string   `json:"task_ref"`
	Level                 string   `json:"level"`
	Trivial               bool     `json:"trivial,omitempty"`
	SelectedModelRef      string   `json:"selected_model_ref"`
	ReasoningEffort       string   `json:"reasoning_effort"`
	PolicyRef             string   `json:"policy_ref"`
	ReasonRef             string   `json:"reason_ref,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
	XHighAuthorizationRef string   `json:"xhigh_authorization_ref,omitempty"`
}

func (receipt ClaudeModelRoutingReceiptV0) validForLaunchV0(model string) bool {
	if strings.TrimSpace(model) == "" || strings.Contains(strings.ToLower(model), "opus") ||
		receipt.SchemaVersion != ClaudeModelRoutingReceiptSchemaVersionV0 ||
		strings.TrimSpace(receipt.RequestID) == "" || strings.TrimSpace(receipt.CorrelationID) == "" ||
		strings.TrimSpace(receipt.TaskRef) == "" || strings.TrimSpace(receipt.PolicyRef) == "" ||
		strings.TrimSpace(receipt.SelectedModelRef) == "" {
		return false
	}
	switch strings.TrimSpace(receipt.Level) {
	case "normal", "complex", "critical":
	default:
		return false
	}
	if strings.TrimSpace(receipt.Level) == "critical" &&
		(strings.TrimSpace(receipt.ReasonRef) == "" || len(receipt.EvidenceRefs) == 0) {
		return false
	}
	switch strings.TrimSpace(receipt.ReasoningEffort) {
	case "low", "medium", "high":
		return true
	case "xhigh":
		return !receipt.Trivial && strings.TrimSpace(receipt.ReasonRef) != "" &&
			len(receipt.EvidenceRefs) > 0 && strings.TrimSpace(receipt.XHighAuthorizationRef) != ""
	default:
		return false
	}
}
