package main

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	commandHTTPRedirectDeniedV0 = "command_http_redirect_denied"

	commandHTTPTransportProfileLoopbackControlPlaneV0 = "loopback_control_plane"
	commandHTTPTransportProfileOPESTemporalV0         = "opes_temporal"
	commandHTTPProxyPolicyDenyV0                      = "deny"
)

type commandHTTPTransportPolicyV0 struct {
	Profile               string
	ProxyPolicy           string
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	KeepAlive             time.Duration
}

func commandHTTPClientWithRedirectPolicyV0(timeout time.Duration, baseURL string) http.Client {
	return http.Client{
		Timeout:       timeout,
		Transport:     commandHTTPTransportV0(commandHTTPTransportProfileLoopbackControlPlaneV0),
		CheckRedirect: commandHTTPRedirectPolicyV0(baseURL),
	}
}

func commandOPESTemporalHTTPClientV0(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: commandHTTPTransportV0(commandHTTPTransportProfileOPESTemporalV0),
	}
}

func commandHTTPTransportPolicyV0ForProfile(profile string) commandHTTPTransportPolicyV0 {
	policy := commandHTTPTransportPolicyV0{
		Profile:               profile,
		ProxyPolicy:           commandHTTPProxyPolicyDenyV0,
		DialTimeout:           3 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		KeepAlive:             30 * time.Second,
	}
	if profile == commandHTTPTransportProfileOPESTemporalV0 {
		policy.DialTimeout = 5 * time.Second
		policy.TLSHandshakeTimeout = 5 * time.Second
		policy.ResponseHeaderTimeout = 10 * time.Second
		policy.MaxIdleConns = 32
		policy.MaxIdleConnsPerHost = 8
	}
	return policy
}

func commandHTTPTransportV0(profile string) *http.Transport {
	policy := commandHTTPTransportPolicyV0ForProfile(profile)
	dialer := &net.Dialer{Timeout: policy.DialTimeout, KeepAlive: policy.KeepAlive}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          policy.MaxIdleConns,
		MaxIdleConnsPerHost:   policy.MaxIdleConnsPerHost,
		IdleConnTimeout:       policy.IdleConnTimeout,
		TLSHandshakeTimeout:   policy.TLSHandshakeTimeout,
		ResponseHeaderTimeout: policy.ResponseHeaderTimeout,
	}
}

func commandHTTPRedirectPolicyV0(baseURL string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !commandHTTPRedirectAllowedV0(req, baseURL) || len(via) >= 10 {
			return commandHTTPRedirectErrorV0{}
		}
		return nil
	}
}

func commandHTTPRedirectAllowedV0(req *http.Request, baseURL string) bool {
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

type commandHTTPRedirectErrorV0 struct{}

func (commandHTTPRedirectErrorV0) Error() string {
	return commandHTTPRedirectDeniedV0
}

func commandHTTPRedirectDeniedErrorV0(err error) bool {
	var redirectErr commandHTTPRedirectErrorV0
	return errors.As(err, &redirectErr)
}
