package orquestamcp

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeMCPRESTBaseURLV0(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", fmt.Errorf("server_url requerida")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("server_url invalida")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("server_url no debe contener credenciales")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("server_url debe usar http o https")
	}
	if strings.TrimSpace(parsed.Host) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("server_url debe ser base http")
	}
	return parsed.String(), nil
}

func joinMCPRESTEndpointV0(baseURL string, endpoint string) (string, error) {
	baseURL, err := normalizeMCPRESTBaseURLV0(baseURL)
	if err != nil {
		return "", err
	}
	endpointPath, err := normalizeMCPRESTEndpointV0(endpoint)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("server_url invalida")
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	if basePath == "" || basePath == "/" {
		parsed.Path = endpointPath
	} else {
		parsed.Path = basePath + endpointPath
	}
	parsed.RawPath = ""
	parsed.ForceQuery = false
	return parsed.String(), nil
}

func normalizeMCPRESTEndpointV0(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.HasPrefix(endpoint, "//") || strings.Contains(endpoint, "\\") {
		return "", fmt.Errorf("endpoint debe ser path relativo")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("endpoint debe ser path relativo")
	}
	parts := strings.Split(strings.TrimLeft(parsed.Path, "/"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("endpoint debe ser path relativo")
		}
	}
	return "/" + strings.Join(parts, "/"), nil
}
