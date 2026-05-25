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

const (
	defaultServerSupervisorMaxExternalWaitsV0 = 1
	maxServerSupervisorMaxExternalWaitsV0     = 70
)

func serverConfigFromEnvV0() (orquestaserver.ConfigV0, error) {
	setDefaultStartupCleanupModeV0()
	projectDir, err := projectDirFromEnvV0()
	if err != nil {
		return orquestaserver.ConfigV0{}, err
	}
	stateDir := absDirEnvOrDefaultV0("ORQUESTA_SERVER_STATE_DIR",
		filepath.Join(defaultControlDirV0(projectDir), "state"))
	runtimeDir := absDirEnvOrDefaultV0("ORQUESTA_CODEX_RUNTIME_WORKDIR",
		filepath.Join(projectDir, ".orquesta-runtime"))
	config := orquestaserver.ConfigV0{
		Addr:          envOrDefaultV0("ORQUESTA_SERVER_ADDR", orquestaserver.DefaultAddrV0),
		StateDir:      stateDir,
		AuditFile:     envOrDefaultV0("ORQUESTA_SERVER_AUDIT_FILE", orquestaserver.DefaultAuditFileV0),
		AuditDisabled: boolEnvOrDefaultV0("ORQUESTA_SERVER_AUDIT_DISABLED", false),
		ControlPlane: orquestaserver.ControlPlaneConfigV0{
			RemoteAccessOptIn: boolEnvOrDefaultV0("ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM", false),
			Token:             os.Getenv("ORQUESTA_SERVER_CONTROL_TOKEN"),
			Principal:         envOrDefaultV0("ORQUESTA_SERVER_CONTROL_PRINCIPAL", "loopback-local"),
			PermissionRef:     envOrDefaultV0("ORQUESTA_SERVER_CONTROL_PERMISSION_REF", "permission-ref-loopback-control-plane"),
			PublicReason:      envOrDefaultV0("ORQUESTA_SERVER_CONTROL_PUBLIC_REASON", "loopback_control_plane"),
		},
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		TickInterval:   time.Duration(intEnvOrDefaultV0("ORQUESTA_SERVER_TICK_INTERVAL_MS", 5000)) * time.Millisecond,
		IdleSelfImprovementAfter: time.Duration(intEnvOrDefaultAllowZeroV0(
			"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS",
			int(orquestaserver.DefaultIdleSelfImprovementAfterV0/time.Second),
		)) * time.Second,
		IdleSelfImprovementDisabled:      strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS")) == "0",
		IdleSelfImprovementProjectRef:    envOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_REF", orquestaserver.DefaultIdleSelfImprovementProjectRefV0),
		IdleSelfImprovementWorktreeRef:   envOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WORKTREE_REF", orquestaserver.DefaultIdleSelfImprovementWorktreeRefV0),
		IdleSelfImprovementBranchRef:     envOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_BRANCH_REF", orquestaserver.DefaultIdleSelfImprovementBranchRefV0),
		IdleSelfImprovementSuggestedArea: envOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AREA", orquestaserver.DefaultIdleSelfImprovementSuggestedAreaV0),
		IdleSelfImprovementWriteSet:      csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WRITE_SET", defaultIdleSelfImprovementWriteSetV0()),
		IdleSelfImprovementRequiredTests: csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_REQUIRED_TESTS", []string{orquestaserver.DefaultIdleSelfImprovementRequiredTestV0}),
		IdleSelfImprovementContextRefs:   csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_CONTEXT_REFS", nil),
		IdleSelfImprovementEvidenceRefs:  csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_EVIDENCE_REFS", nil),
		IdleSelfImprovementAcceptance:    csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_ACCEPTANCE", defaultIdleSelfImprovementAcceptanceV0()),
		IdleSelfImprovementCompactRules:  csvEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_COMPACT_RULES", defaultIdleSelfImprovementCompactRulesV0()),
		IdleSelfImprovementPriorityScore: intEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PRIORITY_SCORE", orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0),
		IdleSelfImprovementMaxRequests:   intEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS", orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0),
		IdleSelfImprovementTargetQueue:   intEnvOrDefaultV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE", orquestaserver.DefaultIdleSelfImprovementTargetQueueV0),
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:          "global",
			MaxRunsPerTick:    intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", defaultCodexServerMaxRunsPerTickV0),
			MaxTicks:          intEnvOrDefaultV0("ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS", orquestaserver.DefaultSupervisorMaxTicksV0),
			MaxExecutions:     intEnvOrDefaultV0("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", defaultCodexServerMaxExecutionsV0),
			StopOnNoExecution: true,
			AllowRepeatedRuns: boolEnvOrDefaultV0("ORQUESTA_SERVER_ALLOW_REPEATED_RUNS", false),
			DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
				MaxBursts:            intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_BURSTS", 4),
				MaxStepsPerBurst:     intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_STEPS", 6),
				MaxDispatchesPerWait: intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES", 4),
				MaxCommands:          intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_COMMANDS", 20),
				MaxOutboxPerCycle:    intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_OUTBOX", 4),
				MaxDecisionCycles:    intEnvOrDefaultV0("ORQUESTA_SERVER_DRAIN_MAX_DECISIONS", 1),
				MaxExternalWaits:     serverSupervisorMaxExternalWaitsV0(),
			},
		},
	}
	config.EffectiveConfig = serverEffectiveConfigFromEnvV0(config)
	return orquestaserver.NormalizeConfigV0(config), orquestaserver.ValidateConfigV0(config)
}

func setDefaultStartupCleanupModeV0() {
	if strings.TrimSpace(os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE")) != "" {
		return
	}
	_ = os.Setenv("ORQUESTA_STARTUP_CLEANUP_MODE", "forced_stop")
}

func defaultIdleSelfImprovementWriteSetV0() []string {
	return []string{
		"modulos/orquesta-server/config_v0.go",
		"modulos/orquesta-server/ports_v0.go",
		"modulos/orquesta-server/supervisor_loop_v0.go",
		"modulos/orquesta-server/status_tracker_v0.go",
		"modulos/orquesta-server/supervisor_loop_v0_test.go",
		"cmd/orquesta-server/config.go",
		"cmd/orquesta-server/config_test.go",
		"cmd/orquesta-server/stack.go",
	}
}

func defaultIdleSelfImprovementAcceptanceV0() []string {
	return []string{
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS por defecto dispara tras 60 segundos sin ejecuciones",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 desactiva automejora idle",
		"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE",
		"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
		"usar evidencia del fallo y corregir la causa general si es posible",
	}
}

func defaultIdleSelfImprovementCompactRulesV0() []string {
	return []string{
		"comunicacion compacta",
		"trabajo secundario: no bloquear ni mezclar con el trabajo principal",
	}
}

func serverSupervisorMaxExternalWaitsV0() int {
	value := intEnvOrDefaultV0(
		"ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS",
		defaultServerSupervisorMaxExternalWaitsV0,
	)
	if value <= 0 {
		return defaultServerSupervisorMaxExternalWaitsV0
	}
	if value > maxServerSupervisorMaxExternalWaitsV0 {
		return maxServerSupervisorMaxExternalWaitsV0
	}
	return value
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

func intEnvOrDefaultAllowZeroV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func csvEnvOrDefaultV0(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
