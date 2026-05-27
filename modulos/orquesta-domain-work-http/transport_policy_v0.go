package orquestadomainworkhttp

import (
	"net"
	"net/http"
	"time"
)

const (
	HTTPTransportProfileDomainEgressV0 = "domain_egress"
	HTTPProxyPolicyDenyV0              = "deny"
)

type HTTPTransportPolicyV0 struct {
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

func DefaultHTTPTransportPolicyV0() HTTPTransportPolicyV0 {
	return HTTPTransportPolicyV0{
		Profile:               HTTPTransportProfileDomainEgressV0,
		ProxyPolicy:           HTTPProxyPolicyDenyV0,
		DialTimeout:           5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       30 * time.Second,
		KeepAlive:             30 * time.Second,
	}
}

func newDomainWorkHTTPClientV0(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newDomainWorkHTTPTransportV0(DefaultHTTPTransportPolicyV0()),
	}
}

func newDomainWorkHTTPTransportV0(policy HTTPTransportPolicyV0) *http.Transport {
	if policy.DialTimeout <= 0 {
		policy.DialTimeout = 5 * time.Second
	}
	if policy.TLSHandshakeTimeout <= 0 {
		policy.TLSHandshakeTimeout = 5 * time.Second
	}
	if policy.ResponseHeaderTimeout <= 0 {
		policy.ResponseHeaderTimeout = 10 * time.Second
	}
	if policy.MaxIdleConns <= 0 {
		policy.MaxIdleConns = 32
	}
	if policy.MaxIdleConnsPerHost <= 0 {
		policy.MaxIdleConnsPerHost = 8
	}
	if policy.IdleConnTimeout <= 0 {
		policy.IdleConnTimeout = 30 * time.Second
	}
	if policy.KeepAlive <= 0 {
		policy.KeepAlive = 30 * time.Second
	}
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
