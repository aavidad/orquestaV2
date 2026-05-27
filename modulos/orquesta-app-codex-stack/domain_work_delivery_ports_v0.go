package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"sync"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	DomainWorkArtifactSubmissionStatusClaimedV0    = "claimed"
	DomainWorkArtifactSubmissionStatusSubmittingV0 = "submitting"
	DomainWorkArtifactSubmissionStatusAcceptedV0   = "accepted"
	DomainWorkArtifactSubmissionStatusRejectedV0   = "rejected"
)

type DomainWorkDeliveryBridgeConfigV0 struct {
	Enabled bool
	Builder DomainWorkArtifactSubmissionBuilderPortV0
	Ledger  DomainWorkArtifactSubmissionLedgerPortV0
}

type DomainWorkArtifactSubmissionBuilderPortV0 interface {
	BuildDomainWorkArtifactSubmissionV0(
		context.Context,
		DomainWorkArtifactSubmissionBuildInputV0,
	) (orquestadomainwork.DomainWorkArtifactSubmissionV0, bool, error)
}

type DomainWorkArtifactSubmissionLedgerPortV0 interface {
	HasDomainWorkArtifactSubmissionV0(context.Context, string) (bool, error)
	RecordDomainWorkArtifactSubmissionV0(context.Context, DomainWorkArtifactSubmissionRecordV0) error
}

type DomainWorkArtifactSubmissionRecordReaderPortV0 interface {
	ListDomainWorkArtifactSubmissionsV0(
		context.Context,
		DomainWorkArtifactSubmissionRecordFilterV0,
	) ([]DomainWorkArtifactSubmissionRecordV0, error)
}

type DomainWorkArtifactSubmissionBuildInputV0 struct {
	Run         orquestacoreworkflow.OrchestrationRunV0
	Task        orquestacoreworkflow.WorkflowTaskV0
	Record      orquestaappchange.AppChangeRecordV0
	Descriptor  orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
	Ack         orquestaruntimecodex.CodexAgentAckV0
	Observation orquestacionnucleoapp.AgentDeliveryObservationV0
	OccurredAt  string
}

