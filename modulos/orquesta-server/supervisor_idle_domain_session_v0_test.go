package orquestaserver

import "testing"

func TestIdleSelfImprovementDomainSessionConfiguredV0IgnoraOPESDefaultedV0(t *testing.T) {
	config := ConfigV0{EffectiveConfig: ServerEffectiveConfigV0{Settings: []ServerConfigSettingV0{
		{Key: "ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED", Value: "false", Source: "defaulted", Scope: "opes_registry_finalpkg"},
		{Key: "ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ID", Value: "informatica-a1-72-padres", Source: "defaulted", Scope: "opes_registry_finalpkg"},
	}}}

	if idleSelfImprovementDomainSessionConfiguredV0(config, StateV0{}) {
		t.Fatalf("settings OPES defaulted no deben suprimir automejora")
	}
}
