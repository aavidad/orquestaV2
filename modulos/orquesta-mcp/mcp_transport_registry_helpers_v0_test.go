package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeMCPTransportV0 struct {
	resources map[string]MCPTransportResourceEnvelopeV0
	tools     map[string]MCPTransportToolEnvelopeV0
}

func newFakeMCPTransportV0() *fakeMCPTransportV0 {
	return &fakeMCPTransportV0{
		resources: map[string]MCPTransportResourceEnvelopeV0{},
		tools:     map[string]MCPTransportToolEnvelopeV0{},
	}
}

func (f *fakeMCPTransportV0) RegisterResourceV0(resource MCPTransportResourceEnvelopeV0) error {
	f.resources[resource.Name] = resource
	return nil
}

func (f *fakeMCPTransportV0) RegisterToolV0(tool MCPTransportToolEnvelopeV0) error {
	f.tools[tool.Name] = tool
	return nil
}

func (f *fakeMCPTransportV0) ReadResourceV0(ctx context.Context, name string) (json.RawMessage, error) {
	return f.resources[name].Handler(ctx)
}

func (f *fakeMCPTransportV0) CallToolV0(ctx context.Context, name string, input any) (json.RawMessage, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return f.tools[name].Handler(ctx, raw)
}

func assertTransportPayloadSaneadoMCPTestV0(t *testing.T, value any, maxBytes int) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(payload) > maxBytes {
		t.Fatalf("payload demasiado grande: got=%d max=%d payload=%s", len(payload), maxBytes, payload)
	}
	text := strings.ToLower(string(payload))
	for _, allowedFalseFlag := range []string{
		`"contains_secret":false`,
		`"contains_transcript":false`,
		`"contains_prompt":false`,
		`"contains_completion":false`,
		`"contains_connection_detail":false`,
	} {
		text = strings.ReplaceAll(text, allowedFalseFlag, "")
	}
	for _, forbidden := range []string{"password", "oauth", "/home/", "home=", "event-store", "internal/", "transcript", "secret", "dsn", "sql"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("payload contiene %q: %s", forbidden, text)
		}
	}
}
