package orquestaagentprocessregistry

import "context"

const (
	ErrAgentProcessRegistryInvalidV0  = "agent_process_registry_invalid"
	ErrAgentProcessRegistryNotFoundV0 = "agent_process_registry_not_found"
)

type AgentProcessRegistryPortV0 interface {
	RecordAgentProcessV0(context.Context, AgentProcessRegistryRecordV0) error
	ResolveAgentProcessV0(context.Context, string, string) (AgentProcessRegistryRecordV0, error)
}

type AgentProcessRegistryListPortV0 interface {
	ListAgentProcessesV0(context.Context, AgentProcessRegistryListFilterV0) ([]AgentProcessRegistryRecordV0, error)
}

type AgentProcessRegistryListFilterV0 struct {
	RunID string
}

type AgentProcessRegistryRecordV0 struct {
	RunID          string
	AgentRequestID string
	ProcessRef     string
	SessionRef     string
	LaunchRef      string
	ReadinessRef   string
	EvidenceRefs   []string
}

type ErrorV0 struct {
	Code    string
	Field   string
	Message string
}

func (err ErrorV0) Error() string {
	return err.Code
}
