package orquestamcp

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	MCPOperationalStatusResourceNameV0    = "orquesta.observability.operational_status.v0"
	MCPOperationalStatusResourceVersionV0 = "v0"
	MCPOperationalStatusResourceURIV0     = "orquesta://observability/operational-status/v0"
	MCPOperationalStatusContentTypeV0     = "application/vnd.orquesta.operational-status.v0+json"
	MCPOperationalStatusEndpointV0        = "/api/v0/operational-status/query"
)

type MCPOperationalStatusResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPOperationalStatusResourceV0 struct {
	URI                 string                              `json:"uri"`
	Version             string                              `json:"version"`
	SummaryKey          string                              `json:"summary_key"`
	ContractResource    string                              `json:"contract_resource_uri"`
	RecommendedEndpoint string                              `json:"recommended_endpoint"`
	CanonicalRefs       []string                            `json:"canonical_refs"`
	AllowedConsumers    []MCPOperationalStatusConsumerV0    `json:"allowed_consumers"`
	AllowedScopes       []string                            `json:"allowed_scopes"`
	AllowedSections     []string                            `json:"allowed_sections"`
	Request             MCPOperationalStatusRequestGuideV0  `json:"request"`
	Response            MCPOperationalStatusResponseGuideV0 `json:"response"`
	PublicErrors        []string                            `json:"errores_publicos"`
	Guardrails          []string                            `json:"guardrails"`
}

type MCPOperationalStatusConsumerV0 struct {
	Module  string `json:"module"`
	Channel string `json:"channel"`
}

type MCPOperationalStatusRequestGuideV0 struct {
	SchemaVersion string   `json:"schema_version"`
	InputShape    string   `json:"input_shape"`
	OpaqueRefs    []string `json:"opaque_refs"`
	Defaults      []string `json:"defaults"`
}

type MCPOperationalStatusResponseGuideV0 struct {
	SchemaVersion string   `json:"schema_version"`
	Estados       []string `json:"estados"`
	Sections      []string `json:"sections"`
	PrivacyFlags  []string `json:"privacy_flags"`
}

func MCPOperationalStatusDescriptorV0() MCPOperationalStatusResourceDescriptorV0 {
	return MCPOperationalStatusResourceDescriptorV0{
		Name:        MCPOperationalStatusResourceNameV0,
		Version:     MCPOperationalStatusResourceVersionV0,
		URI:         MCPOperationalStatusResourceURIV0,
		ContentType: MCPOperationalStatusContentTypeV0,
		SummaryKey:  "mcp.resources.operational_status.summary.v0",
	}
}

