package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func claudeModelRouteForTaskV0(config ClaudeRuntimeConfigV0, taskRef string) (orquestacapacity.ModelRoutingDecisionV0, string, error) {
	config.ModelRouting = normalizeClaudeModelRoutingConfigV0(config.ModelRouting)
	taskRef = strings.TrimSpace(taskRef)
	request := orquestacapacity.ModelRoutingRequestV0{TaskRef: taskRef, Level: orquestacapacity.ModelRoutingLevelNormalV0}
	if declared, ok := config.ModelRouting.TaskRoutes[request.TaskRef]; ok {
		request = declared
		request.TaskRef = taskRef
	}
	decision := orquestacapacity.ResolveModelRoutingV0(config.ModelRouting.Policy, request)
	if decision.Rejected {
		return decision, "", fmt.Errorf("claude_model_routing_rejected:%s", decision.RejectionRef)
	}
	model := strings.TrimSpace(config.ModelRouting.ModelAlias[decision.SelectedModelRef])
	if model == "" {
		return decision, "", fmt.Errorf("claude_model_routing_alias_missing:%s", decision.SelectedModelRef)
	}
	if strings.Contains(strings.ToLower(model), "opus") {
		return decision, "", fmt.Errorf("claude_model_routing_automatic_opus_rejected")
	}
	if !validClaudeRoutingEffortV0(decision.ReasoningEffort) {
		return decision, "", fmt.Errorf("claude_model_routing_effort_rejected:%s", decision.ReasoningEffort)
	}
	return decision, model, nil
}

func validClaudeRoutingEffortV0(effort string) bool {
	switch strings.TrimSpace(effort) {
	case "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}
