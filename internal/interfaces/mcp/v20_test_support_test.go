package mcpinterface

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectOfficialClient(t *testing.T, endpoint string) *sdkmcp.ClientSession {
	t.Helper()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "1"}, nil)
	session, err := client.Connect(testContext(t), &sdkmcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return session
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, arguments map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(testContext(t), &sdkmcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	return result
}

func decodeStructured(t *testing.T, result *sdkmcp.CallToolResult, target any) {
	t.Helper()
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(structured) error = %v", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("Unmarshal(structured=%s) error = %v", payload, err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
