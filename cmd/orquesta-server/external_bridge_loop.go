package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type externalBridgeLoopConfigV0 struct {
	Enabled       bool
	Component     string
	ResultField   string
	FilterSummary []string
	Interval      time.Duration
	InitialDelay  time.Duration
	MaxTicks      int
	EffectTimeout time.Duration
	Observer      externalBridgeLoopObserverV0
	RetryPolicy   externalBridgeRetryPolicyV0
}

type externalBridgeTickV0 func(context.Context) (any, error)
type externalBridgeLoopObserverV0 func(context.Context, externalBridgeLoopEventV0)

type externalBridgeLoopEventV0 struct {
	Component     string
	Status        string
	TickNumber    int
	TickActive    bool
	Success       bool
	ErrorCode     string
	StopReason    string
	FilterSummary []string
	Counters      map[string]int
	OccurredAt    time.Time
}

func runExternalBridgeLoopV0(
	ctx context.Context,
	config externalBridgeLoopConfigV0,
	stderr io.Writer,
	tick externalBridgeTickV0,
) {
	if !config.Enabled || tick == nil {
		notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{Status: "disabled"})
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if stderr == nil {
		stderr = io.Discard
	}
	notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{Status: "waiting_initial_delay"})
	if !waitExternalBridgeDelayV0(ctx, config.InitialDelay) {
		notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{
			Status:     "stopped",
			StopReason: "context_cancelled_before_start",
		})
		return
	}
	consecutiveErrors := 0
	for tickNumber := 1; ; tickNumber++ {
		notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{
			Status:     "running",
			TickNumber: tickNumber,
			TickActive: true,
		})
		tickCtx, cancel := externalBridgeEffectContextV0(ctx, config.EffectTimeout)
		result, err := tick(tickCtx)
		cancel()
		resultEvent := externalBridgeLoopResultEventV0(tickNumber, result, err)
		notifyExternalBridgeLoopV0(ctx, config, resultEvent)
		writeExternalBridgeTickV0(stderr, config, tickNumber, result, err)
		if config.MaxTicks > 0 && tickNumber >= config.MaxTicks {
			notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{
				Status:     "stopped",
				StopReason: "max_ticks_reached",
			})
			return
		}
		delay := config.Interval
		if err != nil {
			consecutiveErrors++
			retryEvent := externalBridgeRetryEventV0(config.RetryPolicy, consecutiveErrors, config.Interval, resultEvent.ErrorCode)
			delay = retryEvent.Delay
			notifyExternalBridgeLoopV0(ctx, config, retryEvent.Event)
		} else {
			consecutiveErrors = 0
		}
		if !waitExternalBridgeDelayV0(ctx, delay) {
			notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{
				Status:     "stopping",
				StopReason: "context_cancelled",
			})
			notifyExternalBridgeLoopV0(ctx, config, externalBridgeLoopEventV0{
				Status:     "stopped",
				StopReason: "context_cancelled",
			})
			return
		}
	}
}

func externalBridgeLoopResultEventV0(tickNumber int, result any, err error) externalBridgeLoopEventV0 {
	event := externalBridgeLoopEventV0{
		Status:     "running",
		TickNumber: tickNumber,
		Success:    err == nil,
		Counters:   externalBridgeResultCountersV0(result),
	}
	if err == nil && event.Counters != nil && event.Counters["submitted"] == 0 {
		event.Status = "idle"
	}
	if err != nil {
		event.Status = "degraded"
		event.ErrorCode = compactExternalBridgeErrorCodeV0(err)
		if errors.Is(err, context.DeadlineExceeded) {
			event.Status = "timeout"
			event.ErrorCode = "external_bridge_tick_timeout"
		}
	}
	return event
}

func notifyExternalBridgeLoopV0(ctx context.Context, config externalBridgeLoopConfigV0, event externalBridgeLoopEventV0) {
	if config.Observer == nil {
		return
	}
	event.Component = firstExternalBridgeValueV0(event.Component, config.Component, "external_bridge_loop")
	event.FilterSummary = compactExternalBridgeStringsV0(append(event.FilterSummary, config.FilterSummary...))
	event.OccurredAt = time.Now().UTC()
	config.Observer(ctx, event)
}

func externalBridgeEffectContextV0(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
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
