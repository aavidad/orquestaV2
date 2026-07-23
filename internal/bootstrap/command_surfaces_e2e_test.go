package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	mcpinterface "orquesta/internal/interfaces/mcp"
	sdkcommands "orquesta/sdk/commands"
)

func TestRealCommandAuditSurvivesRestartAndLifecycleRemainsApplicationOwned(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	var launches atomic.Int64
	body := commandEnvelope(t, "request:http-restart", "project:default", "", map[string]any{
		"statement": "produce restart command evidence", "confirm": true,
	})

	first, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "command-restart", AgentFactory: countingFactory(&launches),
	})
	if err != nil {
		t.Fatalf("build first: %v", err)
	}
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("start first: %v", err)
	}
	original := postRuntimeCommand(t, first, root, "orquesta.goals.create", body)
	if original.Failure != nil || original.AuditRef == "" {
		t.Fatalf("first command=%+v", original)
	}
	var receipt struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	if err := json.Unmarshal(original.Data, &receipt); err != nil || receipt.Goal.GoalRef == "" {
		t.Fatalf("first data=%s err=%v", original.Data, err)
	}
	ref, err := goalRef(receipt.Goal.GoalRef)
	if err != nil {
		t.Fatalf("goal ref: %v", err)
	}
	terminal := waitTerminalGoal(t, first, ref)
	if len(terminal.Artifacts) != 1 || launches.Load() != 1 {
		t.Fatalf("first closure artifacts=%d launches=%d", len(terminal.Artifacts), launches.Load())
	}
	shutdownRuntime(t, first)

	second, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "command-restart", AgentFactory: countingFactory(&launches),
	})
	if err != nil {
		t.Fatalf("build second: %v", err)
	}
	if err := second.Start(context.Background()); err != nil {
		t.Fatalf("start second: %v", err)
	}
	t.Cleanup(func() { shutdownRuntime(t, second) })
	replay := postRuntimeCommand(t, second, root, "orquesta.goals.create", body)
	if replay.Failure != nil || replay.AuditRef != original.AuditRef ||
		!bytes.Equal(replay.Data, original.Data) || launches.Load() != 1 {
		t.Fatalf("replay=%+v original=%+v launches=%d", replay, original, launches.Load())
	}
}

func TestRealHTTPMCPCLIAndSDKParityEndToEnd(t *testing.T) {
	root := t.TempDir()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root), Version: "surface-parity",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { shutdownRuntime(t, runtime) })
	const requestRef = "request:surface-parity"
	httpResult := postRuntimeCommand(t, runtime, root, "orquesta.system.status",
		commandEnvelope(t, requestRef, "project:default", "", map[string]any{}))

	httpClient := authorizedHTTPClient(t, root+"/secrets/local-owner.token")
	sdkClient, err := sdkcommands.New(sdkcommands.Config{
		BaseURL: "http://" + runtime.Address(), HTTPClient: httpClient, MaxResponseBytes: 64 << 10,
	})
	if err != nil {
		t.Fatalf("SDK: %v", err)
	}
	sdkResult, err := sdkClient.Invoke(context.Background(), sdkcommands.Request{
		CommandID: "orquesta.system.status", Version: "1", RequestRef: requestRef,
		ProjectRef: "project:default", Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("SDK invoke: %v", err)
	}
	cliResult := invokeRealCommandCLI(t, runtime, root, requestRef)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "surface-parity", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, root+"/secrets/local-owner.token"),
	}, nil)
	if err != nil {
		t.Fatalf("MCP connect: %v", err)
	}
	defer session.Close()
	mcpCalled := callMCPTool(t, ctx, session, "orquesta.system.status", map[string]any{
		"version": "1", "request_ref": requestRef, "project_ref": "project:default",
		"payload": map[string]any{},
	})
	var mcpOutput mcpinterface.CommandToolOutput
	decodeMCPOutput(t, mcpCalled, &mcpOutput)
	sdkCore := decodeSDKResult(t, sdkResult)
	cliCore := decodeSDKResult(t, cliResult)
	if !reflect.DeepEqual(httpResult, sdkCore) || !reflect.DeepEqual(httpResult, cliCore) ||
		!reflect.DeepEqual(httpResult, mcpOutput.Result) || httpResult.Failure != nil || httpResult.AuditRef == "" {
		t.Fatalf("http=%+v sdk=%+v cli=%+v mcp=%+v", httpResult, sdkCore, cliCore, mcpOutput.Result)
	}
}

func invokeRealCommandCLI(t *testing.T, runtime *Runtime, root, requestRef string) sdkcommands.Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go",
		"run", "-mod=vendor", "./cmd/orquesta", "command",
		"--url", "http://"+runtime.Address(),
		"--credential-file", root+"/secrets/local-owner.token",
		"--max-credential-bytes", "4096",
		"--max-response-bytes", "65536",
		"--timeout", "5s",
		"--request-ref", requestRef,
		"--project-ref", "project:default",
		"--payload", "{}",
		"--", "system", "status",
	)
	command.Dir = "../.."
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("real CLI: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	var result sdkcommands.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode real CLI stdout=%q err=%v", stdout.String(), err)
	}
	return result
}

