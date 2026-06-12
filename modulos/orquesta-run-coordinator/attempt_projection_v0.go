package orquestaruncoordinator

import (
	"strings"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func activeAttemptsByRunRefV0(
	candidates []orquestarunqueue.RunSchedulingCandidateV0,
) map[string]orquestarunqueue.RunQueueAttemptProjectionV0 {
	byRun := map[string]orquestarunqueue.RunQueueAttemptProjectionV0{}
	for _, projection := range orquestarunqueue.ProjectRunQueueAttemptsV0(candidates) {
		for _, runRef := range projection.RunRefs {
			if strings.TrimSpace(runRef) == "" {
				continue
			}
			byRun[runRef] = projection
		}
	}
	return byRun
}
