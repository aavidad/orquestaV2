package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type ReviewReworkReplanCandidateProviderV0 struct {
	Base            CandidateProviderPortV0
	PlanSource      ReviewReworkReplanPlanProviderPortV0
	TaskWriter      WorkflowTaskWriterPortV0
	RequestedBy     string
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

var _ CandidateProviderPortV0 = ReviewReworkReplanCandidateProviderV0{}

func (provider ReviewReworkReplanCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.PlanSource == nil ||
		!reviewReworkPhaseCanPlanV0(request.Run.CurrentPhase) {
		return candidates, nil
	}
	plans, err := provider.PlanSource.BuildReviewReworkReplanPlansV0(
		ctx,
		reviewReworkReplanRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	for _, plan := range plans {
		candidate, ok, err := provider.reviewReworkCandidateV0(ctx, request, plan)
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

func (provider ReviewReworkReplanCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkCandidateV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
) (orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0, bool, error) {
	plan = normalizeReviewReworkReplanPlanV0(plan)
	if reviewReworkSplitPlanSettledV0(request.Run, plan) {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, nil
	}
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 &&
		!reviewReworkRunHasRefV0(request.Run.ReplanDecisions, plan.ReplanRef, "#source:") {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, nil
	}
	if err := validateReviewReworkPlanRefsV0(request, plan); err != nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, err
	}
	proposal, err := reviewReworkReplanProposalV0(request, plan)
	if err != nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, err
	}
	if proposal == nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, nil
	}
	input, err := provider.reviewReworkFollowupsInputV0(ctx, request, plan, *proposal)
	if err != nil {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false, err
	}
	return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{
		CandidateRef:         reviewReworkCandidateRefV0(plan, *proposal),
		ReplanFollowupsInput: input,
		EvidenceRefs:         compactStringsV0(append(plan.EvidenceRefs, proposal.EvidenceRefs...)),
	}, true, nil
}

func reviewReworkReplanProposalV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
) (*orquestacorereplanner.ReplanProposalV0, error) {
	proposal, err := orquestacorereplanner.ReviewResultToReplanProposalV0(
		reviewReworkReplanInputV0(request, plan),
	)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, nil
	}
	proposal.SourceRef = plan.ReworkRequestRef
	normalized, err := orquestacorereplanner.NewReplanProposalV0(*proposal)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkFollowupsInputV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
) (orquestadirector.ReplanFollowupsInputV0, error) {
	action, ok := reviewReworkWorkflowReplanActionV0(proposal.RecommendedAction)
	if !ok {
		return orquestadirector.ReplanFollowupsInputV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "requested_action", "accion de replan de revision no soportada")
	}
	microtasks, err := provider.reviewReworkMicrotaskCandidatesV0(ctx, request, plan, action)
	if err != nil {
		return orquestadirector.ReplanFollowupsInputV0{}, err
	}
	input := orquestadirector.ReplanFollowupsInputV0{
		DecisionCommandMeta: provider.reviewReworkCommandMetaV0(request, "replan", proposal.ReplanRef),
		DecisionPayload: orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      proposal.ReplanRef,
			RunRef:         proposal.RunRef,
			TaskRef:        proposal.TaskRef,
			SourceRef:      proposal.SourceRef,
			AcceptedAction: action,
			FollowupRefs:   reviewReworkFollowupRefsV0(plan, action),
			Summary:        proposal.Summary,
			EvidenceRefs:   proposal.EvidenceRefs,
		},
		SourceKind:           orquestadirector.ReplanFollowupSourceReviewReworkV0,
		OpenPhaseCandidate:   provider.reviewReworkOpenPhaseCandidateV0(request, proposal, action),
		MicrotaskCandidates:  microtasks,
		CapacityCandidate:    provider.reviewReworkCapacityCandidateV0(request, plan, proposal, action),
		AgentCandidate:       provider.reviewReworkAgentCandidateV0(request, plan, proposal, action),
		AskDirectorCandidate: provider.reviewReworkAskDirectorCandidateV0(request, plan, proposal, action),
	}
	return input, nil
}

func validateReviewReworkPlanRefsV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
) error {
	if strings.TrimSpace(request.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	if !reviewReworkRunHasRefV0(request.Run.ReworkRequests, plan.ReworkRequestRef, "#review_result:") {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "rework_request_ref", "retrabajo no reflejado")
	}
	taskRef := plan.TaskRef
	if taskRef == "" {
		taskRef = reviewReworkTaskFromRunV0(request.Run)
	}
	if !reviewReworkRunContainsRefV0(request.Run.Tasks, taskRef) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "task_ref", "tarea no reflejada")
	}
	return nil
}

func reviewReworkPhaseCanPlanV0(phase orquestacoreworkflow.OrchestrationPhaseIDV0) bool {
	return phase == orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		phase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0
}

func reviewReworkWorkflowReplanActionV0(
	action orquestacorereplanner.ReplanRecommendedActionV0,
) (orquestacoreworkflow.ReplanDecisionActionV0, bool) {
	switch action {
	case orquestacorereplanner.ReplanActionRetryTaskV0,
		orquestacorereplanner.ReplanActionAskDirectorV0,
		orquestacorereplanner.ReplanActionSplitTaskV0:
		return orquestacoreworkflow.ReplanDecisionActionV0(action), true
	default:
		return "", false
	}
}
