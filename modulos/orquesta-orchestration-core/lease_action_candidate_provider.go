package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type AgentLeaseActionCandidateProviderV0 struct {
	Base        CandidateProviderPortV0
	LeaseSource AgentLeaseAssessmentProviderPortV0
	RequestedBy string
}

var _ CandidateProviderPortV0 = AgentLeaseActionCandidateProviderV0{}

func (provider AgentLeaseActionCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.LeaseSource == nil {
		return candidates, nil
	}
	assessments, err := provider.LeaseSource.BuildAgentLeaseAssessmentsV0(
		ctx,
		agentLeaseAssessmentRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	for _, assessment := range assessments {
		candidate, ok, err := provider.leaseActionCandidateV0(request, assessment)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		if !ok {
			continue
		}
		candidates.LeaseActionCandidates = append(candidates.LeaseActionCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
	}
	return candidates, nil
}

func (provider AgentLeaseActionCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func agentLeaseAssessmentRequestV0(
	request SchedulerCandidateRequestV0,
) AgentLeaseAssessmentRequestV0 {
	return AgentLeaseAssessmentRequestV0{
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

func (provider AgentLeaseActionCandidateProviderV0) leaseActionCandidateV0(
	request SchedulerCandidateRequestV0,
	assessment orquestacoreleases.AgentTimeoutAssessmentV0,
) (orquestadirectorscheduler.SchedulableLeaseActionCandidateV0, bool, error) {
	if strings.TrimSpace(request.OccurredAt) == "" {
		return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{}, false,
			errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	expired, ok, err := orquestacoreleases.AgentLeaseExpiredFromAssessmentV0(
		normalizeAgentLeaseAssessmentV0(assessment),
	)
	if err != nil {
		return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{}, false,
			errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_lease_assessment", err.Error())
	}
	if !ok || leaseActionAlreadyExpiredV0(request.Run, expired.LeaseRef) {
		return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{}, false, nil
	}
	if strings.TrimSpace(expired.RunRef) != strings.TrimSpace(request.Run.RunID) {
		return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{}, false,
			errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_lease_assessment.run_ref", "run_id no coincide")
	}
	action, ok := leaseRecommendedActionV0(expired.RecommendedAction)
	if !ok {
		return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{}, false,
			errorV0(ErrNucleoOrquestacionInvalidoV0, "recommended_action", "accion de lease no soportada")
	}
	input := orquestadirector.PostLeaseActionInputV0{
		CommandMeta:       provider.leaseActionCommandMetaV0(request, expired.LeaseRef),
		RunRef:            request.Run.RunID,
		AgentRequestID:    expired.AgentRequestID,
		LeaseRef:          expired.LeaseRef,
		ReasonCode:        expired.ReasonCode,
		ObservedAt:        expired.ObservedAt,
		RecommendedAction: action,
		EvidenceRefs:      compactStringsV0(append(request.EvidenceRefs, expired.EvidenceRefs...)),
		QuestionID:        leaseActionQuestionRefV0(action, expired.LeaseRef),
	}
	return orquestadirectorscheduler.SchedulableLeaseActionCandidateV0{
		CandidateRef:         leaseActionCandidateRefV0(expired.LeaseRef),
		PostLeaseActionInput: input,
		EvidenceRefs:         compactStringsV0(append(request.EvidenceRefs, expired.EvidenceRefs...)),
	}, true, nil
}

func (provider AgentLeaseActionCandidateProviderV0) leaseActionCommandMetaV0(
	request SchedulerCandidateRequestV0,
	leaseRef string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := leaseActionSafeRefPartV0(leaseRef)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-lease-expired-" + suffix,
		RunID:          strings.TrimSpace(request.Run.RunID),
		IdempotencyKey: "idem-lease-expired-" + suffix,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    leaseActionRequestedByV0(provider.RequestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
}
