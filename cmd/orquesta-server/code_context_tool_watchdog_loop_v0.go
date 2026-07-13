package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultCodeContextToolWatchdogIntervalV0        = 30 * time.Second
	defaultCodeContextToolWatchdogInitialDelayV0    = 1 * time.Second
	defaultCodeContextToolWatchdogEffectTimeoutV0   = 5 * time.Second
	codeContextToolWatchdogLoopComponentV0          = "code_context_tool_watchdog_loop"
	codeContextToolWatchdogLoopResultFieldV0        = "watchdog"
	codeContextToolWatchdogConfiguredStateDirRefV0  = "state_dir=configured"
	codeContextToolWatchdogProviderCodebaseMCPRefV0 = "provider=codebase_memory_mcp"
)

type serverCodeContextToolWatchdogLoopConfigV0 struct {
	Loop         externalBridgeLoopConfigV0
	StateDir     string
	StopOrphans  bool
	OrphanMinAge time.Duration
}

func serverCodeContextToolWatchdogLoopConfigFromEnvV0() (serverCodeContextToolWatchdogLoopConfigV0, error) {
	projectDir, err := projectDirFromEnvV0()
	if err != nil {
		return serverCodeContextToolWatchdogLoopConfigV0{}, err
	}
	projectConfig := projectConfigFromProjectDirBestEffortV0(projectDir)
	return serverCodeContextToolWatchdogLoopConfigFromProjectConfigV0(projectConfig)
}

func serverCodeContextToolWatchdogLoopConfigFromServerConfigV0(
	serverConfig orquestaserver.ConfigV0,
) (serverCodeContextToolWatchdogLoopConfigV0, error) {
	return serverCodeContextToolWatchdogLoopConfigFromProjectConfigV0(
		projectConfigFromServerConfigBestEffortV0(serverConfig),
	)
}

func serverCodeContextToolWatchdogLoopConfigFromProjectConfigV0(
	projectConfig serverProjectConfigFileV0,
) (serverCodeContextToolWatchdogLoopConfigV0, error) {
	if !codebaseBrokerWatchdogEnabledFromProjectConfigFileV0(projectConfig) {
		return serverCodeContextToolWatchdogLoopConfigV0{}, nil
	}
	stateDir := codebaseBrokerStateDirFromProjectConfigFileV0(projectConfig)
	if stateDir == "" {
		return serverCodeContextToolWatchdogLoopConfigV0{}, fmt.Errorf("codebase_broker_state_dir_required")
	}
	orphanMinAgeSeconds := codebaseBrokerWatchdogOrphanMinAgeSecondsFromProjectConfigFileV0(projectConfig)
	return serverCodeContextToolWatchdogLoopConfigV0{
		StateDir:     stateDir,
		StopOrphans:  codebaseBrokerWatchdogStopOrphansFromProjectConfigFileV0(projectConfig),
		OrphanMinAge: time.Duration(orphanMinAgeSeconds) * time.Second,
		Loop: externalBridgeLoopConfigV0{
			Enabled:       true,
			Component:     codeContextToolWatchdogLoopComponentV0,
			ResultField:   codeContextToolWatchdogLoopResultFieldV0,
			FilterSummary: []string{codeContextToolWatchdogProviderCodebaseMCPRefV0, codeContextToolWatchdogConfiguredStateDirRefV0},
			Interval:      defaultCodeContextToolWatchdogIntervalV0,
			InitialDelay:  defaultCodeContextToolWatchdogInitialDelayV0,
			EffectTimeout: defaultCodeContextToolWatchdogEffectTimeoutV0,
			RetryPolicy: externalBridgeRetryPolicyV0{
				MaxAttempts: 3,
				BaseDelay:   defaultCodeContextToolWatchdogIntervalV0,
				MaxDelay:    5 * time.Minute,
			},
		},
	}, nil
}

func runServerCodeContextToolWatchdogLoopV0(
	ctx context.Context,
	config serverCodeContextToolWatchdogLoopConfigV0,
	stderr io.Writer,
) {
	if !config.Loop.Enabled {
		return
	}
	watchdog := serverCodeContextToolWatchdogFromLoopConfigV0(config)
	runExternalBridgeLoopV0(ctx, config.Loop, stderr, func(ctx context.Context) (any, error) {
		return watchdog.RunOnceV0(ctx)
	})
}

func runServerCodeContextToolWatchdogLoopAsyncV0(
	ctx context.Context,
	config serverCodeContextToolWatchdogLoopConfigV0,
	stderr io.Writer,
) <-chan struct{} {
	if !config.Loop.Enabled {
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runServerCodeContextToolWatchdogLoopV0(ctx, config, stderr)
	}()
	return done
}

func waitServerCodeContextToolWatchdogLoopDoneV0(
	_ context.Context,
	config serverCodeContextToolWatchdogLoopConfigV0,
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
				ErrorCode:  "code_context_tool_watchdog_shutdown_timeout",
				StopReason: "shutdown_timeout",
			})
		}
		return false
	}
}

func serverCodeContextToolWatchdogFromLoopConfigV0(
	config serverCodeContextToolWatchdogLoopConfigV0,
) serverCodeContextToolWatchdogV0 {
	stateDir := strings.TrimSpace(config.StateDir)
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	return serverCodeContextToolWatchdogV0{
		Leases:   store,
		Finisher: store,
		Stopper:  serverFileCodeContextToolOwnerStopperV0{Registry: registry},
		Observer: serverFileCodeContextToolOwnerObserverV0{Registry: registry},
		ProcessGuard: serverCodeContextToolProcessGuardV0{
			Lister:       serverPSCodeContextToolProcessListerV0{},
			Stopper:      serverSignalCodeContextToolProcessStopperV0{},
			OwnerMarkers: registry,
			StopOrphans:  config.StopOrphans,
			OrphanMinAge: config.OrphanMinAge,
		},
	}
}

func serverCodeContextToolWatchdogResultCountersV0(result serverCodeContextToolWatchdogResultV0) map[string]int {
	return map[string]int{
		"observed":                 result.Observed,
		"stopped":                  result.Stopped,
		"errors":                   result.Errors,
		"processes_observed":       result.ProcessObserved,
		"orphan_processes":         result.ProcessOrphans,
		"orphan_processes_stopped": result.ProcessOrphansStopped,
	}
}

func serverCodeContextToolWatchdogResultEvidenceRefsV0(result serverCodeContextToolWatchdogResultV0) []string {
	return compactExternalBridgeStringsV0(result.Evidence)
}
