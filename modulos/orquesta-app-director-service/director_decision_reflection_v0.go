package orquestaappdirectorservice

import (
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

type startAppDirectorCommandEffectMarkerV0 struct {
	IdempotencyKey string `json:"idempotency_key"`
	CausationID    string `json:"causation_id"`
}

func startAppDirectorDecisionAlreadyReflectedV0(
	request StartAppDirectorRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) (bool, error) {
	command, issues := orquestadirectoragentworkflow.BuildDirectorAgentWorkflowCommandV0(
		orquestadirectoragentworkflow.DirectorAgentWorkflowCommandRequestV0{
			Decision:      decision,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			RequestedBy:   request.RequestedBy,
		},
	)
	if len(issues) > 0 {
		return false, nil
	}
	return startAppDirectorRunHasCommandEffectV0(run, command)
}

func startAppDirectorRunHasCommandEffectV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (bool, error) {
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	causationID := strings.TrimSpace(command.CommandID)
	if idempotencyKey == "" || causationID == "" {
		return false, nil
	}
	for _, encoded := range run.CommandEffects {
		marker, ok, err := startAppDirectorDecodeCommandEffectMarkerV0(encoded)
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		if marker.IdempotencyKey == idempotencyKey && marker.CausationID == causationID {
			return true, nil
		}
	}
	return false, nil
}

func startAppDirectorDecodeCommandEffectMarkerV0(
	encoded string,
) (startAppDirectorCommandEffectMarkerV0, bool, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return startAppDirectorCommandEffectMarkerV0{}, false, nil
	}
	var marker startAppDirectorCommandEffectMarkerV0
	if err := json.Unmarshal([]byte(encoded), &marker); err != nil {
		return startAppDirectorCommandEffectMarkerV0{}, false, AppDirectorServiceIssueV0{
			Field: "director_decision.command_effects",
		}
	}
	marker.IdempotencyKey = strings.TrimSpace(marker.IdempotencyKey)
	marker.CausationID = strings.TrimSpace(marker.CausationID)
	return marker, marker.IdempotencyKey != "" && marker.CausationID != "", nil
}
