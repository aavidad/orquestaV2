package mcpinterface

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
)

type commandBinding struct{ CommandID, Version, Path string }

type commandToolInput struct {
	Version             string          `json:"version"`
	RequestRef          string          `json:"request_ref"`
	ProjectRef          string          `json:"project_ref"`
	ClaimedExecutionRef string          `json:"claimed_execution_ref,omitempty"`
	Payload             json.RawMessage `json:"payload"`
}

type CommandToolOutput struct {
	Result commandcore.Result `json:"result"`
}

func RegisterCommandTools(
	server *sdkmcp.Server,
	dispatcher commandcore.Executor,
	provider identity.Provider,
	catalog *i18n.Catalog,
	locale string,
) error {
	if server == nil || dispatcher == nil || provider == nil || catalog == nil ||
		strings.TrimSpace(locale) == "" || strings.TrimSpace(locale) != locale ||
		!dispatcher.Limits().Valid() {
		return errors.New("mcp.command_config_invalid")
	}
	if _, err := catalog.Resolve(locale); err != nil {
		return fmt.Errorf("mcp.command_locale_invalid: %w", err)
	}
	definitions := commandcore.CanonicalDefinitions()
	tools := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if definition.ID == "" || definition.Version == "" || definition.MCP.Tool == "" {
			return errors.New("mcp.command_binding_registry_invalid")
		}
		if _, duplicate := tools[definition.MCP.Tool]; duplicate {
			return errors.New("mcp.command_binding_duplicate")
		}
		tools[definition.MCP.Tool] = struct{}{}
		binding := commandBinding{CommandID: definition.ID, Version: definition.Version, Path: definition.MCP.Tool}
		inputSchema, err := commandInputSchema(definition)
		if err != nil {
			return err
		}
		outputSchema, err := commandOutputSchema(definition)
		if err != nil {
			return err
		}
		description, err := catalog.Text(locale, definition.DescriptionKey)
		if err != nil {
			return fmt.Errorf("mcp.command_description_unavailable: %w", err)
		}
		destructive := definition.MCP.Annotations.Destructive
		openWorld := definition.MCP.Annotations.OpenWorld
		tool := &sdkmcp.Tool{
			Name: binding.Path, Description: description,
			InputSchema: inputSchema, OutputSchema: outputSchema,
			Annotations: &sdkmcp.ToolAnnotations{
				ReadOnlyHint:    definition.MCP.Annotations.ReadOnly,
				DestructiveHint: &destructive,
				IdempotentHint:  definition.MCP.Annotations.Idempotent,
				OpenWorldHint:   &openWorld,
			},
		}
		server.AddTool(tool, func(ctx context.Context, request *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			return callCommandTool(ctx, request, binding, definition, dispatcher, provider), nil
		})
	}
	return nil
}

func callCommandTool(ctx context.Context, request *sdkmcp.CallToolRequest, binding commandBinding,
	definition commandcore.Definition, dispatcher commandcore.Executor, provider identity.Provider,
) *sdkmcp.CallToolResult {
	var input commandToolInput
	if request == nil || request.Params == nil || int64(len(request.Params.Arguments)) > dispatcher.Limits().MaxRequestBytes ||
		decodeStrict(request.Params.Arguments, &input) != nil || input.Version != binding.Version || input.RequestRef == "" ||
		input.ProjectRef == "" || len(input.Payload) == 0 {
		return toolFailure(binding, "", "", commandcore.CodeInvalidRequest)
	}
	if definition.ExecutionBound == (input.ClaimedExecutionRef == "") {
		return toolFailure(binding, input.Version, input.RequestRef, commandcore.CodeInvalidRequest)
	}
	principal, err := provider.Principal(ctx)
	if err != nil {
		return toolFailure(binding, input.Version, input.RequestRef, commandcore.CodeUnauthenticated)
	}
	result := dispatcher.Dispatch(ctx, commandcore.Invocation{
		CommandID: binding.CommandID, CommandVersion: input.Version, RequestRef: input.RequestRef,
		ProjectRef: input.ProjectRef, ClaimedExecutionRef: input.ClaimedExecutionRef,
		Principal: principal, Payload: input.Payload,
	})
	return toolResult(result)
}

func toolFailure(binding commandBinding, version, requestRef, code string) *sdkmcp.CallToolResult {
	if version == "" || version != binding.Version {
		version = binding.Version
	}
	return toolResult(commandcore.Result{
		CommandID: binding.CommandID, CommandVersion: version, RequestRef: requestRef,
		Failure: &commandcore.Failure{Code: code, MessageKey: "error." + code},
	})
}

func toolResult(result commandcore.Result) *sdkmcp.CallToolResult {
	output := CommandToolOutput{Result: result}
	encoded, _ := json.Marshal(output)
	return &sdkmcp.CallToolResult{
		Content:           []sdkmcp.Content{&sdkmcp.TextContent{Text: string(encoded)}},
		StructuredContent: output, IsError: result.Failure != nil,
	}
}

func decodeStrict(encoded []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("mcp.command_input_invalid")
	}
	return nil
}

func commandInputSchema(definition commandcore.Definition) (json.RawMessage, error) {
	var payload any
	if err := json.Unmarshal(definition.InputSchema, &payload); err != nil {
		return nil, err
	}
	properties := map[string]any{
		"version":     map[string]any{"type": "string", "enum": []string{definition.Version}},
		"request_ref": map[string]any{"type": "string"}, "project_ref": map[string]any{"type": "string"},
		"payload": payload,
	}
	required := []string{"version", "request_ref", "project_ref", "payload"}
	if definition.ExecutionBound {
		properties["claimed_execution_ref"] = map[string]any{"type": "string"}
		required = append(required, "claimed_execution_ref")
	}
	return json.Marshal(map[string]any{
		"type": "object", "properties": properties, "required": required, "additionalProperties": false,
	})
}

func commandOutputSchema(definition commandcore.Definition) (json.RawMessage, error) {
	var data any
	if err := json.Unmarshal(definition.OutputSchema, &data); err != nil {
		return nil, err
	}
	failure := map[string]any{
		"type": "object", "properties": map[string]any{
			"code":        map[string]any{"type": "string", "enum": definition.ErrorCodes},
			"message_key": map[string]any{"type": "string"},
		}, "required": []string{"code", "message_key"}, "additionalProperties": false,
	}
	result := map[string]any{
		"type": "object", "properties": map[string]any{
			"command_id":      map[string]any{"type": "string", "enum": []string{definition.ID}},
			"command_version": map[string]any{"type": "string", "enum": []string{definition.Version}},
			"request_ref":     map[string]any{"type": "string"}, "data": data, "failure": failure,
			"audit_ref": map[string]any{"type": "string"},
		}, "required": []string{"command_id", "command_version", "request_ref"}, "additionalProperties": false,
	}
	return json.Marshal(map[string]any{
		"type": "object", "properties": map[string]any{"result": result},
		"required": []string{"result"}, "additionalProperties": false,
	})
}
