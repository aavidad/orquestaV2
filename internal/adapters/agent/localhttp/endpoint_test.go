package localhttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEndpointAcceptsOnlyExplicitLoopback(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:11434",
		"http://127.23.45.67",
		"http://[::1]:8080",
		"http://localhost:8000",
		"https://LOCALHOST.:8443/",
	} {
		if endpoint, err := NewEndpoint(raw, "/models", time.Second); err != nil || endpoint == nil {
			t.Errorf("loopback endpoint %q rejected: endpoint=%v err=%v", raw, endpoint, err)
		}
	}

	for _, raw := range []string{
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://192.168.1.1",
		"http://100.64.0.1",
		"http://169.254.169.254",
		"http://192.0.2.1",
		"http://198.18.0.1",
		"http://240.0.0.1",
		"http://8.8.8.8",
		"http://[fc00::1]",
		"http://[fe80::1]",
		"http://[2001:db8::1]",
		"http://[2606:4700:4700::1111]",
		"http://[::ffff:127.0.0.1]",
		"http://catalog.example",
		"http://localhost.evil",
		"http://localhost:8080/path",
		"http://localhost:8080?query=1",
		"http://user@localhost:8080",
	} {
		if endpoint, err := NewEndpoint(raw, "/models", time.Second); endpoint != nil || !errors.Is(err, ErrEndpointInvalid) {
			t.Errorf("non-loopback endpoint %q accepted: endpoint=%v err=%v", raw, endpoint, err)
		}
	}
}

func TestEndpointPinsTheAuthorizedSocketAndOwnsTransport(t *testing.T) {
	var goodRequests atomic.Int64
	good := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		goodRequests.Add(1)
		if request.URL.Path != "/catalog" {
			t.Errorf("path=%q", request.URL.Path)
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	defer good.Close()
	var poisonRequests atomic.Int64
	poison := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		poisonRequests.Add(1)
	}))
	defer poison.Close()

	endpoint, err := NewEndpoint(good.URL, "/catalog", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	poisonAddress := strings.TrimPrefix(poison.URL, "http://")
	if connection, dialErr := endpoint.transport.DialContext(
		context.Background(), "tcp", poisonAddress,
	); connection != nil || !errors.Is(dialErr, ErrTargetMismatch) {
		t.Fatalf("remapped dial connection=%v err=%v", connection, dialErr)
	}

	response, err := endpoint.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := ReadResponseBody(context.Background(), response, 16)
	if err != nil || string(payload) != "ok" {
		t.Fatalf("payload=%q err=%v", payload, err)
	}
	if goodRequests.Load() != 1 || poisonRequests.Load() != 0 {
		t.Fatalf("good_requests=%d poison_requests=%d", goodRequests.Load(), poisonRequests.Load())
	}

	transport := endpoint.transport
	if transport.Proxy != nil || transport.DialTLS != nil || transport.DialTLSContext != nil ||
		transport.TLSNextProto != nil || transport.MaxResponseHeaderBytes != MaxResponseHeaderBytes {
		t.Fatalf("transport retained external hooks or limits: %+v", transport)
	}
	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil || tlsConfig.InsecureSkipVerify || tlsConfig.VerifyConnection != nil ||
		tlsConfig.VerifyPeerCertificate != nil || tlsConfig.GetClientCertificate != nil {
		t.Fatalf("transport retained TLS callback: %+v", tlsConfig)
	}
}

func TestEndpointMapsLocalhostWithoutDNSAndPreservesAuthority(t *testing.T) {
	var host string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		host = request.Host
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := NewEndpoint("http://localhost:"+port, "/catalog", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	response, err := endpoint.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if host != "localhost:"+port || endpoint.target.Addr().String() != "127.0.0.1" {
		t.Fatalf("host=%q target=%s", host, endpoint.target)
	}
}

func TestEndpointRejectsRedirectsAndOversizedHeaders(t *testing.T) {
	var redirected atomic.Int64
	second := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Add(1)
	}))
	defer second.Close()
	first := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, second.URL, http.StatusFound)
	}))
	defer first.Close()
	endpoint, err := NewEndpoint(first.URL, "/catalog", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	response, err := endpoint.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusFound || redirected.Load() != 0 {
		t.Fatalf("status=%d redirected=%d", response.StatusCode, redirected.Load())
	}

	headers := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Oversized", strings.Repeat("x", int(MaxResponseHeaderBytes)))
		_, _ = writer.Write([]byte("ok"))
	}))
	defer headers.Close()
	endpoint, err = NewEndpoint(headers.URL, "/catalog", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if response, err = endpoint.Get(context.Background()); response != nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("oversized header response=%v err=%v", response, err)
	}
}

func TestReadResponseBodyEnforcesConfiguredAndHardLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", 17)))
	}))
	defer server.Close()
	endpoint, err := NewEndpoint(server.URL, "/catalog", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	response, err := endpoint.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if payload, readErr := ReadResponseBody(context.Background(), response, 16); payload != nil ||
		!errors.Is(readErr, ErrResponseTooLarge) {
		t.Fatalf("payload=%q err=%v", payload, readErr)
	}
	if payload, readErr := ReadResponseBody(context.Background(), response, MaxResponseBodyBytes+1); payload != nil ||
		!errors.Is(readErr, ErrEndpointInvalid) {
		t.Fatalf("hard-limit payload=%q err=%v", payload, readErr)
	}
}
