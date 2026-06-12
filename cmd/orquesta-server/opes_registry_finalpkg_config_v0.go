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
	if strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgEnabledV0)) != "1" {
		return opesRegistryFinalPkgLoopConfigV0{}, nil
	}
	dryRun := opesRegistryFinalPkgDryRunFromEnvV0()
	if !dryRun && strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgConfirmV0)) != "1" {
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
		RegistryPath:           strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgRegistryV0)),
		CourseID:               envOrDefaultV0(envOPESRegistryFinalPkgCourseIDV0, defaultOPESRegistryFinalPkgCourseIDV0),
		CourseRoot:             strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgCourseRootV0)),
		AppChangeState:         filepath.Join(stateDir, "run-state", "app_change_v0.json"),
		OrchestrationStateRoot: orchestrationStateRoot,
		OrchestrationRuns:      filepath.Join(orchestrationStateRoot, "runs"),
		TemplateRunRef:         envOrDefaultV0(envOPESRegistryFinalPkgTemplateRunV0, defaultOPESRegistryFinalPkgTemplateRunRefV0),
		TemplateTopicID:        envOrDefaultV0(envOPESRegistryFinalPkgTemplateTopicV0, defaultOPESRegistryFinalPkgTemplateTopicV0),
		OrquestaBaseURL:        strings.TrimRight(orquestaBaseURL, "/"),
		QueueRef:               envOrDefaultV0(envOPESRegistryFinalPkgQueueRefV0, defaultOPESRegistryFinalPkgQueueRefV0),
		BatchSize:              intEnvOrDefaultV0(envOPESRegistryFinalPkgBatchSizeV0, defaultOPESRegistryFinalPkgBatchSizeV0),
		MaxInFlight:            intEnvOrDefaultV0(envOPESRegistryFinalPkgMaxInFlightV0, defaultOPESRegistryFinalPkgMaxInFlightV0),
		ReconcileCompleted:     strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgReconcileV0)) == "1",
		ReconcileLimit:         intEnvOrDefaultV0(envOPESRegistryFinalPkgReconcileLimitV0, defaultOPESRegistryFinalPkgReconcileLimitV0),
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
			Interval:      durationSecondsEnvOrDefaultV0(envOPESRegistryFinalPkgIntervalV0, defaultOPESRegistryFinalPkgIntervalV0),
			InitialDelay:  defaultOPESRegistryFinalPkgInitialDelayV0,
			MaxTicks:      intEnvOrZeroV0(envOPESRegistryFinalPkgMaxTicksV0),
			EffectTimeout: opesRegistryFinalPkgEffectTimeoutV0(config),
			RetryPolicy: externalBridgeRetryPolicyV0{
				MaxAttempts: 3,
				BaseDelay:   durationSecondsEnvOrDefaultV0(envOPESRegistryFinalPkgIntervalV0, defaultOPESRegistryFinalPkgIntervalV0),
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
	if strings.TrimSpace(config.CourseID) == "" ||
		strings.TrimSpace(config.TemplateRunRef) == "" ||
		strings.TrimSpace(config.TemplateTopicID) == "" ||
		strings.TrimSpace(config.QueueRef) == "" {
		return fmt.Errorf("opes_registry_finalpkg_config_incomplete")
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
	value := strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgDryRunV0))
	if value == "" {
		return true
	}
	return value != "0" && !strings.EqualFold(value, "false")
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

func opesRegistryFinalPkgEffectiveConfigSettingsV0() []orquestaserver.ServerConfigSettingV0 {
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgEnabledV0, strconv.FormatBool(strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgEnabledV0)) == "1")),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgConfirmV0, strconv.FormatBool(strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgConfirmV0)) == "1")),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgDryRunV0, strconv.FormatBool(opesRegistryFinalPkgDryRunFromEnvV0())),
		serverSensitiveConfigSettingFromRegistryV0(envOPESRegistryFinalPkgRegistryV0, configuredEnvValueV0(envOPESRegistryFinalPkgRegistryV0, "opes-registry-finalpkg-registry-path-configured")),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgCourseIDV0, envOrDefaultV0(envOPESRegistryFinalPkgCourseIDV0, defaultOPESRegistryFinalPkgCourseIDV0)),
		serverSensitiveConfigSettingFromRegistryV0(envOPESRegistryFinalPkgCourseRootV0, configuredEnvValueV0(envOPESRegistryFinalPkgCourseRootV0, "opes-registry-finalpkg-course-root-configured")),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgTemplateRunV0, envOrDefaultV0(envOPESRegistryFinalPkgTemplateRunV0, defaultOPESRegistryFinalPkgTemplateRunRefV0)),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgTemplateTopicV0, envOrDefaultV0(envOPESRegistryFinalPkgTemplateTopicV0, defaultOPESRegistryFinalPkgTemplateTopicV0)),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgBatchSizeV0, strconv.Itoa(intEnvOrDefaultV0(envOPESRegistryFinalPkgBatchSizeV0, defaultOPESRegistryFinalPkgBatchSizeV0))),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgMaxInFlightV0, strconv.Itoa(intEnvOrDefaultV0(envOPESRegistryFinalPkgMaxInFlightV0, defaultOPESRegistryFinalPkgMaxInFlightV0))),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgIntervalV0, strconv.Itoa(int(durationSecondsEnvOrDefaultV0(envOPESRegistryFinalPkgIntervalV0, defaultOPESRegistryFinalPkgIntervalV0)/time.Second))),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgMaxTicksV0, strconv.Itoa(intEnvOrZeroV0(envOPESRegistryFinalPkgMaxTicksV0))),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgQueueRefV0, envOrDefaultV0(envOPESRegistryFinalPkgQueueRefV0, defaultOPESRegistryFinalPkgQueueRefV0)),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgReconcileV0, strconv.FormatBool(strings.TrimSpace(os.Getenv(envOPESRegistryFinalPkgReconcileV0)) == "1")),
		serverConfigSettingFromRegistryV0(envOPESRegistryFinalPkgReconcileLimitV0, strconv.Itoa(intEnvOrDefaultV0(envOPESRegistryFinalPkgReconcileLimitV0, defaultOPESRegistryFinalPkgReconcileLimitV0))),
	}
}
