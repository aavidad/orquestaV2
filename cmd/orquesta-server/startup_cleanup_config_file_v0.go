package main

import "strings"

type serverProjectConfigStartupCleanupV0 struct {
	Mode       *string   `json:"mode,omitempty"`
	ScopeRefs  *[]string `json:"scope_refs,omitempty"`
	QueueLimit *int      `json:"queue_limit,omitempty"`
}

type serverStartupCleanupConfigV0 struct {
	Mode       string
	ScopeRefs  []string
	QueueLimit int
}

func startupCleanupConfigFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
) serverStartupCleanupConfigV0 {
	return serverStartupCleanupConfigV0{
		Mode: strings.ToLower(stringProjectConfigOrEnvOrDefaultV0(
			envStartupCleanupModeV0,
			config.StartupCleanup.Mode,
			startupCleanupModeDiagnoseV0,
		)),
		ScopeRefs: stringSliceProjectConfigOrEnvOrDefaultV0(
			envStartupCleanupScopeRefsV0,
			config.StartupCleanup.ScopeRefs,
			nil,
		),
		QueueLimit: intProjectConfigOrEnvOrDefaultV0(
			envStartupQueueLimitV0,
			config.StartupCleanup.QueueLimit,
			0,
		),
	}
}
