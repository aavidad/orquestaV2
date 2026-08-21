// Package localhttp owns the bounded HTTP transport used by local agent
// adapters. It accepts only explicit loopback endpoints and never delegates
// socket selection, proxying, redirects, or TLS policy to callers.
package localhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MaxResponseBodyBytes   = int64(4 << 20)
	MaxResponseHeaderBytes = int64(64 << 10)
)

var (
	ErrEndpointInvalid  = errors.New("localhttp.endpoint_invalid")
	ErrUnavailable      = errors.New("localhttp.unavailable")
	ErrResponseTooLarge = errors.New("localhttp.response_too_large")
	ErrTargetMismatch   = errors.New("localhttp.target_mismatch")
)

// Endpoint is immutable after construction and safe for concurrent queries.
// requestURL contains the authorized numeric address, while authority keeps
// the explicit logical Host header supplied by the local composition.
type Endpoint struct {
	requestURL url.URL
	authority  string
	target     netip.AddrPort
	client     http.Client
	transport  *http.Transport
}

// NewEndpoint builds a transport from scratch. No caller-owned client,
// transport, resolver, proxy, dialer, cookie jar, or TLS hook can enter it.
func NewEndpoint(baseURL, exactPath string, timeout time.Duration) (*Endpoint, error) {
	parsed, target, serverName, err := parseLoopbackEndpoint(baseURL, exactPath)
	if err != nil || timeout <= 0 {
		return nil, ErrEndpointInvalid
	}

	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy:                  nil,
		DialContext:            pinnedDialContext(dialer, target),
		ForceAttemptHTTP2:      false,
		DisableKeepAlives:      true,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName},
		TLSHandshakeTimeout:    timeout,
		ResponseHeaderTimeout:  timeout,
		MaxResponseHeaderBytes: MaxResponseHeaderBytes,
	}
	client := http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Endpoint{
		requestURL: *parsed,
		authority:  parsedAuthority(baseURL),
		target:     target,
		client:     client,
		transport:  transport,
	}, nil
}

// Get performs one exact GET against the endpoint fixed at construction.
// The returned body remains owned by the caller and must be closed.
func (endpoint *Endpoint) Get(ctx context.Context) (*http.Response, error) {
	if endpoint == nil || ctx == nil {
		return nil, ErrEndpointInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.requestURL.String(), nil)
	if err != nil {
		return nil, errors.Join(ErrEndpointInvalid, err)
	}
	request.Host = endpoint.authority
	response, err := endpoint.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errors.Join(ErrUnavailable, err)
	}
	return response, nil
}

// ReadResponseBody applies the shared hard ceiling while streaming the
// response. Callers decode only after the complete bounded body is available.
func ReadResponseBody(ctx context.Context, response *http.Response, maximum int64) ([]byte, error) {
	if ctx == nil || response == nil || response.Body == nil || maximum <= 0 || maximum > MaxResponseBodyBytes {
		return nil, ErrEndpointInvalid
	}
	if response.ContentLength > maximum {
		return nil, ErrResponseTooLarge
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errors.Join(ErrUnavailable, err)
	}
	if int64(len(payload)) > maximum {
		return nil, ErrResponseTooLarge
	}
	return payload, nil
}

func parseLoopbackEndpoint(baseURL, exactPath string) (*url.URL, netip.AddrPort, string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || strings.TrimSpace(baseURL) != baseURL || parsed.Opaque != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" ||
		exactPath == "" || !strings.HasPrefix(exactPath, "/") || strings.ContainsAny(exactPath, "?#") {
		return nil, netip.AddrPort{}, "", ErrEndpointInvalid
	}

	host := parsed.Hostname()
	if host == "" || strings.Contains(host, "%") || strings.HasSuffix(parsed.Host, ":") {
		return nil, netip.AddrPort{}, "", ErrEndpointInvalid
	}
	address, serverName, err := explicitLoopback(host)
	if err != nil {
		return nil, netip.AddrPort{}, "", err
	}
	port, err := endpointPort(parsed)
	if err != nil {
		return nil, netip.AddrPort{}, "", err
	}
	target := netip.AddrPortFrom(address, port)
	parsed.Host = net.JoinHostPort(address.String(), strconv.Itoa(int(port)))
	parsed.Path = exactPath
	return parsed, target, serverName, nil
}

func explicitLoopback(host string) (netip.Addr, string, error) {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return netip.MustParseAddr("127.0.0.1"), "localhost", nil
	}
	address, err := netip.ParseAddr(host)
	if err != nil || address.Is4In6() || !address.IsLoopback() {
		return netip.Addr{}, "", ErrEndpointInvalid
	}
	return address, address.String(), nil
}

func endpointPort(parsed *url.URL) (uint16, error) {
	portText := parsed.Port()
	if portText == "" {
		if parsed.Scheme == "https" {
			return 443, nil
		}
		return 80, nil
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return 0, ErrEndpointInvalid
	}
	return uint16(port), nil
}

func parsedAuthority(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func pinnedDialContext(dialer *net.Dialer, target netip.AddrPort) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, requestedAddress string) (net.Conn, error) {
		requested, err := netip.ParseAddrPort(requestedAddress)
		if err != nil || network != "tcp" || requested != target {
			return nil, ErrTargetMismatch
		}
		return dialer.DialContext(ctx, "tcp", target.String())
	}
}
