package main

func init() {
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPBaseURLV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork HTTP URL",
		Description: "Base URL opt-in del adaptador HTTP neutral de domain_work.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPDomainRefV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork dominio HTTP",
		Description: "Dominio declarado para el destino HTTP neutral; opes productivo queda bloqueado sin adaptador OPES temporal.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkFileDirV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork fichero",
		Description: "Directorio opt-in del backend file de domain_work.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkFileEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork file enabled",
		Description: "Activa el backend file de domain_work usando state_dir si no hay directorio explicito.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPCreatePathV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork create path",
		Description: "Ruta HTTP para crear jobs de domain_work en el adaptador neutral.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPSubmitPathV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork artifact path",
		Description: "Ruta HTTP para entregar artefactos de domain_work en el adaptador neutral.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPTimeoutSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork timeout",
		Description: "Timeout en segundos del adaptador HTTP neutral de domain_work.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPEgressModeV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork egress",
		Description: "Politica de salida HTTP neutral: smoke_local o allowlist.",
	}
	serverEffectiveEnvRegistryV0[envDomainWorkHTTPAllowedHostsV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork hosts",
		Description: "Hosts permitidos para egress allowlist del adaptador HTTP neutral.",
	}
	serverEffectiveEnvRegistryV0[envDomainDeliveryLedgerPathV0] = serverEnvSettingMetadataV0{
		Scope:       "domain_work",
		Label:       "DomainWork ledger",
		Description: "Ruta del ledger de entregas domain_work; se publica solo como referencia configurada.",
	}
}
