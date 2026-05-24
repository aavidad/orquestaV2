package main

import (
	"context"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type fakeIdleSelfRunQueueV0 struct {
	candidates []orquestarunqueue.RunSchedulingCandidateV0
}

func (fake *fakeIdleSelfRunQueueV0) ListRunSchedulingCandidatesV0(
	context.Context,
	orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

func (fake *fakeIdleSelfRunQueueV0) SetRunPriorityV0(
	context.Context,
	orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	return orquestarunqueue.RunSchedulingCandidateV0{}, nil
}
