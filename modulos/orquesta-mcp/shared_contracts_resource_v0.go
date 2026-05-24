package orquestamcp

import "strings"

const (
	MCPSharedContractsResourceNameV0    = "orquesta.contracts.shared.v0"
	MCPSharedContractsResourceVersionV0 = "v0"
	MCPSharedContractsResourceURIV0     = "orquesta://contracts/shared/v0"
	MCPSharedContractsContentTypeV0     = "application/vnd.orquesta.contracts.shared.v0+json"
)

type MCPContractResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPSharedContractsResourceV0 struct {
	URI             string                       `json:"uri"`
	Version         string                       `json:"version"`
	SummaryKey      string                       `json:"summary_key"`
	Freshness       MCPResourceFreshnessV0       `json:"freshness"`
	CanonicalSource string                       `json:"canonical_source"`
	Contracts       []MCPSharedContractCompactV0 `json:"contracts"`
	Guardrails      []string                     `json:"guardrails"`
}

type MCPSharedContractCompactV0 struct {
	Contract      string   `json:"contract"`
	Version       string   `json:"version"`
	ResourceURI   string   `json:"resource_uri"`
	Owner         string   `json:"owner"`
	MCPRole       string   `json:"mcp_role"`
	SummaryKey    string   `json:"summary_key"`
	CanonicalRefs []string `json:"canonical_refs"`
	Input         string   `json:"input"`
	Output        string   `json:"output"`
	PublicErrors  []string `json:"errores_publicos,omitempty"`
	Guardrails    []string `json:"guardrails"`
	ProgressKey   string   `json:"progress_key"`
	BacklogRefs   []string `json:"backlog_refs,omitempty"`
	Verification  []string `json:"verification,omitempty"`
}

func MCPSharedContractsDescriptorV0() MCPContractResourceDescriptorV0 {
	return MCPContractResourceDescriptorV0{
		Name:        MCPSharedContractsResourceNameV0,
		Version:     MCPSharedContractsResourceVersionV0,
		URI:         MCPSharedContractsResourceURIV0,
		ContentType: MCPSharedContractsContentTypeV0,
		SummaryKey:  "mcp.resources.contracts.shared.summary.v0",
	}
}

func MCPSharedContractResourceDescriptorsV0() []MCPContractResourceDescriptorV0 {
	sources := mcpSharedContractSourcesV0()
	out := make([]MCPContractResourceDescriptorV0, 0, len(sources)+1)
	out = append(out, MCPSharedContractsDescriptorV0())
	for _, source := range sources {
		out = append(out, MCPContractResourceDescriptorV0{
			Name:        "orquesta.contracts." + strings.ReplaceAll(source.Slug, "-", "_") + "." + source.Version,
			Version:     source.Version,
			URI:         contractResourceURIMCPV0(source.Slug),
			ContentType: MCPSharedContractsContentTypeV0,
			SummaryKey:  source.SummaryKey,
		})
	}
	return out
}

func NewMCPSharedContractsResourceV0() MCPSharedContractsResourceV0 {
	sources := mcpSharedContractSourcesV0()
	contracts := make([]MCPSharedContractCompactV0, 0, len(sources))
	for _, source := range sources {
		contracts = append(contracts, toMCPSharedContractCompactV0(source))
	}

	return MCPSharedContractsResourceV0{
		URI:             MCPSharedContractsResourceURIV0,
		Version:         MCPSharedContractsResourceVersionV0,
		SummaryKey:      "mcp.resources.contracts.shared.summary.v0",
		Freshness:       mcpResourceFreshnessV0(mcpBacklogT25RefsV0(), mcpMCPResourceVerificationV0()),
		CanonicalSource: "docs/estado_actual_2026-05-17.md",
		Contracts:       contracts,
		Guardrails: []string{
			"mcp.guardrail.adaptador_fino",
			"mcp.guardrail.sin_db_cli_runtime_filesystem_productivo",
			"mcp.guardrail.sin_secretos_transcripts",
			"mcp.guardrail.detalle_extenso_en_modulo_propietario",
		},
	}
}
