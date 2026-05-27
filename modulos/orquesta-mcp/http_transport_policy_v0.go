package orquestamcp

import (
	"net"
	"net/http"
	"time"
)

const (
	MCPHTTPTransportProfileLoopbackControlPlaneV0 = "loopback_control_plane"
	MCPHTTPProxyPolicyDenyV0                      = "deny"
)

type MCPHTTPTransportPolicyV0 struct {
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

func DefaultMCPHTTPTransportPolicyV0() MCPHTTPTransportPolicyV0 {
	return MCPHTTPTransportPolicyV0{
		Profile:               MCPHTTPTransportProfileLoopbackControlPlaneV0,
		ProxyPolicy:           MCPHTTPProxyPolicyDenyV0,
		DialTimeout:           3 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		KeepAlive:             30 * time.Second,
	}
}

func newMCPLoopbackHTTPClientV0(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newMCPHTTPTransportV0(DefaultMCPHTTPTransportPolicyV0()),
	}
}

func newMCPHTTPTransportV0(policy MCPHTTPTransportPolicyV0) *http.Transport {
	if policy.DialTimeout <= 0 {
		policy.DialTimeout = 3 * time.Second
	}
	if policy.TLSHandshakeTimeout <= 0 {
		policy.TLSHandshakeTimeout = 3 * time.Second
	}
	if policy.ResponseHeaderTimeout <= 0 {
		policy.ResponseHeaderTimeout = 5 * time.Second
	}
	if policy.MaxIdleConns <= 0 {
		policy.MaxIdleConns = 16
	}
	if policy.MaxIdleConnsPerHost <= 0 {
		policy.MaxIdleConnsPerHost = 4
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
