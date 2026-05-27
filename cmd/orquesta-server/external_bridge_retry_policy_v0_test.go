package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExternalBridgeLoopPublicaRetryScheduledYBudgetV0(t *testing.T) {
	events := []externalBridgeLoopEventV0{}
	calls := 0

	runExternalBridgeLoopV0(
		context.Background(),
		externalBridgeLoopConfigV0{
			Enabled:  true,
			MaxTicks: 3,
			RetryPolicy: externalBridgeRetryPolicyV0{
				MaxAttempts: 2,
				BaseDelay:   0,
				MaxDelay:    time.Millisecond,
				Jitter: func(_ int, _ time.Duration) time.Duration {
					return 0
				},
			},
			Observer: func(_ context.Context, event externalBridgeLoopEventV0) {
				events = append(events, event)
			},
		},
		nil,
		func(context.Context) (any, error) {
			calls++
			return nil, errors.New("rate_limited")
		},
	)

	if calls != 3 || !containsBridgeStatusForTestV0(events, externalBridgeStatusRateLimitedV0) ||
		!containsBridgeStatusForTestV0(events, externalBridgeStatusRetryBudgetExhaustedV0) {
		t.Fatalf("calls=%d events=%+v", calls, events)
	}
}

func containsBridgeStatusForTestV0(events []externalBridgeLoopEventV0, status string) bool {
	for _, event := range events {
		if event.Status == status {
			return true
		}
	}
	return false
}
