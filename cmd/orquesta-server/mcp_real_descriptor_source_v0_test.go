package main

import (
	"encoding/json"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func assertMCPRealDescriptorSourceNoSensitiveTestV0(
	t *testing.T,
	source orquestamcp.MCPResourceDescriptorSourceV0,
) {
	t.Helper()
	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("descriptor_source marshal: %v", err)
	}
	text := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"/home/", "home=", "token", "password", "oauth", "provider", "model",
		"prompt", "transcript", "completion", "dsn", "sql",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor_source contiene %q: %s", forbidden, text)
		}
	}
}
