package orquestamcp

import (
	"context"
	"encoding/json"
)

type mcpAutoprogrammingSuperviseTransportInputV0 struct {
	MCPRunSupervisorToolInputV0
	OperatorAdvice mcpAutoprogrammingOperatorAdviceListV0 `json:"operator_advice,omitempty"`
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
			if mcpRunSupervisorClassifiedOKResultDespiteExecutorErrorV0(result) {
				normalizedAdvice := input.OperatorAdvice.normalizedMCPV0(
					firstNonEmptyMCPV0(result.RunRef, result.RequestID, result.CorrelationID),
				)
				if len(normalizedAdvice) == 0 {
					return json.Marshal(result)
				}
				diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), result.Diagnostics...)
				diagnostics = append(diagnostics, mcpAutoprogrammingDiagnosticV0(
					"operator_advice_recorded_non_blocking",
					"operator_advice",
					"consejo de operador registrado sin bloquear supervisor",
				))
				payloadResult := result
				payloadResult.Diagnostics = nil
				return json.Marshal(mcpAutoprogrammingSuperviseTransportResultV0{
					MCPRunSupervisorToolResultV0: payloadResult,
					OperatorAdvice:               normalizedAdvice,
					Diagnostics:                  diagnostics,
				})
			}
			if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
				return json.Marshal(result)
			}
			message := publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_supervise_executor_error", err)
			payload := NewMCPRunSupervisorErrorResultV0(
				input.MCPRunSupervisorToolInputV0,
				"autoprogramming_supervise_executor_error",
				"executor",
				message,
			)
			return json.Marshal(payload)
		}
		normalizedAdvice := input.OperatorAdvice.normalizedMCPV0(
			firstNonEmptyMCPV0(result.RunRef, result.RequestID, result.CorrelationID),
		)
		if len(normalizedAdvice) == 0 {
			return json.Marshal(result)
		}
		diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), result.Diagnostics...)
		diagnostics = append(diagnostics, mcpAutoprogrammingDiagnosticV0(
			"operator_advice_recorded_non_blocking",
			"operator_advice",
			"consejo de operador registrado sin bloquear supervisor",
		))
		payloadResult := result
		payloadResult.Diagnostics = nil
		return json.Marshal(mcpAutoprogrammingSuperviseTransportResultV0{
			MCPRunSupervisorToolResultV0: payloadResult,
			OperatorAdvice:               normalizedAdvice,
			Diagnostics:                  diagnostics,
		})
	}
}
