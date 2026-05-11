package orquestamcp

import operator "orquesta/modulos/orquesta-operator-mcp"

const (
	MCPOperatorOperationsResourceNameV0    = "orquesta.operator.operations.v0"
	MCPOperatorOperationsResourceVersionV0 = "v0"
	MCPOperatorOperationsResourceURIV0     = "orquesta://operator/operations/v0"
	MCPOperatorOperationsContentTypeV0     = "application/vnd.orquesta.operator.operations.v0+json"
)

type MCPOperatorOperationsResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPOperatorOperationsResourceV0 struct {
	URI           string                                 `json:"uri"`
	Version       string                                 `json:"version"`
	SchemaVersion string                                 `json:"schema_version"`
	Capabilities  operator.OperatorMCPCapabilitiesV0     `json:"capabilities"`
	Tools         []operator.OperatorMCPToolDescriptorV0 `json:"tools"`
	PublicErrors  []string                               `json:"errores_publicos"`
	Guardrails    []string                               `json:"guardrails"`
}

func MCPOperatorOperationsDescriptorV0() MCPOperatorOperationsResourceDescriptorV0 {
	return MCPOperatorOperationsResourceDescriptorV0{
		Name:        MCPOperatorOperationsResourceNameV0,
		Version:     MCPOperatorOperationsResourceVersionV0,
		URI:         MCPOperatorOperationsResourceURIV0,
		ContentType: MCPOperatorOperationsContentTypeV0,
		SummaryKey:  "mcp.resources.operator.operations.summary.v0",
	}
}

func NewMCPOperatorOperationsResourceV0() MCPOperatorOperationsResourceV0 {
	capabilities := operator.NewOperatorMCPCapabilitiesV0()
	return MCPOperatorOperationsResourceV0{
		URI:           MCPOperatorOperationsResourceURIV0,
		Version:       MCPOperatorOperationsResourceVersionV0,
		SchemaVersion: operator.OperatorMCPSchemaVersionV0,
		Capabilities:  capabilities,
		Tools:         capabilities.Tools,
		PublicErrors: []string{
			operator.ErrOperatorMCPRequiredFieldV0,
			operator.ErrOperatorMCPOpaqueRefV0,
			operator.ErrOperatorMCPBudgetInvalidV0,
			operator.ErrOperatorMCPLimitInvalidV0,
			operator.ErrOperatorMCPQuestionInvalidV0,
			operator.ErrOperatorMCPSectionInvalidV0,
			"operator_mcp_port_unavailable",
			"operator_mcp_port_error",
		},
		Guardrails: []string{
			"adaptador_fino_por_puerto_inyectado",
			"refs_opacas_y_respuestas_compactas",
			"sin_payloads_extensos_ni_internals",
		},
	}
}
