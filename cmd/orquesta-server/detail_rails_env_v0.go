package main

import (
	"os"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	detailProhibitedRailsEnvV0                = envDetailProhibitedRailsV0
	detailProhibitedRailsScopeEnvV0           = envDetailProhibitedRailsScopeV0
	securityModeServerDefaultV0               = orquestarails.SecurityModeProductionV0
	railsModeServerDefaultV0                  = orquestarails.RailsModeOfflineV0
	detailProhibitedRailsServerDefaultV0      = "off"
	detailProhibitedRailsScopeServerDefaultV0 = "core_workflow.*,context_bundle_request.*," +
		"context_materialization.content,context_materialization.ref,director_agent_decision.*"
)

func serverSecurityModeEffectiveValueV0() string {
	return serverSecurityModeEffectiveValueFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func serverRailsModeEffectiveValueV0() string {
	return serverRailsModeEffectiveValueFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func serverDetailRailsEffectiveValueV0() string {
	return serverDetailRailsEffectiveValueFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func serverDetailRailsScopeEffectiveValueV0() string {
	return serverDetailRailsScopeEffectiveValueFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func serverSecurityModeEffectiveValueFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return orquestarails.NormalizeSecurityModeV0(stringProjectConfigOrEnvOrDefaultV0(
		envSecurityModeV0,
		config.RailsSecurity.SecurityMode,
		securityModeServerDefaultV0,
	))
}

func serverRailsModeEffectiveValueFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return orquestarails.NormalizeRailsModeV0(stringProjectConfigOrEnvOrDefaultV0(
		envRailsModeV0,
		config.RailsSecurity.RailsMode,
		railsModeServerDefaultV0,
	))
}

func serverDetailRailsEffectiveValueFromProjectConfigFileV0(serverProjectConfigFileV0) string {
	return "off"
}

func serverDetailRailsScopeEffectiveValueFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		detailProhibitedRailsScopeEnvV0,
		config.RailsSecurity.DetailProhibitedRailsScope,
		detailProhibitedRailsScopeServerDefaultV0,
	)
}

func serverEnvironmentWithDetailRailsDefaultV0(env []string) []string {
	out := append([]string(nil), env...)
	securityMode := orquestarails.NormalizeSecurityModeV0(
		detailRailsEnvValueOrDefaultV0(out, envSecurityModeV0, securityModeServerDefaultV0),
	)
	railsMode := orquestarails.NormalizeRailsModeV0(
		detailRailsEnvValueOrDefaultV0(out, envRailsModeV0, railsModeServerDefaultV0),
	)
	detailRails := detailProhibitedRailsServerDefaultV0
	scope := detailRailsEnvValueOrDefaultV0(out, detailProhibitedRailsScopeEnvV0, detailProhibitedRailsScopeServerDefaultV0)
	out = detailRailsEnvUpsertV0(out, envSecurityModeV0, securityMode)
	out = detailRailsEnvUpsertV0(out, envRailsModeV0, railsMode)
	out = detailRailsEnvUpsertV0(out, detailProhibitedRailsEnvV0, detailRails)
	out = detailRailsEnvUpsertV0(out, detailProhibitedRailsScopeEnvV0, scope)
	return out
}

func applyServerDetailRailsRuntimeDefaultsV0() error {
	env := serverEnvironmentWithDetailRailsDefaultV0(os.Environ())
	for _, key := range []string{envSecurityModeV0, envRailsModeV0, detailProhibitedRailsEnvV0, detailProhibitedRailsScopeEnvV0} {
		value := detailRailsEnvValueOrDefaultV0(env, key, "")
		if value == "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func detailRailsEnvValueOrDefaultV0(env []string, key string, fallback string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			if value := strings.TrimSpace(strings.TrimPrefix(item, prefix)); value != "" {
				return value
			}
			return fallback
		}
	}
	return fallback
}

func detailRailsEnvUpsertV0(env []string, key string, value string) []string {
	prefix := key + "="
	for index, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[index] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func detailRailsEnvPresentV0(env []string, key string) bool {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}
