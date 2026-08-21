package acceptance_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/openaicompat"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

var _ application.ProviderCatalogSource = (*openaicompat.ProviderCatalogSource)(nil)

func TestAcceptanceV25OpenAICompatibleCatalogDoesNotClaimAvailability(t *testing.T) {
	now := time.Date(2026, 8, 21, 17, 0, 0, 0, time.UTC)
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		_, _ = writer.Write([]byte(`{"object":"list","data":[{"id":"tool-looking-model"},{"id":"plain-model"}]}`))
	}))
	defer server.Close()
	source := v25OpenAICompatibleSource(t, server, func() time.Time { return now }, 1024)

	observation, err := source.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requestedPath != "/v1/models" || observation.ProviderRef != openaicompat.ProviderRef ||
		observation.Availability != ports.ProviderAvailabilityUnknown ||
		observation.Quota != ports.ProviderQuotaUnknown || observation.Usage.Known != 0 ||
		len(observation.Models) != 2 {
		t.Fatalf("unexpected observation: %+v path=%q", observation, requestedPath)
	}
	for _, model := range observation.Models {
		if len(model.CapabilityRefs) != 0 || len(model.ReasoningEfforts) != 0 {
			t.Fatalf("nominal compatibility invented facts: %+v", model)
		}
	}
}

func TestAcceptanceV25OpenAICompatibleCatalogFailsClosed(t *testing.T) {
	now := time.Date(2026, 8, 21, 17, 0, 0, 0, time.UTC)
	for _, baseURL := range []string{
		"http://10.0.0.1", "http://100.64.0.1", "http://169.254.169.254",
		"http://192.0.2.1", "http://198.18.0.1", "http://8.8.8.8", "http://[fd00::1]",
	} {
		config := openaicompat.ProviderCatalogConfig{
			BaseURL: baseURL, Now: func() time.Time { return now },
			Freshness: time.Minute, RequestTimeout: time.Second, MaxResponseBytes: 1024,
		}
		if source, err := openaicompat.NewProviderCatalogSource(config); source != nil ||
			!errors.Is(err, openaicompat.ErrProviderCatalogInvalid) {
			t.Fatalf("forbidden base_url=%q source=%v err=%v", baseURL, source, err)
		}
	}

	tests := []struct {
		name    string
		status  int
		body    string
		maximum int64
	}{
		{name: "non-2xx", status: http.StatusServiceUnavailable, body: `{"data":[]}`, maximum: 64},
		{name: "malformed", status: http.StatusOK, body: `{"data":[`, maximum: 64},
		{name: "duplicate", status: http.StatusOK, body: `{"data":[{"id":"same"},{"id":"same"}]}`, maximum: 64},
		{name: "oversized", status: http.StatusOK, body: strings.Repeat("x", 65), maximum: 64},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			source := v25OpenAICompatibleSource(t, server, func() time.Time { return now }, test.maximum)
			if observation, err := source.ObserveProviderCatalog(context.Background()); err == nil || observation.ProviderRef != "" {
				t.Fatalf("failure leaked observation: %+v err=%v", observation, err)
			}
		})
	}
}

func v25OpenAICompatibleSource(t *testing.T, server *httptest.Server, now func() time.Time, maximum int64) *openaicompat.ProviderCatalogSource {
	t.Helper()
	source, err := openaicompat.NewProviderCatalogSource(openaicompat.ProviderCatalogConfig{
		BaseURL: server.URL,
		Now:     now, Freshness: time.Minute, RequestTimeout: time.Second, MaxResponseBytes: maximum,
	})
	if err != nil {
		t.Fatal(err)
	}
	return source
}
