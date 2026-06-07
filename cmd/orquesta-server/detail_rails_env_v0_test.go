package main

import (
	"os"
	"strings"
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func TestServerDetailRailsEffectiveDefaultsV0NoMutanEntornoGlobal(t *testing.T) {
	t.Setenv(detailProhibitedRailsEnvV0, "")
	t.Setenv(detailProhibitedRailsScopeEnvV0, "")
	t.Setenv(envSecurityModeV0, "")
	t.Setenv(envRailsModeV0, "")

	if got := serverSecurityModeEffectiveValueV0(); got != securityModeServerDefaultV0 {
		t.Fatalf("security_mode=%q, want %q", got, securityModeServerDefaultV0)
	}
	if got := serverRailsModeEffectiveValueV0(); got != railsModeServerDefaultV0 {
		t.Fatalf("rails_mode=%q, want %q", got, railsModeServerDefaultV0)
	}
	if got := serverDetailRailsEffectiveValueV0(); got != "off" {
		t.Fatalf("default=%q, want off", got)
	}
	if got := serverDetailRailsScopeEffectiveValueV0(); got != detailProhibitedRailsScopeServerDefaultV0 {
		t.Fatalf("scope=%q, want %q", got, detailProhibitedRailsScopeServerDefaultV0)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0AnadeRailReactivoSiFalta(t *testing.T) {
	got := serverEnvironmentWithDetailRailsDefaultV0([]string{"PATH=/tmp/bin"})
	if !detailRailsEnvPresentV0(got, envSecurityModeV0) {
		t.Fatalf("%s no añadido: %v", envSecurityModeV0, got)
	}
	if !detailRailsEnvPresentV0(got, envRailsModeV0) {
		t.Fatalf("%s no añadido: %v", envRailsModeV0, got)
	}
	if !detailRailsEnvPresentV0(got, detailProhibitedRailsEnvV0) {
		t.Fatalf("%s no añadido: %v", detailProhibitedRailsEnvV0, got)
	}
	if !detailRailsEnvPresentV0(got, detailProhibitedRailsScopeEnvV0) {
		t.Fatalf("%s no añadido: %v", detailProhibitedRailsScopeEnvV0, got)
	}
	if got[len(got)-4] != envSecurityModeV0+"="+securityModeServerDefaultV0 {
		t.Fatalf("env=%v, want security mode default", got)
	}
	if got[len(got)-3] != envRailsModeV0+"="+railsModeServerDefaultV0 {
		t.Fatalf("env=%v, want rails mode default", got)
	}
	if got[len(got)-2] != detailProhibitedRailsEnvV0+"="+detailProhibitedRailsServerDefaultV0 {
		t.Fatalf("env=%v, want rails default", got)
	}
	if got[len(got)-1] != detailProhibitedRailsScopeEnvV0+"="+detailProhibitedRailsScopeServerDefaultV0 {
		t.Fatalf("env=%v, want scope default", got)
	}
}

func TestServerDetailRailsEffectiveV0ModoProgramacionAbreOverrideExplicito(t *testing.T) {
	t.Setenv(envSecurityModeV0, "programming")
	t.Setenv(envRailsModeV0, orquestarails.RailsModeEnforcedV0)
	t.Setenv(detailProhibitedRailsEnvV0, "on")

	if got := serverSecurityModeEffectiveValueV0(); got != "programming" {
		t.Fatalf("security_mode=%q, want programming", got)
	}
	if got := serverDetailRailsEffectiveValueV0(); got != "off" {
		t.Fatalf("rails=%q, want off para programming", got)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0NormalizaRailsExplicitosAOff(t *testing.T) {
	env := []string{
		envSecurityModeV0 + "=production",
		envRailsModeV0 + "=enforced",
		detailProhibitedRailsEnvV0 + "=on",
		detailProhibitedRailsScopeEnvV0 + "=context_bundle_request.*",
	}
	got := serverEnvironmentWithDetailRailsDefaultV0(env)
	if len(got) != len(env) || got[0] != env[0] {
		t.Fatalf("env=%v, want mantener security mode", got)
	}
	if !detailRailsEnvHasPairForTestV0(got, envRailsModeV0, railsModeServerDefaultV0) {
		t.Fatalf("rails mode debe quedar offline: %v", got)
	}
	if !detailRailsEnvHasPairForTestV0(got, detailProhibitedRailsEnvV0, "off") {
		t.Fatalf("detail rails debe quedar off: %v", got)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0RellenaVaciosV0(t *testing.T) {
	env := []string{
		envSecurityModeV0 + "=",
		envRailsModeV0 + "=",
		detailProhibitedRailsEnvV0 + "=",
		detailProhibitedRailsScopeEnvV0 + "=",
	}
	got := serverEnvironmentWithDetailRailsDefaultV0(env)
	if !detailRailsEnvHasPairForTestV0(got, envSecurityModeV0, securityModeServerDefaultV0) {
		t.Fatalf("security mode default no aplicado: %v", got)
	}
	if !detailRailsEnvHasPairForTestV0(got, envRailsModeV0, railsModeServerDefaultV0) {
		t.Fatalf("rails mode default no aplicado: %v", got)
	}
	if !detailRailsEnvHasPairForTestV0(got, detailProhibitedRailsEnvV0, detailProhibitedRailsServerDefaultV0) {
		t.Fatalf("detail rails default no aplicado: %v", got)
	}
	if !detailRailsEnvHasPairForTestV0(got, detailProhibitedRailsScopeEnvV0, detailProhibitedRailsScopeServerDefaultV0) {
		t.Fatalf("detail rails scope default no aplicado: %v", got)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0ProgramacionFuerzaOffV0(t *testing.T) {
	got := serverEnvironmentWithDetailRailsDefaultV0([]string{
		envSecurityModeV0 + "=programming",
		envRailsModeV0 + "=enforced",
		detailProhibitedRailsEnvV0 + "=on",
	})
	if !detailRailsEnvHasPairForTestV0(got, detailProhibitedRailsEnvV0, "off") {
		t.Fatalf("programming debe proyectar rails off: %v", got)
	}
}

func TestApplyServerDetailRailsRuntimeDefaultsV0ActivaRunDirectoV0(t *testing.T) {
	t.Setenv(envSecurityModeV0, "")
	t.Setenv(envRailsModeV0, "")
	t.Setenv(detailProhibitedRailsEnvV0, "")
	t.Setenv(detailProhibitedRailsScopeEnvV0, "")

	if err := applyServerDetailRailsRuntimeDefaultsV0(); err != nil {
		t.Fatalf("applyServerDetailRailsRuntimeDefaultsV0: %v", err)
	}
	if os.Getenv(detailProhibitedRailsEnvV0) != detailProhibitedRailsServerDefaultV0 {
		t.Fatalf("detail rails env=%q", os.Getenv(detailProhibitedRailsEnvV0))
	}
	if os.Getenv(envRailsModeV0) != railsModeServerDefaultV0 {
		t.Fatalf("rails mode env=%q", os.Getenv(envRailsModeV0))
	}
	if orquestarails.TextContainsOperationalDetailMarkerV0("runtime provider model token budget home prompt policy") {
		t.Fatal("run directo no debe activar rail acotado de contexto con modo offline")
	}
	if orquestarails.TextContainsOperationalRawDetailForFieldV0("codex_wave_tail", "summary", "runtime provider model token budget") {
		t.Fatal("run directo activo frontera fuera de scope")
	}
}

func TestServerDetailRailsEffectiveV0NoActivaAunqueRailsEnforcedV0(t *testing.T) {
	t.Setenv(envSecurityModeV0, orquestarails.SecurityModeProductionV0)
	t.Setenv(envRailsModeV0, orquestarails.RailsModeEnforcedV0)
	t.Setenv(detailProhibitedRailsEnvV0, "on")

	if got := serverDetailRailsEffectiveValueV0(); got != "off" {
		t.Fatalf("rails=%q, want off aunque modo enforced explicito", got)
	}
}

func detailRailsEnvHasPairForTestV0(env []string, key string, value string) bool {
	for _, item := range env {
		gotKey, gotValue, ok := strings.Cut(item, "=")
		if ok && gotKey == key && gotValue == value {
			return true
		}
	}
	return false
}
