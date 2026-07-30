package bootstrap

import (
	"context"
	"time"

	"orquesta/internal/application"
)

type scheduler struct {
	orchestrator           *application.Orchestrator
	workerRef              string
	pollInterval           time.Duration
	maxConcurrentLaunches  int64
	report                 func(error)
	claimNextForTesting    func(context.Context, string, application.ActionClaimSelection) (application.ActionClaim, bool, error)
	processClaimForTesting func(context.Context, application.ActionClaim) (application.ProcessResult, error)
}

func (scheduler scheduler) run(ctx context.Context) {
	limit := int(scheduler.maxConcurrentLaunches)
	completions := make(chan error, limit)
	inFlight := 0
	defer func() {
		scheduler.waitForLaunches(completions, inFlight)
	}()

	for {
		scheduler.drainLaunchCompletions(ctx, completions, &inFlight)
		if ctx.Err() != nil {
			return
		}

		selection := application.ActionClaimSelection{ExcludeLaunch: inFlight >= limit}
		claim, found, err := scheduler.claimNext(ctx, selection)
		if err != nil {
			scheduler.reportWhileActive(ctx, err)
			if !scheduler.wait(ctx, completions, &inFlight) {
				return
			}
			continue
		}
		if !found {
			if !scheduler.wait(ctx, completions, &inFlight) {
				return
			}
			continue
		}
		if claim.Action.Kind != application.ActionLaunchAgent {
			_, err := scheduler.processClaim(ctx, claim)
			scheduler.reportWhileActive(ctx, err)
			continue
		}

		// ExcludeLaunch guarantees that a launch is not claimed while every
		// admission slot is occupied. Keep the guard local so a faulty test
		// double cannot make the dispatcher exceed its configured capacity.
		for inFlight >= limit {
			if !scheduler.wait(ctx, completions, &inFlight) {
				return
			}
		}
		inFlight++
		go func(claim application.ActionClaim) {
			_, err := scheduler.processClaim(ctx, claim)
			completions <- err
		}(claim)
	}
}

func (scheduler scheduler) wait(
	ctx context.Context,
	completions <-chan error,
	inFlight *int,
) bool {
	timer := time.NewTimer(scheduler.pollInterval)
	defer timer.Stop()
	if *inFlight == 0 {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return true
		}
	}
	select {
	case <-ctx.Done():
		return false
	case err := <-completions:
		*inFlight--
		scheduler.reportWhileActive(ctx, err)
		return true
	case <-timer.C:
		return true
	}
}

func (scheduler scheduler) drainLaunchCompletions(
	ctx context.Context,
	completions <-chan error,
	inFlight *int,
) {
	for *inFlight > 0 {
		select {
		case err := <-completions:
			*inFlight--
			scheduler.reportWhileActive(ctx, err)
		default:
			return
		}
	}
}

func (scheduler scheduler) waitForLaunches(completions <-chan error, inFlight int) {
	for inFlight > 0 {
		<-completions
		inFlight--
	}
}

func (scheduler scheduler) reportWhileActive(ctx context.Context, err error) {
	if err != nil && ctx.Err() == nil && scheduler.report != nil {
		scheduler.report(err)
	}
}

func (scheduler scheduler) claimNext(
	ctx context.Context,
	selection application.ActionClaimSelection,
) (application.ActionClaim, bool, error) {
	if scheduler.claimNextForTesting != nil {
		return scheduler.claimNextForTesting(ctx, scheduler.workerRef, selection)
	}
	return scheduler.orchestrator.ClaimNextAction(ctx, scheduler.workerRef, selection)
}

func (scheduler scheduler) processClaim(
	ctx context.Context,
	claim application.ActionClaim,
) (application.ProcessResult, error) {
	if scheduler.processClaimForTesting != nil {
		return scheduler.processClaimForTesting(ctx, claim)
	}
	return scheduler.orchestrator.ProcessClaim(ctx, claim)
}
