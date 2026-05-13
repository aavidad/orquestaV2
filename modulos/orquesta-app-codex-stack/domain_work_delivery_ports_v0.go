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
	IdempotencyKey string `json:"idempotency_key"`
	RunRef         string `json:"run_ref,omitempty"`
	TaskRef        string `json:"task_ref,omitempty"`
	DeliveryRef    string `json:"delivery_ref,omitempty"`
	ReceiptRef     string `json:"receipt_ref,omitempty"`
	RecordedAt     string `json:"recorded_at,omitempty"`
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
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	ledger.records[record.IdempotencyKey] = record
	return nil
}
