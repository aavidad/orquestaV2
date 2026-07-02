package orquestai18ndocs

import "strings"

const (
	PublicErrorCatalogVersionV0 = "public_error_catalog.v0"

	PublicErrorSeverityInfoV0  = "info"
	PublicErrorSeverityWarnV0  = "warn"
	PublicErrorSeverityErrorV0 = "error"

	PublicErrorMethodNotAllowedV0                = "metodo_no_permitido"
	PublicErrorMethodUnsupportedV0               = "metodo_no_soportado"
	PublicErrorPathUnsupportedV0                 = "ruta_no_soportada"
	PublicErrorBodyInvalidV0                     = "request_body_invalido"
	PublicErrorBodyTooLargeV0                    = "request_body_too_large"
	PublicErrorBodyTrailingDataV0                = "request_body_trailing_data"
	PublicErrorContentTypeInvalidV0              = "request_content_type_invalido"
	PublicErrorAcceptInvalidV0                   = "request_accept_invalido"
	PublicErrorTransportV0                       = "error_transporte"
	PublicErrorResponseInvalidV0                 = "respuesta_invalida"
	PublicErrorNotConfiguredV0                   = "not_configured"
	PublicErrorUnavailableV0                     = "unavailable"
	PublicErrorExecutorV0                        = "executor_error"
	PublicErrorMCPMethodNotAllowedV0             = "mcp_method_not_allowed"
	PublicErrorMCPInvalidParamsV0                = "mcp_invalid_params"
	PublicErrorMCPToolNotFoundV0                 = "mcp_tool_not_found"
	PublicErrorMCPToolParamsInvalidV0            = "mcp_tool_params_invalid"
	PublicErrorMCPToolArgumentsJSONV0            = "mcp_tool_arguments_invalid_json"
	PublicErrorMCPParamsTooLargeV0               = "mcp_params_too_large"
	PublicErrorMCPOutputTooLargeV0               = "mcp_output_too_large"
	PublicErrorMCPResourceBlockedV0              = "mcp_resource_payload_blocked"
	PublicErrorMCPToolBlockedV0                  = "mcp_tool_payload_blocked"
	PublicErrorMCPTransportUnboundV0             = "mcp_transport_tool_unbound"
	PublicErrorMCPTransportInputV0               = "mcp_transport_tool_input_invalid"
	PublicErrorMCPTransportPortV0                = "mcp_transport_port_unavailable"
	PublicErrorMCPTransportSchemaV0              = "mcp_transport_schema_stale"
	PublicErrorWorkspaceTimelineHTTPV0           = "workspace_timeline_http_error"
	PublicErrorWorkspaceTimelineNACV0            = "workspace_timeline_no_configurada"
	PublicErrorWorkspaceTimelineNAV0             = "workspace_timeline_no_disponible"
	PublicErrorOperatorRequiredFieldV0           = "operator_mcp_required_field"
	PublicErrorOperatorOpaqueRefV0               = "operator_mcp_opaque_ref_invalid"
	PublicErrorOperatorBudgetInvalidV0           = "operator_mcp_budget_invalid"
	PublicErrorOperatorLimitInvalidV0            = "operator_mcp_limit_invalid"
	PublicErrorOperatorQuestionV0                = "operator_mcp_question_invalid"
	PublicErrorOperatorSectionV0                 = "operator_mcp_section_invalid"
	PublicErrorOperatorPortV0                    = "operator_mcp_port_unavailable"
	PublicErrorOperatorPortErrorV0               = "operator_mcp_port_error"
	PublicErrorOperatorConnectorV0               = "operator_mcp_connector_unavailable"
	PublicErrorOperatorTimeoutV0                 = "operator_mcp_timeout"
	PublicErrorOperatorCancelledV0               = "operator_mcp_cancelled"
	PublicErrorCLIOptionInvalidV0                = "opcion_invalida"
	PublicErrorCLIInputTooLargeV0                = "input_too_large"
	PublicErrorCLIConfigInvalidV0                = "configuracion_cli_invalida"
	PublicErrorCLIContractMissingV0              = "contrato_no_configurado"
	PublicErrorCLINotSerializableV0              = "salida_no_serializable"
	PublicErrorWebFormIncompleteV0               = "form_incompleto"
	PublicErrorWebTransportMissingV0             = "transporte_no_configurado"
	PublicErrorRunRefRequiredV0                  = "run_ref_requerido"
	PublicErrorDomainWorkCompletedReviewFailedV0 = "domain_work_completed_review_failed"
	PublicErrorRunControlGoalBackendActiveV0     = "control_not_propagated_to_goal_backend"
)