func TestExecutionResolverIsSharedByGeneratedHTTPAndMCPAndNilFailsClosed(t *testing.T) {
	root := t.TempDir()
	resolver := &recordingExecutionResolver{
		expectedProject:   "project:default",
		expectedExecution: "execution:e2e",
	}
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root), Version: "execution-e2e",
		AgentFactory: countingFactory(new(atomic.Int64)), CommandExecutionResolver: resolver,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { shutdownRuntime(t, runtime) })
	payload := map[string]any{
		"goal_ref": "goal:missing", "message_ref": "mailbox-message:missing",
		"recipient_work_item_ref": "work-item:missing",
	}
	httpResult := postRuntimeCommand(t, runtime, root, "orquesta.mailbox.get",
		commandEnvelope(t, "request:execution:http", "project:default", "execution:e2e", payload))
	if httpResult.Failure == nil || httpResult.Failure.Code != commandcore.CodeNotFound || resolver.CallCount() != 1 {
		t.Fatalf("http=%+v resolver_calls=%d", httpResult, resolver.CallCount())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "execution-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, root+"/secrets/local-owner.token"),
	}, nil)
	if err != nil {
		t.Fatalf("connect MCP: %v", err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != len(commandcore.CanonicalDefinitions()) {
		t.Fatalf("tools=%d definitions=%d err=%v", len(tools.Tools), len(commandcore.CanonicalDefinitions()), err)
	}
	called := callMCPTool(t, ctx, session, "orquesta.mailbox.get", map[string]any{
		"version": "1", "request_ref": "request:execution:mcp", "project_ref": "project:default",
		"claimed_execution_ref": "execution:e2e", "payload": payload,
	})
	var output mcpinterface.CommandToolOutput
	decodeMCPOutput(t, called, &output)
	if output.Result.Failure == nil || output.Result.Failure.Code != commandcore.CodeNotFound ||
		resolver.CallCount() != 2 {
		t.Fatalf("mcp=%+v resolver_calls=%d", output.Result, resolver.CallCount())
	}

	foreignProject := postRuntimeCommand(t, runtime, root, "orquesta.mailbox.get",
		commandEnvelope(t, "request:execution:foreign-project", "project:other", "execution:e2e", payload))
	if foreignProject.Failure == nil || foreignProject.Failure.Code != commandcore.CodeForbidden ||
		foreignProject.AuditRef != "" || resolver.CallCount() != 3 {
		t.Fatalf("foreign project=%+v resolver_calls=%d", foreignProject, resolver.CallCount())
	}
	successor := postRuntimeCommand(t, runtime, root, "orquesta.mailbox.get",
		commandEnvelope(t, "request:execution:successor", "project:default", "execution:successor", payload))
	if successor.Failure == nil || successor.Failure.Code != commandcore.CodeForbidden ||
		successor.AuditRef != "" || resolver.CallCount() != 4 {
		t.Fatalf("successor=%+v resolver_calls=%d", successor, resolver.CallCount())
	}

	nilRoot := t.TempDir()
	nilRuntime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, nilRoot), Version: "execution-nil",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("build nil resolver: %v", err)
	}
	if err := nilRuntime.Start(context.Background()); err != nil {
		t.Fatalf("start nil resolver: %v", err)
	}
	t.Cleanup(func() { shutdownRuntime(t, nilRuntime) })
	forbidden := postRuntimeCommand(t, nilRuntime, nilRoot, "orquesta.mailbox.get",
		commandEnvelope(t, "request:execution:nil", "project:default", "execution:e2e", payload))
	if forbidden.Failure == nil || forbidden.Failure.Code != commandcore.CodeForbidden {
		t.Fatalf("nil resolver result=%+v", forbidden)
	}
}

func commandEnvelope(t *testing.T, requestRef, projectRef, executionRef string, payload any) []byte {
	t.Helper()
	value := map[string]any{
		"version": "1", "request_ref": requestRef, "project_ref": projectRef, "payload": payload,
	}
	if executionRef != "" {
		value["claimed_execution_ref"] = executionRef
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}
	return encoded
}

func postRuntimeCommand(t *testing.T, runtime *Runtime, root, commandID string, body []byte) commandcore.Result {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost,
		"http://"+runtime.Address()+"/api/v1/commands/"+commandID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new command request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := authorizedHTTPClient(t, root+"/secrets/local-owner.token").Do(request)
	if err != nil {
		t.Fatalf("command request: %v", err)
	}
	defer response.Body.Close()
	var result commandcore.Result
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode command response: %v", err)
	}
	return result
}

func decodeSDKResult(t *testing.T, result sdkcommands.Result) commandcore.Result {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal SDK result: %v", err)
	}
	var decoded commandcore.Result
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode SDK result: %v", err)
	}
	return decoded
}

func shutdownRuntime(t *testing.T, runtime *Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func goalRef(value string) (goal.GoalRef, error) {
	return goal.NewGoalRef(value)
}

type recordingExecutionResolver struct {
	mu                sync.Mutex
	expectedProject   string
	expectedExecution string
	calls             int
}

func (resolver *recordingExecutionResolver) ResolveExecution(
	_ context.Context,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	claimed goal.ExecutionRef,
) (goal.ExecutionRef, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.calls++
	if principal.Ref.String() == "" || projectRef.String() != resolver.expectedProject ||
		claimed.String() != resolver.expectedExecution {
		return goal.ExecutionRef{}, errors.New("execution authority rejected")
	}
	return claimed, nil
}

func (resolver *recordingExecutionResolver) CallCount() int {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return resolver.calls
}

var _ commandcore.ExecutionAuthorityResolver = (*recordingExecutionResolver)(nil)
