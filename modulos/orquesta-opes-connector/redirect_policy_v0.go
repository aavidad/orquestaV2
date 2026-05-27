package orquestaopesconnector

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func opesHTTPClientWithRedirectPolicyV0(client *http.Client, timeout time.Duration, baseURL string) *http.Client {
	if client == nil {
		client = newOPESHTTPClientV0(timeout)
	}
	out := *client
	previous := client.CheckRedirect
	out.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateOPESHTTPRedirectV0(req, baseURL); err != nil {
			return err
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return connectorErrorV0{code: ErrOPESHTTPRedirectDeniedV0}
		}
		return nil
	}
	return &out
}

func validateOPESHTTPRedirectV0(req *http.Request, baseURL string) error {
	base, err := url.Parse(trimTrailingSlashV0(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return connectorErrorV0{code: ErrOPESHTTPRedirectDeniedV0}
	}
	if req == nil || req.URL == nil || req.URL.User != nil || req.URL.Fragment != "" {
		return connectorErrorV0{code: ErrOPESHTTPRedirectDeniedV0}
	}
	if req.URL.Scheme != base.Scheme ||
		!strings.EqualFold(req.URL.Hostname(), base.Hostname()) ||
		req.URL.Port() != base.Port() ||
		!opesControlledPathV0(req.URL.RequestURI()) {
		return connectorErrorV0{code: ErrOPESHTTPRedirectDeniedV0}
	}
	return nil
}

func opesHTTPRedirectErrorCodeV0(err error) (string, bool) {
	var publicErr connectorErrorV0
	if errors.As(err, &publicErr) && publicErr.code == ErrOPESHTTPRedirectDeniedV0 {
		return ErrOPESHTTPRedirectDeniedV0, true
	}
	return "", false
}
