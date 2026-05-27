package orquestadomainworkhttp

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeDomainWorkHTTPJSONResponseV0AplicaLimiteContentTypeYTrailing(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "content type", contentType: "text/html", body: `{"status":"ok"}`, want: ErrDomainWorkHTTPResponseContentTypeV0},
		{name: "trailing", contentType: "application/json", body: `{"status":"ok"} {}`, want: ErrDomainWorkHTTPResponseTrailingDataV0},
		{name: "too large", contentType: "application/json", body: strings.Repeat(" ", int(defaultDomainWorkHTTPResponseMaxBytesV0)+1), want: ErrDomainWorkHTTPResponseBodyTooLargeV0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			err := decodeDomainWorkHTTPJSONResponseV0(responseForDomainWorkHTTPTestV0(http.StatusOK, tt.contentType, tt.body), &target)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err=%v want=%s", err, tt.want)
			}
		})
	}
}

func TestDecodeDomainWorkHTTPJSONResponseV0AceptaLegacyTextPlainJSON(t *testing.T) {
	var target map[string]any
	err := decodeDomainWorkHTTPJSONResponseV0(responseForDomainWorkHTTPTestV0(http.StatusOK, "text/plain; charset=utf-8", `{"status":"ok"}`), &target)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if target["status"] != "ok" {
		t.Fatalf("target=%v", target)
	}
}

func responseForDomainWorkHTTPTestV0(status int, contentType string, body string) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
