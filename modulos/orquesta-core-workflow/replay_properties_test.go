package orquestacoreworkflow

import (
	"fmt"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

func TestReplayDurableEventsV0PropDuplicadosExactosConvergenV0(t *testing.T) {
	// Invariant: inserting any number of exact durable duplicates preserves replay state.
	rapid.Check(t, func(rt *rapid.T) {
		blockerCount := rapid.IntRange(0, 8).Draw(rt, "blocker_count")
		base := durableReplayHistoryRapidV0(blockerCount)
		withDuplicates := duplicateDurableReplayEventsRapidV0(rt, base)

		want, err := ReplayDurableEventsV0(base)
		if err != nil {
			rt.Fatalf("base replay failed: %v", err)
		}
		got, err := ReplayDurableEventsV0(withDuplicates)
		if err != nil {
			rt.Fatalf("duplicate replay failed: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			rt.Fatalf("idempotent replay diverged:\nbase=%#v\nduplicates=%#v", want, got)
		}

		again, err := ReplayDurableEventsV0(withDuplicates)
		if err != nil || !reflect.DeepEqual(again, got) {
			rt.Fatalf("replay is not deterministic: again=%#v err=%v", again, err)
		}
	})
}

func TestReplayDurableEventsV0PropMalformedInputIsDeterministicAndDoesNotPanicV0(t *testing.T) {
	// Invariant: untrusted persisted input either replays or returns a stable public error, never panics.
	rapid.Check(t, func(rt *rapid.T) {
		events := rapid.SliceOfN(orchestrationEventRapidV0(), 0, 12).Draw(rt, "events")
		first, firstErr := replayDurableWithoutPanicV0(rt, events)
		second, secondErr := replayDurableWithoutPanicV0(rt, events)
		if !reflect.DeepEqual(first, second) || replayErrorCodeV0(firstErr) != replayErrorCodeV0(secondErr) {
			rt.Fatalf("replay changed for the same input: first=%#v err=%v second=%#v err=%v", first, firstErr, second, secondErr)
		}
	})
}

func TestApplyEventV0PropDoesNotMutateInputV0(t *testing.T) {
	// Invariant: an accepted or rejected event cannot mutate the caller's projection.
	rapid.Check(t, func(rt *rapid.T) {
		current, err := ReplayDurableEventsV0(durableReplayHistoryRapidV0(0))
		if err != nil {
			rt.Fatalf("build current run: %v", err)
		}
		before := cloneRunForReducerV0(current)
		event := orchestrationEventRapidV0().Draw(rt, "event")

		_, _ = applyEventWithoutPanicV0(rt, current, event)
		if !reflect.DeepEqual(current, before) {
			rt.Fatalf("ApplyEventV0 mutated input: before=%#v after=%#v", before, current)
		}
	})
}

func durableReplayHistoryRapidV0(blockerCount int) []OrchestrationEventV0 {
	started, err := NewRunStartedEventV0(propertyEventMetaV0("evt-property-start", 1, "idem-property-start"), RunStartedPayloadV0{
		ProjectRef: "project:property",
		AppSpecRef: "appspec:property",
	})
	if err != nil {
		panic(err)
	}
	events := []OrchestrationEventV0{
		started,
	}
	sequence := int64(2)
	for index := 0; index < blockerCount; index++ {
		blockerID := fmt.Sprintf("blocker-property-%d", index)
		blocked, err := NewRunBlockedEventV0(propertyEventMetaV0(
			fmt.Sprintf("evt-property-block-%d", index), sequence, fmt.Sprintf("idem-property-block-%d", index),
		), RunBlockedPayloadV0{
			BlockerID:  blockerID,
			ReasonCode: "property_blocked",
			Summary:    "Property replay blocker.",
		})
		if err != nil {
			panic(err)
		}
		events = append(events, blocked)
		sequence++
		resolved, err := NewRunBlockerResolvedEventV0(propertyEventMetaV0(
			fmt.Sprintf("evt-property-resolve-%d", index), sequence, fmt.Sprintf("idem-property-resolve-%d", index),
		), RunBlockerResolvedPayloadV0{
			BlockerID:  blockerID,
			ReasonCode: "property_resolved",
			Summary:    "Property replay blocker resolved.",
		})
		if err != nil {
			panic(err)
		}
		events = append(events, resolved)
		sequence++
	}
	return events
}

func propertyEventMetaV0(eventID string, sequence int64, idempotencyKey string) OrchestrationEventMetaV0 {
	return OrchestrationEventMetaV0{
		EventID:        eventID,
		RunID:          "run-property",
		Sequence:       sequence,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-property",
		CausationID:    "cmd-property",
		OccurredAt:     "2026-07-11T12:00:00Z",
	}
}

func duplicateDurableReplayEventsRapidV0(rt *rapid.T, events []OrchestrationEventV0) []OrchestrationEventV0 {
	result := make([]OrchestrationEventV0, 0, len(events)*2)
	for index, event := range events {
		result = append(result, event)
		if rapid.Bool().Draw(rt, fmt.Sprintf("duplicate_%d", index)) {
			result = append(result, event)
		}
	}
	return result
}

func orchestrationEventRapidV0() *rapid.Generator[OrchestrationEventV0] {
	return rapid.Custom(func(rt *rapid.T) OrchestrationEventV0 {
		return OrchestrationEventV0{
			EventID:        rapid.SampledFrom([]string{"", "evt-random", "evt-random-2"}).Draw(rt, "event_id"),
			EventType:      rapid.SampledFrom([]string{"", OrchestrationEventRunStartedV0, OrchestrationEventRunBlockedV0, "Unknown"}).Draw(rt, "event_type"),
			RunID:          rapid.SampledFrom([]string{"", "run-property", "run-other"}).Draw(rt, "run_id"),
			Sequence:       int64(rapid.IntRange(-1, 4).Draw(rt, "sequence")),
			IdempotencyKey: rapid.SampledFrom([]string{"", "idem-random"}).Draw(rt, "idempotency_key"),
			CorrelationID:  rapid.SampledFrom([]string{"", "corr-random"}).Draw(rt, "correlation_id"),
			CausationID:    rapid.SampledFrom([]string{"", "cmd-random"}).Draw(rt, "causation_id"),
			OccurredAt:     rapid.SampledFrom([]string{"", "2026-07-11T12:00:00Z"}).Draw(rt, "occurred_at"),
			PayloadVersion: rapid.SampledFrom([]string{"", OrchestrationEventPayloadVersionV0, "v-other"}).Draw(rt, "payload_version"),
			Payload:        []byte(rapid.SampledFrom([]string{"", "{", "{}", `{"project_ref":"project","app_spec_ref":"app"}`, `{"blocker_id":"blocker","reason_code":"reason","summary":"summary"}`}).Draw(rt, "payload")),
		}
	})
}

func replayDurableWithoutPanicV0(rt *rapid.T, events []OrchestrationEventV0) (run OrchestrationRunV0, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			rt.Fatalf("ReplayDurableEventsV0 panicked: %v", recovered)
		}
	}()
	return ReplayDurableEventsV0(events)
}

func applyEventWithoutPanicV0(rt *rapid.T, current OrchestrationRunV0, event OrchestrationEventV0) (run OrchestrationRunV0, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			rt.Fatalf("ApplyEventV0 panicked: %v", recovered)
		}
	}()
	return ApplyEventV0(current, event)
}

func replayErrorCodeV0(err error) string {
	if err == nil {
		return ""
	}
	if publicErr, ok := err.(OrchestrationEventErrorV0); ok {
		return publicErr.Code
	}
	return err.Error()
}
