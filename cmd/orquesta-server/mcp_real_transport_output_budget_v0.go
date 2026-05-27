package main

import (
	"encoding/json"
	"strconv"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	mcpOutputKindResourceV0 = "resource"
	mcpOutputKindToolV0     = "tool"
)

func mcpOutputBudgetRPCErrorV0(
	kind string,
	name string,
	budget orquestamcp.MCPTransportOutputBudgetV0,
	payload json.RawMessage,
) *mcpJSONRPCErrorV0 {
	budget = normalizeMCPRealOutputBudgetV0(kind, budget)
	if budget.Mode == orquestamcp.MCPTransportOutputModeBlockedV0 || len(payload) > budget.MaxBytes {
		code := strings.TrimSpace(budget.Overflow)
		if code == "" {
			code = orquestamcp.MCPTransportOutputTooLargeV0
		}
		return &mcpJSONRPCErrorV0{
			Code:    -32000,
			Message: orquestamcp.MCPTransportOutputTooLargeV0,
			Data: map[string]string{
				"error_code":     code,
				"output_kind":    kind,
				"output_ref":     publicMCPRegistrationRefV0(name, false),
				"output_mode":    orquestamcp.MCPTransportOutputModeBlockedV0,
				"freshness":      budget.Freshness,
				"redaction":      budget.Redaction,
				"max_bytes":      strconv.Itoa(budget.MaxBytes),
				"observed_bytes": strconv.Itoa(len(payload)),
			},
		}
	}
	return nil
}

func normalizeMCPRealOutputBudgetV0(
	kind string,
	budget orquestamcp.MCPTransportOutputBudgetV0,
) orquestamcp.MCPTransportOutputBudgetV0 {
	switch kind {
	case mcpOutputKindResourceV0:
		return orquestamcp.NormalizeMCPTransportOutputBudgetV0(
			budget,
			orquestamcp.MCPTransportDefaultResourceOutputMaxBytesV0,
			orquestamcp.MCPTransportResourcePayloadBlockedV0,
		)
	default:
		return orquestamcp.NormalizeMCPTransportOutputBudgetV0(
			budget,
			orquestamcp.MCPTransportDefaultToolOutputMaxBytesV0,
			orquestamcp.MCPTransportToolPayloadBlockedV0,
		)
	}
}
