package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type AssessmentReplanSourceV0 struct {
	Store    orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	Capacity CapacityConfigV0
}

var _ orquestacionnucleoapp.AgentAssessmentReplanPlanProviderPortV0 = AssessmentReplanSourceV0{}

func (source AssessmentReplanSourceV0) BuildAgentAssessmentReplanPlansV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0,
) ([]orquestacionnucleoapp.AgentAssessmentReplanPlanV0, error) {
	if source.Store == nil {
		return nil, fmt.Errorf("assessment_replan_source: receipt_store requerido")
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.Run.RunID},
	)
	if err != nil {
		return nil, err
	}
	taskByAgent := assessmentReplanTaskByAgentV0(descriptors)
	plans := make([]orquestacionnucleoapp.AgentAssessmentReplanPlanV0, 0, len(request.Run.ConfirmedStoppedAgents)+len(request.Run.LostAgents))
	for _, projection := range assessmentReplanTerminalProjectionsV0(request.Run) {
		if !assessmentReplanProjectionReadyV0(request.Run, projection) {
			continue
		}
		taskRef := assessmentReplanTaskRefV0(projection, taskByAgent)
		if taskRef == "" {
			continue
		}
		if assessmentReplanTaskTerminalFailedFollowupCountV0(request.Run, taskRef) >= assessmentReplanMaxTerminalFailedFollowupsV0 {
			continue
		}
		if assessmentReplanTaskHasMaterializedFollowupV0(request.Run, taskRef) {
			continue
		}
		plan := source.planForAssessmentV0(request, projection, taskRef)
		if assessmentReplanReplacementAlreadyRequestedV0(request.Run, plan.AgentRequestID) {
			continue
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func assessmentReplanTaskByAgentV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) map[string]string {
	tasks := map[string]string{}
	for _, descriptor := range descriptors {
		agentRef := strings.TrimSpace(descriptor.AgentRef)
		taskRef := strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef)
		if agentRef != "" && taskRef != "" {
			tasks[agentRef] = taskRef
		}
	}
	return tasks
}

func assessmentReplanTerminalProjectionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	projections := make([]orquestacoreworkflow.AgentWorkAssessmentProjectionV0, 0, len(run.AgentAssessments))
	for _, raw := range run.AgentAssessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok || !assessmentReplanProjectionActionCanRepairV0(projection) {
			continue
		}
		projections = append(projections, projection)
	}
	return projections
}

func assessmentReplanProjectionActionCanRepairV0(
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) bool {
	switch strings.TrimSpace(projection.Action) {
	case orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
		orquestacoreworkflow.AgentAssessmentActionAskDirectorV0:
		return true
	default:
		return false
	}
}

func assessmentReplanProjectionReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) bool {
	agentRef := strings.TrimSpace(projection.AgentRequestID)
	if agentRef == "" {
		return false
	}
	if projection.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 {
		return stringInSetV0(run.LostAgents, agentRef) ||
			stringInSetV0(run.ConfirmedStoppedAgents, agentRef)
	}
	return stringInSetV0(run.StoppedAgents, agentRef) &&
		(stringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
			stringInSetV0(run.LostAgents, agentRef))
}

func assessmentReplanTaskRefV0(
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskByAgent map[string]string,
) string {
	if strings.TrimSpace(projection.TaskRef) != "" {
		return strings.TrimSpace(projection.TaskRef)
	}
	return strings.TrimSpace(taskByAgent[strings.TrimSpace(projection.AgentRequestID)])
}

func (source AssessmentReplanSourceV0) planForAssessmentV0(
	request orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
) orquestacionnucleoapp.AgentAssessmentReplanPlanV0 {
	suffix := assessmentReplanSuffixV0(request.Run.RunID, projection, taskRef)
	evidence := assessmentReplanEvidenceRefsV0(request, projection)
	summary := "Reemplazar agente bloqueado tras evaluacion de progreso."
	return orquestacionnucleoapp.AgentAssessmentReplanPlanV0{
		CandidateRef:               "assessment-replan-candidate-ref-" + suffix,
		ReplanRef:                  "replan-ref-" + suffix,
		SignalRef:                  "assessment-replan-signal-ref-" + suffix,
		TaskRef:                    taskRef,
		ReasonRef:                  "reason-ref-" + suffix,
		RequestedAction:            orquestacorereplanner.ReplanActionReplaceAgentV0,
		ReplacementRole:            "implementacion",
		CapacityRequestRef:         "capacity-ref-assessment-" + suffix,
		AgentRequestID:             "agent-ref-assessment-" + suffix,
		MinimumRecommendedCapacity: assessmentReplanCapacityV0(source.Capacity),
		Summary:                    summary,
		EvidenceRefs:               evidence,
		Assessment:                 assessmentReplanPayloadV0(projection, taskRef, summary, evidence),
	}
}

func assessmentReplanPayloadV0(
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	taskRef string,
	summary string,
	evidence []string,
) orquestacoreworkflow.AgentWorkAssessedPayloadV0 {
	phaseID := strings.TrimSpace(projection.PhaseID)
	if phaseID == "" {
		phaseID = string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	}
	return orquestacoreworkflow.AgentWorkAssessedPayloadV0{
		AssessmentRef:  projection.AssessmentRef,
		PhaseID:        phaseID,
		AgentRequestID: projection.AgentRequestID,
		TaskRef:        taskRef,
		DeliveryRef:    projection.DeliveryRef,
		Verdict:        projection.Verdict,
		Action:         projection.Action,
		Severity:       projection.Severity,
		Summary:        summary,
		EvidenceRefs:   evidence,
	}
}

func assessmentReplanReplacementAlreadyRequestedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	return stringInSetV0(run.Agents, agentRef) ||
		stringInSetV0(run.StartedAgents, agentRef)
}

func assessmentReplanTaskHasMaterializedFollowupV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return false
	}
	needle := "#task:" + taskRef + "#action:" +
		string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	for _, projection := range run.ReplanDecisions {
		projection = strings.TrimSpace(projection)
		if !strings.Contains(projection, needle) {
			continue
		}
		for _, followupRef := range assessmentReplanFollowupAgentRefsV0(projection) {
			if assessmentReplanFollowupStillBlocksTaskV0(run, followupRef) {
				return true
			}
		}
	}
	return false
}

func assessmentReplanCapacityV0(
	config CapacityConfigV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(config.Tier)) != "" {
		return config.Tier
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func assessmentReplanEvidenceRefsV0(
	request orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) []string {
	refs := []string{
		"evidence-ref-assessment-replan",
		projection.AssessmentRef,
		projection.AgentRequestID,
	}
	return compactStringsV0(append(refs, request.EvidenceRefs...))
}
