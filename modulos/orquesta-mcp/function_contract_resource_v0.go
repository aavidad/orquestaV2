package orquestamcp

import orquestacore "orquesta/modulos/orquesta-core"

const (
	MCPFunctionContractResourceNameV0    = "orquesta.core.function_contracts.v0"
	MCPFunctionContractResourceVersionV0 = "v0"
	MCPFunctionContractResourceURIV0     = "orquesta://core/function-contracts/v0"
	MCPFunctionContractContentTypeV0     = "application/vnd.orquesta.function-contracts.v0+json"
)

type MCPFunctionContractResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPFunctionContractResourceV0 struct {
	URI               string   `json:"uri"`
	Version           string   `json:"version"`
	SummaryKey        string   `json:"summary_key"`
	ListEndpoint      string   `json:"list_endpoint"`
	ViewEndpoint      string   `json:"view_endpoint"`
	AllowedOperations []string `json:"allowed_operations"`
	BlockedOperations []string `json:"blocked_operations"`
	PublicErrors      []string `json:"errores_publicos"`
	Guardrails        []string `json:"guardrails"`
}

func MCPFunctionContractDescriptorV0() MCPFunctionContractResourceDescriptorV0 {
	return MCPFunctionContractResourceDescriptorV0{
		Name:        MCPFunctionContractResourceNameV0,
		Version:     MCPFunctionContractResourceVersionV0,
		URI:         MCPFunctionContractResourceURIV0,
		ContentType: MCPFunctionContractContentTypeV0,
		SummaryKey:  "mcp.resources.function_contracts.summary.v0",
	}
}

func NewMCPFunctionContractResourceV0() MCPFunctionContractResourceV0 {
	return MCPFunctionContractResourceV0{
		URI:          MCPFunctionContractResourceURIV0,
		Version:      MCPFunctionContractResourceVersionV0,
		SummaryKey:   "mcp.resources.function_contracts.summary.v0",
		ListEndpoint: MCPFunctionContractListHTTPPathV0,
		ViewEndpoint: MCPFunctionContractViewHTTPPathV0,
		AllowedOperations: []string{
			"listar",
			"ver",
		},
		BlockedOperations: []string{
			"registrar",
			"archivar",
			"reemplazar",
		},
		PublicErrors: functionContractPublicErrorsMCPV0(),
		Guardrails: []string{
			"read_only_por_puerto_inyectado",
			"estado_vivo_desde_eventos_o_stores_causales",
			"sin_markdown_historico_ni_db_v1_como_canon",
			"registro_mutante_bloqueado_hasta_decision_director_workflow_outbox",
		},
	}
}

func functionContractPublicErrorsMCPV0() []string {
	return []string{
		orquestacore.FunctionContractQueryErrIncompletoV0,
		orquestacore.FunctionContractQueryErrConsultaNoDisponibleV0,
		orquestacore.FunctionContractQueryErrFiltroNoSoportadoV0,
		orquestacore.FunctionContractQueryErrCursorInvalidoV0,
		orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0,
		orquestacore.FunctionContractQueryErrNoEncontradoV0,
	}
}
