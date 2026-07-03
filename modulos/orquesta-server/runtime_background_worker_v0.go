package orquestaserver

import "context"

type RuntimeBackgroundWorkerPortV0 interface {
	RunBackgroundV0(context.Context)
}

func compactRuntimeBackgroundWorkersV0(
	workers []RuntimeBackgroundWorkerPortV0,
) []RuntimeBackgroundWorkerPortV0 {
	if len(workers) == 0 {
		return nil
	}
	out := make([]RuntimeBackgroundWorkerPortV0, 0, len(workers))
	for _, worker := range workers {
		if worker != nil {
			out = append(out, worker)
		}
	}
	return out
}
