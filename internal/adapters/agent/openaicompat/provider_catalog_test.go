package openaicompat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/localhttp"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestProviderCatalogPublishesOnlyObservedModelIDs(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	var method, path, host string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method, path, host = request.Method, request.URL.Path, request.Host
		_, _ = writer.Write([]byte(`{"object":"list","data":[{"id":"tool-looking-model"},{"id":"plain-model"}]}`))
	}))
	defer server.Close()
	source := newLoopbackTestSource(t, server.URL, func() time.Time { return now }, time.Minute, 1024)

	observation, err := source.ObserveProviderCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != "/v1/models" || host != strings.TrimPrefix(server.URL, "http://") {
		t.Fatalf("request=%s %s host=%q", method, path, host)
	}
	if observation.Availability != ports.ProviderAvailabilityUnknown ||
		observation.Quota != ports.ProviderQuotaUnknown ||
		observation.Usage != (governance.ResourceUsage{Quality: governance.UsageQualityUnknown}) ||
		len(observation.Models) != 2 {
		t.Fatalf("models endpoint invented facts: %+v", observation)
	}
	for _, model := range observation.Models {
		if len(model.CapabilityRefs) != 0 || len(model.ReasoningEfforts) != 0 {
			t.Fatalf("nominal compatibility invented facts: %+v", model)
		}
	}
}

func TestProviderCatalogRejectsEveryNonLoopbackClass(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC)
	for _, baseURL := range []string{
		"http://10.0.0.1", "http://100.64.0.1", "http://169.254.169.254",
		"http://192.0.2.1", "http://198.18.0.1", "http://240.0.0.1",
		"http://8.8.8.8", "http://[fc00::1]", "http://[2001:db8::1]",
		"http://catalog.example", "http://localhost.evil",
	} {
		if source, err := NewProviderCatalogSource(validConfig(baseURL, func() time.Time { return now })); source != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
			t.Errorf("base_url=%q source=%v err=%v", baseURL, source, err)
		}
	}
}

func TestProviderCatalogRejectsInvalidConfiguration(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 30, 0, 0, time.UTC)
	valid := validConfig("http://127.0.0.1:8000", func() time.Time { return now })
	for name, mutate := range map[string]func(*ProviderCatalogConfig){
		"nil clock":       func(config *ProviderCatalogConfig) { config.Now = nil },
		"zero freshness":  func(config *ProviderCatalogConfig) { config.Freshness = 0 },
		"zero timeout":    func(config *ProviderCatalogConfig) { config.RequestTimeout = 0 },
		"oversized limit": func(config *ProviderCatalogConfig) { config.MaxResponseBytes = localhttp.MaxResponseBodyBytes + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			config := valid
			mutate(&config)
			if source, err := NewProviderCatalogSource(config); source != nil || !errors.Is(err, ErrProviderCatalogInvalid) {
				t.Fatalf("source=%v err=%v", source, err)
			}
		})
	}
}

func TestProviderCatalogRejectsRedirectBeforeSecondRequest(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	secondRequests := 0
	second := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { secondRequests++ }))
	defer second.Close()
	first := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, second.URL, http.StatusFound)
	}))
	defer first.Close()
	source := newLoopbackTestSource(t, first.URL, func() time.Time { return now }, time.Minute, 100)
	if observation, err := source.ObserveProviderCatalog(context.Background()); !errors.Is(err, ErrProviderCatalogUnavailable) || observation.ProviderRef != "" || secondRequests != 0 {
		t.Fatalf("observation=%+v err=%v second_requests=%d", observation, err, secondRequests)
	}
}

func TestProviderCatalogEnforcesOwnDeadline(t *testing.T) {
	now := time.Date(2026, 8, 21, 13, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()
	config := validConfig(server.URL, func() time.Time { return now })
	config.RequestTimeout = 20 * time.Millisecond
	source, err := NewProviderCatalogSource(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.ObserveProviderCatalog(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error=%v", err)
	}
}

func TestProviderCatalogFailsClosedOnPayloadAndFreshnessFailures(t *testing.T) {
	now := time.Date(2026, 8, 21, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		body    string
		maximum int64
		want    error
	}{
		{name: "malformed", body: `{"data":[`, maximum: 100, want: ErrProviderCatalogInvalid},
		{name: "duplicate", body: `{"data":[{"id":"m"},{"id":"m"}]}`, maximum: 100, want: ErrProviderCatalogInvalid},
		{name: "trailing", body: `{"data":[]} {}`, maximum: 100, want: ErrProviderCatalogInvalid},
		{name: "oversized", body: `{"data":[]}` + strings.Repeat(" ", 20), maximum: 11, want: ErrProviderCatalogTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			source := newLoopbackTestSource(t, server.URL, func() time.Time { return now }, time.Minute, test.maximum)
			if _, err := source.ObserveProviderCatalog(context.Background()); !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
		})
	}

	clockCalls := 0
	clock := func() time.Time {
		clockCalls++
		if clockCalls == 1 {
			return now
		}
		return now.Add(time.Minute)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	source := newLoopbackTestSource(t, server.URL, clock, time.Minute, 100)
	if _, err := source.ObserveProviderCatalog(context.Background()); !errors.Is(err, ErrProviderCatalogInvalid) {
		t.Fatalf("expired during observation err=%v", err)
	}
}

func TestProviderCatalogSupportsConcurrentQueries(t *testing.T) {
	now := time.Date(2026, 8, 21, 16, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	source := newLoopbackTestSource(t, server.URL, func() time.Time { return now }, time.Minute, 100)
	var wait sync.WaitGroup
	for index := 0; index < 20; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := source.ObserveProviderCatalog(context.Background()); err != nil {
				t.Errorf("observe: %v", err)
			}
		}()
	}
	wait.Wait()
}

func validConfig(baseURL string, now func() time.Time) ProviderCatalogConfig {
	return ProviderCatalogConfig{
		BaseURL: baseURL, Now: now, Freshness: time.Minute,
		RequestTimeout: time.Second, MaxResponseBytes: 1024,
	}
}

func newLoopbackTestSource(
	t *testing.T,
	baseURL string,
	now func() time.Time,
	freshness time.Duration,
	maximum int64,
) *ProviderCatalogSource {
	t.Helper()
	config := validConfig(baseURL, now)
	config.Freshness = freshness
	config.MaxResponseBytes = maximum
	source, err := NewProviderCatalogSource(config)
	if err != nil {
		t.Fatal(err)
	}
	return source
}
