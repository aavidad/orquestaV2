package bootstrap

import (
	"context"
	"time"

	"orquesta/internal/application"
)

type scheduler struct {
	orchestrator *application.Orchestrator
	workerRef    string
	pollInterval time.Duration
	report       func(error)
}

func (scheduler scheduler) run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		for {
			result, err := scheduler.orchestrator.ProcessNext(ctx, scheduler.workerRef)
			if err != nil && ctx.Err() == nil && scheduler.report != nil {
				scheduler.report(err)
			}
			if ctx.Err() != nil {
				return
			}
			if !result.Processed {
				break
			}
		}
		timer.Reset(scheduler.pollInterval)
	}
}