type PublicErrorDescriptorV0 struct {
	Code        string `json:"code"`
	I18nKey     string `json:"i18n_key"`
	Boundary    string `json:"boundary"`
	Adapter     string `json:"adapter"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	JSONRPCCode int    `json:"jsonrpc_code,omitempty"`
	Retryable   bool   `json:"retryable"`
	Severity    string `json:"severity"`
}

func PublicErrorCatalogV0() []PublicErrorDescriptorV0 {
	return append([]PublicErrorDescriptorV0(nil), publicErrorCatalogV0...)
}

func PublicErrorDescriptorByCodeV0(code string) (PublicErrorDescriptorV0, bool) {
	code = strings.TrimSpace(code)
	for _, entry := range publicErrorCatalogV0 {
		if entry.Code == code {
			return entry, true
		}
	}
	return PublicErrorDescriptorV0{}, false
}

func PublicErrorCodeKnownV0(code string) bool {
	_, ok := PublicErrorDescriptorByCodeV0(code)
	return ok
}

func NormalizePublicErrorCodeV0(code string, fallback string) string {
	code = strings.TrimSpace(code)
	if PublicErrorCodeKnownV0(code) {
		return code
	}
	fallback = strings.TrimSpace(fallback)
	if PublicErrorCodeKnownV0(fallback) {
		return fallback
	}
	return PublicErrorExecutorV0
}

var publicErrorCatalogV0 = []PublicErrorDescriptorV0{
	publicErrorV0(PublicErrorMethodNotAllowedV0, "http", "shared", 405, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMethodUnsupportedV0, "http", "web", 405, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorPathUnsupportedV0, "http", "shared", 404, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorBodyInvalidV0, "http", "shared", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorBodyTooLargeV0, "http", "shared", 413, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorBodyTrailingDataV0, "http", "shared", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorContentTypeInvalidV0, "http", "shared", 415, -32600, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorAcceptInvalidV0, "http", "mcp-jsonrpc", 406, -32600, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorTransportV0, "rest-client", "cli-web", 0, 0, true, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorResponseInvalidV0, "rest-client", "cli-web", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorNotConfiguredV0, "port", "shared", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorUnavailableV0, "port", "shared", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorExecutorV0, "executor", "shared", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPMethodNotAllowedV0, "jsonrpc-http", "mcp", 405, -32600, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPInvalidParamsV0, "jsonrpc", "mcp", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPToolNotFoundV0, "jsonrpc", "mcp", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPToolParamsInvalidV0, "jsonrpc", "mcp", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPToolArgumentsJSONV0, "jsonrpc", "mcp", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPParamsTooLargeV0, "jsonrpc", "mcp", 413, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPOutputTooLargeV0, "mcp-output", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPResourceBlockedV0, "mcp-output", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPToolBlockedV0, "mcp-output", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPTransportUnboundV0, "mcp-transport", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorMCPTransportInputV0, "mcp-transport", "mcp", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorMCPTransportPortV0, "mcp-transport", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorMCPTransportSchemaV0, "mcp-transport", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("autoprogramming_prepare_run_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("autoprogramming_prepare_run_executor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("autoprogramming_status_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("autoprogramming_status_executor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("autoprogramming_supervise_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("autoprogramming_supervise_executor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("autoprogramming_observe_goal_timeout", "http", "mcp", 504, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("domain_work_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("domain_work_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("external_work_run_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("external_work_run_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("external_work_run_timeout", "http", "mcp", 504, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("run_control_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("run_control_timeout", "http", "mcp", 504, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorRunControlGoalBackendActiveV0, "http", "mcp", 409, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("run_supervisor_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("run_queue_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("director_stats_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("director_stats_executor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("server_shutdown_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("server_shutdown_executor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("request_change_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("run_supervisor_execute_error", "mcp-transport", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorDomainWorkCompletedReviewFailedV0, "mcp-transport", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0("run_supervisor_error", "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorWorkspaceTimelineHTTPV0, "http", "mcp", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorWorkspaceTimelineNACV0, "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorWorkspaceTimelineNAV0, "mcp-tool", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0("arrancar_director_no_configurado", "http", "mcp", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorOperatorRequiredFieldV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorOpaqueRefV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorBudgetInvalidV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorLimitInvalidV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorQuestionV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorSectionV0, "operator-mcp", "operator", 400, -32602, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorPortV0, "operator-mcp", "operator", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorOperatorPortErrorV0, "operator-mcp", "operator", 500, -32000, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorOperatorConnectorV0, "operator-mcp", "operator", 503, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorOperatorTimeoutV0, "operator-mcp", "operator", 504, -32000, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorOperatorCancelledV0, "operator-mcp", "operator", 499, -32000, true, PublicErrorSeverityInfoV0),
	publicErrorV0(PublicErrorCLIOptionInvalidV0, "cli", "cli", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorCLIInputTooLargeV0, "cli", "cli", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorCLIConfigInvalidV0, "cli", "cli", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorCLIContractMissingV0, "cli", "cli", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorCLINotSerializableV0, "cli", "cli", 0, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorWebFormIncompleteV0, "web", "web", 400, 0, false, PublicErrorSeverityErrorV0),
	publicErrorV0(PublicErrorWebTransportMissingV0, "web", "web", 503, 0, true, PublicErrorSeverityWarnV0),
	publicErrorV0(PublicErrorRunRefRequiredV0, "web-mcp", "shared", 400, -32602, false, PublicErrorSeverityErrorV0),
}

func publicErrorV0(
	code string,
	boundary string,
	adapter string,
	httpStatus int,
	jsonRPCCode int,
	retryable bool,
	severity string,
) PublicErrorDescriptorV0 {
	return PublicErrorDescriptorV0{
		Code:        code,
		I18nKey:     "orquesta.public_errors." + code,
		Boundary:    boundary,
		Adapter:     adapter,
		HTTPStatus:  httpStatus,
		JSONRPCCode: jsonRPCCode,
		Retryable:   retryable,
		Severity:    severity,
	}
}
