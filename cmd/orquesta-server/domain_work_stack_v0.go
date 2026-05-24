package main

import (
	"fmt"
	"net/http"
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
	baseURL := firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL")
	httpBaseURL := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL"))
	fileDir := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_FILE_DIR"))
	fileEnabled := strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED")) == "1" ||
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
			BaseURL: baseURL,
			HTTPClient: &http.Client{
				Timeout: time.Duration(intEnvOrDefaultV0("ORQUESTA_OPES_TIMEOUT_SECONDS", 30)) * time.Second,
			},
			DefaultMaxAttempts: intEnvOrDefaultV0("ORQUESTA_OPES_DEFAULT_MAX_ATTEMPTS", 1),
		})
		return orquestamcp.NewMCPDomainWorkToolExecutorV0(client, client), nil
	}
	if httpBaseURL != "" {
		client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
			BaseURL:            httpBaseURL,
			CreateJobPath:      strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH")),
			SubmitArtifactPath: strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH")),
			Timeout:            time.Duration(intEnvOrDefaultV0("ORQUESTA_DOMAIN_WORK_HTTP_TIMEOUT_SECONDS", 30)) * time.Second,
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

func domainWorkDeliveryEnabledFromEnvV0() bool {
	return firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL") != "" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL")) != ""
}

func domainWorkRequiredTestConfigFromEnvV0() orquestaappcodexstack.DomainWorkRequiredTestConfigV0 {
	if firstNonEmptyEnvV0("ORQUESTA_OPES_BASE_URL", "OPES_BASE_URL") == "" {
		return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{}
	}
	return orquestaappcodexstack.DomainWorkRequiredTestConfigV0{
		Policy: orquestaopesbridge.OPESRequiredTestPolicyV0{},
	}
}
