package orquestadomainwork

import (
	"context"
	"encoding/json"
)

const (
	DomainWorkJobRequestSchemaV0         = "domain_work_job_request.v0"
	DomainWorkJobSchemaV0                = "domain_work_job.v0"
	DomainWorkArtifactSubmissionSchemaV0 = "domain_work_artifact_submission.v0"
	DomainWorkArtifactReceiptSchemaV0    = "domain_work_artifact_receipt.v0"

	DomainWorkStatusAcceptedV0 = "accepted"
	DomainWorkStatusInvalidV0  = "invalid"

	DomainWorkDefaultRequestedByV0 = "orquesta"

	ErrDomainWorkDomainRefRequiredV0    = "domain_work_domain_ref_required"
	ErrDomainWorkDomainRefInvalidV0     = "domain_work_domain_ref_invalid"
	ErrDomainWorkWorkKindRequiredV0     = "domain_work_work_kind_required"
	ErrDomainWorkWorkKindInvalidV0      = "domain_work_work_kind_invalid"
	ErrDomainWorkObjectiveRequiredV0    = "domain_work_objective_required"
	ErrDomainWorkCorrelationRequiredV0  = "domain_work_correlation_id_required"
	ErrDomainWorkIdempotencyRequiredV0  = "domain_work_idempotency_key_required"
	ErrDomainWorkRequestedByRequiredV0  = "domain_work_requested_by_required"
	ErrDomainWorkRefInvalidV0           = "domain_work_ref_invalid"
	ErrDomainWorkFieldNameInvalidV0     = "domain_work_field_name_invalid"
	ErrDomainWorkJobRefRequiredV0       = "domain_work_job_ref_required"
	ErrDomainWorkArtifactRefRequiredV0  = "domain_work_artifact_ref_required"
	ErrDomainWorkArtifactTypeRequiredV0 = "domain_work_artifact_type_required"
	ErrDomainWorkFieldJSONInvalidV0     = "domain_work_field_json_invalid"
)

type DomainWorkJobRequestV0 struct {
	SchemaVersion      string                    `json:"schema_version"`
	RequestID          string                    `json:"request_id,omitempty"`
	CorrelationID      string                    `json:"correlation_id"`
	IdempotencyKey     string                    `json:"idempotency_key"`
	RequestedBy        string                    `json:"requested_by"`
	DomainRef          string                    `json:"domain_ref"`
	InterfaceRefs      []string                  `json:"interface_refs,omitempty"`
	WorkKind           string                    `json:"work_kind"`
	WorkRefs           []string                  `json:"work_refs,omitempty"`
	Objective          string                    `json:"objective"`
	InputFields        []DomainWorkFieldV0       `json:"input_fields,omitempty"`
	InputRefs          []string                  `json:"input_refs,omitempty"`
	Constraints        []string                  `json:"constraints,omitempty"`
	AcceptanceCriteria []string                  `json:"acceptance_criteria,omitempty"`
	ExternalRefs       []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs       []string                  `json:"evidence_refs,omitempty"`
}

type DomainWorkJobV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	Status         string                    `json:"status"`
	JobRef         string                    `json:"job_ref,omitempty"`
	DomainRef      string                    `json:"domain_ref,omitempty"`
	WorkKind       string                    `json:"work_kind,omitempty"`
	CorrelationID  string                    `json:"correlation_id,omitempty"`
	IdempotencyKey string                    `json:"idempotency_key,omitempty"`
	ExternalRefs   []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs   []string                  `json:"evidence_refs,omitempty"`
	Issues         []DomainWorkIssueV0       `json:"issues,omitempty"`
}

type DomainWorkArtifactSubmissionV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	RequestID      string                    `json:"request_id,omitempty"`
	CorrelationID  string                    `json:"correlation_id"`
	IdempotencyKey string                    `json:"idempotency_key"`
	RequestedBy    string                    `json:"requested_by"`
	DomainRef      string                    `json:"domain_ref"`
	JobRef         string                    `json:"job_ref"`
	ArtifactRef    string                    `json:"artifact_ref"`
	ArtifactType   string                    `json:"artifact_type"`
	Summary        string                    `json:"summary,omitempty"`
	PayloadFields  []DomainWorkFieldV0       `json:"payload_fields,omitempty"`
	PayloadRefs    []string                  `json:"payload_refs,omitempty"`
	ExternalRefs   []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs   []string                  `json:"evidence_refs,omitempty"`
	CompleteJob    bool                      `json:"complete_job,omitempty"`
}

type DomainWorkArtifactReceiptV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	Status         string                    `json:"status"`
	JobRef         string                    `json:"job_ref,omitempty"`
	ArtifactRef    string                    `json:"artifact_ref,omitempty"`
	ReceiptRef     string                    `json:"receipt_ref,omitempty"`
	CorrelationID  string                    `json:"correlation_id,omitempty"`
	IdempotencyKey string                    `json:"idempotency_key,omitempty"`
	ExternalRefs   []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs   []string                  `json:"evidence_refs,omitempty"`
	Issues         []DomainWorkIssueV0       `json:"issues,omitempty"`
}

type DomainWorkExternalRefV0 struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type DomainWorkFieldV0 struct {
	Name      string          `json:"name"`
	Value     string          `json:"value,omitempty"`
	Values    []string        `json:"values,omitempty"`
	ValueJSON json.RawMessage `json:"value_json,omitempty"`
}

type DomainWorkIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type DomainWorkJobCreatorPortV0 interface {
	CreateDomainWorkJobV0(context.Context, DomainWorkJobRequestV0) (DomainWorkJobV0, error)
}

type DomainWorkArtifactSubmitterPortV0 interface {
	SubmitDomainWorkArtifactV0(context.Context, DomainWorkArtifactSubmissionV0) (DomainWorkArtifactReceiptV0, error)
}
