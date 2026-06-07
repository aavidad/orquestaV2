package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type opesBridgeLoopConfigV0 struct {
	Loop        externalBridgeLoopConfigV0
	DrainConfig opesDrainConfigV0
}

type opesBridgeDrainerV0 func(context.Context, opesDrainConfigV0) (opesDrainSummaryV0, error)

func opesBridgeLoopConfigFromEnvV0(
	orquestaBaseURLFallback string,
) (opesBridgeLoopConfigV0, error) {
	if strings.TrimSpace(os.Getenv(envOPESBridgeEnabledV0)) != "1" {
		return opesBridgeLoopConfigV0{}, nil
	}
	if strings.TrimSpace(os.Getenv(envOPESBridgeConfirmV0)) != "1" {
		return opesBridgeLoopConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_CONFIRM requerido para opes bridge loop")
	}
	drainConfig, err := opesDrainConfigFromEnvWithBaseURLV0(orquestaBaseURLFallback)
	if err != nil {
		return opesBridgeLoopConfigV0{}, err
	}
	if !opesBridgeHasSafeFilterV0(drainConfig) {
		return opesBridgeLoopConfigV0{}, fmt.Errorf("ORQUESTA_OPES_BRIDGE_JOB_TYPE, ORQUESTA_OPES_BRIDGE_JOB_REF u ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE requerido para opes bridge loop")
	}
	drainConfig.DryRun = false
	component := "opes_bridge_loop"
	if len(drainConfig.JobTypeSequence) > 0 {
		component = "opes_bridge_sequence_loop"
	}
	return opesBridgeLoopConfigV0{
		Loop: externalBridgeLoopConfigV0{
			Enabled:       true,
			Component:     component,
			ResultField:   "summary",
			FilterSummary: opesBridgeFilterSummaryV0(drainConfig),
			Interval:      durationSecondsEnvOrDefaultV0(envOPESBridgeIntervalSecondsV0, 60*time.Second),
			InitialDelay:  durationSecondsEnvOrDefaultV0(envOPESBridgeInitialDelaySecondsV0, 2*time.Second),
			MaxTicks:      intEnvOrZeroV0(envOPESBridgeMaxTicksV0),
			EffectTimeout: drainConfig.HTTPTimeout,
			RetryPolicy: externalBridgeRetryPolicyV0{
				MaxAttempts: 3,
				BaseDelay:   durationSecondsEnvOrDefaultV0(envOPESBridgeIntervalSecondsV0, 60*time.Second),
				MaxDelay:    5 * time.Minute,
			},
		},
		DrainConfig: drainConfig,
	}, nil
}

func runOPESBridgeLoopV0(
	ctx context.Context,
	config opesBridgeLoopConfigV0,
	stderr io.Writer,
	drainer opesBridgeDrainerV0,
) {
	if drainer == nil {
		return
	}
	runExternalBridgeLoopV0(ctx, config.Loop, stderr, func(ctx context.Context) (any, error) {
		return drainer(ctx, config.DrainConfig)
	})
}

func runOPESBridgeLoopAsyncV0(
	ctx context.Context,
	config opesBridgeLoopConfigV0,
	stderr io.Writer,
	drainer opesBridgeDrainerV0,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runOPESBridgeLoopV0(ctx, config, stderr, drainer)
	}()
	return done
}

func waitOPESBridgeLoopDoneV0(
	_ context.Context,
	config opesBridgeLoopConfigV0,
	done <-chan struct{},
) bool {
	if done == nil {
		return true
	}
	timeout := externalBridgeLoopShutdownTimeoutV0(config.Loop)
	waitCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-done:
		return true
	case <-waitCtx.Done():
		if config.Loop.Observer != nil {
			config.Loop.Observer(context.Background(), externalBridgeLoopEventV0{
				Component:  config.Loop.Component,
				Status:     "timeout",
				ErrorCode:  "external_bridge_shutdown_timeout",
				StopReason: "shutdown_timeout",
			})
		}
		return false
	}
}

func opesBridgeHasSafeFilterV0(config opesDrainConfigV0) bool {
	if strings.TrimSpace(os.Getenv(envOPESBridgeAllowUnfilteredV0)) == "1" {
		return true
	}
	return strings.TrimSpace(config.JobType) != "" ||
		strings.TrimSpace(config.JobRef) != "" ||
		strings.TrimSpace(config.ProgramID) != "" ||
		strings.TrimSpace(config.TopicID) != "" ||
		strings.TrimSpace(config.CorrelationID) != "" ||
		len(config.JobTypeSequence) > 0
}

func opesBridgeFilterSummaryV0(config opesDrainConfigV0) []string {
	filters := []string{}
	if strings.TrimSpace(config.JobType) != "" {
		filters = append(filters, "job_type="+strings.TrimSpace(config.JobType))
	}
	if strings.TrimSpace(config.JobRef) != "" {
		filters = append(filters, "job_ref=configured")
	}
	if strings.TrimSpace(config.ProgramID) != "" {
		filters = append(filters, "program_id=configured")
	}
	if strings.TrimSpace(config.TopicID) != "" {
		filters = append(filters, "topic_id=configured")
	}
	if strings.TrimSpace(config.CorrelationID) != "" {
		filters = append(filters, "correlation_id=configured")
	}
	if len(config.JobTypeSequence) > 0 {
		filters = append(filters, "job_type_sequence_count="+fmt.Sprint(len(config.JobTypeSequence)))
	}
	if config.Limit > 0 {
		filters = append(filters, "limit="+fmt.Sprint(config.Limit))
	}
	return filters
}
