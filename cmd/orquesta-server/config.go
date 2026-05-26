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
	stateDir := absDirEnvOrDefaultV0(envServerStateDirV0,
		filepath.Join(defaultControlDirV0(projectDir), "state"))
	runtimeDir := absDirEnvOrDefaultV0(envCodexRuntimeWorkDirV0,
		filepath.Join(projectDir, ".orquesta-runtime"))
	config := orquestaserver.ConfigV0{
		Addr:          envOrDefaultV0(envServerAddrV0, orquestaserver.DefaultAddrV0),
		StateDir:      stateDir,
		AuditFile:     envOrDefaultV0(envServerAuditFileV0, orquestaserver.DefaultAuditFileV0),
		AuditDisabled: boolEnvOrDefaultV0(envServerAuditDisabledV0, false),
		ControlPlane: orquestaserver.ControlPlaneConfigV0{
			RemoteAccessOptIn: boolEnvOrDefaultV0(envServerRemoteControlPlaneConfirmV0, false),
			Token:             os.Getenv(envServerControlTokenV0),
			Principal:         envOrDefaultV0(envServerControlPrincipalV0, "loopback-local"),
			PermissionRef:     envOrDefaultV0(envServerControlPermissionRefV0, "permission-ref-loopback-control-plane"),
			PublicReason:      envOrDefaultV0(envServerControlPublicReasonV0, "loopback_control_plane"),
		},
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		TickInterval:   time.Duration(intEnvOrDefaultV0(envServerTickIntervalMSV0, 5000)) * time.Millisecond,
		IdleSelfImprovementAfter: time.Duration(intEnvOrDefaultAllowZeroV0(
			envServerIdleSelfImprovementAfterV0,
			int(orquestaserver.DefaultIdleSelfImprovementAfterV0/time.Second),
		)) * time.Second,
		IdleSelfImprovementDisabled:      strings.TrimSpace(os.Getenv(envServerIdleSelfImprovementAfterV0)) == "0",
		IdleSelfImprovementProjectRef:    envOrDefaultV0(envServerIdleSelfImprovementProjectRefV0, orquestaserver.DefaultIdleSelfImprovementProjectRefV0),
		IdleSelfImprovementWorktreeRef:   envOrDefaultV0(envServerIdleSelfImprovementWorktreeRefV0, orquestaserver.DefaultIdleSelfImprovementWorktreeRefV0),
		IdleSelfImprovementBranchRef:     envOrDefaultV0(envServerIdleSelfImprovementBranchRefV0, orquestaserver.DefaultIdleSelfImprovementBranchRefV0),
		IdleSelfImprovementSuggestedArea: envOrDefaultV0(envServerIdleSelfImprovementAreaV0, orquestaserver.DefaultIdleSelfImprovementSuggestedAreaV0),
		IdleSelfImprovementWriteSet:      csvEnvOrDefaultV0(envServerIdleSelfImprovementWriteSetV0, defaultIdleSelfImprovementWriteSetV0()),
		IdleSelfImprovementRequiredTests: csvEnvOrDefaultV0(envServerIdleSelfImprovementRequiredTestsV0, []string{orquestaserver.DefaultIdleSelfImprovementRequiredTestV0}),
		IdleSelfImprovementContextRefs:   csvEnvOrDefaultV0(envServerIdleSelfImprovementContextRefsV0, nil),
		IdleSelfImprovementEvidenceRefs:  csvEnvOrDefaultV0(envServerIdleSelfImprovementEvidenceRefsV0, nil),
		IdleSelfImprovementAcceptance:    csvEnvOrDefaultV0(envServerIdleSelfImprovementAcceptanceV0, defaultIdleSelfImprovementAcceptanceV0()),
		IdleSelfImprovementCompactRules:  csvEnvOrDefaultV0(envServerIdleSelfImprovementCompactRulesV0, defaultIdleSelfImprovementCompactRulesV0()),
		IdleSelfImprovementPriorityScore: intEnvOrDefaultV0(envServerIdleSelfImprovementPriorityScoreV0, orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0),
		IdleSelfImprovementMaxRequests:   intEnvOrDefaultV0(envServerIdleSelfImprovementMaxRequestsV0, orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0),
		IdleSelfImprovementTargetQueue:   intEnvOrDefaultV0(envServerIdleSelfImprovementTargetQueueV0, orquestaserver.DefaultIdleSelfImprovementTargetQueueV0),
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:          "global",
			MaxRunsPerTick:    intEnvOrDefaultV0(envServerMaxRunsPerTickV0, defaultCodexServerMaxRunsPerTickV0),
			MaxTicks:          intEnvOrDefaultV0(envServerSupervisorMaxTicksV0, orquestaserver.DefaultSupervisorMaxTicksV0),
			MaxExecutions:     intEnvOrDefaultV0(envServerMaxExecutionsPerTickV0, defaultCodexServerMaxExecutionsV0),
			StopOnNoExecution: true,
			AllowRepeatedRuns: boolEnvOrDefaultV0(envServerAllowRepeatedRunsV0, false),
			DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
				MaxBursts:            intEnvOrDefaultV0(envServerDrainMaxBurstsV0, 4),
				MaxStepsPerBurst:     intEnvOrDefaultV0(envServerDrainMaxStepsV0, 6),
				MaxDispatchesPerWait: intEnvOrDefaultV0(envServerDrainMaxDispatchesV0, 4),
				MaxCommands:          intEnvOrDefaultV0(envServerDrainMaxCommandsV0, 20),
				MaxOutboxPerCycle:    intEnvOrDefaultV0(envServerDrainMaxOutboxV0, 4),
				MaxDecisionCycles:    intEnvOrDefaultV0(envServerDrainMaxDecisionsV0, 1),
				MaxExternalWaits:     serverSupervisorMaxExternalWaitsV0(),
			},
		},
	}
	config.EffectiveConfig = serverEffectiveConfigFromEnvV0(config)
	return orquestaserver.NormalizeConfigV0(config), orquestaserver.ValidateConfigV0(config)
}

func setDefaultStartupCleanupModeV0() {
	if strings.TrimSpace(os.Getenv(envStartupCleanupModeV0)) != "" {
		return
	}
	_ = os.Setenv(envStartupCleanupModeV0, "forced_stop")
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
		envServerIdleSelfImprovementAfterV0 + " por defecto dispara tras 60 segundos sin ejecuciones",
		envServerIdleSelfImprovementAfterV0 + "=0 desactiva automejora idle",
		"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de " + envServerIdleSelfImprovementTargetQueueV0,
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
		envServerDrainMaxExternalWaitsV0,
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
	value := strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0))
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