type DomainWorkArtifactSubmissionRecordV0 struct {
	IdempotencyKey string   `json:"idempotency_key"`
	Status         string   `json:"status,omitempty"`
	RunRef         string   `json:"run_ref,omitempty"`
	TaskRef        string   `json:"task_ref,omitempty"`
	DeliveryRef    string   `json:"delivery_ref,omitempty"`
	DomainRef      string   `json:"domain_ref,omitempty"`
	JobRef         string   `json:"job_ref,omitempty"`
	ArtifactRef    string   `json:"artifact_ref,omitempty"`
	ArtifactType   string   `json:"artifact_type,omitempty"`
	CompleteJob    bool     `json:"complete_job,omitempty"`
	ReceiptRef     string   `json:"receipt_ref,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	IssueRefs      []string `json:"issue_refs,omitempty"`
	RecordedAt     string   `json:"recorded_at,omitempty"`
}

type DomainWorkArtifactSubmissionRecordFilterV0 struct {
	IdempotencyKey string
	RunRef         string
	TaskRef        string
	DeliveryRef    string
	Status         string
}

func normalizeDomainWorkDeliveryBridgeConfigV0(
	config DomainWorkDeliveryBridgeConfigV0,
) DomainWorkDeliveryBridgeConfigV0 {
	if config.Builder == nil {
		config.Builder = defaultDomainWorkArtifactSubmissionBuilderV0{}
	}
	if config.Ledger == nil {
		config.Ledger = NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	}
	return config
}

type InMemoryDomainWorkArtifactSubmissionLedgerV0 struct {
	mu      sync.Mutex
	records map[string]DomainWorkArtifactSubmissionRecordV0
}

func NewInMemoryDomainWorkArtifactSubmissionLedgerV0() *InMemoryDomainWorkArtifactSubmissionLedgerV0 {
	return &InMemoryDomainWorkArtifactSubmissionLedgerV0{
		records: map[string]DomainWorkArtifactSubmissionRecordV0{},
	}
}

func (ledger *InMemoryDomainWorkArtifactSubmissionLedgerV0) HasDomainWorkArtifactSubmissionV0(
	_ context.Context,
	idempotencyKey string,
) (bool, error) {
	if ledger == nil {
		return false, nil
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	_, ok := ledger.records[strings.TrimSpace(idempotencyKey)]
	return ok, nil
}

func (ledger *InMemoryDomainWorkArtifactSubmissionLedgerV0) RecordDomainWorkArtifactSubmissionV0(
	_ context.Context,
	record DomainWorkArtifactSubmissionRecordV0,
) error {
	if ledger == nil {
		return fmt.Errorf("domain_work_artifact_ledger_requerido")
	}
	record.IdempotencyKey = strings.TrimSpace(record.IdempotencyKey)
	if record.IdempotencyKey == "" {
		return fmt.Errorf("domain_work_artifact_idempotency_key_requerida")
	}
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.records[record.IdempotencyKey]; ok {
		merged, err := mergeDomainWorkArtifactSubmissionRecordV0(existing, record)
		if err != nil {
			return err
		}
		ledger.records[record.IdempotencyKey] = merged
		return nil
	}
	ledger.records[record.IdempotencyKey] = record
	return nil
}

func (ledger *InMemoryDomainWorkArtifactSubmissionLedgerV0) ListDomainWorkArtifactSubmissionsV0(
	_ context.Context,
	filter DomainWorkArtifactSubmissionRecordFilterV0,
) ([]DomainWorkArtifactSubmissionRecordV0, error) {
	if ledger == nil {
		return nil, nil
	}
	filter = normalizeDomainWorkArtifactSubmissionRecordFilterV0(filter)
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(ledger.records))
	for _, record := range ledger.records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if !domainWorkArtifactSubmissionRecordMatchesFilterV0(record, filter) {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func normalizeDomainWorkArtifactSubmissionRecordV0(
	record DomainWorkArtifactSubmissionRecordV0,
) DomainWorkArtifactSubmissionRecordV0 {
	record.IdempotencyKey = strings.TrimSpace(record.IdempotencyKey)
	record.Status = strings.TrimSpace(record.Status)
	record.RunRef = strings.TrimSpace(record.RunRef)
	record.TaskRef = strings.TrimSpace(record.TaskRef)
	record.DeliveryRef = strings.TrimSpace(record.DeliveryRef)
	record.DomainRef = strings.TrimSpace(record.DomainRef)
	record.JobRef = strings.TrimSpace(record.JobRef)
	record.ArtifactRef = strings.TrimSpace(record.ArtifactRef)
	record.ArtifactType = strings.TrimSpace(record.ArtifactType)
	record.ReceiptRef = strings.TrimSpace(record.ReceiptRef)
	record.EvidenceRefs = compactStringsV0(record.EvidenceRefs)
	record.IssueRefs = compactStringsV0(record.IssueRefs)
	record.RecordedAt = strings.TrimSpace(record.RecordedAt)
	return record
}

func normalizeDomainWorkArtifactSubmissionRecordFilterV0(
	filter DomainWorkArtifactSubmissionRecordFilterV0,
) DomainWorkArtifactSubmissionRecordFilterV0 {
	filter.IdempotencyKey = strings.TrimSpace(filter.IdempotencyKey)
	filter.RunRef = strings.TrimSpace(filter.RunRef)
	filter.TaskRef = strings.TrimSpace(filter.TaskRef)
	filter.DeliveryRef = strings.TrimSpace(filter.DeliveryRef)
	filter.Status = strings.TrimSpace(filter.Status)
	return filter
}

func domainWorkArtifactSubmissionRecordMatchesFilterV0(
	record DomainWorkArtifactSubmissionRecordV0,
	filter DomainWorkArtifactSubmissionRecordFilterV0,
) bool {
	return (filter.IdempotencyKey == "" || record.IdempotencyKey == filter.IdempotencyKey) &&
		(filter.RunRef == "" || record.RunRef == filter.RunRef) &&
		(filter.TaskRef == "" || record.TaskRef == filter.TaskRef) &&
		(filter.DeliveryRef == "" || record.DeliveryRef == filter.DeliveryRef) &&
		(filter.Status == "" || record.Status == filter.Status)
}
