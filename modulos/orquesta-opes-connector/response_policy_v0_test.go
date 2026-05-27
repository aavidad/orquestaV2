package orquestaopesconnector

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeOPESJSONResponseV0AplicaLimiteContentTypeYTrailing(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "content type", contentType: "text/html", body: `{"id":"job-ref-001"}`, want: ErrOPESResponseContentTypeV0},
		{name: "trailing", contentType: "application/json", body: `{"id":"job-ref-001"} {}`, want: ErrOPESResponseTrailingDataV0},
		{name: "too large", contentType: "application/json", body: strings.Repeat(" ", int(defaultOPESResponseMaxBytesV0)+1), want: ErrOPESResponseBodyTooLargeV0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			err := decodeOPESJSONResponseV0(responseForOPESTestV0(tt.contentType, tt.body), &target)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err=%v want=%s", err, tt.want)
			}
		})
	}
}

func TestDecodeOPESJSONResponseV0AceptaLegacyTextPlainJSON(t *testing.T) {
	var target map[string]any
	err := decodeOPESJSONResponseV0(responseForOPESTestV0("text/plain; charset=utf-8", `{"id":"job-ref-001"}`), &target)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if target["id"] != "job-ref-001" {
		t.Fatalf("target=%v", target)
	}
}

func responseForOPESTestV0(contentType string, body string) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
