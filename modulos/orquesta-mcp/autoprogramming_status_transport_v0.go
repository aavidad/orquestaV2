package orquestamcp

import (
	"context"
	"encoding/json"
)

type mcpAutoprogrammingStatusTransportInputV0 struct {
	MCPAutoprogrammingStatusToolInputV0
	TelemetryFlags string                                 `json:"telemetry_flags,omitempty"`
	OperatorAdvice mcpAutoprogrammingOperatorAdviceListV0 `json:"operator_advice,omitempty"`
}

type mcpAutoprogrammingStatusTransportResultV0 struct {
	MCPAutoprogrammingStatusToolResultV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
}

func mcpAutoprogrammingStatusTransportHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input mcpAutoprogrammingStatusTransportInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutoprogrammingStatusToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input.MCPAutoprogrammingStatusToolInputV0)
		if err != nil {
			payload := newMCPAutoprogrammingStatusExecutorErrorResultV0(
				input.MCPAutoprogrammingStatusToolInputV0,
				publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_status_executor_error", err),
			)
			return json.Marshal(payload)
		}
		normalizedAdvice := input.OperatorAdvice.normalizedMCPV0(
			firstNonEmptyMCPV0(result.RunRef, result.QueueRef, result.RequestID, result.CorrelationID),
		)
		if len(normalizedAdvice) == 0 {
			return json.Marshal(result)
		}
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"operator_advice_recorded_non_blocking",
			"operator_advice",
			"consejo de operador registrado sin bloquear estado de autoprogramacion",
		))
		return json.Marshal(mcpAutoprogrammingStatusTransportResultV0{
			MCPAutoprogrammingStatusToolResultV0: result,
			OperatorAdvice:                       normalizedAdvice,
		})
	}
}
