package orquestadomainworkhttp

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func domainWorkHTTPClientWithRedirectPolicyV0(
	client *http.Client,
	timeout time.Duration,
	destination DestinationV0,
	allowedPaths ...string,
) *http.Client {
	if client == nil {
		client = newDomainWorkHTTPClientV0(timeout)
	}
	out := *client
	previous := client.CheckRedirect
	out.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateDomainWorkHTTPRedirectV0(req, destination, allowedPaths...); err != nil {
			return err
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return ErrorV0{Code: ErrDomainWorkHTTPRedirectDeniedV0}
		}
		return nil
	}
	return &out
}

func validateDomainWorkHTTPRedirectV0(
	req *http.Request,
	destination DestinationV0,
	allowedPaths ...string,
) error {
	if req == nil || req.URL == nil || req.URL.User != nil || req.URL.Fragment != "" {
		return ErrorV0{Code: ErrDomainWorkHTTPRedirectDeniedV0}
	}
	if !sameDomainWorkHTTPOriginV0(req.URL, destination) {
		return ErrorV0{Code: ErrDomainWorkHTTPRedirectDeniedV0}
	}
	path, err := normalizePathV0(req.URL.RequestURI(), "", ErrDomainWorkHTTPRedirectDeniedV0)
	if err != nil {
		return err
	}
	for _, allowed := range allowedPaths {
		if path == allowed {
			return nil
		}
	}
	return ErrorV0{Code: ErrDomainWorkHTTPRedirectDeniedV0}
}

func sameDomainWorkHTTPOriginV0(target *url.URL, destination DestinationV0) bool {
	return target.Scheme == destination.Scheme &&
		strings.EqualFold(target.Hostname(), destination.Hostname) &&
		target.Port() == destination.Port
}

func domainWorkHTTPRedirectErrorCodeV0(err error) (string, bool) {
	var publicErr ErrorV0
	if errors.As(err, &publicErr) && publicErr.Code == ErrDomainWorkHTTPRedirectDeniedV0 {
		return ErrDomainWorkHTTPRedirectDeniedV0, true
	}
	return "", false
}
