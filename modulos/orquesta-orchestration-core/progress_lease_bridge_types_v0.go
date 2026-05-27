package orquestacionnucleoapp

import (
	"context"
	"sync"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type AgentProgressLeasePolicyProviderPortV0 interface {
	BuildAgentProgressLeasePolicyV0(
		ctx context.Context,
		request AgentProgressLeasePolicyRequestV0,
	) (AgentProgressLeasePolicyV0, bool, error)
}

type AgentProgressLeasePolicyRequestV0 struct {
	Run           orquestacoreworkflow.OrchestrationRunV0 `json:"run"`
	Observation   AgentProgressObservationV0              `json:"observation"`
	OccurredAt    string                                  `json:"occurred_at"`
	CorrelationID string                                  `json:"correlation_id,omitempty"`
	EvidenceRefs  []string                                `json:"evidence_refs,omitempty"`
}

type AgentProgressLeasePolicyV0 struct {
	LeaseRef         string                                `json:"lease_ref,omitempty"`
	LaunchObservedAt string                                `json:"launch_observed_at,omitempty"`
	Policy           orquestacoreleases.AgentLeasePolicyV0 `json:"policy"`
	EvidenceRefs     []string                              `json:"evidence_refs,omitempty"`
}

type StaticAgentProgressLeasePolicyProviderV0 struct {
	Policy           orquestacoreleases.AgentLeasePolicyV0
	LeaseRef         string
	LaunchObservedAt string
	EvidenceRefs     []string
}

type AgentProgressLeaseBridgeV0 struct {
	ProgressSource AgentProgressObservationProviderPortV0
	PolicySource   AgentProgressLeasePolicyProviderPortV0

	mu    sync.Mutex
	cache agentProgressLeaseBridgeCacheV0
}

type agentProgressLeaseBridgeCacheV0 struct {
	Key          string
	Valid        bool
	Observations []AgentProgressObservationV0
}

type AgentProgressLeaseAssessmentInputV0 struct {
	AssessmentRef    string
	LeaseRef         string
	ObservedAt       string
	LaunchObservedAt string
	Report           AgentProgressObservationV0
	Policy           orquestacoreleases.AgentLeasePolicyV0
	EvidenceRefs     []string
}
