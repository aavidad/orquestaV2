package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverConfigFromEnvV0() (orquestaserver.ConfigV0, error) {
	projectDir, err := projectDirFromEnvV0()
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	stateDir := absDirEnvOrDefaultV0("ORQUESTA_SERVER_STATE_DIR",
		filepath.Join(defaultControlDirV0(projectDir), "state"))
	runtimeDir := absDirEnvOrDefaultV0("ORQUESTA_CODEX_RUNTIME_WORKDIR",
		filepath.Join(defaultControlDirV0(projectDir), "runtime"))
	config := orquestaserver.ConfigV0{
		Addr:           envOrDefaultV0("ORQUESTA_SERVER_ADDR", orquestaserver.DefaultAddrV0),
		StateDir:       stateDir,
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		TickInterval:   time.Duration(intEnvOrDefaultV0("ORQUESTA_SERVER_TICK_INTERVAL_MS", 5000)) * time.Millisecond,
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:          "global",
			MaxRunsPerTick:    intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", 2),
			MaxExecutions:     intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", 2),
			StopOnNoExecution: true,
			DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
				MaxBursts:            intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_BURSTS", 4),
				MaxStepsPerBurst:     intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_STEPS", 6),
				MaxDispatchesPerWait: intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES", 4),
				MaxCommands:          intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_COMMANDS", 20),
				MaxOutboxPerCycle:    intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_OUTBOX", 4),
				MaxDecisionCycles:    intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_DECISIONS", 1),
				MaxExternalWaits:     intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", 1),
			},
		},
	}
	return orquestaserver.NormalizeConfigV0(config), orquestaserver.ValidateConfigV0(config)
}

func defaultControlDirV0(projectDir string) string {
	cleanProject := filepath.Clean(projectDir)
	base := filepath.Base(cleanProject)
	if base == "" || base == "." || base == string(os.PathSeparator) {
		base = "project"
	}
	return filepath.Join(filepath.Dir(cleanProject), ".orquesta-control", base)
}

func projectDirFromEnvV0() (string, error) {
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROJECT_WORKDIR"))
	if value == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("project_work_dir_unavailable")
		}
		value = cwd
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("project_work_dir_invalid")
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", fmt.Errorf("project_work_dir_unavailable")
	}
	return abs, nil
}

func absDirEnvOrDefaultV0(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return fallback
	}
	_ = os.MkdirAll(abs, 0o700)
	return abs
}

func envOrDefaultV0(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnvOrDefaultV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
