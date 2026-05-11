package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxReviewResultPayloadBytesV0 = 2048
	maxReviewResultStringV0       = 600
	maxReviewResultEvidenceRefsV0 = 20
)

var forbiddenReviewResultFragmentsV0 = []string{
	"db", "database", "sql", "dsn", "runtime",
	"provider", "providers", "proveedor", "proveedores",
	"model", "models", "modelo", "modelos",
	"home", "oauth", "prompt", "prompts",
	"transcript", "transcripts", "completion",
	"raw_text", "full_text", "codex", "claude",
	"ollama", "vllm", "adapter", "adaptador",
	"filesystem", "git", "docker", "tmux",
	"secret", "secreto", "token", "password",
	"credential", "credencial", "api_key",
}

func validateReviewResultRequiredFieldsV0(result ReviewResultV0) error {
	fields := map[string]string{
		"review_result_ref": result.ReviewResultRef,
		"review_request_id": result.ReviewRequestID,
		"delivery_ref":      result.DeliveryRef,
		"status":            string(result.Status),
		"summary":           result.Summary,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return reviewResultErrorV0(ErrReviewResultInvalidoV0, field)
		}
	}
	return nil
}

func validateReviewResultCollectionsV0(result ReviewResultV0) error {
	if len(result.EvidenceRefs) > maxReviewResultEvidenceRefsV0 {
		return reviewResultErrorV0(ErrReviewResultInvalidoV0, "evidence_refs")
	}
	for _, ref := range result.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return reviewResultErrorV0(ErrReviewResultInvalidoV0, "evidence_refs")
		}
	}
	return nil
}

func validateReviewResultCompactPayloadV0(result ReviewResultV0) error {
	data, err := json.Marshal(result)
	if err != nil {
		return reviewResultErrorV0(ErrReviewResultPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewResultPayloadBytesV0 {
		return reviewResultErrorV0(ErrReviewResultPayloadInvalidoV0, "payload")
	}
	return nil
}

func reviewResultTextFieldsV0(result ReviewResultV0) []string {
	values := []string{
		result.ReviewResultRef,
		result.ReviewRequestID,
		result.DeliveryRef,
		string(result.Status),
		result.Summary,
		result.QualityGateRef,
	}
	return append(values, result.EvidenceRefs...)
}

func reviewResultHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReviewResultStringV0 {
			return true
		}
	}
	return false
}

func reviewResultHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenReviewResultFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func normalizeReviewResultStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		normalized[index] = strings.TrimSpace(value)
	}
	return normalized
}

func reviewResultErrorV0(code string, field string) ReviewResultErrorV0 {
	return ReviewResultErrorV0{Code: code, Field: field}
}
