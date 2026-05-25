package main

import (
	"os"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	detailProhibitedRailsEnvV0                = orquestarails.DetailProhibitedRailsEnvV0
	detailProhibitedRailsScopeEnvV0           = orquestarails.DetailProhibitedRailsScopeEnvV0
	detailProhibitedRailsServerDefaultV0      = "off"
	detailProhibitedRailsScopeServerDefaultV0 = "core_workflow.*,context_bundle_request.*," +
		"context_materialization.content,context_materialization.ref,director_agent_decision.*"
)

func ensureServerDetailRailsDefaultV0() {
	if strings.TrimSpace(os.Getenv(detailProhibitedRailsEnvV0)) == "" {
		_ = os.Setenv(detailProhibitedRailsEnvV0, detailProhibitedRailsServerDefaultV0)
	}
	if strings.TrimSpace(os.Getenv(detailProhibitedRailsScopeEnvV0)) == "" {
		_ = os.Setenv(detailProhibitedRailsScopeEnvV0, detailProhibitedRailsScopeServerDefaultV0)
	}
}

func serverEnvironmentWithDetailRailsDefaultV0(env []string) []string {
	out := append([]string(nil), env...)
	if !detailRailsEnvPresentV0(out, detailProhibitedRailsEnvV0) {
		out = append(out, detailProhibitedRailsEnvV0+"="+detailProhibitedRailsServerDefaultV0)
	}
	if !detailRailsEnvPresentV0(out, detailProhibitedRailsScopeEnvV0) {
		out = append(out, detailProhibitedRailsScopeEnvV0+"="+detailProhibitedRailsScopeServerDefaultV0)
	}
	return out
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
