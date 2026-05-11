package orquestamcp

const (
	MCPBootstrapResourceNameV0    = "orquesta.director.bootstrap_appspec.v0"
	MCPBootstrapResourceVersionV0 = "v0"
	MCPBootstrapResourceURIV0     = "orquesta://contracts/bootstrap-proyecto-desde-appspec/v0"
	MCPBootstrapContentTypeV0     = "application/vnd.orquesta.bootstrap-appspec.v0+json"
)

type MCPBootstrapResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPBootstrapResourceV0 struct {
	URI                string                 `json:"uri"`
	Version            string                 `json:"version"`
	SummaryKey         string                 `json:"summary_key"`
	Owner              string                 `json:"owner"`
	CanonicalRefs      []string               `json:"canonical_refs"`
	ToolName           string                 `json:"tool_name"`
	Input              MCPBootstrapShapeV0    `json:"input"`
	Output             MCPBootstrapShapeV0    `json:"output"`
	ComposedContracts  []string               `json:"composed_contracts"`
	PublicErrors       []string               `json:"errores_publicos"`
	Guardrails         []string               `json:"guardrails"`
	CompactResultHints MCPBootstrapResultHint `json:"compact_result_hints"`
}

type MCPBootstrapShapeV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	Shape         string   `json:"shape"`
	Required      []string `json:"required,omitempty"`
	OpaqueRefs    []string `json:"opaque_refs,omitempty"`
}

type MCPBootstrapResultHint struct {
	Registro []string `json:"registro"`
	Workflow []string `json:"workflow"`
	Excluded []string `json:"excluded"`
}

func MCPBootstrapDescriptorV0() MCPBootstrapResourceDescriptorV0 {
	return MCPBootstrapResourceDescriptorV0{
		Name:        MCPBootstrapResourceNameV0,
		Version:     MCPBootstrapResourceVersionV0,
		URI:         MCPBootstrapResourceURIV0,
		ContentType: MCPBootstrapContentTypeV0,
		SummaryKey:  "mcp.resources.bootstrap_appspec.summary.v0",
	}
}

func NewMCPBootstrapResourceV0() MCPBootstrapResourceV0 {
	return MCPBootstrapResourceV0{
		URI:        MCPBootstrapResourceURIV0,
		Version:    MCPBootstrapResourceVersionV0,
		SummaryKey: "mcp.resources.bootstrap_appspec.summary.v0",
		Owner:      "orquesta-director",
		CanonicalRefs: compactStringsMCPV0([]string{
			"../CONTRATOS.md#BootstrapProyectoDesdeAppSpec-v0",
			"orquesta-director/docs/contratos.md",
		}),
		ToolName: MCPBootstrapToolNameV0,
		Input: MCPBootstrapShapeV0{
			Shape:      "envelope:{request_id?,correlation_id?,respuesta?,command:BootstrapProyectoDesdeAppSpecCommandV0}",
			Required:   []string{"command.idempotency_key", "command.app_spec", "command.backlog", "command.occurred_at"},
			OpaqueRefs: []string{"request_id", "correlation_id", "project_ref", "app_spec_ref", "run_id"},
		},
		Output: MCPBootstrapShapeV0{
			Shape:      "ok:{registro_aceptado compacto,project_ref,app_spec_ref,workflow compacto}|error:{errores_publicos}",
			OpaqueRefs: []string{"project_ref", "app_spec_ref", "run_id", "command_id", "event_id"},
		},
		ComposedContracts: []string{
			"SolicitarNuevaApp v0",
			"RegistrarProyectoDesdeAppSpec v0",
			"StartRunFromAppSpecV0",
			"OrchestrationCommandV0",
		},
		PublicErrors: []string{
			"director_bootstrap_invalido",
			"errores_publicos_de_orquesta_core",
			"errores_publicos_de_orquesta_core_workflow",
		},
		Guardrails: []string{
			"tool_puro_sin_persistencia_runtime_db_filesystem_productivo",
			"invoca_solo_orquesta_director_bootstrap_v0",
			"usa_dtos_publicos_del_director",
			"resultado_compacto_para_ia_sin_payloads_completos",
		},
		CompactResultHints: MCPBootstrapResultHint{
			Registro: []string{"registro_id", "project_ref", "app_spec_ref", "contadores", "warnings"},
			Workflow: []string{"start_run_command_ref", "eventos_emitidos", "outbox_count", "idempotent"},
			Excluded: []string{"entrada_extensa", "plan_extenso", "payloads_json", "detalles_operativos"},
		},
	}
}
