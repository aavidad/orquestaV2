package mcpinterface

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	cliinterface "orquesta/internal/interfaces/cli"
	"orquesta/internal/interfaces/httpapi"
	sdkcommands "orquesta/sdk/commands"
)

type v20Executor struct {
	invocations []commandcore.Invocation
	result      commandcore.Result
	limits      commandcore.APILimits
}

func (executor *v20Executor) Dispatch(_ context.Context, invocation commandcore.Invocation) commandcore.Result {
	executor.invocations = append(executor.invocations, invocation)
	result := executor.result
	result.CommandID, result.CommandVersion, result.RequestRef = invocation.CommandID, invocation.CommandVersion, invocation.RequestRef
	return result
}
func (executor *v20Executor) Definitions() []commandcore.Definition {
	return commandcore.CanonicalDefinitions()
}
func (executor *v20Executor) Limits() commandcore.APILimits {
	if executor.limits.Valid() {
		return executor.limits
	}
	return commandcore.APILimits{MaxRequestBytes: 4096, MaxListLimit: 100}
}

type v20Identity struct {
	principal identity.Principal
	err       error
	calls     *int
}

func (provider v20Identity) Principal(context.Context) (identity.Principal, error) {
	if provider.calls != nil {
		*provider.calls++
	}
	return provider.principal, provider.err
}

