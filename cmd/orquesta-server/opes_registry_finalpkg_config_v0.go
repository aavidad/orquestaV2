package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultOPESRegistryFinalPkgCourseIDV0         = "informatica-a1-72-padres"
	defaultOPESRegistryFinalPkgTemplateRunRefV0   = "run-ref-opes-a1-t001-finalpkg-20260612"
	defaultOPESRegistryFinalPkgTemplateTopicV0    = "001"
	defaultOPESRegistryFinalPkgBatchSizeV0        = 6
	defaultOPESRegistryFinalPkgMaxInFlightV0      = 6
	defaultOPESRegistryFinalPkgReconcileLimitV0   = 70
	defaultOPESRegistryFinalPkgReconcileTimeoutV0 = 5 * time.Minute
	defaultOPESRegistryFinalPkgIntervalV0         = 60 * time.Second
	defaultOPESRegistryFinalPkgInitialDelayV0     = 2 * time.Second
	defaultOPESRegistryFinalPkgHTTPTimeoutV0      = 30 * time.Second
	defaultOPESRegistryFinalPkgQueueRefV0         = "global"
)

type opesRegistryFinalPkgLoopConfigV0 struct {
	Loop     externalBridgeLoopConfigV0
	Producer opesRegistryFinalPkgConfigV0
}

type opesRegistryFinalPkgConfigV0 struct {
	RegistryPath           string
	CourseID               string
	CourseRoot             string
	AppChangeState         string
	OrchestrationStateRoot string
	OrchestrationRuns      string
	TemplateRunRef         string
	TemplateTopicID        string
	OrquestaBaseURL        string
	QueueRef               string
	BatchSize              int
	MaxInFlight            int
	ReconcileCompleted     bool
	ReconcileLimit         int
	DryRun                 bool
	HTTPTimeout            time.Duration
}

func opesRegistryFinalPkgLoopConfigFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	orquestaBaseURLFallback string,
) (opesRegistryFinalPkgLoopConfigV0, error) {
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	if !opesRegistryFinalPkgEnabledFromProjectConfigFileV0(projectConfig) {
		return opesRegistryFinalPkgLoopConfigV0{}, nil
	}
	dryRun := opesRegistryFinalPkgDryRunFromProjectConfigFileV0(projectConfig)
	if !dryRun && !opesRegistryFinalPkgConfirmFromProjectConfigFileV0(projectConfig) {
		return opesRegistryFinalPkgLoopConfigV0{}, fmt.Errorf("%s requerido para crear runs finalpkg desde registro OPES", envOPESRegistryFinalPkgConfirmV0)
	}
	stateDir := strings.TrimSpace(serverConfig.StateDir)
	if stateDir == "" {
		return opesRegistryFinalPkgLoopConfigV0{}, fmt.Errorf("%s requerido para productor finalpkg", envServerStateDirV0)
	}
	orquestaBaseURL, err := orquestaBaseURLFromEnvOrStateOrFallbackV0(dryRun, orquestaBaseURLFallback)
	if err != nil {
		return opesRegistryFinalPkgLoopConfigV0{}, err
	}
	orchestrationStateRoot := filepath.Join(stateDir, "orchestration-state")
	config := opesRegistryFinalPkgConfigV0{
		RegistryPath:           opesRegistryFinalPkgRegistryPathFromProjectConfigFileV0(projectConfig),
		CourseID:               opesRegistryFinalPkgCourseIDFromProjectConfigFileV0(projectConfig),
		CourseRoot:             opesRegistryFinalPkgCourseRootFromProjectConfigFileV0(projectConfig),
		AppChangeState:         filepath.Join(stateDir, "run-state", "app_change_v0.json"),
		OrchestrationStateRoot: orchestrationStateRoot,
		OrchestrationRuns:      filepath.Join(orchestrationStateRoot, "runs"),
		TemplateRunRef:         opesRegistryFinalPkgTemplateRunRefFromProjectConfigFileV0(projectConfig),
		TemplateTopicID:        opesRegistryFinalPkgTemplateTopicIDFromProjectConfigFileV0(projectConfig),
		OrquestaBaseURL:        strings.TrimRight(orquestaBaseURL, "/"),
		QueueRef:               opesRegistryFinalPkgQueueRefFromProjectConfigFileV0(projectConfig),
		BatchSize:              opesRegistryFinalPkgBatchSizeFromProjectConfigFileV0(projectConfig),
		MaxInFlight:            opesRegistryFinalPkgMaxInFlightFromProjectConfigFileV0(projectConfig),
		ReconcileCompleted:     opesRegistryFinalPkgReconcileFromProjectConfigFileV0(projectConfig),
		ReconcileLimit:         opesRegistryFinalPkgReconcileLimitFromProjectConfigFileV0(projectConfig),
		DryRun:                 dryRun,
		HTTPTimeout:            defaultOPESRegistryFinalPkgHTTPTimeoutV0,
	}
	if err := validateOPESRegistryFinalPkgConfigV0(config); err != nil {
		return opesRegistryFinalPkgLoopConfigV0{}, err
	}
	return opesRegistryFinalPkgLoopConfigV0{
		Loop: externalBridgeLoopConfigV0{
			Enabled:       true,
			Component:     "opes_registry_finalpkg_loop",
			ResultField:   "summary",
			FilterSummary: opesRegistryFinalPkgFilterSummaryV0(config),
			Interval:      opesRegistryFinalPkgIntervalFromProjectConfigFileV0(projectConfig),
			InitialDelay:  defaultOPESRegistryFinalPkgInitialDelayV0,
			MaxTicks:      opesRegistryFinalPkgMaxTicksFromProjectConfigFileV0(projectConfig),
			EffectTimeout: opesRegistryFinalPkgEffectTimeoutV0(config),
			RetryPolicy: externalBridgeRetryPolicyV0{
				MaxAttempts: 3,
				BaseDelay:   opesRegistryFinalPkgIntervalFromProjectConfigFileV0(projectConfig),
				MaxDelay:    5 * time.Minute,
			},
		},
		Producer: config,
	}, nil
}

func validateOPESRegistryFinalPkgConfigV0(config opesRegistryFinalPkgConfigV0) error {
	if strings.TrimSpace(config.RegistryPath) == "" {
		return fmt.Errorf("%s requerido", envOPESRegistryFinalPkgRegistryV0)
	}
	if strings.TrimSpace(config.CourseRoot) == "" {
		return fmt.Errorf("%s requerido", envOPESRegistryFinalPkgCourseRootV0)
	}
	missing := make([]string, 0, 4)
	if strings.TrimSpace(config.CourseID) == "" {
		missing = append(missing, envOPESRegistryFinalPkgCourseIDV0)
	}
	if strings.TrimSpace(config.TemplateRunRef) == "" {
		missing = append(missing, envOPESRegistryFinalPkgTemplateRunV0)
	}
	if strings.TrimSpace(config.TemplateTopicID) == "" {
		missing = append(missing, envOPESRegistryFinalPkgTemplateTopicV0)
	}
	if strings.TrimSpace(config.QueueRef) == "" {
		missing = append(missing, envOPESRegistryFinalPkgQueueRefV0)
	}
	if len(missing) > 0 {
		return fmt.Errorf("opes_registry_finalpkg_config_incomplete: %s", strings.Join(missing, ","))
	}
	if config.BatchSize < 1 || config.MaxInFlight < 1 || config.ReconcileLimit < 1 {
		return fmt.Errorf("opes_registry_finalpkg_limits_invalid")
	}
	if config.ReconcileCompleted && strings.TrimSpace(config.OrchestrationStateRoot) == "" {
		return fmt.Errorf("opes_registry_finalpkg_orchestration_state_root_required")
	}
	return nil
}

