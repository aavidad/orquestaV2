package orquestaopesdirector

import (
	"context"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	OPESCausalProducerResultSchemaV0 = "opes_causal_producer_result.v0"

	OPESCausalProducerStatusCompletedV0 = "completed"
	OPESCausalProducerStatusInvalidV0   = "invalid"

	OPESCausalProducerDefaultDomainRefV0   = "opes"
	OPESCausalProducerDefaultRequestedByV0 = "orquesta-opes-causal-producer"

	ErrOPESCausalArtifactSourceRequiredV0 = "opes_causal_artifact_source_required"
	ErrOPESCausalJobCreatorRequiredV0     = "opes_causal_job_creator_required"
	ErrOPESCausalArtifactInvalidV0        = "opes_causal_artifact_invalid"
	ErrOPESCausalJobRejectedV0            = "opes_causal_job_rejected"
)

type OPESCausalProducerRequestV0 struct {
	DomainRef     string
	CorrelationID string
	MaxActions    int
	EvidenceRefs  []string
}

type OPESCausalProducerPortsV0 struct {
	ArtifactSource OPESCausalArtifactRecordSourcePortV0
	JobCreator     orquestadomainwork.DomainWorkJobCreatorPortV0
	JobRecords     orquestadomainwork.DomainWorkJobRecordSourcePortV0
}

type OPESCausalArtifactRecordSourcePortV0 interface {
	ListOPESCausalArtifactRecordsV0(
		context.Context,
		OPESCausalArtifactRecordFilterV0,
	) ([]OPESCausalArtifactRecordV0, error)
}

type OPESCausalArtifactRecordFilterV0 struct {
	DomainRef string
	Limit     int
}

type OPESCausalArtifactRecordV0 struct {
	IdempotencyKey string
	Status         string
	RunRef         string
	TaskRef        string
	DeliveryRef    string
	CorrelationID  string
	DomainRef      string
	JobRef         string
	ArtifactRef    string
	ArtifactType   string
	Summary        string
	PayloadFields  []orquestadomainwork.DomainWorkFieldV0
	PayloadRefs    []string
	ExternalRefs   []orquestadomainwork.DomainWorkExternalRefV0
	CompleteJob    bool
	ReceiptRef     string
	EvidenceRefs   []string
	IssueRefs      []string
	RecordedAt     string
}

type OPESCausalProducerResultV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	Status        string                                      `json:"status"`
	ProcessedRefs []string                                    `json:"processed_refs,omitempty"`
	SkippedRefs   []string                                    `json:"skipped_refs,omitempty"`
	RequestedJobs []orquestadomainwork.DomainWorkJobRequestV0 `json:"requested_jobs,omitempty"`
	CreatedJobs   []orquestadomainwork.DomainWorkJobV0        `json:"created_jobs,omitempty"`
	EvidenceRefs  []string                                    `json:"evidence_refs,omitempty"`
	Issues        []orquestadomainwork.DomainWorkIssueV0      `json:"issues,omitempty"`
}
