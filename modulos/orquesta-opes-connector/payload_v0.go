package orquestaopesconnector

import (
	"net/url"

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
	ID             string            `json:"id"`
	ArtifactID     string            `json:"artifact_id"`
	JobID          string            `json:"job_id"`
	CorrelationID  string            `json:"correlation_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	ExternalRefs   map[string]string `json:"external_refs"`
}

func opesCreateJobPayloadV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	maxAttempts int,
) opesCreateJobRequestV0 {
	input := domainWorkFieldsToObjectV0(request.InputFields)
	addStringSliceV0(input, "work_refs", request.WorkRefs)
	addStringSliceV0(input, "input_refs", request.InputRefs)
	addStringSliceV0(input, "interface_refs", request.InterfaceRefs)
	addStringSliceV0(input, "constraints", request.Constraints)
	addStringSliceV0(input, "acceptance_criteria", request.AcceptanceCriteria)
	addStringSliceV0(input, "evidence_refs", request.EvidenceRefs)
	addStringV0(input, "objective", request.Objective)
	return opesCreateJobRequestV0{
		JobType:        request.WorkKind,
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

func domainWorkFieldsToObjectV0(fields []orquestadomainwork.DomainWorkFieldV0) map[string]any {
	out := map[string]any{}
	for _, field := range fields {
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
