package main

import (
	"fmt"
	"net/url"
	"strings"
)

func commandRESTBaseURLFromAddrV0(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", fmt.Errorf("base_url_required")
	}
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	return commandRESTBaseURLV0(addr)
}

func commandRESTBaseURLV0(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", fmt.Errorf("base_url_required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil {
		return "", fmt.Errorf("base_url_invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("base_url_scheme_not_allowed")
	}
	if strings.TrimSpace(parsed.Host) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("base_url_must_be_http_base")
	}
	return parsed.String(), nil
}

func commandRESTEndpointURLV0(baseURL string, endpoint string) (string, error) {
	baseURL, err := commandRESTBaseURLV0(baseURL)
	if err != nil {
		return "", err
	}
	endpointPath, err := commandRESTEndpointPathV0(endpoint)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("base_url_invalid")
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

func commandRESTEndpointPathV0(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.HasPrefix(endpoint, "//") || strings.Contains(endpoint, "\\") {
		return "", fmt.Errorf("endpoint_must_be_relative_path")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("endpoint_must_be_relative_path")
	}
	parts := strings.Split(strings.TrimLeft(parsed.Path, "/"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("endpoint_must_be_relative_path")
		}
	}
	return "/" + strings.Join(parts, "/"), nil
}
