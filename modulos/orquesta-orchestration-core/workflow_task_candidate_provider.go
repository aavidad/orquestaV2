package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcandidates "orquesta/modulos/orquesta-director-candidates"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type WorkflowTaskCandidateProviderV0 struct {
	Base            CandidateProviderPortV0
	TaskStore       WorkflowTaskStorePortV0
	RequestedBy     string
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	MaxFrontier     int
}

var _ CandidateProviderPortV0 = WorkflowTaskCandidateProviderV0{}

func (provider WorkflowTaskCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(request.Run.Tasks) == 0 {
		return candidates, nil
	}
	taskRefs := workflowTaskRefsForSchedulingV0(request.Run)
	if len(taskRefs) == 0 {
		return candidates, nil
	}
	if provider.TaskStore == nil {
		return SchedulerCandidateSetV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_store",
			"workflow_task_store requerido",
		)
	}
	tasks, err := provider.TaskStore.LoadWorkflowTasksV0(ctx, request.Run.RunID, taskRefs)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	var activeClaims []orquestacoreconcurrency.WorksetClaimV0
	pending := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, len(tasks))
	for _, task := range tasks {
		if !workflowTaskSchedulableV0(request.Run, task) {
			continue
		}
		candidate, err := provider.workflowTaskCandidateV0(request, task)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		if workflowTaskAgentActiveV0(request.Run, task.TaskID) {
			activeClaims = workflowTaskActiveClaimsV0(activeClaims, candidate.Claims)
			continue
		}
		pending = append(pending, candidate)
	}
	frontier := workflowTaskCandidateFrontierV0(activeClaims, pending, provider.frontierLimitV0())
	candidates.WorkCandidates = append(candidates.WorkCandidates, frontier...)
	candidates.WorkClaims = append(candidates.WorkClaims, workflowTaskClaimsForSchedulerV0(activeClaims, frontier)...)
	for _, candidate := range frontier {
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
	}
	return candidates, nil
}

func (provider WorkflowTaskCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func (provider WorkflowTaskCandidateProviderV0) workflowTaskCandidateV0(
	request SchedulerCandidateRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestadirectorscheduler.SchedulableWorkCandidateV0, error) {
	normalized, err := orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task",
			err.Error(),
		)
	}
	task = normalized
	suffix := workflowTaskCandidateSafeRefPartV0(task.TaskID)
	claimRef := WorkflowTaskClaimRefV0(task.TaskID)
	agentRef := WorkflowTaskAgentRequestRefV0(task.TaskID)
	capacityRef := WorkflowTaskCapacityRequestRefV0(task.TaskID)
	candidate, err := orquestadirectorcandidates.BuildSchedulableWorkCandidateV0(
		orquestadirectorcandidates.SchedulableWorkCandidateInputV0{
			CandidateRef:     "work-ref-" + suffix,
			RunRef:           request.Run.RunID,
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:          task.TaskID,
			SubjectClaimRefs: []string{claimRef},
			ScopeClaims: []orquestadirectorcandidates.WorkCandidateScopeClaimV0{{
				ClaimRef:       claimRef,
				AgentRequestID: agentRef,
				WriteScopes:    task.WriteSet,
				EvidenceRefs:   []string{"evidence-ref-claim-" + suffix},
			}},
			Commands: orquestadirectorcandidates.WorkCandidateCommandsV0{
				CapacityCommandID:      "cmd-capacity-" + suffix,
				CapacityIdempotencyKey: "idem-capacity-" + suffix,
				GateCommandID:          "cmd-gate-" + suffix,
				GateIdempotencyKey:     "idem-gate-" + suffix,
				AgentCommandID:         "cmd-agent-" + suffix,
				AgentIdempotencyKey:    "idem-agent-" + suffix,
				CorrelationID:          strings.TrimSpace(request.CorrelationID),
				RequestedBy:            workflowTaskRequestedByV0(provider.RequestedBy),
				OccurredAt:             strings.TrimSpace(request.OccurredAt),
			},
			Capacity: orquestadirectorcandidates.WorkCandidateCapacityInputV0{
				CapacityRequestID:          capacityRef,
				ReasonCode:                 "programacion_siguiente_paso",
				Summary:                    "Capacidad para microtarea acotada.",
				MinimumRecommendedCapacity: provider.capacityV0(),
				EvidenceRefs:               []string{"evidence-ref-capacity-" + suffix},
			},
			Agent: orquestadirectorcandidates.WorkCandidateAgentInputV0{
				AgentRequestID: agentRef,
				ClaimRef:       claimRef,
				Role:           "implementacion",
				Summary:        "Microtarea acotada lista.",
				EvidenceRefs:   []string{"evidence-ref-agent-" + suffix},
			},
			GateEvidenceRefs: []string{"evidence-ref-gate-" + suffix},
			EvidenceRefs:     []string{"evidence-ref-work-" + suffix},
		},
	)
	if err != nil {
		return orquestadirectorscheduler.SchedulableWorkCandidateV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_candidate",
			err.Error(),
		)
	}
	return candidate, nil
}

func (provider WorkflowTaskCandidateProviderV0) capacityV0() orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(provider.DefaultCapacity)) != "" {
		return provider.DefaultCapacity
	}
	return orquestacoreworkflow.OrchestrationCapacityMediumV0
}

func (provider WorkflowTaskCandidateProviderV0) frontierLimitV0() int {
	if provider.MaxFrontier > 0 {
		return provider.MaxFrontier
	}
	return defaultWorkflowTaskCandidateFrontierLimitV0
}

func workflowTaskSchedulableV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	return task.RunID == run.RunID &&
		task.PhaseID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 &&
		workflowTaskDependenciesSatisfiedV0(run, task)
}

func workflowTaskRefsForSchedulingV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if stringInSetV0(taskRef, run.ClosedTasks) {
			continue
		}
		refs = append(refs, taskRef)
	}
	return refs
}

func WorkflowTaskAgentRequestRefV0(taskRef string) string {
	return "agent-ref-" + workflowTaskCandidateSafeRefPartV0(taskRef)
}

func WorkflowTaskCapacityRequestRefV0(taskRef string) string {
	return "capacity-ref-" + workflowTaskCandidateSafeRefPartV0(taskRef)
}

func WorkflowTaskClaimRefV0(taskRef string) string {
	return "claim-ref-" + workflowTaskCandidateSafeRefPartV0(taskRef)
}

func workflowTaskCandidateSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func workflowTaskRequestedByV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "orquesta-nucleo-workitems"
}

func stringInSetV0(value string, set []string) bool {
	value = strings.TrimSpace(value)
	for _, item := range set {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func workflowTaskAgentActiveV0(run orquestacoreworkflow.OrchestrationRunV0, taskRef string) bool {
	agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
	if stringInSetV0(agentRef, run.FailedAgents) || stringInSetV0(agentRef, run.StoppedAgents) {
		return false
	}
	return stringInSetV0(agentRef, run.Agents) || stringInSetV0(agentRef, run.StartedAgents)
}
