package orquestaweb

import (
	"net"
	"net/http"
	"sync"
	"time"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
)

const (
	WebHTTPTransportProfileLoopbackControlPlaneV0 = "loopback_control_plane"
	WebHTTPProxyPolicyDenyV0                      = "deny"
)

type WebHTTPTransportPolicyV0 struct {
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

func DefaultWebHTTPTransportPolicyV0() WebHTTPTransportPolicyV0 {
	return WebHTTPTransportPolicyV0{
		Profile:               WebHTTPTransportProfileLoopbackControlPlaneV0,
		ProxyPolicy:           WebHTTPProxyPolicyDenyV0,
		DialTimeout:           3 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		KeepAlive:             30 * time.Second,
	}
}

func newWebLoopbackHTTPClientV0(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: webInProcessRegistryTransportV0{base: newWebHTTPTransportV0(DefaultWebHTTPTransportPolicyV0())},
	}
}

func newWebHTTPTransportV0(policy WebHTTPTransportPolicyV0) *http.Transport {
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

var webInProcessHTTPRegistryV0 sync.Map

type webInProcessRegistryTransportV0 struct {
	base http.RoundTripper
}

func registerWebInProcessHTTPHandlerV0(host string, handler http.Handler) {
	if host == "" || handler == nil {
		return
	}
	webInProcessHTTPRegistryV0.Store(host, handler)
}

func unregisterWebInProcessHTTPHandlerV0(host string) {
	if host == "" {
		return
	}
	webInProcessHTTPRegistryV0.Delete(host)
}

func (transport webInProcessRegistryTransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		if handler, ok := webInProcessHTTPRegistryV0.Load(req.URL.Host); ok {
			response, err := inprocesshttp.TransportV0{Handler: handler.(http.Handler)}.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			if response.Header.Get("Content-Type") == "" && response.ContentLength > 0 {
				response.Header.Set("Content-Type", "application/json")
			}
			return response, nil
		}
	}
	if transport.base != nil {
		return transport.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
