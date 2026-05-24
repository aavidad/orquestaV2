package orquestamcp

import (
	"context"
	"encoding/json"
)

type mcpAutoprogrammingSuperviseTransportInputV0 struct {
	MCPRunSupervisorToolInputV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
}

type mcpAutoprogrammingSuperviseTransportResultV0 struct {
	MCPRunSupervisorToolResultV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
	Diagnostics    []MCPAutoprogrammingDiagnosticV0     `json:"diagnostics,omitempty"`
}

func mcpAutoprogrammingSuperviseTransportHandlerV0(
	port MCPTransportRunSupervisorExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input mcpAutoprogrammingSuperviseTransportInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutoprogrammingSuperviseToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input.MCPRunSupervisorToolInputV0)
		if err != nil {
			return nil, err
		}
		normalizedAdvice := normalizeMCPAutoprogrammingOperatorAdviceV0(
			input.OperatorAdvice,
			firstNonEmptyMCPV0(result.RunRef, result.RequestID, result.CorrelationID),
		)
		if len(normalizedAdvice) == 0 {
			return json.Marshal(result)
		}
		return json.Marshal(mcpAutoprogrammingSuperviseTransportResultV0{
			MCPRunSupervisorToolResultV0: result,
			OperatorAdvice:               normalizedAdvice,
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{mcpAutoprogrammingDiagnosticV0(
				"operator_advice_recorded_non_blocking",
				"operator_advice",
				"consejo de operador registrado sin bloquear supervisor",
			)},
		})
	}
}
