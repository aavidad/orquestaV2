package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type externalBridgeLoopConfigV0 struct {
	Enabled      bool
	Component    string
	ResultField  string
	Interval     time.Duration
	InitialDelay time.Duration
	MaxTicks     int
}

type externalBridgeTickV0 func(context.Context) (any, error)

func runExternalBridgeLoopV0(
	ctx context.Context,
	config externalBridgeLoopConfigV0,
	stderr io.Writer,
	tick externalBridgeTickV0,
) {
	if !config.Enabled || tick == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if !waitExternalBridgeDelayV0(ctx, config.InitialDelay) {
		return
	}
	for tickNumber := 1; ; tickNumber++ {
		result, err := tick(ctx)
		writeExternalBridgeTickV0(stderr, config, tickNumber, result, err)
		if config.MaxTicks > 0 && tickNumber >= config.MaxTicks {
			return
		}
		if !waitExternalBridgeDelayV0(ctx, config.Interval) {
			return
		}
	}
}

func waitExternalBridgeDelayV0(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func writeExternalBridgeTickV0(
	w io.Writer,
	config externalBridgeLoopConfigV0,
	tick int,
	result any,
	err error,
) {
	component := strings.TrimSpace(config.Component)
	if component == "" {
		component = "external_bridge_loop"
	}
	resultField := strings.TrimSpace(config.ResultField)
	if resultField == "" {
		resultField = "result"
	}
	payload := map[string]any{
		"component": component,
		"tick":      tick,
		resultField: result,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		_, _ = fmt.Fprintf(w, "%s tick=%d marshal_error=%v\n", component, tick, marshalErr)
		return
	}
	_, _ = fmt.Fprintln(w, string(data))
}

func durationSecondsEnvOrDefaultV0(key string, fallback time.Duration) time.Duration {
	value := intEnvOrDefaultV0(key, int(fallback/time.Second))
	if value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Second
}

func intEnvOrZeroV0(key string) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}
