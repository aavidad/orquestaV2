package orquestaweb

import (
	"net/url"
	"strings"
)

const webRESTURLPolicyInvalidV0 = "web_rest_url_policy_invalid"

func normalizeWebRESTBaseURLStringV0(raw string) string {
	parsed, ok := normalizeWebRESTBaseURLV0(raw)
	if !ok {
		return strings.TrimRight(strings.TrimSpace(raw), "/")
	}
	return parsed.String()
}

func webRESTEndpointURLV0(baseURL string, endpoint string, fallbackEndpoint string) string {
	parsed, ok := normalizeWebRESTBaseURLV0(baseURL)
	if !ok {
		return webRESTURLPolicyInvalidV0
	}
	endpointPath, ok := normalizeWebRESTEndpointV0(endpoint, fallbackEndpoint)
	if !ok {
		return webRESTURLPolicyInvalidV0
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	if basePath == "" || basePath == "/" {
		parsed.Path = endpointPath
	} else {
		parsed.Path = basePath + endpointPath
	}
	parsed.RawPath = ""
	parsed.ForceQuery = false
	return parsed.String()
}

func normalizeWebRESTBaseURLV0(raw string) (*url.URL, bool) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return nil, false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil {
		return nil, false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, false
	}
	if strings.TrimSpace(parsed.Host) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, false
	}
	return parsed, true
}

func normalizeWebRESTEndpointV0(endpoint string, fallbackEndpoint string) (string, bool) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(fallbackEndpoint)
	}
	if endpoint == "" || strings.HasPrefix(endpoint, "//") || strings.Contains(endpoint, "\\") {
		return "", false
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	parts := strings.Split(strings.TrimLeft(parsed.Path, "/"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false
		}
	}
	return "/" + strings.Join(parts, "/"), true
}
