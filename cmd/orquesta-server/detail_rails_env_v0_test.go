package main

import (
	"os"
	"testing"
)

func TestEnsureServerDetailRailsDefaultV0MantieneRailAbiertoPorDefecto(t *testing.T) {
	t.Setenv(detailProhibitedRailsEnvV0, "")
	t.Setenv(detailProhibitedRailsScopeEnvV0, "")

	ensureServerDetailRailsDefaultV0()

	if got := os.Getenv(detailProhibitedRailsEnvV0); got != "off" {
		t.Fatalf("default=%q, want off", got)
	}
	if got := os.Getenv(detailProhibitedRailsScopeEnvV0); got != detailProhibitedRailsScopeServerDefaultV0 {
		t.Fatalf("scope=%q, want %q", got, detailProhibitedRailsScopeServerDefaultV0)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0AddReactivaSiFalta(t *testing.T) {
	got := serverEnvironmentWithDetailRailsDefaultV0([]string{"PATH=/tmp/bin"})
	if !detailRailsEnvPresentV0(got, detailProhibitedRailsEnvV0) {
		t.Fatalf("%s no añadido: %v", detailProhibitedRailsEnvV0, got)
	}
	if !detailRailsEnvPresentV0(got, detailProhibitedRailsScopeEnvV0) {
		t.Fatalf("%s no añadido: %v", detailProhibitedRailsScopeEnvV0, got)
	}
	if got[len(got)-2] != detailProhibitedRailsEnvV0+"="+detailProhibitedRailsServerDefaultV0 {
		t.Fatalf("env=%v, want rails default", got)
	}
	if got[len(got)-1] != detailProhibitedRailsScopeEnvV0+"="+detailProhibitedRailsScopeServerDefaultV0 {
		t.Fatalf("env=%v, want scope default", got)
	}
}

func TestServerEnvironmentWithDetailRailsDefaultV0NoPisaValorExplicito(t *testing.T) {
	env := []string{
		detailProhibitedRailsEnvV0 + "=off",
		detailProhibitedRailsScopeEnvV0 + "=context_bundle_request.*",
	}
	got := serverEnvironmentWithDetailRailsDefaultV0(env)
	if len(got) != len(env) || got[0] != env[0] || got[1] != env[1] {
		t.Fatalf("env=%v, want %v", got, env)
	}
}
