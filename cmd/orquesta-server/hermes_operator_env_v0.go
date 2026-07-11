package main

const (
	envHermesEnabledV0            = "ORQUESTA_HERMES_ENABLED"
	envHermesBaseURLV0            = "ORQUESTA_HERMES_BASE_URL"
	envHermesMCPPathV0            = "ORQUESTA_HERMES_MCP_PATH"
	envHermesAPIKeyV0             = "ORQUESTA_HERMES_API_KEY"
	envHermesStatusToolV0         = "ORQUESTA_HERMES_STATUS_TOOL"
	envHermesBurstToolV0          = "ORQUESTA_HERMES_BURST_TOOL"
	envHermesOutboxToolV0         = "ORQUESTA_HERMES_OUTBOX_TOOL"
	envHermesQueryToolV0          = "ORQUESTA_HERMES_QUERY_TOOL"
	envHermesStatusConnectorRefV0 = "ORQUESTA_HERMES_STATUS_CONNECTOR_REF"
	envHermesBurstConnectorRefV0  = "ORQUESTA_HERMES_BURST_CONNECTOR_REF"
	envHermesOutboxConnectorRefV0 = "ORQUESTA_HERMES_OUTBOX_CONNECTOR_REF"
	envHermesQueryConnectorRefV0  = "ORQUESTA_HERMES_QUERY_CONNECTOR_REF"
	envHermesTimeoutSecondsV0     = "ORQUESTA_HERMES_TIMEOUT_SECONDS"
	envHermesMaxRequestBytesV0    = "ORQUESTA_HERMES_MAX_REQUEST_BYTES"
	envHermesMaxResponseBytesV0   = "ORQUESTA_HERMES_MAX_RESPONSE_BYTES"
)

func init() {
	for key, metadata := range hermesOperatorEnvRegistryV0 {
		serverEffectiveEnvRegistryV0[key] = metadata
	}
}

var hermesOperatorEnvRegistryV0 = map[string]serverEnvSettingMetadataV0{
	envHermesEnabledV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes habilitado",
		Description: "Activa el conector externo Hermes API/MCP.",
	},
	envHermesBaseURLV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes base URL",
		Description: "Endpoint base del operador Hermes API/MCP; se publica redacted.",
	},
	envHermesMCPPathV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes MCP path",
		Description: "Ruta MCP JSON-RPC del operador Hermes.",
	},
	envHermesAPIKeyV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes API key",
		Description: "Override legacy/deprecated del token Hermes; nunca se publica en claro.",
	},
	envHermesStatusToolV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes status tool",
		Description: "Nombre remoto del tool Hermes de estado.",
	},
	envHermesBurstToolV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes burst tool",
		Description: "Nombre remoto del tool Hermes de burst supervisado.",
	},
	envHermesOutboxToolV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes outbox tool",
		Description: "Nombre remoto del tool Hermes de outbox pendiente.",
	},
	envHermesQueryToolV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes query tool",
		Description: "Nombre remoto del tool Hermes de consulta dirigida.",
	},
	envHermesStatusConnectorRefV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes status ref",
		Description: "Ref opaca remota para consultas de estado Hermes.",
	},
	envHermesBurstConnectorRefV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes burst ref",
		Description: "Ref opaca remota para burst supervisado Hermes.",
	},
	envHermesOutboxConnectorRefV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes outbox ref",
		Description: "Ref opaca remota para outbox Hermes.",
	},
	envHermesQueryConnectorRefV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes query ref",
		Description: "Ref opaca remota para consultas dirigidas Hermes.",
	},
	envHermesTimeoutSecondsV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes timeout",
		Description: "Timeout por llamada Hermes API/MCP.",
	},
	envHermesMaxRequestBytesV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes request bytes",
		Description: "Maximo de bytes por request JSON-RPC Hermes.",
	},
	envHermesMaxResponseBytesV0: {
		Scope:       "hermes_operator",
		Label:       "Hermes response bytes",
		Description: "Maximo de bytes por respuesta JSON-RPC Hermes.",
	},
}
