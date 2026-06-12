package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExternalBridgeLoopDisabledDoesNotTickV0(t *testing.T) {
	calls := 0

	runExternalBridgeLoopV0(
		context.Background(),
		externalBridgeLoopConfigV0{},
		nil,
		func(context.Context) (any, error) {
			calls++
			return nil, nil
		},
	)

	if calls != 0 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestExternalBridgeLoopKeepsRunningAfterTickErrorV0(t *testing.T) {
	calls := 0
	var stderr bytes.Buffer

	runExternalBridgeLoopV0(
		context.Background(),
		externalBridgeLoopConfigV0{
			Enabled:      true,
			Component:    "test_bridge",
			ResultField:  "summary",
			Interval:     time.Millisecond,
			InitialDelay: 0,
			MaxTicks:     2,
		},
		&stderr,
		func(context.Context) (any, error) {
			calls++
			if calls == 1 {
				return map[string]int{"seen": 1}, errors.New("temporary_error")
			}
			return map[string]int{"seen": 2}, nil
		},
	)

	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	output := stderr.String()
	if !strings.Contains(output, `"component":"test_bridge"`) ||
		!strings.Contains(output, `"error":"temporary_error"`) ||
		!strings.Contains(output, `"seen":2`) {
		t.Fatalf("stderr=%s", output)
	}
}

func TestExternalBridgeLoopObserverRecibeLifecycleCompactoV0(t *testing.T) {
	events := []externalBridgeLoopEventV0{}
	config := externalBridgeLoopConfigV0{
		Enabled:       true,
		Component:     "test_bridge",
		ResultField:   "summary",
		FilterSummary: []string{"job_type=plan_temario", "job_ref=configured"},
		MaxTicks:      1,
		Observer: func(_ context.Context, event externalBridgeLoopEventV0) {
			events = append(events, event)
		},
	}

	runExternalBridgeLoopV0(context.Background(), config, nil, func(context.Context) (any, error) {
		return opesRegistryFinalPkgSummaryV0{
			Seen:                 2,
			Submitted:            0,
			Skipped:              2,
			CompletedNonTerminal: 1,
			CompletedNonTerminalRefs: []opesRegistryFinalPkgCompletionDriftV0{{
				RunRef: "run-ref-opes-a1-t002-finalpkg-20260612",
				Reason: "package_complete_run_non_terminal",
			}},
		}, nil
	})

	if len(events) < 4 {
		t.Fatalf("events=%+v", events)
	}
	lastTick := events[2]
	if lastTick.Status != "idle" ||
		lastTick.Counters["seen"] != 2 ||
		lastTick.Counters["completed_nonterminal"] != 1 ||
		len(lastTick.EvidenceRefs) != 1 ||
		lastTick.EvidenceRefs[0] != "run-ref-opes-a1-t002-finalpkg-20260612" ||
		!strings.Contains(strings.Join(lastTick.FilterSummary, ","), "job_ref=configured") {
		t.Fatalf("last_tick=%+v", lastTick)
	}
	if events[len(events)-1].Status != "stopped" ||
		events[len(events)-1].StopReason != "max_ticks_reached" {
		t.Fatalf("stop=%+v", events[len(events)-1])
	}
}

func TestExternalBridgeLoopObserverMarcaTimeoutV0(t *testing.T) {
	events := []externalBridgeLoopEventV0{}
	runExternalBridgeLoopV0(
		context.Background(),
		externalBridgeLoopConfigV0{
			Enabled:       true,
			Component:     "test_bridge",
			EffectTimeout: time.Nanosecond,
			MaxTicks:      1,
			Observer: func(_ context.Context, event externalBridgeLoopEventV0) {
				events = append(events, event)
			},
		},
		nil,
		func(ctx context.Context) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	)

	found := false
	for _, event := range events {
		if event.Status == "timeout" &&
			event.ErrorCode == "external_bridge_tick_timeout" {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%+v", events)
	}
}
