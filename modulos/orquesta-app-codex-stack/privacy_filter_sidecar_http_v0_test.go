package orquestaappcodexstack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrivacyFilterSidecarHTTPPortV0PostsSimpleLocalContract(t *testing.T) {
	observed := PrivacyFilterSidecarHTTPRequestV0{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/filter" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("content-type=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&observed); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(PrivacyFilterSidecarHTTPResponseV0{
			SchemaVersion:    PrivacyFilterSidecarHTTPResponseSchemaVersionV0,
			Content:          "public sanitized-ref-001",
			Sanitized:        true,
			EvidenceRefs:     []string{"/home/alberto/private/evidence.json"},
			Categories:       []string{"credential_value"},
			ReplacementCount: 1,
		})
	}))
	defer server.Close()

	port, err := NewPrivacyFilterSidecarHTTPPortV0(PrivacyFilterSidecarHTTPConfigV0{
		Enabled:          true,
		EndpointURL:      server.URL + "/filter",
		Timeout:          time.Second,
		MaxResponseBytes: 4096,
		EvidenceRef:      "evidence-ref-privacy-filter-test",
	})
	if err != nil {
		t.Fatalf("NewPrivacyFilterSidecarHTTPPortV0: %v", err)
	}

	result := port.FilterEgressPayloadV0(PrivacyFilterSidecarRequestV0{
		RequestRef: "task-ref-privacy-filter-http",
		EntryRef:   "entry-ref-privacy-filter-http",
		SourceRef:  "source-ref-privacy-filter-http",
		Payload:    `token="sk-secret-local" public=true`,
	})

	if observed.SchemaVersion != PrivacyFilterSidecarHTTPRequestSchemaVersionV0 ||
		observed.RequestRef != "task-ref-privacy-filter-http" ||
		!strings.Contains(observed.Payload, "sk-secret-local") {
		t.Fatalf("unexpected sidecar request: %+v", observed)
	}
	if !result.Sanitized || result.ReviewRequired || result.Content != "public sanitized-ref-001" {
		t.Fatalf("unexpected sidecar result: %+v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "/home/alberto") {
		t.Fatalf("sidecar evidence leaked local path: %s", string(raw))
	}
}

func TestPrivacyFilterSidecarHTTPPortV0RejectsNonLocalEndpoint(t *testing.T) {
	_, err := NewPrivacyFilterSidecarHTTPPortV0(PrivacyFilterSidecarHTTPConfigV0{
		Enabled:     true,
		EndpointURL: "https://example.com/privacy-filter",
	})
	if err == nil || !strings.Contains(err.Error(), "not_local") {
		t.Fatalf("expected non-local endpoint error, got %v", err)
	}
}

func TestPrivacyFilterSidecarHTTPConfigV0RedactsOperationalEndpoint(t *testing.T) {
	public := RedactedPrivacyFilterSidecarHTTPConfigV0(PrivacyFilterSidecarHTTPConfigV0{
		Enabled:     true,
		EndpointURL: "http://127.0.0.1:48123/filter",
		SidecarRef:  "sidecar-ref-test",
		AdapterRef:  "adapter-ref-test",
		EvidenceRef: "/home/alberto/private/evidence.json",
	})

	raw, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshal public config: %v", err)
	}
	if !public.LocalEndpointConfigured {
		t.Fatalf("endpoint should be projected as configured")
	}
	for _, forbidden := range []string{"127.0.0.1", "48123", "/filter", "/home/alberto"} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(forbidden)) {
			t.Fatalf("public config leaked %q: %s", forbidden, string(raw))
		}
	}
}
