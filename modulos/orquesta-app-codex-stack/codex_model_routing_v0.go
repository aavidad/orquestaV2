package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

func codexModelRouteForTaskV0(config CodexRuntimeConfigV0, taskRef string) (orquestacapacity.ModelRoutingDecisionV0, string, error) {
	config.ModelRouting = normalizeCodexModelRoutingConfigV0(config.ModelRouting)
	taskRef = strings.TrimSpace(taskRef)
	request := orquestacapacity.ModelRoutingRequestV0{TaskRef: taskRef, Level: orquestacapacity.ModelRoutingLevelNormalV0}
	if declared, ok := config.ModelRouting.TaskRoutes[request.TaskRef]; ok {
		request = declared
		request.TaskRef = taskRef
	}
	decision := orquestacapacity.ResolveModelRoutingV0(config.ModelRouting.Policy, request)
	if decision.Rejected {
		return decision, "", fmt.Errorf("model_routing_rejected:%s", decision.RejectionRef)
	}
	model := strings.TrimSpace(config.ModelRouting.ModelAlias[decision.SelectedModelRef])
	if model == "" {
		return decision, "", fmt.Errorf("model_routing_alias_missing:%s", decision.SelectedModelRef)
	}
	return decision, model, nil
}