func NewMCPOperationalStatusResourceV0() MCPOperationalStatusResourceV0 {
	return MCPOperationalStatusResourceV0{
		URI:                 MCPOperationalStatusResourceURIV0,
		Version:             MCPOperationalStatusResourceVersionV0,
		SummaryKey:          "mcp.resources.operational_status.summary.v0",
		ContractResource:    "orquesta://contracts/operational-status-query/v0",
		RecommendedEndpoint: MCPOperationalStatusEndpointV0,
		CanonicalRefs: compactStringsMCPV0([]string{
			"../CONTRATOS.md",
			"orquesta-observability/docs/contratos.md",
			"orquesta-cli/operational_status_client_v0.go",
		}),
		AllowedConsumers: []MCPOperationalStatusConsumerV0{
			{Module: "orquesta-cli", Channel: orquestaobservability.OperationalStatusConsumerCLIChannelV0},
			{Module: "orquesta-mcp", Channel: orquestaobservability.OperationalStatusConsumerMCPChannelV0},
			{Module: "orquesta-web", Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0},
			{Module: "orquesta-core", Channel: orquestaobservability.OperationalStatusConsumerCoreChannelV0},
		},
		AllowedScopes: []string{
			orquestaobservability.OperationalStatusScopeSistemaV0,
			orquestaobservability.OperationalStatusScopeProyectoV0,
			orquestaobservability.OperationalStatusScopeFlujoV0,
			orquestaobservability.OperationalStatusScopeTareaV0,
			orquestaobservability.OperationalStatusScopeRuntimeV0,
			orquestaobservability.OperationalStatusScopeCapacityV0,
			orquestaobservability.OperationalStatusScopeReviewV0,
		},
		AllowedSections: []string{
			orquestaobservability.OperationalStatusSectionEstadoV0,
			orquestaobservability.OperationalStatusSectionProgresoV0,
			orquestaobservability.OperationalStatusSectionBloqueosV0,
			orquestaobservability.OperationalStatusSectionSaludV0,
			orquestaobservability.OperationalStatusSectionActividadRecienteV0,
			orquestaobservability.OperationalStatusSectionContadoresV0,
			orquestaobservability.OperationalStatusSectionReferenciasV0,
		},
		Request: MCPOperationalStatusRequestGuideV0{
			SchemaVersion: orquestaobservability.OperationalStatusQuerySchemaVersionV0,
			InputShape:    "query:{request_id,correlation_id,consumer,module/channel,locale,scope,subject_ref?,trace_ref?,time_window?,include_sections,limit,freshness?}",
			OpaqueRefs:    []string{"request_id", "correlation_id", "subject_ref", "trace_ref", "watermark_ref"},
			Defaults:      []string{"locale=es si el adaptador no recibe otro valor", "include_sections acotado", "limit compacto y nunca acceso directo a DB"},
		},
		Response: MCPOperationalStatusResponseGuideV0{
			SchemaVersion: orquestaobservability.DiagnosticoCompactoSchemaVersionV0,
			Estados: []string{
				orquestaobservability.DiagnosticoEstadoOKV0,
				orquestaobservability.DiagnosticoEstadoDegradedV0,
				orquestaobservability.DiagnosticoEstadoBlockedV0,
				orquestaobservability.DiagnosticoEstadoFailedV0,
				orquestaobservability.DiagnosticoEstadoUnknownV0,
			},
			Sections:     []string{"progreso", "salud", "bloqueos", "actividad_reciente", "contadores", "referencias", "warnings"},
			PrivacyFlags: []string{"contains_secret=false", "contains_transcript=false", "contains_prompt=false", "contains_completion=false", "contains_connection_detail=false"},
		},
		PublicErrors: []string{
			orquestaobservability.ErrOperationalStatusQueryInvalidaV0,
			orquestaobservability.ErrConsumidorNoAutorizadoV0,
			orquestaobservability.ErrScopeNoSoportadoV0,
			orquestaobservability.ErrReferenciaNoOpacaV0,
			orquestaobservability.ErrConsultaDemasiadoAmpliaV0,
			orquestaobservability.ErrProyeccionNoDisponibleV0,
			orquestaobservability.ErrDiagnosticoNoDisponibleV0,
			orquestaobservability.ErrFrescuraNoGarantizadaV0,
			"secreto_detectado",
			"transcript_no_permitido",
		},
		Guardrails: []string{
			"resource_puro_sin_tool_ni_servidor_mcp",
			"read_only_sin_db_runtime_filesystem_productivo",
			"diagnostico_compacto_sin_secretos_transcripts_sql_dsn_home",
			"contrato_canonico_en_observability_y_contratos_globales",
		},
	}
}

func MCPOperationalStatusConsumerByKeyV0(value string) (MCPOperationalStatusConsumerV0, bool) {
	needle := normalizeOperationalStatusLookupMCPV0(value)
	for _, item := range NewMCPOperationalStatusResourceV0().AllowedConsumers {
		if needle == normalizeOperationalStatusLookupMCPV0(item.Module) ||
			needle == normalizeOperationalStatusLookupMCPV0(item.Channel) ||
			needle == normalizeOperationalStatusLookupMCPV0(item.Module+"/"+item.Channel) {
			return item, true
		}
	}
	return MCPOperationalStatusConsumerV0{}, false
}

func normalizeOperationalStatusLookupMCPV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}
