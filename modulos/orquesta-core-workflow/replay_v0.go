package orquestacoreworkflow

import "strings"

func ReplayDurableEventsV0(events []OrchestrationEventV0) (OrchestrationRunV0, error) {
	uniqueEvents, err := strictReplayPreflightV0(events)
	if err != nil {
		return OrchestrationRunV0{}, err
	}
	var run OrchestrationRunV0
	for _, event := range uniqueEvents {
		next, err := applyValidatedEventV0(run, event)
		if err != nil {
			return run, err
		}
		run = next
	}
	return run, nil
}

func ValidateStrictEventSequenceV0(events []OrchestrationEventV0) error {
	_, err := strictReplayPreflightV0(events)
	return err
}

func strictReplayPreflightV0(events []OrchestrationEventV0) ([]OrchestrationEventV0, error) {
	tracker := newStrictReplayPreflightTrackerV0()
	uniqueEvents := make([]OrchestrationEventV0, 0, len(events))
	for _, event := range events {
		duplicate, err := tracker.validateEventV0(event)
		if err != nil {
			return nil, err
		}
		if !duplicate {
			uniqueEvents = append(uniqueEvents, event)
		}
	}
	return uniqueEvents, nil
}

type strictReplayPreflightTrackerV0 struct {
	expectedSequence int64
	runID            string
	idempotency      replayIdempotencyIndexV0
}

func newStrictReplayPreflightTrackerV0() strictReplayPreflightTrackerV0 {
	return strictReplayPreflightTrackerV0{
		expectedSequence: 1,
		idempotency:      newReplayIdempotencyIndexV0(),
	}
}

func (tracker *strictReplayPreflightTrackerV0) validateEventV0(event OrchestrationEventV0) (bool, error) {
	if err := ValidateOrchestrationEventV0(event); err != nil {
		return false, err
	}
	duplicate, err := tracker.idempotency.registerEventV0(event)
	if err != nil {
		return false, err
	}
	if duplicate {
		return true, nil
	}
	if err := tracker.validateNextEventV0(event); err != nil {
		return false, err
	}
	tracker.expectedSequence++
	return false, nil
}

func (tracker *strictReplayPreflightTrackerV0) validateNextEventV0(event OrchestrationEventV0) error {
	if err := tracker.validateRunIDV0(event); err != nil {
		return err
	}
	if event.Sequence != tracker.expectedSequence {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "sequence")
	}
	return nil
}

func (tracker *strictReplayPreflightTrackerV0) validateRunIDV0(event OrchestrationEventV0) error {
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
