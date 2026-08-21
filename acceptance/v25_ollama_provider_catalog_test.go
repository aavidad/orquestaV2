package acceptance_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/ollama"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

var _ application.ProviderCatalogSource = (*ollama.ProviderCatalogSource)(nil)

func TestAcceptanceV25OllamaCatalogIsLocalExplicitAndObservationOnly(t *testing.T) {
	now := time.Date(2026, 8, 21, 17, 0, 0, 0, time.UTC)
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		_, _ = writer.Write([]byte(`{"models":[{"name":"coder-looking:latest"},{"name":"plain:latest"}]}`))
	}))
	defer server.Close()
	baseConfig := ollama.ProviderCatalogConfig{
		BaseURL: server.URL, Now: func() time.Time { return now },
		Freshness: time.Minute, RequestTimeout: time.Second, MaxResponseBytes: 4096,
	}
	source, err := ollama.NewProviderCatalogSource(baseConfig)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := source.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requestedPath != "/api/tags" || observation.ProviderRef != ollama.ProviderRef ||
		observation.Availability != ports.ProviderAvailabilityUnknown ||
		observation.Quota != ports.ProviderQuotaUnknown || observation.Usage.Known != 0 ||
		len(observation.Models) != 2 {
		t.Fatalf("unexpected Ollama observation: %+v path=%q", observation, requestedPath)
	}
	for _, model := range observation.Models {
		if len(model.CapabilityRefs) != 0 || len(model.ReasoningEfforts) != 0 {
			t.Fatalf("catalog listing invented capability facts: %+v", model)
		}
	}
	remote := baseConfig
	remote.BaseURL = "http://8.8.8.8:11434"
	if source, err := ollama.NewProviderCatalogSource(remote); source != nil ||
		!errors.Is(err, ollama.ErrProviderCatalogInvalid) {
		t.Fatalf("remote endpoint source=%v err=%v", source, err)
	}
}

func TestAcceptanceV25OllamaCatalogFailsClosedOffline(t *testing.T) {
	now := time.Date(2026, 8, 21, 17, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		status  int
		body    string
		maximum int64
	}{
		{name: "non-2xx", status: http.StatusServiceUnavailable, body: `{"models":[]}`, maximum: 64},
		{name: "malformed", status: http.StatusOK, body: `{"models":`, maximum: 64},
		{name: "duplicate", status: http.StatusOK, body: `{"models":[{"name":"same"},{"name":"same"}]}`, maximum: 64},
		{name: "oversized", status: http.StatusOK, body: strings.Repeat("x", 65), maximum: 64},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			source, err := ollama.NewProviderCatalogSource(ollama.ProviderCatalogConfig{
				BaseURL: server.URL, Now: func() time.Time { return now },
				Freshness: time.Minute, RequestTimeout: time.Second,
				MaxResponseBytes: test.maximum,
			})
			if err != nil {
				t.Fatal(err)
			}
			if observation, err := source.ObserveProviderCatalog(context.Background()); err == nil || observation.ProviderRef != "" {
				t.Fatalf("invalid response accepted: observation=%+v err=%v", observation, err)
			}
		})
	}
}
