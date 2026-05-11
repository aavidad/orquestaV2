package orquestamcp

import (
	"context"
	"encoding/json"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

func mcpOperatorTransportHandlerV0(name string, port any) MCPTransportToolHandlerV0 {
	return func(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
		switch name {
		case operator.OperatorMCPStatusToolNameV0:
			var input operator.OperatorStatusQueryV0
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil, err
			}
			result := ExecuteMCPOperatorStatusToolV0(portAsOperatorStatusMCPV0(port), input)
			return json.Marshal(result)
		case operator.OperatorMCPBurstToolNameV0:
			var input operator.OperatorSupervisedBurstRequestV0
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil, err
			}
			result := ExecuteMCPOperatorBurstToolV0(portAsOperatorBurstMCPV0(port), input)
			return json.Marshal(result)
		case operator.OperatorMCPOutboxToolNameV0:
			var input operator.OperatorPendingOutboxQueryV0
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil, err
			}
			result := ExecuteMCPOperatorOutboxToolV0(portAsOperatorOutboxMCPV0(port), input)
			return json.Marshal(result)
		default:
			var input operator.OperatorDirectedQueryV0
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil, err
			}
			result := ExecuteMCPOperatorDirectedQueryToolV0(portAsOperatorQueryMCPV0(port), input)
			return json.Marshal(result)
		}
	}
}

func mcpTransportToolErrorPayloadV0(tool string, code string) (json.RawMessage, error) {
	return json.Marshal(MCPTransportToolErrorV0{Estado: MCPOperatorToolEstadoErrorV0, Tool: tool, ErrorCode: code})
}

func portAsOperatorStatusMCPV0(port any) operator.OperatorMCPStatusPortV0 {
	value, _ := port.(operator.OperatorMCPStatusPortV0)
	return value
}

func portAsOperatorBurstMCPV0(port any) operator.OperatorMCPBurstPortV0 {
	value, _ := port.(operator.OperatorMCPBurstPortV0)
	return value
}

func portAsOperatorOutboxMCPV0(port any) operator.OperatorMCPOutboxPortV0 {
	value, _ := port.(operator.OperatorMCPOutboxPortV0)
	return value
}

func portAsOperatorQueryMCPV0(port any) operator.OperatorMCPDirectedQueryPortV0 {
	value, _ := port.(operator.OperatorMCPDirectedQueryPortV0)
	return value
}
