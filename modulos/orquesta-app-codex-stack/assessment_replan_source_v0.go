package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
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
	plans := make([]orquestacionnucleoapp.AgentAssessmentReplanPlanV0, 0, len(request.Run.ConfirmedStoppedAgents))
	for _, projection := range assessmentReplanStoppedProjectionsV0(request.Run) {
		if !assessmentReplanProjectionReadyV0(request.Run, projection) {
			continue
		}
		taskRef := assessmentReplanTaskRefV0(projection, taskByAgent)
		if taskRef == "" {
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

func assessmentReplanStoppedProjectionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	projections := make([]orquestacoreworkflow.AgentWorkAssessmentProjectionV0, 0, len(run.AgentAssessments))
	for _, raw := range run.AgentAssessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok || projection.Action != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
			continue
		}
		projections = append(projections, projection)
	}
	return projections
}

func assessmentReplanProjectionReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) bool {
	agentRef := strings.TrimSpace(projection.AgentRequestID)
	return agentRef != "" &&
		stringInSetV0(run.StoppedAgents, agentRef) &&
		stringInSetV0(run.ConfirmedStoppedAgents, agentRef)
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
	suffix := assessmentReplanSafeRefV0(projection.AssessmentRef)
	evidence := assessmentReplanEvidenceRefsV0(request, projection)
	summary := "Reemplazar agente detenido tras evaluacion de progreso."
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

func assessmentReplanSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "sin-ref"
	}
	return value
}
