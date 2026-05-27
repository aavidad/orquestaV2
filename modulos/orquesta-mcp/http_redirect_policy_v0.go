package orquestamcp

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const mcpHTTPRedirectDeniedV0 = "mcp_http_redirect_denied"

func mcpHTTPClientWithRedirectPolicyV0(client *http.Client, timeout time.Duration, baseURL string) *http.Client {
	if client == nil {
		client = newMCPLoopbackHTTPClientV0(timeout)
	}
	out := *client
	previous := client.CheckRedirect
	out.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !mcpHTTPRedirectAllowedV0(req, baseURL) || len(via) >= 10 {
			return mcpHTTPRedirectErrorV0{}
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
	return &out
}

func mcpHTTPRedirectAllowedV0(req *http.Request, baseURL string) bool {
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return false
	}
	if req == nil || req.URL == nil || req.URL.User != nil || req.URL.Fragment != "" {
		return false
	}
	return req.URL.Scheme == base.Scheme &&
		strings.EqualFold(req.URL.Hostname(), base.Hostname()) &&
		req.URL.Port() == base.Port()
}

type mcpHTTPRedirectErrorV0 struct{}

func (mcpHTTPRedirectErrorV0) Error() string {
	return mcpHTTPRedirectDeniedV0
}
