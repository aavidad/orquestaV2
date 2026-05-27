package main

import (
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerEffectiveConfigV0PublicaRailsDetalleCanonicos(t *testing.T) {
	t.Setenv(envDetailProhibitedRailsV0, "")
	t.Setenv(envDetailProhibitedRailsScopeV0, "")
	t.Setenv(envSecurityModeV0, "")
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envSecurityModeV0:               securityModeServerDefaultV0,
		envDetailProhibitedRailsV0:      detailProhibitedRailsServerDefaultV0,
		envDetailProhibitedRailsScopeV0: detailProhibitedRailsScopeServerDefaultV0,
	} {
		if got := effectiveSettingValueForDetailRailsTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q", key, got, want)
		}
		if source := effectiveSettingSourceForDetailRailsTestV0(settings, key); source != "defaulted" {
			t.Fatalf("%s source=%q want defaulted", key, source)
		}
	}
}

func effectiveSettingValueForDetailRailsTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func effectiveSettingSourceForDetailRailsTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Source
		}
	}
	return ""
}
