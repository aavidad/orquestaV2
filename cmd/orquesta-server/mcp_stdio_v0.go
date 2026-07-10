package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaserver "orquesta/modulos/orquesta-server"
)

// mcpStdioCommandV0 serves one JSON-RPC request per input line. It deliberately
// does not construct the HTTP application handler or start a listener.
func mcpStdioCommandV0(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	options, err := parseServerCommandConfigOptionsV0("mcp-stdio", args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: %v\n", err)
		return 2
	}
	config, err := serverConfigFromEnvWithProjectConfigPathV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: %v\n", err)
		return 1
	}
	bindings, err := mcpStdioBindingsFromConfigV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: %v\n", err)
		return 1
	}
	return serveMCPStdioV0(stdin, stdout, stderr, bindings)
}

func mcpStdioBindingsFromConfigV0(config orquestaserver.ConfigV0) (orquestamcp.MCPTransportBindingsV0, error) {
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		return orquestamcp.MCPTransportBindingsV0{}, err
	}
	return mcpStdioBindingsFromStackV0(stack), nil
}

func mcpStdioBindingsFromStackV0(stack orquestaappcodexstack.StackV0) orquestamcp.MCPTransportBindingsV0 {
	bindings := stack.MCPTransportBindings
	eventReader, _ := stack.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	bindings.WorkspaceTimeline = newServerWorkspaceTimelineSourceWithEventsV0(bindings, eventReader)
	return bindings
}

func serveMCPStdioV0(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	bindings orquestamcp.MCPTransportBindingsV0,
) int {
	registry, err := newMCPRealTransportRegistryForStdioV0(bindings)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: registry: %v\n", err)
		return 1
	}
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64<<10), mcpJSONRPCMaxBodyBytesV0+1)
	for scanner.Scan() {
		if err := serveMCPStdioLineV0(registry, scanner.Bytes(), stdout, stderr); err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: stdout: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		if err := writeMCPStdioDecodeErrorV0(stdout, serverPublicErrBodyTooLargeV0); err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: stdout: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: stdin: %v\n", err)
		return 1
	}
	return 0
}

func newMCPRealTransportRegistryForStdioV0(bindings orquestamcp.MCPTransportBindingsV0) (*mcpRealTransportRegistryV0, error) {
	registry := &mcpRealTransportRegistryV0{
		resourcesByName: map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		resourcesByURI:  map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		tools:           map[string]orquestamcp.MCPTransportToolEnvelopeV0{},
	}
	if err := orquestamcp.RegisterMCPTransportV0(registry, bindings); err != nil {
		return nil, err
	}
	return registry, nil
}

func serveMCPStdioLineV0(registry *mcpRealTransportRegistryV0, line []byte, stdout io.Writer, stderr io.Writer) error {
	var request mcpJSONRPCRequestV0
	if code := decodeMCPStdioJSONRPCRequestV0(line, &request); code != "" {
		_, _ = fmt.Fprintf(stderr, "orquesta-server mcp-stdio: request_rejected reason_code=%s\n", code)
		return writeMCPStdioDecodeErrorV0(stdout, code)
	}
	safeID, validationErr, notificationAccepted := validateMCPJSONRPCRequestV0(request)
	if validationErr != nil {
		return writeMCPStdioResponseV0(stdout, mcpJSONRPCResponseV0{
			JSONRPC: mcpJSONRPCVersionV0,
			ID:      safeID,
			Error:   validationErr,
		})
	}
	if notificationAccepted {
		return nil
	}
	ctx := context.WithValue(context.Background(), mcpRealCorrelationContextKeyV0{}, mcpJSONRPCIDCorrelationV0(safeID))
	result, rpcErr := registry.handleJSONRPCV0(ctx, request)
	response := mcpJSONRPCResponseV0{JSONRPC: mcpJSONRPCVersionV0, ID: safeID}
	if rpcErr != nil {
		response.Error = rpcErr
	} else {
		response.Result = result
	}
	return writeMCPStdioResponseV0(stdout, response)
}

func decodeMCPStdioJSONRPCRequestV0(line []byte, request *mcpJSONRPCRequestV0) string {
	if len(line) > mcpJSONRPCMaxBodyBytesV0 {
		return serverPublicErrBodyTooLargeV0
	}
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return "mcp_request_must_be_object"
	}
	switch trimmed[0] {
	case '[':
		return "mcp_batch_unsupported"
	case '{':
	default:
		return "mcp_request_must_be_object"
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if err := decoder.Decode(request); err != nil {
		return "mcp_json_invalid"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return serverPublicErrBodyTrailingDataV0
	}
	return ""
}

func writeMCPStdioDecodeErrorV0(stdout io.Writer, code string) error {
	return writeMCPStdioResponseV0(stdout, mcpJSONRPCResponseV0{
		JSONRPC: mcpJSONRPCVersionV0,
		Error:   mcpRPCErrorV0(mcpRPCCodeForDecodeErrorV0(code), "mcp_parse_error", code),
	})
}

func writeMCPStdioResponseV0(stdout io.Writer, response mcpJSONRPCResponseV0) error {
	return json.NewEncoder(stdout).Encode(response)
}