func v20Principal(t *testing.T) identity.Principal {
	t.Helper()
	ref, _ := identity.NewPrincipalRef("principal:v20")
	actor, _ := goal.NewActorRef("actor:v20")
	principal, err := identity.NewPrincipal(ref, actor, identity.PrincipalKindHuman, "local_token")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func TestHTTPMCPCLIAndSDKReturnEquivalentSuccessEnvelope(t *testing.T) {
	executor := &v20Executor{result: commandcore.Result{Data: json.RawMessage(`{"ready":true}`), AuditRef: "audit:parity"}}
	provider := v20Identity{principal: v20Principal(t)}
	httpHandler, err := httpapi.New(httpapi.Config{Dispatcher: executor, Identity: provider})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(httpHandler)
	defer httpServer.Close()
	body := []byte(`{"version":"1","request_ref":"request:parity","project_ref":"project:v20","payload":{}}`)
	response, err := http.Post(httpServer.URL+"/api/v1/commands/orquesta.system.status", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var httpResult commandcore.Result
	if err := json.NewDecoder(response.Body).Decode(&httpResult); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	sdkClient, err := sdkcommands.New(sdkcommands.Config{BaseURL: httpServer.URL, HTTPClient: httpServer.Client(), MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	sdkResult, err := sdkClient.Invoke(context.Background(), sdkcommands.Request{CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:parity", ProjectRef: "project:v20", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	cli, err := cliinterface.New(sdkClient)
	if err != nil {
		t.Fatal(err)
	}
	cliResult, err := cli.Run(context.Background(), []string{"system", "status"}, "request:parity", "project:v20", "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	mcpServer := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, mcpServer, executor, provider, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return mcpServer }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mcpHTTP := httptest.NewServer(handler)
	t.Cleanup(mcpHTTP.Close)
	session := connectOfficialClient(t, mcpHTTP.URL)
	called := callTool(t, session, "orquesta.system.status", map[string]any{"version": "1", "request_ref": "request:parity", "project_ref": "project:v20", "payload": map[string]any{}})
	var output CommandToolOutput
	decodeStructured(t, called, &output)
	sdkCoreResult := decodeSDKCommandResult(t, sdkResult)
	cliCoreResult := decodeSDKCommandResult(t, cliResult)
	if !reflect.DeepEqual(httpResult, sdkCoreResult) || !reflect.DeepEqual(httpResult, cliCoreResult) ||
		!reflect.DeepEqual(httpResult, output.Result) {
		t.Fatalf("http=%+v sdk=%+v cli=%+v mcp=%+v", httpResult, sdkResult, cliResult, output.Result)
	}
}

func TestExecutionBoundCommandsResolveClaimedExecutionAgainstAuthenticatedPrincipal(t *testing.T) {
	executor := &v20Executor{result: commandcore.Result{Data: json.RawMessage(`{}`), AuditRef: "audit:mcp"}}
	provider := v20Identity{principal: v20Principal(t)}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, server, executor, provider, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return server }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	endpoint := httptest.NewServer(handler)
	t.Cleanup(endpoint.Close)
	session := connectOfficialClient(t, endpoint.URL)
	result := callTool(t, session, "orquesta.mailbox.get", map[string]any{"version": "1", "request_ref": "request:mcp", "project_ref": "project:v20", "claimed_execution_ref": "execution:v20", "payload": map[string]any{"goal_ref": "goal:g", "message_ref": "mailbox-message:m", "recipient_work_item_ref": "work-item:w"}})
	var output CommandToolOutput
	decodeStructured(t, result, &output)
	if output.Result.Failure != nil || len(executor.invocations) != 1 ||
		executor.invocations[0].Principal.Ref.String() != "principal:v20" ||
		executor.invocations[0].ClaimedExecutionRef != "execution:v20" {
		t.Fatalf("result=%+v invocation=%+v", output.Result, executor.invocations)
	}
}

func TestHTTPMCPCLIAndSDKReturnEquivalentInvalidRequest(t *testing.T) {
	authorityCalls := 0
	executor := &v20Executor{result: commandcore.Result{Failure: &commandcore.Failure{
		Code: commandcore.CodeInvalidRequest, MessageKey: "error.invalid_request",
	}}}
	results := invokeAllCommandSurfaces(t, executor, v20Identity{
		principal: v20Principal(t), calls: &authorityCalls,
	}, "request:invalid", json.RawMessage(`{}`))
	assertEquivalentCommandResults(t, results)
	if results[0].Failure == nil || results[0].Failure.Code != commandcore.CodeInvalidRequest ||
		results[0].Failure.MessageKey != "error.invalid_request" ||
		authorityCalls != 4 || len(executor.invocations) != 4 {
		t.Fatalf("result=%+v authority=%d dispatch=%d", results[0], authorityCalls, len(executor.invocations))
	}
}

func TestHTTPMCPCLIAndSDKReturnEquivalentUnauthenticatedForbiddenNotFoundConflict(t *testing.T) {
	tests := []struct {
		code        string
		identityErr error
	}{
		{code: commandcore.CodeUnauthenticated, identityErr: errors.New("missing identity")},
		{code: commandcore.CodeForbidden},
		{code: commandcore.CodeNotFound},
		{code: commandcore.CodeConflict},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			executor := &v20Executor{}
			if test.identityErr == nil {
				executor.result.Failure = &commandcore.Failure{Code: test.code, MessageKey: "error." + test.code}
			}
			results := invokeAllCommandSurfaces(t, executor, v20Identity{
				principal: v20Principal(t), err: test.identityErr,
			}, "request:parity-error", json.RawMessage(`{}`))
			assertEquivalentCommandResults(t, results)
			if results[0].Failure == nil || results[0].Failure.Code != test.code ||
				results[0].Failure.MessageKey != "error."+test.code {
				t.Fatalf("result=%+v", results[0])
			}
		})
	}
}

func TestCommandErrorsUseStableCodeAndCatalogKeyAcrossBindings(t *testing.T) {
	stable := []struct {
		core string
		sdk  string
	}{
		{commandcore.CodeInvalidRequest, sdkcommands.CodeInvalidRequest},
		{commandcore.CodeUnauthenticated, sdkcommands.CodeUnauthenticated},
		{commandcore.CodeForbidden, sdkcommands.CodeForbidden},
		{commandcore.CodeNotFound, sdkcommands.CodeNotFound},
		{commandcore.CodeConflict, sdkcommands.CodeConflict},
		{commandcore.CodeUnavailable, sdkcommands.CodeUnavailable},
		{commandcore.CodeInternal, sdkcommands.CodeInternal},
	}
	for _, code := range stable {
		if code.core != code.sdk {
			t.Fatalf("core=%q sdk=%q", code.core, code.sdk)
		}
		var output CommandToolOutput
		decodeStructured(t, toolFailure(commandBinding{
			CommandID: "orquesta.system.status", Version: "1", Path: "orquesta.system.status",
		}, "1", "request:error", code.core), &output)
		if output.Result.Failure == nil || output.Result.Failure.Code != code.core ||
			output.Result.Failure.MessageKey != "error."+code.core {
			t.Fatalf("code=%q output=%+v", code.core, output)
		}
	}
}

func invokeAllCommandSurfaces(t *testing.T, executor *v20Executor, provider v20Identity, requestRef string, payload json.RawMessage) []commandcore.Result {
	t.Helper()
	httpHandler, err := httpapi.New(httpapi.Config{Dispatcher: executor, Identity: provider})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(httpHandler)
	t.Cleanup(httpServer.Close)
	encoded, err := json.Marshal(map[string]any{
		"version": "1", "request_ref": requestRef, "project_ref": "project:v20", "payload": payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(httpServer.URL+"/api/v1/commands/orquesta.system.status", "application/json", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	var direct commandcore.Result
	if err := json.NewDecoder(response.Body).Decode(&direct); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()

	sdkClient, err := sdkcommands.New(sdkcommands.Config{
		BaseURL: httpServer.URL, HTTPClient: httpServer.Client(), MaxResponseBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	sdkResult, err := sdkClient.Invoke(context.Background(), sdkcommands.Request{
		CommandID: "orquesta.system.status", Version: "1", RequestRef: requestRef,
		ProjectRef: "project:v20", Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	cli, err := cliinterface.New(sdkClient)
	if err != nil {
		t.Fatal(err)
	}
	cliResult, err := cli.Run(context.Background(), []string{"system", "status"}, requestRef, "project:v20", "", payload)
	if err != nil {
		t.Fatal(err)
	}

	mcpServer := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, mcpServer, executor, provider, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return mcpServer }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mcpHTTP := httptest.NewServer(handler)
	t.Cleanup(mcpHTTP.Close)
	called := callTool(t, connectOfficialClient(t, mcpHTTP.URL), "orquesta.system.status", map[string]any{
		"version": "1", "request_ref": requestRef, "project_ref": "project:v20", "payload": payload,
	})
	var output CommandToolOutput
	decodeStructured(t, called, &output)
	return []commandcore.Result{
		direct, decodeSDKCommandResult(t, sdkResult), decodeSDKCommandResult(t, cliResult), output.Result,
	}
}

func assertEquivalentCommandResults(t *testing.T, results []commandcore.Result) {
	t.Helper()
	for index := 1; index < len(results); index++ {
		if !reflect.DeepEqual(results[0], results[index]) {
			t.Fatalf("results=%+v", results)
		}
	}
}

func decodeSDKCommandResult(t *testing.T, input sdkcommands.Result) commandcore.Result {
	t.Helper()
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var result commandcore.Result
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMCPPublishesExactlyCanonicalSchemasForAllCommands(t *testing.T) {
	executor := &v20Executor{}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, server, executor, v20Identity{principal: v20Principal(t)}, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return server }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	endpoint := httptest.NewServer(handler)
	t.Cleanup(endpoint.Close)
	listed, err := connectOfficialClient(t, endpoint.URL).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != len(commandcore.CanonicalDefinitions()) {
		t.Fatalf("tools=%d definitions=%d", len(listed.Tools), len(commandcore.CanonicalDefinitions()))
	}
	definitions := make(map[string]commandcore.Definition)
	for _, definition := range executor.Definitions() {
		definitions[definition.ID] = definition
	}
	for _, tool := range listed.Tools {
		definition, ok := definitions[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool=%q", tool.Name)
		}
		encoded, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(encoded, &schema); err != nil {
			t.Fatal(err)
		}
		if !jsonSemanticallyEqual(schema.Properties["payload"], definition.InputSchema) {
			t.Fatalf("%s payload schema drift", definition.ID)
		}
		delete(definitions, tool.Name)
	}
	if len(definitions) != 0 {
		t.Fatalf("missing tools=%v", definitions)
	}
}

func TestMCPRejectsOversizedArgumentsBeforeAuthorityOrDispatch(t *testing.T) {
	executor := &v20Executor{limits: commandcore.APILimits{MaxRequestBytes: 96, MaxListLimit: 10}}
	provider := v20Identity{principal: v20Principal(t)}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, server, executor, provider, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return server }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	endpoint := httptest.NewServer(handler)
	t.Cleanup(endpoint.Close)
	result := callTool(t, connectOfficialClient(t, endpoint.URL), "orquesta.system.status", map[string]any{
		"version": "1", "request_ref": "request:" + string(make([]byte, 200)), "project_ref": "project:v20", "payload": map[string]any{},
	})
	var output CommandToolOutput
	decodeStructured(t, result, &output)
	if output.Result.Failure == nil || output.Result.Failure.Code != commandcore.CodeInvalidRequest || len(executor.invocations) != 0 {
		t.Fatalf("output=%+v calls=%d", output, len(executor.invocations))
	}
}

func TestMCPRejectsMissingCanonicalPayloadBeforeAuthorityOrDispatch(t *testing.T) {
	executor := &v20Executor{}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v20", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, server, executor, v20Identity{principal: v20Principal(t)}, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return server }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	endpoint := httptest.NewServer(handler)
	t.Cleanup(endpoint.Close)
	result := callTool(t, connectOfficialClient(t, endpoint.URL), "orquesta.system.status", map[string]any{
		"version": "1", "request_ref": "request:missing", "project_ref": "project:v20",
	})
	var output CommandToolOutput
	decodeStructured(t, result, &output)
	if output.Result.Failure == nil || output.Result.Failure.Code != commandcore.CodeInvalidRequest || len(executor.invocations) != 0 {
		t.Fatalf("output=%+v calls=%d", output, len(executor.invocations))
	}
}

func jsonSemanticallyEqual(left, right []byte) bool {
	var one, two any
	return json.Unmarshal(left, &one) == nil && json.Unmarshal(right, &two) == nil && reflect.DeepEqual(one, two)
}
