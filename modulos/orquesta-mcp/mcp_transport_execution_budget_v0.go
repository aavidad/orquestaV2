package orquestamcp

import (
	"encoding/json"
	"time"
)

const (
	MCPTransportExecutionProfileResourceReadV0           = "resource_read"
	MCPTransportExecutionProfileControlPlaneMutationV0   = "control_plane_mutation"
	MCPTransportExecutionProfileAutoprogrammingLongV0    = "autoprogramming_long"
	MCPTransportExecutionProfileDefaultToolV0            = "default_tool"
	MCPTransportExecutionDefaultResourceReadTimeoutV0    = 2 * time.Second
	MCPTransportExecutionDefaultControlPlaneTimeoutV0    = 10 * time.Second
	MCPTransportExecutionDefaultAutoprogrammingTimeoutV0 = 30 * time.Second
	MCPTransportExecutionDefaultToolTimeoutV0            = 5 * time.Second
	MCPTransportExecutionTimeoutV0                       = "mcp_tool_timeout"
	MCPTransportExecutionCancelledV0                     = "mcp_tool_cancelled"
)

type MCPTransportExecutionBudgetV0 struct {
	Profile       string        `json:"profile"`
	MaxDuration   time.Duration `json:"max_duration"`
	TimeoutCode   string        `json:"timeout_code"`
	CancelledCode string        `json:"cancelled_code"`
}

type mcpTransportExecutionBudgetJSONV0 struct {
	Profile       string `json:"profile"`
	MaxDurationMS int64  `json:"max_duration_ms"`
	TimeoutCode   string `json:"timeout_code"`
	CancelledCode string `json:"cancelled_code"`
}

func MCPTransportResourceExecutionBudgetV0() MCPTransportExecutionBudgetV0 {
	return mcpTransportExecutionBudgetV0(
		MCPTransportExecutionProfileResourceReadV0,
		MCPTransportExecutionDefaultResourceReadTimeoutV0,
	)
}

func MCPTransportToolExecutionBudgetV0(profile string) MCPTransportExecutionBudgetV0 {
	switch profile {
	case MCPTransportExecutionProfileAutoprogrammingLongV0:
		return mcpTransportExecutionBudgetV0(profile, MCPTransportExecutionDefaultAutoprogrammingTimeoutV0)
	case MCPTransportExecutionProfileControlPlaneMutationV0:
		return mcpTransportExecutionBudgetV0(profile, MCPTransportExecutionDefaultControlPlaneTimeoutV0)
	default:
		return mcpTransportExecutionBudgetV0(MCPTransportExecutionProfileDefaultToolV0, MCPTransportExecutionDefaultToolTimeoutV0)
	}
}

func NormalizeMCPTransportExecutionBudgetV0(
	budget MCPTransportExecutionBudgetV0,
	defaultProfile string,
) MCPTransportExecutionBudgetV0 {
	if budget.Profile == "" {
		budget.Profile = defaultProfile
	}
	switch budget.Profile {
	case MCPTransportExecutionProfileResourceReadV0,
		MCPTransportExecutionProfileControlPlaneMutationV0,
		MCPTransportExecutionProfileAutoprogrammingLongV0,
		MCPTransportExecutionProfileDefaultToolV0:
	default:
		budget.Profile = defaultProfile
	}
	if budget.MaxDuration <= 0 {
		budget = MCPTransportToolExecutionBudgetV0(budget.Profile)
		if defaultProfile == MCPTransportExecutionProfileResourceReadV0 {
			budget = MCPTransportResourceExecutionBudgetV0()
		}
	}
	if budget.TimeoutCode == "" {
		budget.TimeoutCode = MCPTransportExecutionTimeoutV0
	}
	if budget.CancelledCode == "" {
		budget.CancelledCode = MCPTransportExecutionCancelledV0
	}
	return budget
}

func mcpTransportExecutionBudgetV0(profile string, timeout time.Duration) MCPTransportExecutionBudgetV0 {
	return MCPTransportExecutionBudgetV0{
		Profile:       profile,
		MaxDuration:   timeout,
		TimeoutCode:   MCPTransportExecutionTimeoutV0,
		CancelledCode: MCPTransportExecutionCancelledV0,
	}
}

func (budget MCPTransportExecutionBudgetV0) MarshalJSON() ([]byte, error) {
	return json.Marshal(mcpTransportExecutionBudgetJSONV0{
		Profile:       budget.Profile,
		MaxDurationMS: budget.MaxDuration.Milliseconds(),
		TimeoutCode:   budget.TimeoutCode,
		CancelledCode: budget.CancelledCode,
	})
}
