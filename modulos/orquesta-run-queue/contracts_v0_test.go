package orquestarunqueue

import "context"

type fakeRunQueueReaderPortV0 struct {
	candidates []RunSchedulingCandidateV0
}

var _ RunQueueReaderPortV0 = fakeRunQueueReaderPortV0{}

func (fake fakeRunQueueReaderPortV0) ListRunSchedulingCandidatesV0(context.Context, RunQueueReadRequestV0) ([]RunSchedulingCandidateV0, error) {
	return fake.candidates, nil
}