func opesRegistryFinalPkgDryRunFromEnvV0() bool {
	return opesRegistryFinalPkgDryRunFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func opesRegistryFinalPkgEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgEnabledV0, config.OPESRegistryFinalPkg.Enabled, false)
}

func opesRegistryFinalPkgConfirmFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgConfirmV0, config.OPESRegistryFinalPkg.Confirm, false)
}

func opesRegistryFinalPkgDryRunFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	value := strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgDryRunV0))
	if value != "" {
		return boolEnvOrDefaultV0(envOPESRegistryFinalPkgDryRunV0, true)
	}
	if config.OPESRegistryFinalPkg.DryRun != nil {
		return *config.OPESRegistryFinalPkg.DryRun
	}
	return true
}

func opesRegistryFinalPkgRegistryPathFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgRegistryV0, config.OPESRegistryFinalPkg.RegistryPath, "")
}

func opesRegistryFinalPkgCourseIDFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return opesRegistryFinalPkgStringWithDryRunDefaultFromProjectConfigFileV0(
		config,
		envOPESRegistryFinalPkgCourseIDV0,
		config.OPESRegistryFinalPkg.CourseID,
		defaultOPESRegistryFinalPkgCourseIDV0,
	)
}

func opesRegistryFinalPkgCourseRootFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgCourseRootV0, config.OPESRegistryFinalPkg.CourseRoot, "")
}

func opesRegistryFinalPkgTemplateRunRefFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return opesRegistryFinalPkgStringWithDryRunDefaultFromProjectConfigFileV0(
		config,
		envOPESRegistryFinalPkgTemplateRunV0,
		config.OPESRegistryFinalPkg.TemplateRunRef,
		defaultOPESRegistryFinalPkgTemplateRunRefV0,
	)
}

func opesRegistryFinalPkgTemplateTopicIDFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return opesRegistryFinalPkgStringWithDryRunDefaultFromProjectConfigFileV0(
		config,
		envOPESRegistryFinalPkgTemplateTopicV0,
		config.OPESRegistryFinalPkg.TemplateTopicID,
		defaultOPESRegistryFinalPkgTemplateTopicV0,
	)
}

func opesRegistryFinalPkgStringWithDryRunDefaultFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	key string,
	fileValue *string,
	dryRunFallback string,
) string {
	fallback := ""
	if opesRegistryFinalPkgDryRunFromProjectConfigFileV0(config) {
		fallback = dryRunFallback
	}
	return stringProjectConfigOrEnvOrDefaultV0(key, fileValue, fallback)
}

func opesRegistryFinalPkgBatchSizeFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgBatchSizeV0, config.OPESRegistryFinalPkg.BatchSize, defaultOPESRegistryFinalPkgBatchSizeV0)
}

func opesRegistryFinalPkgMaxInFlightFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgMaxInFlightV0, config.OPESRegistryFinalPkg.MaxInFlight, defaultOPESRegistryFinalPkgMaxInFlightV0)
}

func opesRegistryFinalPkgIntervalFromProjectConfigFileV0(config serverProjectConfigFileV0) time.Duration {
	return durationSecondsProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgIntervalV0, config.OPESRegistryFinalPkg.IntervalSeconds, defaultOPESRegistryFinalPkgIntervalV0)
}

func opesRegistryFinalPkgMaxTicksFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrZeroV0(envOPESRegistryFinalPkgMaxTicksV0, config.OPESRegistryFinalPkg.MaxTicks)
}

func opesRegistryFinalPkgQueueRefFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgQueueRefV0, config.OPESRegistryFinalPkg.QueueRef, defaultOPESRegistryFinalPkgQueueRefV0)
}

func opesRegistryFinalPkgReconcileFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgReconcileV0, config.OPESRegistryFinalPkg.ReconcileEnabled, false)
}

func opesRegistryFinalPkgReconcileLimitFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(envOPESRegistryFinalPkgReconcileLimitV0, config.OPESRegistryFinalPkg.ReconcileLimit, defaultOPESRegistryFinalPkgReconcileLimitV0)
}

func durationSecondsProjectConfigOrEnvOrDefaultV0(key string, fileValue *int, fallback time.Duration) time.Duration {
	value := intProjectConfigOrEnvOrDefaultV0(key, fileValue, int(fallback/time.Second))
	if value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Second
}

func intProjectConfigOrEnvOrZeroV0(key string, fileValue *int) int {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return intEnvOrZeroV0(key)
	}
	if fileValue != nil && *fileValue >= 0 {
		return *fileValue
	}
	return 0
}

func opesRegistryFinalPkgEffectTimeoutV0(config opesRegistryFinalPkgConfigV0) time.Duration {
	if config.ReconcileCompleted && !config.DryRun {
		return defaultOPESRegistryFinalPkgReconcileTimeoutV0
	}
	return config.HTTPTimeout
}

func opesRegistryFinalPkgFilterSummaryV0(config opesRegistryFinalPkgConfigV0) []string {
	return compactExternalBridgeStringsV0([]string{
		"course_id=" + strings.TrimSpace(config.CourseID),
		"queue_ref=" + strings.TrimSpace(config.QueueRef),
		"registry_path=configured",
		"course_root=configured",
		"dry_run=" + strconv.FormatBool(config.DryRun),
	})
}

func opesRegistryFinalPkgEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgEnabledV0, strconv.FormatBool(opesRegistryFinalPkgEnabledFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgEnabledV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgConfirmV0, strconv.FormatBool(opesRegistryFinalPkgConfirmFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgConfirmV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgDryRunV0, strconv.FormatBool(opesRegistryFinalPkgDryRunFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgDryRunV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envOPESRegistryFinalPkgRegistryV0,
			sensitiveConfigValueFromSourceV0(opesRegistryFinalPkgRegistryPathFromProjectConfigFileV0(projectConfig), "opes-registry-finalpkg-registry-path-configured", configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgRegistryV0)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgRegistryV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgCourseIDV0, opesRegistryFinalPkgCourseIDFromProjectConfigFileV0(projectConfig), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgCourseIDV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envOPESRegistryFinalPkgCourseRootV0,
			sensitiveConfigValueFromSourceV0(opesRegistryFinalPkgCourseRootFromProjectConfigFileV0(projectConfig), "opes-registry-finalpkg-course-root-configured", configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgCourseRootV0)),
			configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgCourseRootV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgTemplateRunV0, opesRegistryFinalPkgTemplateRunRefFromProjectConfigFileV0(projectConfig), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgTemplateRunV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgTemplateTopicV0, opesRegistryFinalPkgTemplateTopicIDFromProjectConfigFileV0(projectConfig), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgTemplateTopicV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgBatchSizeV0, strconv.Itoa(opesRegistryFinalPkgBatchSizeFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgBatchSizeV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgMaxInFlightV0, strconv.Itoa(opesRegistryFinalPkgMaxInFlightFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgMaxInFlightV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgIntervalV0, strconv.Itoa(int(opesRegistryFinalPkgIntervalFromProjectConfigFileV0(projectConfig)/time.Second)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgIntervalV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgMaxTicksV0, strconv.Itoa(opesRegistryFinalPkgMaxTicksFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgMaxTicksV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgQueueRefV0, opesRegistryFinalPkgQueueRefFromProjectConfigFileV0(projectConfig), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgQueueRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgReconcileV0, strconv.FormatBool(opesRegistryFinalPkgReconcileFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgReconcileV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOPESRegistryFinalPkgReconcileLimitV0, strconv.Itoa(opesRegistryFinalPkgReconcileLimitFromProjectConfigFileV0(projectConfig)), configSettingSourceFromConfigOrProjectConfigV0(config, envOPESRegistryFinalPkgReconcileLimitV0)),
	}
}
