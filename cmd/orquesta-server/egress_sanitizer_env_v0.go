package main

import orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"

const (
	envEgressSanitizerEnabledV0              = "ORQUESTA_EGRESS_SANITIZER_ENABLED"
	envEgressSanitizerRefV0                  = "ORQUESTA_EGRESS_SANITIZER_REF"
	envEgressSanitizerLocalModelEnabledV0    = "ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_ENABLED"
	envEgressSanitizerLocalModelRefV0        = "ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF"
	envEgressSanitizerLocalModelPathV0       = "ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_PATH"
	envEgressSanitizerLocalRuntimeRefV0      = "ORQUESTA_EGRESS_SANITIZER_LOCAL_RUNTIME_REF"
	envEgressSanitizerLocalEvidenceRefV0     = "ORQUESTA_EGRESS_SANITIZER_LOCAL_EVIDENCE_REF"
	envEgressSanitizerSidecarEnabledV0       = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_ENABLED"
	envEgressSanitizerSidecarRefV0           = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_REF"
	envEgressSanitizerSidecarAdapterRefV0    = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_ADAPTER_REF"
	envEgressSanitizerSidecarTransportRefV0  = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_TRANSPORT_REF"
	envEgressSanitizerSidecarEvidenceRefV0   = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_EVIDENCE_REF"
	envEgressSanitizerSidecarCommandV0       = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND"
	envEgressSanitizerSidecarLocalEndpointV0 = "ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT"
)

func init() {
	serverEffectiveEnvRegistryV0[envEgressSanitizerEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Sanitizer egress",
		Description: "Activa el sanitizer local canonico para contexto/egress de apps IA lanzadas por Orquesta.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Ref sanitizer egress",
		Description: "Ref opaca del sanitizer local canonico usado en egress.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerLocalModelEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Modelo local sanitizer",
		Description: "Activa el filtro local de privacidad como apoyo del sanitizer canonico.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerLocalModelRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Modelo sanitizer",
		Description: "Ref opaca del modelo local de privacidad; el valor concreto no se publica.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerLocalModelPathV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Ruta modelo sanitizer",
		Description: "Presencia de ruta local del modelo de privacidad; la ruta concreta no se publica.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerLocalRuntimeRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Runtime sanitizer",
		Description: "Ref opaca del runtime local que ejecuta el filtro de privacidad.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerLocalEvidenceRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Evidencia sanitizer",
		Description: "Ref opaca de evidencia/configuracion del filtro local de privacidad.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Sidecar privacy filter",
		Description: "Activa el sidecar local opt-in OpenAI Privacy Filter para egress.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Ref sidecar privacy filter",
		Description: "Ref opaca del sidecar local de privacidad.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarAdapterRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Adaptador privacy filter",
		Description: "Ref opaca del adaptador que conecta el sidecar local.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarTransportRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Transporte privacy filter",
		Description: "Ref opaca del transporte usado por el sidecar local.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarEvidenceRefV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Evidencia sidecar privacy filter",
		Description: "Ref opaca de evidencia/configuracion del sidecar local.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarCommandV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Comando sidecar privacy filter",
		Description: "Presencia del comando local del sidecar; el comando concreto no se publica.",
	}
	serverEffectiveEnvRegistryV0[envEgressSanitizerSidecarLocalEndpointV0] = serverEnvSettingMetadataV0{
		Scope:       "egress_sanitizer",
		Label:       "Endpoint sidecar privacy filter",
		Description: "Presencia del endpoint local del sidecar; el endpoint concreto no se publica.",
	}
}

func egressSanitizerConfigFromEnvV0() orquestaappcodexstack.EgressSanitizerConfigV0 {
	return orquestaappcodexstack.NormalizeEgressSanitizerConfigV0(orquestaappcodexstack.EgressSanitizerConfigV0{
		Enabled:      boolEnvOrDefaultV0(envEgressSanitizerEnabledV0, false),
		SanitizerRef: envOrDefaultV0(envEgressSanitizerRefV0, ""),
		LocalModel: orquestaappcodexstack.PrivacyFilterModelConfigV0{
			Enabled:     boolEnvOrDefaultV0(envEgressSanitizerLocalModelEnabledV0, false),
			ModelRef:    envOrDefaultV0(envEgressSanitizerLocalModelRefV0, ""),
			RuntimeRef:  envOrDefaultV0(envEgressSanitizerLocalRuntimeRefV0, ""),
			EvidenceRef: envOrDefaultV0(envEgressSanitizerLocalEvidenceRefV0, ""),
		},
		Sidecar: orquestaappcodexstack.PrivacyFilterSidecarConfigV0{
			Enabled:                 boolEnvOrDefaultV0(envEgressSanitizerSidecarEnabledV0, false),
			SidecarRef:              envOrDefaultV0(envEgressSanitizerSidecarRefV0, ""),
			AdapterRef:              envOrDefaultV0(envEgressSanitizerSidecarAdapterRefV0, ""),
			TransportRef:            envOrDefaultV0(envEgressSanitizerSidecarTransportRefV0, ""),
			EvidenceRef:             envOrDefaultV0(envEgressSanitizerSidecarEvidenceRefV0, ""),
			CommandConfigured:       envOrDefaultV0(envEgressSanitizerSidecarCommandV0, "") != "",
			LocalEndpointConfigured: envOrDefaultV0(envEgressSanitizerSidecarLocalEndpointV0, "") != "",
		},
	})
}

func egressSanitizerConfigWithSidecarPortFromEnvV0() (orquestaappcodexstack.EgressSanitizerConfigV0, error) {
	config := egressSanitizerConfigFromEnvV0()
	endpoint := envOrDefaultV0(envEgressSanitizerSidecarLocalEndpointV0, "")
	if !config.Enabled || !config.Sidecar.Enabled || endpoint == "" {
		return config, nil
	}
	port, err := orquestaappcodexstack.NewPrivacyFilterSidecarHTTPPortV0(
		orquestaappcodexstack.PrivacyFilterSidecarHTTPConfigV0{
			Enabled:          true,
			EndpointURL:      endpoint,
			SidecarRef:       config.Sidecar.SidecarRef,
			AdapterRef:       config.Sidecar.AdapterRef,
			TransportRef:     config.Sidecar.TransportRef,
			EvidenceRef:      config.Sidecar.EvidenceRef,
			MaxResponseBytes: 64 << 10,
		},
	)
	if err != nil {
		return orquestaappcodexstack.EgressSanitizerConfigV0{}, err
	}
	config.Sidecar.Port = port
	return config, nil
}
