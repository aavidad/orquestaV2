package orquestamcp

import (
	"context"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	MCPCodebaseStatusToolNameV0    = "orquesta.codebase.status.v0"
	MCPCodebaseStatusToolVersionV0 = "v0"
	MCPCodebaseStatusResourceURIV0 = "orquesta://contracts/codebase-status/v0"
	MCPCodebaseStatusHTTPPathV0    = "/api/v0/codebase/status"

	MCPCodebaseStatusHTTPErrorCodeV0             = "codebase_status_http_error"
	MCPCodebaseStatusHTTPNotConfiguredCodeV0     = "codebase_status_no_configurado"
	MCPCodebaseStatusHTTPExecutorErrorCodeV0     = "codebase_status_error"
	MCPCodebaseStatusHTTPUnsupportedPathCodeV0   = MCPPublicErrPathUnsupportedV0
	MCPCodebaseStatusHTTPUnsupportedMethodCodeV0 = MCPPublicErrMethodNotAllowedV0
)

type MCPCodebaseStatusToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPCodebaseStatusToolInputV0 struct {
	SchemaVersion         string                                    `json:"schema_version,omitempty"`
	RequestRef            string                                    `json:"request_ref,omitempty"`
	CorrelationID         string                                    `json:"correlation_id,omitempty"`
	RepositoryRef         string                                    `json:"repository_ref,omitempty"`
	ToolRef               string                                    `json:"tool_ref,omitempty"`
	ObservedAt            string                                    `json:"observed_at,omitempty"`
	IncludeTerminal       bool                                      `json:"include_terminal,omitempty"`
	DefaultCPUHighPercent int                                       `json:"default_cpu_high_percent,omitempty"`
	Observations          []MCPCodebaseStatusToolObservationInputV0 `json:"observations,omitempty"`
}

