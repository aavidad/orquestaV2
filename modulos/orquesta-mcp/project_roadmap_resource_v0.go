package orquestamcp

const (
	MCPProjectRoadmapResourceNameV0    = "orquesta.project.roadmap.v0"
	MCPProjectRoadmapResourceVersionV0 = "v0"
	MCPProjectRoadmapResourceURIV0     = "orquesta://project/roadmap/v0"
	MCPProjectRoadmapContentTypeV0     = "application/vnd.orquesta.project.roadmap.v0+json"
)

type MCPProjectRoadmapResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPProjectRoadmapResourceV0 struct {
	URI           string                        `json:"uri"`
	Version       string                        `json:"version"`
	SummaryKey    string                        `json:"summary_key"`
	Scope         string                        `json:"scope"`
	Freshness     MCPResourceFreshnessV0        `json:"freshness"`
	CanonicalRefs []string                      `json:"canonical_refs"`
	Roadmap       []MCPProjectRoadmapItemV0     `json:"roadmap"`
	Decisions     []MCPProjectDecisionCompactV0 `json:"decisiones"`
	Guardrails    []string                      `json:"guardrails"`
}

type MCPProjectRoadmapItemV0 struct {
	ID            string   `json:"id"`
	Area          string   `json:"area"`
	Status        string   `json:"estado"`
	Owner         string   `json:"owner"`
	Focus         string   `json:"focus"`
	SummaryKey    string   `json:"summary_key"`
	ProgressKey   string   `json:"progress_key"`
	Contracts     []string `json:"contracts"`
	Dependencies  []string `json:"dependencies,omitempty"`
	Guardrails    []string `json:"guardrails"`
	CanonicalRefs []string `json:"canonical_refs"`
	BacklogRefs   []string `json:"backlog_refs,omitempty"`
	Verification  []string `json:"verification,omitempty"`
}

type MCPProjectDecisionCompactV0 struct {
	ID            string   `json:"id"`
	Status        string   `json:"estado"`
	Decision      string   `json:"decision"`
	DecisionKey   string   `json:"decision_key"`
	Motivo        string   `json:"motivo"`
	MotivoKey     string   `json:"motivo_key"`
	AppliesTo     []string `json:"applies_to"`
	Guardrails    []string `json:"guardrails"`
	CanonicalRefs []string `json:"canonical_refs"`
}

func MCPProjectRoadmapDescriptorV0() MCPProjectRoadmapResourceDescriptorV0 {
	return MCPProjectRoadmapResourceDescriptorV0{
		Name:        MCPProjectRoadmapResourceNameV0,
		Version:     MCPProjectRoadmapResourceVersionV0,
		URI:         MCPProjectRoadmapResourceURIV0,
		ContentType: MCPProjectRoadmapContentTypeV0,
		SummaryKey:  "mcp.project.roadmap.summary.v0",
	}
}

func NewMCPProjectRoadmapResourceV0() MCPProjectRoadmapResourceV0 {
	items := mcpProjectRoadmapSourcesV0()
	roadmap := make([]MCPProjectRoadmapItemV0, 0, len(items))
	for _, item := range items {
		roadmap = append(roadmap, toMCPProjectRoadmapItemV0(item))
	}

	decisions := mcpProjectDecisionSourcesV0()
	compactDecisions := make([]MCPProjectDecisionCompactV0, 0, len(decisions))
	for _, decision := range decisions {
		compactDecisions = append(compactDecisions, toMCPProjectDecisionCompactV0(decision))
	}

	return MCPProjectRoadmapResourceV0{
		URI:        MCPProjectRoadmapResourceURIV0,
		Version:    MCPProjectRoadmapResourceVersionV0,
		SummaryKey: "mcp.project.roadmap.summary.v0",
		Scope:      "orquesta-nucleo-reutilizable",
		Freshness:  mcpResourceFreshnessV0(mcpBacklogT25RefsV0(), mcpMCPResourceVerificationV0()),
		CanonicalRefs: []string{
			"docs/estado_actual_2026-05-17.md",
			"docs/guia_nucleo_orquestacion_2026-05-17.md",
			"modulos/orquesta-core-workflow/docs/contratos.md",
			"modulos/orquesta-orchestration-core/docs/contratos.md",
		},
		Roadmap:   roadmap,
		Decisions: compactDecisions,
		Guardrails: []string{
			"roadmap_estatico_con_freshness_y_refs_vivas",
			"sin_db_cli_runtime_filesystem_productivo",
			"detalle_canonico_en_docs_vigentes_y_modulos_propietarios",
			"mcp_es_adaptador_de_lectura_compacta",
		},
	}
}
