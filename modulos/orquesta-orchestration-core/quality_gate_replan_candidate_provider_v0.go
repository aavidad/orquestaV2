package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type QualityGateReplanCandidateProviderV0 struct {
	Base            CandidateProviderPortV0
	RequestedBy     string
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

var _ CandidateProviderPortV0 = QualityGateReplanCandidateProviderV0{}

func (provider QualityGateReplanCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return candidates, nil
	}
	for _, parts := range qualityGateReplanCausalDecisionsV0(request.Run) {
		candidate, ok := provider.qualityGateReplanFollowupCandidateV0(request, parts)
		if !ok {
			continue
		}
		candidates.ReplanFollowupCandidates = append(candidates.ReplanFollowupCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
	}
	return candidates, nil
}

func (provider QualityGateReplanCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}
