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
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_ENABLED")) != "1" {
		return opesBridgeLoopConfigV0{}, nil
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_CONFIRM")) != "1" {
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
			Enabled:      true,
			Component:    component,
			ResultField:  "summary",
			Interval:     durationSecondsEnvOrDefaultV0("ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS", 60*time.Second),
			InitialDelay: durationSecondsEnvOrDefaultV0("ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS", 2*time.Second),
			MaxTicks:     intEnvOrZeroV0("ORQUESTA_OPES_BRIDGE_MAX_TICKS"),
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

func opesBridgeHasSafeFilterV0(config opesDrainConfigV0) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED")) == "1" {
		return true
	}
	return strings.TrimSpace(config.JobType) != "" ||
		strings.TrimSpace(config.JobRef) != "" ||
		len(config.JobTypeSequence) > 0
}
