package orquestaagentprocessregistrymemory

import (
	"context"
	"sort"
	"sync"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
)

type AgentProcessRecordV0 = orquestaagentprocessregistry.AgentProcessRegistryRecordV0

var _ orquestaagentprocessregistry.AgentProcessRegistryPortV0 = (*InMemoryAgentProcessRegistryV0)(nil)
var _ orquestaagentprocessregistry.AgentProcessRegistryListPortV0 = (*InMemoryAgentProcessRegistryV0)(nil)

type InMemoryAgentProcessRegistryV0 struct {
	mu      sync.Mutex
	records map[agentProcessRegistryKeyV0]AgentProcessRecordV0
}

func NewInMemoryAgentProcessRegistryV0() *InMemoryAgentProcessRegistryV0 {
	return &InMemoryAgentProcessRegistryV0{
		records: map[agentProcessRegistryKeyV0]AgentProcessRecordV0{},
	}
}

func (registry *InMemoryAgentProcessRegistryV0) RecordAgentProcessV0(
	ctx context.Context,
	record AgentProcessRecordV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	record = orquestaagentprocessregistry.NormalizeAgentProcessRegistryRecordV0(record)
	if err := orquestaagentprocessregistry.ValidateAgentProcessRegistryRecordV0(record); err != nil {
		return err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.ensureRecordsLockedV0()
	key := agentProcessRegistryKeyFromRecordV0(record)
	existing, ok := registry.records[key]
	if ok {
		if agentProcessRecordEqualV0(existing, record) {
			return nil
		}
		return registryMemoryErrorV0(
			orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0,
			"agent_process_registry",
			"agent_process conflict",
		)
	}
	registry.records[key] = record
	return nil
}

func (registry *InMemoryAgentProcessRegistryV0) ResolveAgentProcessV0(
	ctx context.Context,
	runID string,
	agentRequestID string,
) (AgentProcessRecordV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AgentProcessRecordV0{}, err
	}
	runID, agentRequestID = orquestaagentprocessregistry.NormalizeAgentProcessRegistryLookupV0(runID, agentRequestID)
	key := agentProcessRegistryKeyV0{
		runID:          runID,
		agentRequestID: agentRequestID,
	}
	if err := validateAgentProcessLookupV0(key); err != nil {
		return AgentProcessRecordV0{}, err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.ensureRecordsLockedV0()
	record, ok := registry.records[key]
	if !ok {
		return AgentProcessRecordV0{}, registryMemoryErrorV0(
			orquestaagentprocessregistry.ErrAgentProcessRegistryNotFoundV0,
			"agent_process_registry",
			"agent_process no encontrado",
		)
	}
	return record, nil
}

func (registry *InMemoryAgentProcessRegistryV0) ListAgentProcessesV0(
	ctx context.Context,
	filter orquestaagentprocessregistry.AgentProcessRegistryListFilterV0,
) ([]AgentProcessRecordV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filter = orquestaagentprocessregistry.NormalizeAgentProcessRegistryListFilterV0(filter)
	if err := orquestaagentprocessregistry.ValidateAgentProcessRegistryListFilterV0(filter); err != nil {
		return nil, err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.ensureRecordsLockedV0()
	records := make([]AgentProcessRecordV0, 0, len(registry.records))
	for _, record := range registry.records {
		if filter.RunID != "" && record.RunID != filter.RunID {
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i int, j int) bool {
		if records[i].RunID != records[j].RunID {
			return records[i].RunID < records[j].RunID
		}
		return records[i].AgentRequestID < records[j].AgentRequestID
	})
	return records, nil
}

func validateAgentProcessLookupV0(key agentProcessRegistryKeyV0) error {
	return orquestaagentprocessregistry.ValidateAgentProcessRegistryLookupV0(key.runID, key.agentRequestID)
}

func registryMemoryErrorV0(code string, field string, message string) orquestaagentprocessregistry.ErrorV0 {
	return orquestaagentprocessregistry.ErrorV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}

func (registry *InMemoryAgentProcessRegistryV0) ensureRecordsLockedV0() {
	if registry.records == nil {
		registry.records = map[agentProcessRegistryKeyV0]AgentProcessRecordV0{}
	}
}

type agentProcessRegistryKeyV0 struct {
	runID          string
	agentRequestID string
}

func agentProcessRegistryKeyFromRecordV0(
	record AgentProcessRecordV0,
) agentProcessRegistryKeyV0 {
	return agentProcessRegistryKeyV0{
		runID:          record.RunID,
		agentRequestID: record.AgentRequestID,
	}
}

func agentProcessRecordEqualV0(
	left AgentProcessRecordV0,
	right AgentProcessRecordV0,
) bool {
	if left.RunID != right.RunID ||
		left.AgentRequestID != right.AgentRequestID ||
		left.ProcessRef != right.ProcessRef ||
		left.SessionRef != right.SessionRef ||
		left.LaunchRef != right.LaunchRef ||
		left.ReadinessRef != right.ReadinessRef ||
		len(left.EvidenceRefs) != len(right.EvidenceRefs) {
		return false
	}
	for i := range left.EvidenceRefs {
		if left.EvidenceRefs[i] != right.EvidenceRefs[i] {
			return false
		}
	}
	return true
}
