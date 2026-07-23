package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/adapters/auth/localtoken"
	commandcore "orquesta/internal/commands"
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

	createdResult := callMCPTool(t, ctx, session, "orquesta.goals.create", map[string]any{
		"version": "1", "project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e",
		"payload": map[string]any{"statement": "produce API evidence", "confirm": true},
	})
	var created mcpiface.CommandToolOutput
	decodeMCPOutput(t, createdResult, &created)
	var createdData struct {
		Goal struct {
			GoalRef    string `json:"goal_ref"`
			ProjectRef string `json:"project_ref"`
			SpecHash   string `json:"spec_hash"`
		} `json:"goal"`
	}
	decodeCommandData(t, created.Result, &createdData)
	if createdResult.IsError || created.Result.Failure != nil || created.Result.AuditRef == "" ||
		createdData.Goal.GoalRef == "" || createdData.Goal.SpecHash == "" ||
		createdData.Goal.ProjectRef != hierarchy.ProjectRef().String() {
		t.Fatalf("create = %+v result=%+v", created, createdResult)
	}

	var closedGoalRef string
	deadline := time.Now().Add(5 * time.Second)
	poll := 0
	for time.Now().Before(deadline) {
		poll++
		result := callMCPTool(t, ctx, session, "orquesta.goals.get", map[string]any{
			"version": "1", "project_ref": hierarchy.ProjectRef().String(),
			"request_ref": fmt.Sprintf("request:mcp-e2e:get:%d", poll),
			"payload":     map[string]any{"goal_ref": createdData.Goal.GoalRef},
		})
		var output mcpiface.CommandToolOutput
		decodeMCPOutput(t, result, &output)
		var data struct {
			Goal struct {
				GoalRef string `json:"goal_ref"`
				State   string `json:"state"`
			} `json:"goal"`
			ExecutionCount int `json:"execution_count"`
			ArtifactCount  int `json:"artifact_count"`
		}
		decodeCommandData(t, output.Result, &data)
		if !result.IsError && output.Result.Failure == nil && data.Goal.State == string(goal.GoalStateSucceeded) &&
			data.ExecutionCount == 1 && data.ArtifactCount == 1 {
			closedGoalRef = data.Goal.GoalRef
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	ref, err := goal.NewGoalRef(createdData.Goal.GoalRef)
	if err != nil {
		t.Fatalf("created goal ref: %v", err)
	}
	closed := waitTerminalGoal(t, runtime, ref)
	if closedGoalRef == "" || len(closed.Artifacts) != 1 || len(closed.Attestations) != 1 || launches.Load() != 1 {
		t.Fatalf("closure missing: goal=%+v launches=%d", closed, launches.Load())
	}

	artifactResult := callMCPTool(t, ctx, session, "orquesta.artifacts.read", map[string]any{
		"version": "1", "project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e:artifact",
		"payload": map[string]any{
			"goal_ref": closed.Goal.Ref().String(), "artifact_ref": closed.Artifacts[0].Stored.Ref.String(),
		},
	})
	var artifact mcpiface.CommandToolOutput
	decodeMCPOutput(t, artifactResult, &artifact)
	var artifactData struct {
		ContentBase64 string `json:"content_base64"`
	}
	decodeCommandData(t, artifact.Result, &artifactData)
	if artifactResult.IsError || artifact.Result.Failure != nil || artifactData.ContentBase64 == "" {
		t.Fatalf("artifact read = %+v result=%+v", artifact, artifactResult)
	}
	content, err := base64.StdEncoding.DecodeString(artifactData.ContentBase64)
	if err != nil || !strings.Contains(string(content), "produce API evidence") {
		t.Fatalf("artifact content = %q err=%v", content, err)
	}

	replayResult := callMCPTool(t, ctx, session, "orquesta.goals.create", map[string]any{
		"version": "1", "project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e",
		"payload": map[string]any{"statement": "produce API evidence", "confirm": true},
	})
	var replay mcpiface.CommandToolOutput
	decodeMCPOutput(t, replayResult, &replay)
	var replayData struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	decodeCommandData(t, replay.Result, &replayData)
	if replayResult.IsError || replay.Result.Failure != nil || replay.Result.AuditRef != created.Result.AuditRef ||
		replayData.Goal.GoalRef != closed.Goal.Ref().String() || launches.Load() != 1 {
		t.Fatalf("idempotent replay = %+v launches=%d", replay, launches.Load())
	}

	statusResult := callMCPTool(t, ctx, session, "orquesta.system.status", map[string]any{
		"version": "1", "project_ref": hierarchy.ProjectRef().String(), "request_ref": "request:mcp-e2e:status",
		"payload": map[string]any{},
	})
	var status mcpiface.CommandToolOutput
	decodeMCPOutput(t, statusResult, &status)
	var statusData struct {
		Goals          int64 `json:"goals"`
		RunningGoals   int64 `json:"running_goals"`
		PendingActions int64 `json:"pending_actions"`
	}
	decodeCommandData(t, status.Result, &statusData)
	if statusResult.IsError || status.Result.Failure != nil || statusData.Goals != 1 ||
		statusData.RunningGoals != 0 || statusData.PendingActions != 0 {
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

func decodeCommandData(t *testing.T, result commandcore.Result, target any) {
	t.Helper()
	if result.Failure != nil {
		t.Fatalf("command failure: %+v", result.Failure)
	}
	if err := json.Unmarshal(result.Data, target); err != nil {
		t.Fatalf("decode command data %s: %v", result.Data, err)
	}
}
