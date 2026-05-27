package orquestaserver

const (
	ServerStatusEndpointV0       = "/api/v0/server/status"
	ServerStatusLegacyEndpointV0 = "/api/status"

	ServerStatusCanonicalHeaderV0     = "X-Orquesta-Status-Canonical"
	ServerStatusCompatibilityHeaderV0 = "X-Orquesta-Status-Compatibility"
	ServerStatusOwnerHeaderV0         = "X-Orquesta-Status-Owner"
	ServerStatusSunsetHeaderV0        = "X-Orquesta-Status-Sunset-Policy"
	ServerStatusCompatibilityLegacyV0 = "legacy_alias"
	ServerStatusOwnerServerV0         = "orquesta-server"
	ServerStatusSunsetNoNewUseV0      = "no_new_use"
)
