package orquestamcp

import "strings"

const (
	MCPResourceDescriptorStatusActiveV0 = "active"
	MCPResourceDescriptorStatusStaleV0  = "stale"
	MCPResourceDescriptorStatusLegacyV0 = "legacy"
)

type MCPResourceDescriptorSourceV0 struct {
	Status             string   `json:"status"`
	Owner              string   `json:"owner"`
	CanonicalSource    string   `json:"canonical_source"`
	Freshness          string   `json:"freshness"`
	DTOOrValidator     string   `json:"dto_or_validator"`
	PublicErrorsSource string   `json:"public_errors_source"`
	Verification       []string `json:"verification,omitempty"`
}

func mcpResourceDescriptorSourceForResourceV0(name string) MCPResourceDescriptorSourceV0 {
	switch strings.TrimSpace(name) {
	case MCPSharedContractsResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-mcp", "docs/estado_actual_2026-05-17.md", "mcpSharedContractSourcesV0", "mcpSharedContractSourcesV0")
	case MCPProjectRoadmapResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-mcp", "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T198", "mcpProjectRoadmapSourcesV0", "mcpProjectDecisionSourcesV0")
	case MCPOperationalStatusResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-observability", "modulos/orquesta-observability/operational_status_types_v0.go", "OperationalStatusQueryV0+ValidateOperationalStatusQueryV0", "orquesta-observability error constants")
	case MCPFunctionContractResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-core", "modulos/orquesta-core/function_contract_query_v0.go", "FunctionContractReadIndexPortV0", "orquesta-core function contract errors")
	case MCPWorkspaceTimelineResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-observability", "modulos/orquesta-observability/workspace_timeline_types_v0.go", "WorkspaceTimelineQueryV0+ValidateWorkspaceTimelineQueryV0", "orquesta-observability workspace errors")
	case MCPBootstrapResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-director", "modulos/orquesta-mcp/bootstrap_appspec_tool_v0.go", "MCPBootstrapToolInputV0+ExecuteMCPBootstrapToolV0", "director/core workflow public errors")
	case MCPCoreWorkflowContractsResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-core-workflow", "modulos/orquesta-core-workflow/docs/contratos.md", "OrchestrationRunV0+HandleOrchestrationCommandV0", "orquesta-core-workflow error constants")
	case MCPOperatorOperationsResourceNameV0:
		return mcpResourceDescriptorSourceV0("orquesta-operator-mcp", "modulos/orquesta-operator-mcp/operator_capabilities_v0.go", "OperatorMCPCapabilitiesV0", "operatorToolPublicErrorsV0")
	default:
		return MCPResourceDescriptorSourceV0{Status: MCPResourceDescriptorStatusStaleV0}
	}
}

func mcpResourceDescriptorSourceV0(
	owner string,
	canonicalSource string,
	dtoOrValidator string,
	publicErrorsSource string,
) MCPResourceDescriptorSourceV0 {
	return MCPResourceDescriptorSourceV0{
		Status:             MCPResourceDescriptorStatusActiveV0,
		Owner:              strings.TrimSpace(owner),
		CanonicalSource:    strings.TrimSpace(canonicalSource),
		Freshness:          MCPTransportOutputFreshnessStaticV0,
		DTOOrValidator:     strings.TrimSpace(dtoOrValidator),
		PublicErrorsSource: strings.TrimSpace(publicErrorsSource),
		Verification: []string{
			"mcp_resource_descriptor_source_sync_v0_test",
			"transport_output_budget_v0",
		},
	}
}

func ValidateMCPResourceDescriptorSourceV0(source MCPResourceDescriptorSourceV0) bool {
	switch source.Status {
	case MCPResourceDescriptorStatusActiveV0, MCPResourceDescriptorStatusStaleV0, MCPResourceDescriptorStatusLegacyV0:
	default:
		return false
	}
	if source.Status != MCPResourceDescriptorStatusActiveV0 {
		return strings.TrimSpace(source.CanonicalSource) != ""
	}
	return strings.TrimSpace(source.Owner) != "" &&
		strings.TrimSpace(source.CanonicalSource) != "" &&
		strings.TrimSpace(source.Freshness) != "" &&
		strings.TrimSpace(source.DTOOrValidator) != "" &&
		strings.TrimSpace(source.PublicErrorsSource) != "" &&
		len(compactStringsMCPV0(source.Verification)) > 0
}
