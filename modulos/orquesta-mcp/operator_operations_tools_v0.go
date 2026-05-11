package orquestamcp

import operator "orquesta/modulos/orquesta-operator-mcp"

const (
	MCPOperatorToolEstadoOKV0    = "ok"
	MCPOperatorToolEstadoErrorV0 = "error"
)

type MCPOperatorToolResultV0 struct {
	Estado        string                                     `json:"estado"`
	Tool          string                                     `json:"tool"`
	Issues        []operator.OperatorMCPIssueV0              `json:"issues,omitempty"`
	Status        *operator.OperatorMCPStatusResultV0        `json:"status,omitempty"`
	Burst         *operator.OperatorMCPBurstResultV0         `json:"burst,omitempty"`
	Outbox        *operator.OperatorMCPOutboxResultV0        `json:"outbox,omitempty"`
	DirectedQuery *operator.OperatorMCPDirectedQueryResultV0 `json:"directed_query,omitempty"`
	ErrorCode     string                                     `json:"error_code,omitempty"`
}

func ExecuteMCPOperatorStatusToolV0(
	port operator.OperatorMCPStatusPortV0,
	input operator.OperatorStatusQueryV0,
) MCPOperatorToolResultV0 {
	if issues := operator.ValidateOperatorStatusQueryV0(input); len(issues) > 0 {
		return operatorValidationErrorMCPV0(operator.OperatorMCPStatusToolNameV0, issues)
	}
	if port == nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPStatusToolNameV0, "operator_mcp_port_unavailable")
	}
	output, err := port.QueryOperatorStatusV0(input)
	if err != nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPStatusToolNameV0, "operator_mcp_port_error")
	}
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoOKV0, Tool: operator.OperatorMCPStatusToolNameV0, Status: &output}
}

func ExecuteMCPOperatorBurstToolV0(
	port operator.OperatorMCPBurstPortV0,
	input operator.OperatorSupervisedBurstRequestV0,
) MCPOperatorToolResultV0 {
	if issues := operator.ValidateOperatorSupervisedBurstV0(input); len(issues) > 0 {
		return operatorValidationErrorMCPV0(operator.OperatorMCPBurstToolNameV0, issues)
	}
	if port == nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPBurstToolNameV0, "operator_mcp_port_unavailable")
	}
	output, err := port.RequestOperatorSupervisedBurstV0(input)
	if err != nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPBurstToolNameV0, "operator_mcp_port_error")
	}
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoOKV0, Tool: operator.OperatorMCPBurstToolNameV0, Burst: &output}
}

func ExecuteMCPOperatorOutboxToolV0(
	port operator.OperatorMCPOutboxPortV0,
	input operator.OperatorPendingOutboxQueryV0,
) MCPOperatorToolResultV0 {
	if issues := operator.ValidateOperatorPendingOutboxV0(input); len(issues) > 0 {
		return operatorValidationErrorMCPV0(operator.OperatorMCPOutboxToolNameV0, issues)
	}
	if port == nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPOutboxToolNameV0, "operator_mcp_port_unavailable")
	}
	output, err := port.ListOperatorPendingOutboxV0(input)
	if err != nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPOutboxToolNameV0, "operator_mcp_port_error")
	}
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoOKV0, Tool: operator.OperatorMCPOutboxToolNameV0, Outbox: &output}
}

func ExecuteMCPOperatorDirectedQueryToolV0(
	port operator.OperatorMCPDirectedQueryPortV0,
	input operator.OperatorDirectedQueryV0,
) MCPOperatorToolResultV0 {
	if issues := operator.ValidateOperatorDirectedQueryV0(input); len(issues) > 0 {
		return operatorValidationErrorMCPV0(operator.OperatorMCPDirectedQueryToolV0, issues)
	}
	if port == nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPDirectedQueryToolV0, "operator_mcp_port_unavailable")
	}
	output, err := port.RaiseOperatorDirectedQueryV0(input)
	if err != nil {
		return operatorPortErrorMCPV0(operator.OperatorMCPDirectedQueryToolV0, "operator_mcp_port_error")
	}
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoOKV0, Tool: operator.OperatorMCPDirectedQueryToolV0, DirectedQuery: &output}
}

func operatorValidationErrorMCPV0(tool string, issues []operator.OperatorMCPIssueV0) MCPOperatorToolResultV0 {
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoErrorV0, Tool: tool, Issues: issues}
}

func operatorPortErrorMCPV0(tool string, code string) MCPOperatorToolResultV0 {
	return MCPOperatorToolResultV0{Estado: MCPOperatorToolEstadoErrorV0, Tool: tool, ErrorCode: code}
}
