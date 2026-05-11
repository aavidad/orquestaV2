package orquestacoreworkflow

import "strings"

func ReplayDurableEventsV0(events []OrchestrationEventV0) (OrchestrationRunV0, error) {
	var run OrchestrationRunV0
	tracker := newDurableReplayTrackerV0()
	for _, event := range events {
		next, err := tracker.applyEventV0(run, event)
		if err != nil {
			return run, err
		}
		run = next
	}
	return run, nil
}

func ValidateStrictEventSequenceV0(events []OrchestrationEventV0) error {
	_, err := ReplayDurableEventsV0(events)
	return err
}

type durableReplayTrackerV0 struct {
	expectedSequence int64
	runID            string
	idempotency      replayIdempotencyIndexV0
}

func newDurableReplayTrackerV0() durableReplayTrackerV0 {
	return durableReplayTrackerV0{
		expectedSequence: 1,
		idempotency:      newReplayIdempotencyIndexV0(),
	}
}

func (tracker *durableReplayTrackerV0) applyEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	if err := ValidateOrchestrationEventV0(event); err != nil {
		return current, err
	}
	duplicate, err := tracker.idempotency.registerEventV0(event)
	if err != nil {
		return current, err
	}
	if duplicate {
		return current, nil
	}
	if err := tracker.validateNextEventV0(event); err != nil {
		return current, err
	}
	next, err := ApplyEventV0(current, event)
	if err != nil {
		return current, err
	}
	tracker.expectedSequence++
	return next, nil
}

func (tracker *durableReplayTrackerV0) validateNextEventV0(event OrchestrationEventV0) error {
	if err := tracker.validateRunIDV0(event); err != nil {
		return err
	}
	if event.Sequence != tracker.expectedSequence {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "sequence")
	}
	return nil
}

func (tracker *durableReplayTrackerV0) validateRunIDV0(event OrchestrationEventV0) error {
	runID := strings.TrimSpace(event.RunID)
	if tracker.runID == "" {
		tracker.runID = runID
		return nil
	}
	if tracker.runID != runID {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "run_id")
	}
	return nil
}
