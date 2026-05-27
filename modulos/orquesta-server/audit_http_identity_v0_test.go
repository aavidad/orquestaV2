package orquestaserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuditHTTPHandlerV0RedactaIdentidadClienteV0(t *testing.T) {
	stateDir := t.TempDir()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir: stateDir,
		Addr:     "127.0.0.1:8787",
	}, RuntimeDepsV0{
		StateStore: &memoryStateStoreV0{},
		Clock:      fixedClockV0{now: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	req.Header.Set("Forwarded", "for=203.0.113.10;proto=https")

	runtime.HandlerV0().ServeHTTP(rec, req)

	events := readAuditEventsForTestV0(t, AuditPathV0(runtime.config))
	last := events[len(events)-1]
	identity, ok := last.Payload["client_identity"].(map[string]interface{})
	if !ok {
		t.Fatalf("client_identity ausente: %+v", last.Payload)
	}
	if identity["schema_version"] != auditClientIdentitySchemaV0 ||
		identity["category"] != "loopback" ||
		identity["redaction"] != auditClientIdentityRedactionV0 ||
		identity["raw_address_persisted"] != false ||
		identity["forwarded_header_policy"] != "ignored_untrusted" ||
		identity["authorization_scope"] != "loopback_bind" {
		t.Fatalf("identity=%+v", identity)
	}
	headers, ok := identity["forwarded_headers_present"].([]interface{})
	if !ok || len(headers) != 2 || headers[0] != "Forwarded" || headers[1] != "X-Forwarded-For" {
		t.Fatalf("headers=%+v", identity["forwarded_headers_present"])
	}
	encoded, _ := json.Marshal(last.Payload)
	for _, forbidden := range []string{"127.0.0.1", "45678", "203.0.113.10", "for=203", "remote_addr"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("payload filtra identidad %q: %s", forbidden, string(encoded))
		}
	}
}

func TestAuditRemoteAddrCategoryV0(t *testing.T) {
	cases := []struct {
		name string
		host string
		want string
	}{
		{name: "loopback", host: "127.0.0.1", want: "loopback"},
		{name: "localhost", host: "localhost", want: "loopback"},
		{name: "private", host: "10.0.0.5", want: "private"},
		{name: "external", host: "8.8.8.8", want: "external"},
		{name: "unknown", host: "example.invalid", want: "unknown"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := auditRemoteAddrCategoryV0(tt.host); got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}
