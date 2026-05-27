package orquestaappgateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
)

func TestInProcessTransportV0AcotaResponseBody(t *testing.T) {
	transport := InProcessTransportV0{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", 64)))
		}),
		MaxResponseBytes: 8,
	}

	_, err := transport.RoundTrip(newInProcessTestRequestV0(t, context.Background()))

	if err == nil || !strings.Contains(err.Error(), inprocesshttp.CodeResponseTooLargeV0) {
		t.Fatalf("error=%v", err)
	}
}

func TestInProcessTransportV0DevuelveTimeoutSiHandlerIgnoraContexto(t *testing.T) {
	release := make(chan struct{})
	transport := InProcessTransportV0{
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			<-release
		}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	defer close(release)

	_, err := transport.RoundTrip(newInProcessTestRequestV0(t, ctx))

	if err == nil || !strings.Contains(err.Error(), inprocesshttp.CodeTimeoutV0) {
		t.Fatalf("error=%v", err)
	}
}

func TestInProcessTransportV0RecuperaPanicSinStack(t *testing.T) {
	transport := InProcessTransportV0{
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("secret stack payload")
		}),
	}

	_, err := transport.RoundTrip(newInProcessTestRequestV0(t, context.Background()))

	if err == nil || err.Error() != inprocesshttp.CodeHandlerPanicV0 {
		t.Fatalf("error=%v", err)
	}
}

func TestInProcessTransportV0PreservaParidadHTTPVisible(t *testing.T) {
	transport := InProcessTransportV0{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch || r.Header.Get("X-Correlation-ID") != "corr-inprocess" {
				t.Fatalf("request=%s headers=%v", r.Method, r.Header)
			}
			w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
			w.Header().Set("X-Correlation-ID", r.Header.Get("X-Correlation-ID"))
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ok"))
		}),
	}
	req := newInProcessTestRequestV0(t, context.Background())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-inprocess")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted || resp.Status != "202 Accepted" ||
		resp.Header.Get("Content-Type") != "application/json" ||
		resp.Header.Get("X-Correlation-ID") != "corr-inprocess" ||
		string(body) != "ok" {
		t.Fatalf("response status=%s headers=%v body=%q", resp.Status, resp.Header, body)
	}
}

func newInProcessTestRequestV0(t *testing.T, ctx context.Context) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, InternalBaseURLV0+"/api/v0/test", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return req
}
