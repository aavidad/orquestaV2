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
