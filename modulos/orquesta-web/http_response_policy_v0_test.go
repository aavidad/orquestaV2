package orquestaweb

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeWebHTTPJSONResponseV0AplicaLimiteContentTypeYTrailing(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "content type", contentType: "text/html", body: `{"ok":true}`},
		{name: "trailing", contentType: "application/json", body: `{"ok":true} {}`},
		{name: "too large", contentType: "application/json", body: strings.Repeat(" ", int(webHTTPResponseMaxBytesV0)+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			if decodeWebHTTPJSONResponseV0(webResponseForTestV0(tt.contentType, tt.body), &target) {
				t.Fatalf("respuesta aceptada")
			}
		})
	}
}

func TestDecodeWebHTTPJSONResponseV0AceptaLegacyTextPlainJSON(t *testing.T) {
	var target map[string]any
	if !decodeWebHTTPJSONResponseV0(webResponseForTestV0("text/plain; charset=utf-8", `{"ok":true}`), &target) {
		t.Fatalf("respuesta rechazada")
	}
	if target["ok"] != true {
		t.Fatalf("target=%v", target)
	}
}

func webResponseForTestV0(contentType string, body string) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
