package orquestacionnucleoapp

import "context"

type CandidateProviderFuncV0 func(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error)

func (fn CandidateProviderFuncV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	return fn(ctx, request)
}

type StaticCandidateProviderV0 struct {
	Candidates SchedulerCandidateSetV0
}

func (provider StaticCandidateProviderV0) BuildSchedulerCandidatesV0(
	context.Context,
	SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	return provider.Candidates, nil
}
