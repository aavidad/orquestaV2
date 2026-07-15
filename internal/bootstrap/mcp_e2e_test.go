package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	mcpiface "orquesta/internal/interfaces/mcp"
)

func TestRealMCPAPIClosesDurableGoalThroughSQLiteAndArtifactStore(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	var launches atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "test-e2e", AgentFactory: countingFactory(&launches),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	principal, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatalf("compose local identity: %v", err)
	}
	membership, err := runtime.repository.Membership(context.Background(), principal.Ref, hierarchy.ProjectRef())
	if err != nil || !membership.IsActive() || membership.Role() != identity.RoleProjectOwner ||
		principal.Ref.String() != runtime.config.IdentityLocalActor() ||
		principal.Method != localtoken.AuthenticationMethod {
		t.Fatalf("local RBAC provisioning: principal=%+v membership=%+v err=%v", principal, membership, err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	unauthorized, err := http.Get(runtime.MCPURL())
	if err != nil {
		t.Fatalf("unauthenticated request: %v", err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized || unauthorized.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("unauthenticated status = %d headers=%v", unauthorized.StatusCode, unauthorized.Header)
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "orquesta-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, root+"/secrets/local-owner.token"),
	}, nil)
	if err != nil {
		t.Fatalf("connect official MCP client: %v", err)
	}
	defer session.Close()

	createdResult := callMCPTool(t, ctx, session, mcpiface.ToolGoalsCreate, map[string]any{
		"project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e",
		"statement": "produce API evidence", "confirm": true,
	})
	var created mcpiface.CreateGoalOutput
	decodeMCPOutput(t, createdResult, &created)
	if createdResult.IsError || !created.Created || created.Goal == nil ||
		created.Goal.ActorRef != principal.ActorRef.String() ||
		created.Goal.ProjectRef != hierarchy.ProjectRef().String() {
		t.Fatalf("create = %+v result=%+v", created, createdResult)
	}

	var closed mcpiface.GoalView
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result := callMCPTool(t, ctx, session, mcpiface.ToolGoalsGet, map[string]any{
			"project_ref": hierarchy.ProjectRef().String(), "goal_ref": created.Goal.GoalRef,
		})
		var output mcpiface.GetGoalOutput
		decodeMCPOutput(t, result, &output)
		if !result.IsError && output.Goal != nil && output.Goal.State == string(goal.GoalStateSucceeded) {
			closed = *output.Goal
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if closed.GoalRef == "" || len(closed.Artifacts) != 1 || len(closed.Attestations) != 1 || launches.Load() != 1 {
		t.Fatalf("closure missing: goal=%+v launches=%d", closed, launches.Load())
	}

	artifactResult := callMCPTool(t, ctx, session, mcpiface.ToolArtifactsRead, map[string]any{
		"project_ref": hierarchy.ProjectRef().String(), "goal_ref": closed.GoalRef,
		"artifact_ref": closed.Artifacts[0].ArtifactRef,
	})
	var artifact mcpiface.ReadArtifactOutput
	decodeMCPOutput(t, artifactResult, &artifact)
	if artifactResult.IsError || artifact.Artifact == nil {
		t.Fatalf("artifact read = %+v result=%+v", artifact, artifactResult)
	}
	content, err := base64.StdEncoding.DecodeString(artifact.Artifact.ContentBase64)
	if err != nil || !strings.Contains(string(content), "produce API evidence") {
		t.Fatalf("artifact content = %q err=%v", content, err)
	}

	replayResult := callMCPTool(t, ctx, session, mcpiface.ToolGoalsCreate, map[string]any{
		"project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e",
		"statement": "produce API evidence", "confirm": true,
	})
	var replay mcpiface.CreateGoalOutput
	decodeMCPOutput(t, replayResult, &replay)
	if replayResult.IsError || replay.Created || replay.Goal == nil || replay.Goal.GoalRef != closed.GoalRef || launches.Load() != 1 {
		t.Fatalf("idempotent replay = %+v launches=%d", replay, launches.Load())
	}

	statusResult := callMCPTool(t, ctx, session, mcpiface.ToolSystemStatus, map[string]any{
		"project_ref": hierarchy.ProjectRef().String(),
	})
	var status mcpiface.SystemStatusOutput
	decodeMCPOutput(t, statusResult, &status)
	if statusResult.IsError || !status.Ready || status.Version != "test-e2e" || status.Goals != 1 ||
		status.RunningGoals != 0 || status.PendingActions != 0 {
		t.Fatalf("status = %+v result=%+v", status, statusResult)
	}
}

func callMCPTool(
	t *testing.T,
	ctx context.Context,
	session *sdkmcp.ClientSession,
	name string,
	arguments map[string]any,
) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return result
}

func decodeMCPOutput(t *testing.T, result *sdkmcp.CallToolResult, target any) {
	t.Helper()
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured output: %v", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("decode structured output %s: %v", payload, err)
	}
}
