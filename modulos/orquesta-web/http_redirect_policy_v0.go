package orquestaweb

import (
	"net/http"
	"strings"
	"time"
)

const webHTTPRedirectDeniedV0 = "web_http_redirect_denied"

func webHTTPClientWithRedirectPolicyV0(client *http.Client, timeout time.Duration, baseURL string) *http.Client {
	if client == nil {
		client = newWebLoopbackHTTPClientV0(timeout)
	}
	out := *client
	previous := client.CheckRedirect
	out.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !webHTTPRedirectAllowedV0(req, baseURL) || len(via) >= 10 {
			return webHTTPRedirectErrorV0{}
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
	return &out
}

func webHTTPRedirectAllowedV0(req *http.Request, baseURL string) bool {
	base, ok := normalizeWebRESTBaseURLV0(baseURL)
	if !ok {
		return false
	}
	if req == nil || req.URL == nil || req.URL.User != nil || req.URL.Fragment != "" {
		return false
	}
	return req.URL.Scheme == base.Scheme &&
		strings.EqualFold(req.URL.Hostname(), base.Hostname()) &&
		req.URL.Port() == base.Port()
}

type webHTTPRedirectErrorV0 struct{}

func (webHTTPRedirectErrorV0) Error() string {
	return webHTTPRedirectDeniedV0
}
