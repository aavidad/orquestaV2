package main

import (
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	domainWorkHTTPBaseURLConfiguredRefV0        = "domain-work-http-base-url-configured"
	domainWorkFileDirConfiguredRefV0            = "domain-work-file-dir-configured"
	domainWorkDeliveryLedgerPathConfiguredRefV0 = "domain-work-delivery-ledger-path-configured"
)

func domainWorkHTTPBaseURLFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPBaseURLV0,
		config.DomainWork.HTTPBaseURL,
		"",
	)
}

func domainWorkHTTPDomainRefFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPDomainRefV0,
		config.DomainWork.HTTPDomainRef,
		"",
	)
}

func domainWorkFileDirFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkFileDirV0,
		config.DomainWork.FileDir,
		"",
	)
}

func domainWorkFileEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envDomainWorkFileEnabledV0,
		config.DomainWork.FileEnabled,
		false,
	)
}

func domainWorkHTTPCreatePathFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPCreatePathV0,
		config.DomainWork.HTTPCreatePath,
		"",
	)
}

func domainWorkHTTPSubmitPathFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPSubmitPathV0,
		config.DomainWork.HTTPSubmitPath,
		"",
	)
}

func domainWorkHTTPTimeoutSecondsFromProjectConfigFileV0(config serverProjectConfigFileV0) int {
	return intProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPTimeoutSecondsV0,
		config.DomainWork.HTTPTimeoutSeconds,
		30,
	)
}

func domainWorkHTTPEgressModeFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPEgressModeV0,
		config.DomainWork.HTTPEgressMode,
		"",
	)
}

func domainWorkHTTPAllowedHostsFromProjectConfigFileV0(config serverProjectConfigFileV0) []string {
	return stringSliceProjectConfigOrEnvOrDefaultV0(
		envDomainWorkHTTPAllowedHostsV0,
		config.DomainWork.HTTPAllowedHosts,
		nil,
	)
}

func domainWorkDeliveryLedgerPathFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envDomainDeliveryLedgerPathV0,
		config.DomainWork.DeliveryLedgerPath,
		"",
	)
}

func domainWorkDeliveryEnabledFromProjectConfigV0(projectDir string) bool {
	return domainWorkDeliveryEnabledFromProjectConfigFileV0(projectConfigFromProjectDirBestEffortV0(projectDir))
}

func domainWorkDeliveryEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	fileDir := strings.TrimSpace(domainWorkFileDirFromProjectConfigFileV0(config))
	return firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) != "" ||
		strings.TrimSpace(domainWorkHTTPBaseURLFromProjectConfigFileV0(config)) != "" ||
		domainWorkFileEnabledFromProjectConfigFileV0(config) ||
		fileDir != ""
}

func domainWorkEffectiveSettingsV0(
	serverConfig orquestaserver.ConfigV0,
	config serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	baseURLSource := configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPBaseURLV0)
	fileDirSource := configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkFileDirV0)
	ledgerSource := configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainDeliveryLedgerPathV0)
	return []orquestaserver.ServerConfigSettingV0{
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPBaseURLV0,
			sensitiveConfigValueFromSourceV0(
				domainWorkHTTPBaseURLFromProjectConfigFileV0(config),
				domainWorkHTTPBaseURLConfiguredRefV0,
				baseURLSource,
			),
			baseURLSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPDomainRefV0,
			domainWorkHTTPDomainRefFromProjectConfigFileV0(config),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPDomainRefV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envDomainWorkFileDirV0,
			sensitiveConfigValueFromSourceV0(
				domainWorkFileDirFromProjectConfigFileV0(config),
				domainWorkFileDirConfiguredRefV0,
				fileDirSource,
			),
			fileDirSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkFileEnabledV0,
			strconv.FormatBool(domainWorkFileEnabledFromProjectConfigFileV0(config)),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkFileEnabledV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPCreatePathV0,
			domainWorkHTTPCreatePathFromProjectConfigFileV0(config),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPCreatePathV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPSubmitPathV0,
			domainWorkHTTPSubmitPathFromProjectConfigFileV0(config),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPSubmitPathV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPTimeoutSecondsV0,
			strconv.Itoa(domainWorkHTTPTimeoutSecondsFromProjectConfigFileV0(config)),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPTimeoutSecondsV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPEgressModeV0,
			domainWorkHTTPEgressModeFromProjectConfigFileV0(config),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPEgressModeV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envDomainWorkHTTPAllowedHostsV0,
			strings.Join(domainWorkHTTPAllowedHostsFromProjectConfigFileV0(config), ","),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envDomainWorkHTTPAllowedHostsV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envDomainDeliveryLedgerPathV0,
			sensitiveConfigValueFromSourceV0(
				domainWorkDeliveryLedgerPathFromProjectConfigFileV0(config),
				domainWorkDeliveryLedgerPathConfiguredRefV0,
				ledgerSource,
			),
			ledgerSource,
		),
	}
}
