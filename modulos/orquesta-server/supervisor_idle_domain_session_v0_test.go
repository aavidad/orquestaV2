package orquestaserver

import "testing"

func TestIdleSelfImprovementDomainSessionConfiguredV0IgnoraBridgeDisabledPersistidoV0(t *testing.T) {
	state := StateV0{
		ExternalBridgeComponent: "external_bridge_loop",
		ExternalBridgeStatus:    "disabled",
	}

	if idleSelfImprovementDomainSessionConfiguredV0(ConfigV0{}, state) {
		t.Fatalf("bridge disabled persistido no debe suprimir automejora")
	}
}

func TestIdleSelfImprovementDomainSessionConfiguredV0IgnoraBridgeComponentSinEstadoActivoV0(t *testing.T) {
	state := StateV0{ExternalBridgeComponent: "external_bridge_loop"}

	if idleSelfImprovementDomainSessionConfiguredV0(ConfigV0{}, state) {
		t.Fatalf("componente bridge historico sin estado activo no debe suprimir automejora")
	}
}

func TestIdleSelfImprovementDomainSessionConfiguredV0IgnoraEstadosBridgeSegurosV0(t *testing.T) {
	for _, status := range []string{"", "disabled", "inactive", "stopped", "idle", "off", "false", "0", "absent", "not_configured"} {
		t.Run(status, func(t *testing.T) {
			state := StateV0{
				ExternalBridgeComponent: "external_bridge_loop",
				ExternalBridgeStatus:    status,
			}
			if idleSelfImprovementDomainSessionConfiguredV0(ConfigV0{}, state) {
				t.Fatalf("bridge status %q no debe suprimir automejora", status)
			}
		})
	}
}

func TestIdleSelfImprovementDomainSessionConfiguredV0BloqueaEstadosBridgeActivosV0(t *testing.T) {
	for _, status := range []string{"running", "enabled", "active", "submitted", "waiting", "draining", "error", "recovering"} {
		t.Run(status, func(t *testing.T) {
			state := StateV0{
				ExternalBridgeComponent: "external_bridge_loop",
				ExternalBridgeStatus:    status,
			}
			if !idleSelfImprovementDomainSessionConfiguredV0(ConfigV0{}, state) {
				t.Fatalf("bridge status %q debe suprimir automejora", status)
			}
		})
	}
}

func TestIdleSelfImprovementDomainSessionConfiguredV0IgnoraOPESDefaultedV0(t *testing.T) {
	config := ConfigV0{EffectiveConfig: ServerEffectiveConfigV0{Settings: []ServerConfigSettingV0{
		{Key: "ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED", Value: "false", Source: "defaulted", Scope: "opes_registry_finalpkg"},
		{Key: "ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ID", Value: "informatica-a1-72-padres", Source: "defaulted", Scope: "opes_registry_finalpkg"},
	}}}

	if idleSelfImprovementDomainSessionConfiguredV0(config, StateV0{}) {
		t.Fatalf("settings OPES defaulted no deben suprimir automejora")
	}
}
