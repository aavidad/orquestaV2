package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func domainWorkExecutorFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfigs ...serverProjectConfigFileV0,
) (orquestamcp.MCPDomainWorkExecutorPortV0, error) {
	projectConfig := serverProjectConfigFileV0{}
	if len(projectConfigs) > 0 {
		projectConfig = projectConfigs[0]
	} else {
		projectConfig = projectConfigFromServerConfigBestEffortV0(serverConfig)
	}
	opesConfig := serverOPESConfigSnapshotFromProjectConfigFileV0(projectConfig)
	baseURL := firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0)
	httpBaseURL := strings.TrimSpace(domainWorkHTTPBaseURLFromProjectConfigFileV0(projectConfig))
	fileDir := strings.TrimSpace(domainWorkFileDirFromProjectConfigFileV0(projectConfig))
	explicitFileBackend := domainWorkFileEnabledFromProjectConfigFileV0(projectConfig) ||
		fileDir != ""
	configuredBackends := 0
	for _, enabled := range []bool{baseURL != "", httpBaseURL != "", explicitFileBackend} {
		if enabled {
			configuredBackends++
		}
	}
	if configuredBackends > 1 {
		return nil, fmt.Errorf("domain_work_backend_ambiguous")
	}
	if baseURL != "" {
		if err := opesDomainWorkDestinationPolicyFromEnvV0(baseURL); err != nil {
			return nil, err
		}
		client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
			BaseURL:            baseURL,
			HTTPClient:         commandOPESTemporalHTTPClientV0(time.Duration(opesConfig.TimeoutSeconds) * time.Second),
			DefaultMaxAttempts: opesConfig.DefaultMaxAttempts,
		})
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if httpBaseURL != "" {
		if err := domainWorkHTTPDestinationPolicyFromProjectConfigFileV0(projectConfig); err != nil {
			return nil, err
		}
		egressPolicy, err := domainWorkHTTPEgressPolicyFromProjectConfigFileV0(projectConfig)
		if err != nil {
			return nil, err
		}
		client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
			BaseURL:            httpBaseURL,
			CreateJobPath:      domainWorkHTTPCreatePathFromProjectConfigFileV0(projectConfig),
			SubmitArtifactPath: domainWorkHTTPSubmitPathFromProjectConfigFileV0(projectConfig),
			EgressPolicy:       egressPolicy,
			Timeout:            time.Duration(domainWorkHTTPTimeoutSecondsFromProjectConfigFileV0(projectConfig)) * time.Second,
		})
		if err != nil {
			return nil, err
		}
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if fileDir == "" {
		fileDir = filepath.Join(serverConfig.StateDir, "domain-work-jobs")
	}
	fileDir, err := filepath.Abs(fileDir)
	if err != nil {
		return nil, fmt.Errorf("domain_work_file_dir_invalid")
	}
	creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(fileDir)
	if err != nil {
		return nil, err
	}
	return orquestamcp.NewMCPDomainWorkToolExecutorV0(creator, creator), nil
}

func domainWorkHTTPDestinationPolicyFromProjectConfigFileV0(config serverProjectConfigFileV0) error {
	if strings.EqualFold(domainWorkHTTPDomainRefFromProjectConfigFileV0(config), "opes") {
		return fmt.Errorf("opes_destination_productive_not_allowed")
	}
	return nil
}

func domainWorkHTTPEgressPolicyFromProjectConfigFileV0(config serverProjectConfigFileV0) (orquestadomainworkhttp.EgressPolicyV0, error) {
	mode := strings.TrimSpace(domainWorkHTTPEgressModeFromProjectConfigFileV0(config))
	switch mode {
	case orquestadomainworkhttp.EgressModeSmokeLocalV0:
		return orquestadomainworkhttp.SmokeLocalEgressPolicyV0(), nil
	case orquestadomainworkhttp.EgressModeAllowlistV0:
		allowedHosts := domainWorkHTTPAllowedHostsFromProjectConfigFileV0(config)
		if len(allowedHosts) == 0 {
			return orquestadomainworkhttp.EgressPolicyV0{}, errors.New(orquestadomainworkhttp.ErrDomainWorkHTTPEgressPolicyRequiredV0)
		}
		return orquestadomainworkhttp.AllowlistEgressPolicyV0(allowedHosts...), nil
	default:
		return orquestadomainworkhttp.EgressPolicyV0{}, errors.New(orquestadomainworkhttp.ErrDomainWorkHTTPEgressPolicyRequiredV0)
	}
}

func opesDomainWorkDestinationPolicyFromEnvV0(baseURL string) error {
	config := opesProjectConfigFromEnvBestEffortV0()
	if opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeProductiveConfirmV0, false) {
		return fmt.Errorf("opes_destination_productive_not_allowed")
	}
	destination, err := opesBridgeDestinationFromProjectConfigFileV0(config, "opes", baseURL, false)
	if err != nil {
		return err
	}
	if err := opesBridgeRequireRealOPESConfirmationFromProjectConfigFileV0(config, destination, false); err != nil {
		return err
	}
	evidenceRef := opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeDestinationEvidenceV0)
	if evidenceRef != "" && !compactEvidenceRefV0(evidenceRef) {
		return fmt.Errorf("opes_destination_evidence_ref_invalid")
	}
	if destination.Category == "temporal" && evidenceRef == "" {
		return fmt.Errorf("opes_destination_evidence_ref_required")
	}
	return nil
}

func domainWorkDeliveryEnabledFromEnvV0() bool {
	return domainWorkDeliveryEnabledFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func domainWorkRequiredTestConfigFromEnvV0() orquestaappcodexstack.DomainWorkRequiredTestConfigV0 {
	if firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) == "" {
		return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{}
	}
	return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{
		Policy: orquestaopesbridge.OPESRequiredTestPolicyV0{},
	}
}