type MCPCodebaseStatusToolObservationInputV0 struct {
	ObservationRef string   `json:"observation_ref,omitempty"`
	LeaseRef       string   `json:"lease_ref,omitempty"`
	ObservedAt     string   `json:"observed_at,omitempty"`
	LastRequestAt  string   `json:"last_request_at,omitempty"`
	CPUPercent     int      `json:"cpu_percent,omitempty"`
	ActiveRequests int      `json:"active_requests,omitempty"`
	CPUHighPercent int      `json:"cpu_high_percent,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type MCPCodebaseStatusToolResultV0 = orquestacontext.CodeContextToolingStatusV0

type MCPCodebaseStatusToolExecutorV0 struct {
	Leases orquestacontext.CodeContextToolLeaseListPortV0
	Clock  func() time.Time
}

func MCPCodebaseStatusDescriptorV0() MCPCodebaseStatusToolDescriptorV0 {
	return MCPCodebaseStatusToolDescriptorV0{
		Name:        MCPCodebaseStatusToolNameV0,
		Version:     MCPCodebaseStatusToolVersionV0,
		InputSchema: "codebase_status:{schema_version?,request_ref?,repository_ref?,tool_ref?,observed_at?,include_terminal?,default_cpu_high_percent?,observations?[{lease_ref,observed_at?,cpu_percent?,active_requests?,cpu_high_percent?,evidence_refs?}]}",
		Output:      "code_context_tooling_status:{estado,total_leases,active_leases,stop_requested,high_cpu_stop_requested,entries,next_actions,issues}",
		ResourceURI: MCPCodebaseStatusResourceURIV0,
		Invariantes: []string{
			"publica estado compacto de leases del broker central de contexto",
			"no arranca codebase-memory-mcp ni mata procesos desde este tool",
			"no publica PID, HOME, tokens, command line ni rutas privadas de runtime",
			"la parada real queda en watchdog de servidor/composicion con owner marker",
			"las observaciones de owner marker corrigen la evaluacion publica sin ejecutar parada desde MCP",
		},
	}
}

func (executor MCPCodebaseStatusToolExecutorV0) Execute(
	ctx context.Context,
	input MCPCodebaseStatusToolInputV0,
) (MCPCodebaseStatusToolResultV0, error) {
	input = normalizeMCPCodebaseStatusInputV0(input)
	if input.SchemaVersion != "" && input.SchemaVersion != orquestacontext.CodeContextToolingStatusSchemaVersionV0 {
		return newMCPCodebaseStatusErrorResultV0(input, orquestacontext.ErrCodeContextSchemaNoSoportadoV0, "schema_version"), nil
	}
	if executor.Leases == nil {
		return newMCPCodebaseStatusErrorResultV0(input, orquestacontext.ErrCodeContextProveedorNoConfiguradoV0, "leases"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	filter := orquestacontext.CodeContextToolLeaseListFilterV0{
		RepositoryRef: input.RepositoryRef,
		ToolRef:       input.ToolRef,
	}
	if !input.IncludeTerminal {
		filter.Status = orquestacontext.CodeContextToolLeaseStatusActiveV0
	}
	leases, err := executor.Leases.ListCodeContextToolLeasesV0(ctx, filter)
	if err != nil {
		return newMCPCodebaseStatusErrorResultV0(input, orquestacontext.ErrCodeContextLeaseErrorV0, "leases"), nil
	}
	request := orquestacontext.CodeContextToolingStatusRequestV0{
		RequestRef:            input.RequestRef,
		CorrelationID:         input.CorrelationID,
		RepositoryRef:         input.RepositoryRef,
		ObservedAt:            firstNonEmptyMCPV0(input.ObservedAt, executor.nowV0().UTC().Format(time.RFC3339)),
		Leases:                leases,
		Observations:          codebaseStatusToolObservationsMCPV0(input.Observations),
		DefaultCPUHighPercent: input.DefaultCPUHighPercent,
		EvidenceRefs:          []string{input.RequestRef},
	}
	status, buildErr := orquestacontext.BuildCodeContextToolingStatusV0(request)
	if buildErr != nil {
		return status, nil
	}
	return status, nil
}

func normalizeMCPCodebaseStatusInputV0(input MCPCodebaseStatusToolInputV0) MCPCodebaseStatusToolInputV0 {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.RepositoryRef = strings.TrimSpace(input.RepositoryRef)
	input.ToolRef = strings.TrimSpace(input.ToolRef)
	input.ObservedAt = strings.TrimSpace(input.ObservedAt)
	input.DefaultCPUHighPercent = nonNegativeMCPCodebaseStatusIntV0(input.DefaultCPUHighPercent)
	for idx := range input.Observations {
		input.Observations[idx].ObservationRef = strings.TrimSpace(input.Observations[idx].ObservationRef)
		input.Observations[idx].LeaseRef = strings.TrimSpace(input.Observations[idx].LeaseRef)
		input.Observations[idx].ObservedAt = strings.TrimSpace(input.Observations[idx].ObservedAt)
		input.Observations[idx].LastRequestAt = strings.TrimSpace(input.Observations[idx].LastRequestAt)
		input.Observations[idx].CPUPercent = nonNegativeMCPCodebaseStatusIntV0(input.Observations[idx].CPUPercent)
		input.Observations[idx].ActiveRequests = nonNegativeMCPCodebaseStatusIntV0(input.Observations[idx].ActiveRequests)
		input.Observations[idx].CPUHighPercent = nonNegativeMCPCodebaseStatusIntV0(input.Observations[idx].CPUHighPercent)
		input.Observations[idx].EvidenceRefs = compactStringsMCPV0(input.Observations[idx].EvidenceRefs)
	}
	return input
}

func codebaseStatusToolObservationsMCPV0(
	observations []MCPCodebaseStatusToolObservationInputV0,
) []orquestacontext.CodeContextToolLeaseObservationV0 {
	if len(observations) == 0 {
		return nil
	}
	out := make([]orquestacontext.CodeContextToolLeaseObservationV0, 0, len(observations))
	for _, observation := range observations {
		leaseRef := strings.TrimSpace(observation.LeaseRef)
		if leaseRef == "" {
			continue
		}
		out = append(out, orquestacontext.CodeContextToolLeaseObservationV0{
			ObservationRef: observation.ObservationRef,
			ObservedAt:     observation.ObservedAt,
			Lease: orquestacontext.CodeContextToolLeaseV0{
				LeaseRef: leaseRef,
			},
			LastRequestAt:  observation.LastRequestAt,
			CPUPercent:     observation.CPUPercent,
			ActiveRequests: observation.ActiveRequests,
			CPUHighPercent: observation.CPUHighPercent,
			EvidenceRefs:   compactStringsMCPV0(observation.EvidenceRefs),
		})
	}
	return out
}

func nonNegativeMCPCodebaseStatusIntV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func (executor MCPCodebaseStatusToolExecutorV0) nowV0() time.Time {
	if executor.Clock == nil {
		return time.Now().UTC()
	}
	return executor.Clock().UTC()
}

func newMCPCodebaseStatusErrorResultV0(
	input MCPCodebaseStatusToolInputV0,
	code string,
	field string,
) MCPCodebaseStatusToolResultV0 {
	return orquestacontext.CodeContextToolingStatusV0{
		SchemaVersion: orquestacontext.CodeContextToolingStatusSchemaVersionV0,
		Estado:        orquestacontext.CodeContextToolingEstadoErrorV0,
		RequestRef:    strings.TrimSpace(input.RequestRef),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
		RepositoryRef: strings.TrimSpace(input.RepositoryRef),
		ObservedAt:    strings.TrimSpace(input.ObservedAt),
		Issues: []orquestacontext.CodeContextIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}
