package orquestadirectoragentfilesource

import (
	"context"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

const (
	DirectorAgentDecisionFileSchemaVersionV0   = "director_agent_decisions_file.v0"
	DefaultDirectorAgentDecisionFileMaxBytesV0 = 64 * 1024
)

type DirectorAgentDecisionFileDescriptorV0 struct {
	DescriptorRef string `json:"descriptor_ref,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	Path          string `json:"path"`
}

type DirectorAgentDecisionFileListRequestV0 struct {
	RunID          string   `json:"run_id"`
	PhaseArtifacts []string `json:"phase_artifacts,omitempty"`
	Deliveries     []string `json:"deliveries,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	RequestedBy    string   `json:"requested_by,omitempty"`
}

type DirectorAgentDecisionFileDescriptorProviderPortV0 interface {
	ListDirectorAgentDecisionFilesV0(
		ctx context.Context,
		request DirectorAgentDecisionFileListRequestV0,
	) ([]DirectorAgentDecisionFileDescriptorV0, error)
}

type DirectorAgentDecisionFileReaderPortV0 interface {
	ReadDirectorAgentDecisionFileV0(
		ctx context.Context,
		path string,
		maxBytes int,
	) ([]byte, error)
}

type DirectorAgentDecisionFileSourceV0 struct {
	DescriptorProvider DirectorAgentDecisionFileDescriptorProviderPortV0
	Reader             DirectorAgentDecisionFileReaderPortV0
	MaxBytes           int
}

type DirectorAgentDecisionFileEnvelopeV0 struct {
	SchemaVersion string                                          `json:"schema_version"`
	Decisions     []orquestadirectoragent.DirectorAgentDecisionV0 `json:"decisions"`
}

type DirectorAgentFileSourceIssueV0 struct {
	Field string
}

func (issue DirectorAgentFileSourceIssueV0) Error() string {
	return "director_agent_file_source_invalido: " + issue.Field
}
