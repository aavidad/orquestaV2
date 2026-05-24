package orquestamcp

const (
	MCPResourceFreshnessUpdatedAtV0 = "2026-05-24"
	MCPResourceFreshnessStatusV0    = "static_projection_with_live_refs"
)

type MCPResourceFreshnessV0 struct {
	Status        string   `json:"status"`
	UpdatedAt     string   `json:"updated_at"`
	AuthorityRefs []string `json:"authority_refs"`
	BacklogRefs   []string `json:"backlog_refs"`
	Verification  []string `json:"verification"`
}

func mcpResourceFreshnessV0(backlogRefs []string, verification []string) MCPResourceFreshnessV0 {
	return MCPResourceFreshnessV0{
		Status:    MCPResourceFreshnessStatusV0,
		UpdatedAt: MCPResourceFreshnessUpdatedAtV0,
		AuthorityRefs: []string{
			"docs/estado_actual_2026-05-17.md",
			"docs/guia_nucleo_orquestacion_2026-05-17.md",
			"docs/principio_orquesta_piensa_director.md",
		},
		BacklogRefs:  compactStringsMCPV0(backlogRefs),
		Verification: compactStringsMCPV0(verification),
	}
}

func mcpBacklogT25RefsV0() []string {
	return []string{
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T25-mcp-roadmap-backlog-state-sync",
	}
}

func mcpRoadmapBacklogRefsV0(extra ...string) []string {
	return compactStringsMCPV0(append(mcpBacklogT25RefsV0(), extra...))
}

func mcpMCPResourceVerificationV0(extra ...string) []string {
	return compactStringsMCPV0(append([]string{"go test -count=1 ./modulos/orquesta-mcp"}, extra...))
}
