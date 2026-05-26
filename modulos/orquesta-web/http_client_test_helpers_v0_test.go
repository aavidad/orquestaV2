package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const webHTTPClientTestBaseURLV0 = "http://orquesta-web.test"

var (
	webHTTPClientTestServerSeqV0      atomic.Uint64
	webHTTPClientTestServerRegistryV0 sync.Map
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
	webHTTPClientTestServerRegistryV0.Store(host, handler)
	return &webHTTPClientTestServerV0{
		URL:  "http://" + host,
		host: host,
	}
}

func (server *webHTTPClientTestServerV0) Close() {
	if server == nil || server.host == "" {
		return
	}
	webHTTPClientTestServerRegistryV0.Delete(server.host)
}

func newWebHTTPClientForHandlerV0(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: webHTTPClientRoundTripperV0{handler: handler},
	}
}

func (transport webHTTPClientRoundTripperV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if deadline, ok := req.Context().Deadline(); ok && time.Until(deadline) <= 5*time.Millisecond {
		responseCh := make(chan *http.Response, 1)
		go func() {
			responseCh <- transport.roundTripNowV0(req)
		}()
		select {
		case response := <-responseCh:
			return response, nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	return transport.roundTripNowV0(req), nil
}

func (transport webHTTPClientRoundTripperV0) roundTripNowV0(req *http.Request) *http.Response {
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, req)
	response := recorder.Result()
	response.Request = req
	return response
}

func (transport webHTTPClientRegistryRoundTripperV0) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		if handler, ok := webHTTPClientTestServerRegistryV0.Load(req.URL.Host); ok {
			return webHTTPClientRoundTripperV0{handler: handler.(http.Handler)}.RoundTrip(req)
		}
	}
	if transport.base != nil {
		return transport.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
