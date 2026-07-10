package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPStdioV0ServesHandshakeListsAndResourceReadV0(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"` + orquestamcp.MCPOperatorOperationsResourceURIV0 + `"}}`,
	}, "\n")
	responses, stderr, code := runMCPStdioForTestV0(t, input)
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if len(responses) != 3 {
		t.Fatalf("responses=%d", len(responses))
	}
	for _, response := range responses {
		if response.Error != nil {
			t.Fatalf("response error=%+v", response.Error)
		}
	}
	var initialized mcpInitializeResultV0
	if err := json.Unmarshal(responses[0].Result, &initialized); err != nil || initialized.ProtocolVersion == "" {
		t.Fatalf("initialize result=%s err=%v", responses[0].Result, err)
	}
	var tools mcpToolListResultV0
	if err := json.Unmarshal(responses[1].Result, &tools); err != nil || len(tools.Tools) == 0 {
		t.Fatalf("tools result=%s err=%v", responses[1].Result, err)
	}
	var resource mcpResourceReadResultV0
	if err := json.Unmarshal(responses[2].Result, &resource); err != nil || len(resource.Contents) != 1 {
		t.Fatalf("resource result=%s err=%v", responses[2].Result, err)
	}
}

func TestMCPStdioV0NotificationDoesNotWriteResponseV0(t *testing.T) {
	responses, stderr, code := runMCPStdioForTestV0(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if code != 0 || stderr != "" || len(responses) != 0 {
		t.Fatalf("code=%d stderr=%q responses=%d", code, stderr, len(responses))
	}
}

func TestMCPStdioV0RejectsInvalidAndTrailingJSONWithoutStdoutLogsV0(t *testing.T) {
	responses, stderr, code := runMCPStdioForTestV0(t, strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"} {}`,
	}, "\n"))
	if code != 0 || len(responses) != 2 {
		t.Fatalf("code=%d stderr=%q responses=%d", code, stderr, len(responses))
	}
	if !strings.Contains(stderr, "mcp_json_invalid") || !strings.Contains(stderr, serverPublicErrBodyTrailingDataV0) {
		t.Fatalf("stderr=%q", stderr)
	}
	for _, response := range responses {
		if response.Error == nil || response.Error.Message != "mcp_parse_error" {
			t.Fatalf("response=%+v", response)
		}
	}
}

func runMCPStdioForTestV0(t *testing.T, input string) ([]mcpJSONRPCRawResponseV0, string, int) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := serveMCPStdioV0(strings.NewReader(input), &stdout, &stderr, orquestamcp.MCPTransportBindingsV0{})
	var responses []mcpJSONRPCRawResponseV0
	decoder := json.NewDecoder(&stdout)
	for {
		var response mcpJSONRPCRawResponseV0
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode stdout=%q: %v", stdout.String(), err)
		}
		responses = append(responses, response)
	}
	return responses, stderr.String(), code
}
