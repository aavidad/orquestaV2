package orquestaweb

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
)

const webHTTPClientTestBaseURLV0 = "http://orquesta-web.test"

var (
	webHTTPClientTestServerSeqV0 atomic.Uint64
)

type webHTTPClientRoundTripperV0 struct {
	handler http.Handler
}

type webHTTPClientTestServerV0 struct {
	URL  string
	host string
}

type webHTTPClientRegistryRoundTripperV0 struct {
	base http.RoundTripper
}

func init() {
	http.DefaultTransport = webHTTPClientRegistryRoundTripperV0{base: http.DefaultTransport}
}

func newWebHTTPTestServerV0(t *testing.T, handler http.Handler) *webHTTPClientTestServerV0 {
	t.Helper()
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	id := webHTTPClientTestServerSeqV0.Add(1)
	host := "orquesta-web-test-" + strconv.FormatUint(id, 10) + ".local"
	registerWebInProcessHTTPHandlerV0(host, handler)
	return &webHTTPClientTestServerV0{
		URL:  "http://" + host,
		host: host,
	}
}

func (server *webHTTPClientTestServerV0) Close() {
	if server == nil || server.host == "" {
		return
	}
	unregisterWebInProcessHTTPHandlerV0(server.host)
}

func newWebHTTPClientForHandlerV0(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: webHTTPClientRoundTripperV0{handler: handler},
	}
}

func (transport webHTTPClientRoundTripperV0) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := inprocesshttp.TransportV0{Handler: transport.handler}.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if response.Header.Get("Content-Type") == "" && response.ContentLength > 0 {
		response.Header.Set("Content-Type", "application/json")
	}
	return response, nil
}

func (transport webHTTPClientRegistryRoundTripperV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		if handler, ok := webInProcessHTTPRegistryV0.Load(req.URL.Host); ok {
			return webHTTPClientRoundTripperV0{handler: handler.(http.Handler)}.RoundTrip(req)
		}
	}
	if transport.base != nil {
		return transport.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
