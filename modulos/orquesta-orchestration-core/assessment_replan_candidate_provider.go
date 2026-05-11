package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type AgentAssessmentReplanCandidateProviderV0 struct {
	Base            CandidateProviderPortV0
	PlanSource      AgentAssessmentReplanPlanProviderPortV0
	RequestedBy     string
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

var _ CandidateProviderPortV0 = AgentAssessmentReplanCandidateProviderV0{}

func (provider AgentAssessmentReplanCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.PlanSource == nil ||
		request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return candidates, nil
	}
	plans, err := provider.PlanSource.BuildAgentAssessmentReplanPlansV0(
		ctx,
		agentAssessmentReplanPlanRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	for _, plan := range plans {
		candidate, ok, err := provider.assessmentReplanCandidateV0(request, plan)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		if !ok {
			continue
		}
		candidates.ReplanFollowupCandidates = append(candidates.ReplanFollowupCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
	}
	return candidates, nil
}

func (provider AgentAssessmentReplanCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func agentAssessmentReplanPlanRequestV0(
	request SchedulerCandidateRequestV0,
) AgentAssessmentReplanPlanRequestV0 {
	return AgentAssessmentReplanPlanRequestV0{
		Run:              request.Run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       request.OccurredAt,
		PreviousStep:     request.PreviousStep,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: request.PreviousDecision,
	}
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanCandidateV0(
	request SchedulerCandidateRequestV0,
	plan AgentAssessmentReplanPlanV0,
) (orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0, bool, error) {
	plan = normalizeAgentAssessmentReplanPlanV0(plan)
	proposal, err := orquestacorereplanner.AgentWorkAssessmentToReplanProposalV0(
		agentAssessmentReplanInputV0(request, plan),
	)
	if err != nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, err
	}
	if proposal == nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, nil
	}
	input, err := provider.assessmentReplanFollowupsInputV0(request, plan, *proposal)
	if err != nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, err
	}
	return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{
		CandidateRef:         assessmentReplanCandidateRefV0(plan, *proposal),
		ReplanFollowupsInput: input,
		EvidenceRefs:         compactStringsV0(append(plan.EvidenceRefs, proposal.EvidenceRefs...)),
	}, true, nil
}

func agentAssessmentReplanInputV0(
	request SchedulerCandidateRequestV0,
	plan AgentAssessmentReplanPlanV0,
) orquestacorereplanner.AgentWorkAssessmentReplanInputV0 {
	taskRef := plan.TaskRef
	if taskRef == "" {
		taskRef = plan.Assessment.TaskRef
	}
	return orquestacorereplanner.AgentWorkAssessmentReplanInputV0{
		ReplanRef:       plan.ReplanRef,
		SignalRef:       plan.SignalRef,
		RunRef:          request.Run.RunID,
		TaskRef:         taskRef,
		RequestedAction: plan.RequestedAction,
		ReplacementRole: plan.ReplacementRole,
		ReasonRef:       plan.ReasonRef,
		Summary:         plan.Summary,
		EvidenceRefs:    compactStringsV0(append(request.EvidenceRefs, plan.EvidenceRefs...)),
		Assessment:      plan.Assessment,
	}
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanFollowupsInputV0(
	request SchedulerCandidateRequestV0,
	plan AgentAssessmentReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
) (orquestadirector.ReplanFollowupsInputV0, error) {
	action, ok := workflowReplanActionFromProposalV0(proposal.RecommendedAction)
	if !ok {
		return orquestadirector.ReplanFollowupsInputV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "requested_action", "accion de replan no soportada")
	}
	input := orquestadirector.ReplanFollowupsInputV0{
		DecisionCommandMeta: provider.assessmentReplanCommandMetaV0(
			request,
			"replan",
			proposal.ReplanRef,
		),
		DecisionPayload: orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      proposal.ReplanRef,
			RunRef:         proposal.RunRef,
			TaskRef:        proposal.TaskRef,
			SourceRef:      proposal.SourceRef,
			AcceptedAction: action,
			FollowupRefs:   assessmentReplanFollowupRefsV0(plan),
			Summary:        proposal.Summary,
			EvidenceRefs:   proposal.EvidenceRefs,
		},
		CapacityCandidate: provider.assessmentReplanCapacityCandidateV0(request, plan, proposal, action),
		AgentCandidate:    provider.assessmentReplanAgentCandidateV0(request, plan, proposal, action),
		BlockedAgentRefs:  compactStringsV0([]string{plan.Assessment.AgentRequestID}),
	}
	return input, nil
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanCapacityCandidateV0(
	request SchedulerCandidateRequestV0,
	plan AgentAssessmentReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanCapacityCandidateV0 {
	if !assessmentReplanActionRequiresCapacityV0(action) || plan.CapacityRequestRef == "" {
		return nil
	}
	return &orquestadirector.ReplanCapacityCandidateV0{
		CommandMeta: provider.assessmentReplanCommandMetaV0(request, "capacity", plan.CapacityRequestRef),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          plan.CapacityRequestRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    proposal.TaskRef,
			ReasonCode:                 proposal.ReasonCode,
			Summary:                    "Capacidad para replan por evaluacion de agente.",
			MinimumRecommendedCapacity: provider.assessmentReplanCapacityV0(plan),
			EvidenceRefs:               compactStringsV0(proposal.EvidenceRefs),
		},
	}
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanAgentCandidateV0(
	request SchedulerCandidateRequestV0,
	plan AgentAssessmentReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanAgentCandidateV0 {
	if !assessmentReplanActionRequiresAgentV0(action) || plan.AgentRequestID == "" {
		return nil
	}
	return &orquestadirector.ReplanAgentCandidateV0{
		CommandMeta: provider.assessmentReplanCommandMetaV0(request, "agent", plan.AgentRequestID),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     plan.AgentRequestID,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            proposal.TaskRef,
			CapacityRequestRef: plan.CapacityRequestRef,
			Role:               assessmentReplanRoleV0(plan, proposal),
			Summary:            "Agente de reemplazo para replan por evaluacion.",
			EvidenceRefs:       compactStringsV0(proposal.EvidenceRefs),
		},
	}
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanCommandMetaV0(
	request SchedulerCandidateRequestV0,
	kind string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := assessmentReplanSafeRefPartV0(kind + "-" + ref)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-assessment-" + suffix,
		RunID:          request.Run.RunID,
		IdempotencyKey: "idem-assessment-" + suffix,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    assessmentReplanRequestedByV0(provider.RequestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
}
