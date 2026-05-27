package orquestamcp

import orquestaobservability "orquesta/modulos/orquesta-observability"

const (
	MCPWorkspaceTimelineResourceNameV0    = MCPWorkspaceTimelineToolNameV0
	MCPWorkspaceTimelineResourceVersionV0 = MCPWorkspaceTimelineToolVersionV0
	MCPWorkspaceTimelineResourceURIV0     = "orquesta://observability/workspace-timeline/v0"
	MCPWorkspaceTimelineHTTPPathV0        = "/api/v0/observability/workspace-timeline"
	MCPWorkspaceTimelineContentTypeV0     = "application/vnd.orquesta.workspace-timeline.v0+json"
	MCPWorkspaceTimelineEndpointV0        = MCPWorkspaceTimelineHTTPPathV0
)

type MCPWorkspaceTimelineResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPWorkspaceTimelineResourceV0 struct {
	URI                 string   `json:"uri"`
	Version             string   `json:"version"`
	SummaryKey          string   `json:"summary_key"`
	ContractResource    string   `json:"contract_resource_uri"`
	RecommendedEndpoint string   `json:"recommended_endpoint"`
	AllowedSources      []string `json:"allowed_sources"`
	InputShape          string   `json:"input_shape"`
	OutputShape         string   `json:"output_shape"`
	Guardrails          []string `json:"guardrails"`
	PublicErrors        []string `json:"errores_publicos"`
}

func MCPWorkspaceTimelineDescriptorV0() MCPWorkspaceTimelineResourceDescriptorV0 {
	return MCPWorkspaceTimelineResourceDescriptorV0{
		Name:        MCPWorkspaceTimelineResourceNameV0,
		Version:     MCPWorkspaceTimelineResourceVersionV0,
		URI:         MCPWorkspaceTimelineResourceURIV0,
		ContentType: MCPWorkspaceTimelineContentTypeV0,
		SummaryKey:  "mcp.resources.workspace_timeline.summary.v0",
	}
}

func NewMCPWorkspaceTimelineResourceV0() MCPWorkspaceTimelineResourceV0 {
	return MCPWorkspaceTimelineResourceV0{
		URI:                 MCPWorkspaceTimelineResourceURIV0,
		Version:             MCPWorkspaceTimelineResourceVersionV0,
		SummaryKey:          "mcp.resources.workspace_timeline.summary.v0",
		ContractResource:    "orquesta://contracts/workspace-timeline-query/v0",
		RecommendedEndpoint: MCPWorkspaceTimelineEndpointV0,
		AllowedSources: []string{
			orquestaobservability.WorkspaceTimelineSourceEventsV0,
			orquestaobservability.WorkspaceTimelineSourceAuditV0,
			orquestaobservability.WorkspaceTimelineSourceRunQueueV0,
			orquestaobservability.WorkspaceTimelineSourceRuntimeProgressV0,
			orquestaobservability.WorkspaceTimelineSourceGitStatsV0,
			orquestaobservability.WorkspaceTimelineSourceUsageCostV0,
			orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0,
		},
		InputShape:  "query:{request_id,correlation_id,consumer,locale,scope=workspace,agent_ref?,project_ref?,task_ref?,time_window,page,sources}",
		OutputShape: "timeline:{sources[].status,items[],privacy}",
		Guardrails: []string{
			"read_only_por_puerto_workspace_timeline",
			"sin_shell_git_local_runtime_filesystem_directo",
			"fuente_ausente_devuelve_not_available",
			"sin_secretos_prompts_completions_transcripts_home_tokens",
		},
		PublicErrors: []string{
			orquestaobservability.ErrWorkspaceTimelineQueryInvalidaV0,
			orquestaobservability.ErrWorkspaceTimelineNoDisponibleV0,
			orquestaobservability.ErrConsultaDemasiadoAmpliaV0,
			orquestaobservability.ErrReferenciaNoOpacaV0,
			orquestaobservability.ErrSecretoDetectadoV0,
			orquestaobservability.ErrTranscriptNoPermitidoV0,
		},
	}
}
