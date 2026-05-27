package orquestacli

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeCLIRESTJSONBodyV0AplicaLimiteContentTypeYTrailing(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "content type", contentType: "text/html", body: `{"ok":true}`, want: "response_content_type"},
		{name: "trailing", contentType: "application/json", body: `{"ok":true} {}`, want: "response_trailing_data"},
		{name: "too large", contentType: "application/json", body: strings.Repeat(" ", int(cliRESTResponseMaxBytesV0)+1), want: "response_body_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			if got := decodeCLIRESTJSONBodyV0(cliResponseForTestV0(tt.contentType, tt.body), &target); got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestDecodeCLIRESTJSONBodyV0AceptaLegacyTextPlainJSON(t *testing.T) {
	var target map[string]any
	if got := decodeCLIRESTJSONBodyV0(cliResponseForTestV0("text/plain; charset=utf-8", `{"ok":true}`), &target); got != "" {
		t.Fatalf("got=%q", got)
	}
	if target["ok"] != true {
		t.Fatalf("target=%v", target)
	}
}

func TestDecodeCLIRESTJSONBodyForCommandV0AplicaLimitePorComando(t *testing.T) {
	body := strings.Repeat(" ", int(cliRESTResponseMaxBytesForCommandV0(CliDefaultCommandServerStatusV0))+1)
	var target map[string]any
	if got := decodeCLIRESTJSONBodyForCommandV0(cliResponseForTestV0("application/json", body), CliDefaultCommandServerStatusV0, &target); got != "response_body_too_large" {
		t.Fatalf("got=%q", got)
	}
}

func TestCLIRESTNo2xxDetailForCommandV0NoPropagaBodyCrudo(t *testing.T) {
	body := `<html>token=secret&url=http://user:pass@example.invalid/private</html>`
	got := cliRESTNo2xxDetailForCommandV0(cliResponseForTestV0("text/html", body), CliDefaultCommandSolicitarAppV0)
	if got != "status_no_2xx" {
		t.Fatalf("got=%q", got)
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "example.invalid") || strings.Contains(got, "user:pass") {
		t.Fatalf("detail leaks body: %q", got)
	}
}

func cliResponseForTestV0(contentType string, body string) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}
