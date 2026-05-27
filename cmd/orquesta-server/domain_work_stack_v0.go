package main

import (
	"errors"
	"fmt"
	"os"
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
) (orquestamcp.MCPDomainWorkExecutorPortV0, error) {
	baseURL := firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0)
	httpBaseURL := strings.TrimSpace(os.Getenv(envDomainWorkHTTPBaseURLV0))
	fileDir := strings.TrimSpace(os.Getenv(envDomainWorkFileDirV0))
	fileEnabled := strings.TrimSpace(os.Getenv(envDomainWorkFileEnabledV0)) == "1" ||
		fileDir != ""
	configuredBackends := 0
	for _, enabled := range []bool{baseURL != "", httpBaseURL != "", fileEnabled} {
		if enabled {
			configuredBackends++
		}
	}
	if configuredBackends > 1 {
		return nil, fmt.Errorf("domain_work_backend_ambiguous")
	}
	if baseURL != "" {
		client := orquestaopesconnector.NewRESTClientV0(orquestaopesconnector.RESTClientConfigV0{
			BaseURL:            baseURL,
			HTTPClient:         commandOPESTemporalHTTPClientV0(time.Duration(intEnvOrDefaultV0(envOPESTimeoutSecondsV0, 30)) * time.Second),
			DefaultMaxAttempts: intEnvOrDefaultV0(envOPESDefaultMaxAttemptsV0, 1),
		})
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if httpBaseURL != "" {
		egressPolicy, err := domainWorkHTTPEgressPolicyFromEnvV0()
		if err != nil {
			return nil, err
		}
		client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
			BaseURL:            httpBaseURL,
			CreateJobPath:      strings.TrimSpace(os.Getenv(envDomainWorkHTTPCreatePathV0)),
			SubmitArtifactPath: strings.TrimSpace(os.Getenv(envDomainWorkHTTPSubmitPathV0)),
			EgressPolicy:       egressPolicy,
			Timeout:            time.Duration(intEnvOrDefaultV0(envDomainWorkHTTPTimeoutSecondsV0, 30)) * time.Second,
		})
		if err != nil {
			return nil, err
		}
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if !fileEnabled {
		return nil, nil
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
	return orquestamcp.NewMCPDomainWorkToolExecutorV0(creator, nil), nil
}

func domainWorkHTTPEgressPolicyFromEnvV0() (orquestadomainworkhttp.EgressPolicyV0, error) {
	mode := strings.TrimSpace(os.Getenv(envDomainWorkHTTPEgressModeV0))
	switch mode {
	case orquestadomainworkhttp.EgressModeSmokeLocalV0:
		return orquestadomainworkhttp.SmokeLocalEgressPolicyV0(), nil
	case orquestadomainworkhttp.EgressModeAllowlistV0:
		allowedHosts := splitCSVEnvV0(os.Getenv(envDomainWorkHTTPAllowedHostsV0))
		if len(allowedHosts) == 0 {
			return orquestadomainworkhttp.EgressPolicyV0{}, errors.New(orquestadomainworkhttp.ErrDomainWorkHTTPEgressPolicyRequiredV0)
		}
		return orquestadomainworkhttp.AllowlistEgressPolicyV0(allowedHosts...), nil
	default:
		return orquestadomainworkhttp.EgressPolicyV0{}, errors.New(orquestadomainworkhttp.ErrDomainWorkHTTPEgressPolicyRequiredV0)
	}
}

func splitCSVEnvV0(raw string) []string {
	var values []string
	for _, item := range strings.Split(raw, ",") {
		value := strings.TrimSpace(item)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func domainWorkDeliveryEnabledFromEnvV0() bool {
	return firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) != "" ||
		strings.TrimSpace(os.Getenv(envDomainWorkHTTPBaseURLV0)) != ""
}

func domainWorkRequiredTestConfigFromEnvV0() orquestaappcodexstack.DomainWorkRequiredTestConfigV0 {
	if firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) == "" {
		return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{}
	}
	return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{
		Policy: orquestaopesbridge.OPESRequiredTestPolicyV0{},
	}
}
