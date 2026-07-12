package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
)

func TestMCPBootstrapComposicionCanonicaCableaCatalogoYSuperficiesV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)

	if err := verifyCanonicalMCPBootstrapV0(server.URL, stack); err != nil {
		t.Fatal(err)
	}

	broken := stack
	broken.MCPTransportBindings.OperatorDirectorMessage = channel.OperatorDirectorChannelServiceV0{}
	if err := verifyCanonicalMCPBindingsV0(broken); err == nil || !strings.Contains(err.Error(), "OperatorDirectorMessage") {
		t.Fatalf("binding desactivado no produjo rojo causal: %v", err)
	}
}

func buildCanonicalMCPBootstrapStackForTestV0(t *testing.T) orquestaappcodexstack.StackV0 {
	t.Helper()
	projectDir := t.TempDir()
	if err := writeServerCodeContextFixtureV0(projectDir); err != nil {
		t.Fatalf("write codebase fixture: %v", err)
	}
	canonicalConfig, err := os.ReadFile(filepath.Join("..", "..", serverProjectConfigFileNameV0))
	if err != nil {
		t.Fatalf("read canonical project config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), canonicalConfig, 0o600); err != nil {
		t.Fatalf("write canonical project config: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, t.TempDir())
	t.Setenv(envCodexRuntimeWorkDirV0, filepath.Join(t.TempDir(), "runtime"))
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	return stack
}

func verifyCanonicalMCPBootstrapV0(baseURL string, stack orquestaappcodexstack.StackV0) error {
	if err := verifyCanonicalMCPBindingsV0(stack); err != nil {
		return err
	}

	var listedTools mcpToolListResultV0
	if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/list", map[string]any{}, &listedTools); err != nil {
		return err
	}
	actualTools := map[string]bool{}
	for _, tool := range listedTools.Tools {
		actualTools[tool.Name] = true
	}
	// La obligación canónica es fija e independiente de los bindings. Si una
	// composición omite un puerto, la tool debe faltar aquí y el guard ponerse
	// rojo; derivar esta lista de los bindings ocultaría exactamente el fallo.
	for _, expected := range orquestamcp.MCPTransportToolsV0(stack.MCPTransportBindings) {
		if !actualTools[expected.Name] {
			return fmt.Errorf("mcp bootstrap: binding declarado sin tool registrada: %s", expected.Name)
		}
	}

	// Guard exhaustivo: registrado != cableado. Llama a TODAS las tools
	// registradas y falla si alguna responde con puerto sin cablear. Sin este
	// barrido, una tool declarada con binding nil (p.ej.
	// orquesta.tool.capabilities.list.v0) pasaba el bootstrap: aparecia en
	// tools/list y nadie la llamaba nunca.
	unbound := []string{}
	for _, tool := range listedTools.Tools {
		var raw json.RawMessage
		if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/call", map[string]any{
			"name":      tool.Name,
			"arguments": map[string]any{},
		}, &raw); err != nil {
			continue // error de transporte/validacion: no es puerto sin cablear
		}
		if reason := bootstrapMissingPortReasonV0(string(raw)); reason != "" {
			unbound = append(unbound, tool.Name+" ("+reason+")")
		}
	}

	if len(unbound) > 0 {
		sort.Strings(unbound)
		return fmt.Errorf("mcp bootstrap: tools registradas pero NO cableadas (%d): %s", len(unbound), strings.Join(unbound, ", "))
	}

	var listedResources mcpResourceListResultV0
	if err := callMCPJSONRPCBootstrapV0(baseURL, "resources/list", map[string]any{}, &listedResources); err != nil {
		return err
	}
	actualResources := map[string]bool{}
	for _, resource := range listedResources.Resources {
		actualResources[resource.Name] = true
	}
	for _, expected := range orquestamcp.MCPTransportResourcesV0() {
		if !actualResources[expected.Name] {
			return fmt.Errorf("mcp bootstrap: resource declarado no registrado: %s", expected.Name)
		}
	}

	calls := []struct {
		group    string
		tool     string
		httpPath string
		input    any
	}{
		{group: "goal_observe", tool: orquestamcp.MCPObserveAppDirectorGoalToolNameV0, httpPath: orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0, input: map[string]any{"run_ref": "run-ref-h0b-missing"}},
		{group: "autoprogramming_status", tool: orquestamcp.MCPAutoprogrammingStatusToolNameV0, httpPath: orquestamcp.MCPAutoprogrammingStatusHTTPPathV0, input: map[string]any{"queue_ref": "global"}},
		{group: "run_control", tool: orquestamcp.MCPRunControlToolNameV0, httpPath: orquestamcp.MCPRunControlHTTPPathV0, input: map[string]any{"action": "pause", "run_ref": "run-ref-h0b-missing", "requested_by": "h0b-bootstrap", "reason": "binding probe"}},
		{group: "director_stats", tool: orquestamcp.MCPDirectorStatsToolNameV0, httpPath: orquestamcp.MCPDirectorStatsHTTPPathV0, input: map[string]any{"run_ref": "run-ref-h0b-missing"}},
		{group: "operator_director_message", tool: channel.OperatorDirectorMessageToolNameV0, input: map[string]any{"request_ref": "request-ref-h0b", "target_ref": "director", "intent": "status", "body": "estado del bootstrap"}},
		{group: "codebase", tool: orquestamcp.MCPCodebaseQueryToolNameV0, httpPath: orquestamcp.MCPCodebaseQueryHTTPPathV0, input: serverCodeContextQueryForTestV0("request-ref-h0b-codebase")},
	}
	for _, call := range calls {
		var result mcpToolCallResultV0
		if err := callMCPJSONRPCBootstrapV0(baseURL, "tools/call", map[string]any{"name": call.tool, "arguments": call.input}, &result); err != nil {
			return fmt.Errorf("grupo %s MCP: %w", call.group, err)
		}
		encoded, _ := json.Marshal(result)
		if reason := bootstrapMissingPortReasonV0(string(encoded)); reason != "" {
			return fmt.Errorf("grupo %s MCP binding muerto: %s payload=%s", call.group, reason, string(encoded))
		}
		if call.httpPath == "" {
			continue
		}
		body, err := json.Marshal(call.input)
		if err != nil {
			return err
		}
		response, err := http.Post(baseURL+call.httpPath, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("grupo %s HTTP: %w", call.group, err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		_ = response.Body.Close()
		if readErr != nil {
			return fmt.Errorf("grupo %s HTTP read: %w", call.group, readErr)
		}
		if response.StatusCode == http.StatusNotFound {
			return fmt.Errorf("grupo %s HTTP ruta no montada: %s", call.group, call.httpPath)
		}
		if reason := bootstrapMissingPortReasonV0(string(responseBody)); reason != "" {
			return fmt.Errorf("grupo %s HTTP binding muerto: %s", call.group, reason)
		}
	}
	return nil
}

func verifyCanonicalMCPBindingsV0(stack orquestaappcodexstack.StackV0) error {
	bindings := stack.MCPTransportBindings
	checks := []struct {
		name string
		ok   bool
	}{
		{name: "ObserveDirectorGoal", ok: bindings.ObserveDirectorGoal != nil},
		{name: "AutoprogrammingStatus.Queue", ok: bindings.RunQueuePriority != nil},
		{name: "AutoprogrammingStatus.Stats", ok: bindings.DirectorStats != nil},
		{name: "RunControl", ok: bindings.RunControl != nil},
		{name: "DirectorStats", ok: bindings.DirectorStats != nil},
		{name: "OperatorDirectorMessage", ok: bindings.OperatorDirectorMessage.Dispatcher != nil && bindings.OperatorDirectorMessage.Store != nil},
		{name: "CodebaseQuery", ok: bindings.CodebaseQuery != nil},
		{name: "CodebaseStatus", ok: bindings.CodebaseStatus != nil},
	}
	for _, check := range checks {
		if !check.ok {
			return fmt.Errorf("mcp bootstrap: binding canonico no cableado: %s", check.name)
		}
	}
	return nil
}

func callMCPJSONRPCBootstrapV0(baseURL, method string, params, output any) error {
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "request-ref-h0b-bootstrap", "method": method, "params": params})
	if err != nil {
		return err
	}
	response, err := http.Post(baseURL+mcpRealHTTPPathV0, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("POST /mcp status=%d", response.StatusCode)
	}
	var rpc mcpJSONRPCRawResponseV0
	if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
		return err
	}
	if rpc.Error != nil {
		return fmt.Errorf("POST /mcp method=%s error=%+v", method, rpc.Error)
	}
	return json.Unmarshal(rpc.Result, output)
}

func bootstrapMissingPortReasonV0(payload string) string {
	lower := strings.ToLower(payload)
	for _, reason := range []string{"port_unavailable", "port_no_disponible", "puerto_no_disponible", "transport_unbound", "mcp_transport_tool_unbound", "operator_message_port_unavailable"} {
		if strings.Contains(lower, reason) {
			return reason
		}
	}
	return ""
}
