package bootstrap

import (
	"context"
	"time"

	"orquesta/internal/application"
)

type scheduler struct {
	orchestrator          *application.Orchestrator
	workerRef             string
	pollInterval          time.Duration
	maxLaunchesPerCycle   int64
	report                func(error)
	processNextForTesting func(context.Context, string) (application.ProcessResult, error)
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

		nextDelay, keepRunning := scheduler.processCycle(ctx)
		if !keepRunning {
			return
		}
		timer.Reset(nextDelay)
	}
}

func (scheduler scheduler) processCycle(ctx context.Context) (time.Duration, bool) {
	launches := int64(0)
	for {
		result, err := scheduler.processNext(ctx)
		if err != nil && ctx.Err() == nil && scheduler.report != nil {
			scheduler.report(err)
		}
		if ctx.Err() != nil {
			return 0, false
		}
		if !result.Processed {
			return scheduler.pollInterval, true
		}
		if result.Action != application.ActionLaunchAgent {
			continue
		}
		launches++
		if launches >= scheduler.maxLaunchesPerCycle {
			// Start a fresh cycle immediately. This bounds launch bursts without
			// making stop/observe work wait for the normal poll interval.
			return 0, true
		}
	}
}

func (scheduler scheduler) processNext(ctx context.Context) (application.ProcessResult, error) {
	if scheduler.processNextForTesting != nil {
		return scheduler.processNextForTesting(ctx, scheduler.workerRef)
	}
	return scheduler.orchestrator.ProcessNext(ctx, scheduler.workerRef)
}
