package orquestaopesconnector

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type opesCreateJobRequestV0 struct {
	JobType        string            `json:"job_type"`
	Input          map[string]any    `json:"input,omitempty"`
	MaxAttempts    int               `json:"max_attempts"`
	CorrelationID  string            `json:"correlation_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	RequestedBy    string            `json:"requested_by"`
	ExternalRefs   map[string]string `json:"external_refs,omitempty"`
}

type opesArtifactRequestV0 struct {
	ArtifactType   string            `json:"artifact_type"`
	Summary        string            `json:"summary,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	PayloadJSON    map[string]any    `json:"payload_json"`
	ExternalRefs   map[string]string `json:"external_refs,omitempty"`
	CompleteJob    bool              `json:"complete_job"`
}

type opesJobResponseV0 struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	CorrelationID  string            `json:"correlation_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	ExternalRefs   map[string]string `json:"external_refs"`
}

type opesArtifactResponseV0 struct {
	ID             string             `json:"id"`
	ArtifactID     string             `json:"artifact_id"`
	JobID          string             `json:"job_id"`
	CorrelationID  string             `json:"correlation_id"`
	IdempotencyKey string             `json:"idempotency_key"`
	ExternalRefs   map[string]string  `json:"external_refs"`
	Artifact       opesArtifactInfoV0 `json:"artifact"`
	Job            opesArtifactJobV0  `json:"job"`
}

type opesArtifactInfoV0 struct {
	ID    string `json:"id"`
	JobID string `json:"job_id"`
	Type  string `json:"type"`
}

type opesArtifactJobV0 struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func opesCreateJobPayloadV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	maxAttempts int,
) opesCreateJobRequestV0 {
	input := domainWorkFieldsToObjectV0(request.InputFields)
	addStringV0(input, "work_kind", request.WorkKind)
	addStringSliceV0(input, "work_refs", request.WorkRefs)
	addStringSliceV0(input, "input_refs", request.InputRefs)
	addStringSliceV0(input, "interface_refs", request.InterfaceRefs)
	addStringSliceV0(input, "constraints", request.Constraints)
	addStringSliceV0(input, "acceptance_criteria", request.AcceptanceCriteria)
	addStringSliceV0(input, "evidence_refs", request.EvidenceRefs)
	addStringV0(input, "objective", request.Objective)
	transportJobType := opesTransportJobTypeForWorkKindV0(request.WorkKind)
	if transportJobType != request.WorkKind {
		addStringV0(input, "transport_job_type", transportJobType)
	}
	return opesCreateJobRequestV0{
		JobType:        transportJobType,
		Input:          input,
		MaxAttempts:    maxAttempts,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		RequestedBy:    request.RequestedBy,
		ExternalRefs:   domainWorkExternalRefsMapV0(request.ExternalRefs),
	}
}

func opesArtifactPayloadV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) opesArtifactRequestV0 {
	payload := domainWorkFieldsToObjectV0(submission.PayloadFields)
	payload = canonicalOPESArtifactPayloadObjectV0(submission.ArtifactType, payload)
	addStringSliceV0(payload, "payload_refs", submission.PayloadRefs)
	addStringSliceV0(payload, "evidence_refs", submission.EvidenceRefs)
	return opesArtifactRequestV0{
		ArtifactType:   submission.ArtifactType,
		Summary:        submission.Summary,
		IdempotencyKey: submission.IdempotencyKey,
		PayloadJSON:    payload,
		ExternalRefs:   domainWorkExternalRefsMapV0(submission.ExternalRefs),
		CompleteJob:    submission.CompleteJob,
	}
}

func canonicalOPESArtifactPayloadObjectV0(
	artifactType string,
	payload map[string]any,
) map[string]any {
	if strings.TrimSpace(artifactType) != orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0 {
		return payload
	}
	normalizeOPESAudioAssetPayloadObjectV0(payload)
	return payload
}

func normalizeOPESAudioAssetPayloadObjectV0(payload map[string]any) {
	if payload == nil {
		return
	}
	if value, ok := payload["duration_seconds"]; ok {
		if numeric, ok := opesPositiveIntFromAnyV0(value); ok {
			payload["duration_seconds"] = numeric
		} else {
			if _, exists := payload["duration_seconds_raw"]; !exists {
				payload["duration_seconds_raw"] = value
			}
			delete(payload, "duration_seconds")
		}
	}
	if value, ok := payload["source_refs"]; ok {
		if !opesSourceRefsAlreadyCompactV0(value) {
			if _, exists := payload["source_ref_details"]; !exists {
				payload["source_ref_details"] = value
			}
		}
		if refs, ok := opesCompactRefsFromAnyV0(value); ok {
			payload["source_refs"] = refs
			return
		}
		delete(payload, "source_refs")
	}
}

func opesSourceRefsAlreadyCompactV0(value any) bool {
	switch typed := value.(type) {
	case []string:
		return true
	case []any:
		for _, item := range typed {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func opesPositiveIntFromAnyV0(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, typed > 0
	case int64:
		return int(typed), typed > 0
	case float64:
		if typed <= 0 {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil && parsed > 0 {
			return int(parsed), true
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return parsed, true
		}
	}
	return 0, false
}

func opesCompactRefsFromAnyV0(value any) ([]string, bool) {
	refs := []string{}
	seen := map[string]struct{}{}
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, exists := seen[ref]; exists {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	switch typed := value.(type) {
	case []string:
		for _, ref := range typed {
			add(ref)
		}
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				add(text)
				continue
			}
			if object, ok := item.(map[string]any); ok {
				for _, key := range opesSourceRefCandidateKeysV0() {
					if text, ok := object[key].(string); ok {
						add(text)
					}
				}
			}
		}
	case map[string]any:
		for _, key := range opesSourceRefCandidateKeysV0() {
			if text, ok := typed[key].(string); ok {
				add(text)
			}
		}
	default:
		return nil, false
	}
	return refs, len(refs) > 0
}

func opesSourceRefCandidateKeysV0() []string {
	return []string{
		"source_ref",
		"source_artifact_ref",
		"source_content_artifact_ref",
		"source_content_ref",
		"assembled_topic_artifact_ref",
		"assembled_topic_artifact_id",
		"artifact_ref",
		"topic_id",
		"program_id",
		"course_id",
	}
}

func domainWorkFieldsToObjectV0(fields []orquestadomainwork.DomainWorkFieldV0) map[string]any {
	out := map[string]any{}
	for _, field := range fields {
		if len(field.ValueJSON) > 0 {
			var value any
			if err := json.Unmarshal(field.ValueJSON, &value); err == nil {
				out[field.Name] = value
				continue
			}
		}
		if len(field.Values) > 0 {
			out[field.Name] = append([]string(nil), field.Values...)
			continue
		}
		out[field.Name] = field.Value
	}
	return out
}

func domainWorkExternalRefsMapV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
) map[string]string {
	out := map[string]string{}
	for _, ref := range refs {
		out[ref.Kind] = ref.Ref
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func opesArtifactPathV0(jobRef string) string {
	return "/api/jobs/" + url.PathEscape(jobRef) + "/artifacts"
}

func addStringV0(out map[string]any, key string, value string) {
	if value != "" {
		out[key] = value
	}
}

func addStringSliceV0(out map[string]any, key string, values []string) {
	if len(values) > 0 {
		out[key] = append([]string(nil), values...)
	}
}
