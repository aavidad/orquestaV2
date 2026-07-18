package ports

import (
	"strings"

	"orquesta/internal/governance"
)

func validateAgentGovernanceRequest(request AgentLaunchRequest) error {
	if governance.ValidateBudgetDemand(request.BudgetDemand) != nil {
		return &AgentContractError{Code: "agent.budget_demand_invalid"}
	}
	if governance.ValidateSecurityCriticality(request.SecurityCriticality) != nil {
		return &AgentContractError{Code: "agent.security_criticality_invalid"}
	}
	if governance.ValidateReasoningEffort(request.ReasoningEffort) != nil {
		return &AgentContractError{Code: "agent.reasoning_effort_invalid"}
	}
	return nil
}

func validAgentReceiptRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}
