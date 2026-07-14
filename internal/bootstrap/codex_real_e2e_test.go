package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/config"
	"orquesta/internal/goal"
	mcpiface "orquesta/internal/interfaces/mcp"
)

var realCodexConfig = flag.String(
	"orquesta-real-codex-config",
	"",
	"opt-in TOML config for the real Codex MCP end-to-end test",
)

func TestRealCodexAdapterClosesGoalThroughProductionMCPServer(t *testing.T) {
	if *realCodexConfig == "" {
		t.Skip("real Codex E2E is opt-in")
	}
	snapshot, err := config.Load(config.LoadOptions{FilePath: *realCodexConfig})
	if err != nil {
		t.Fatalf("load production config: %v", err)
	}
	runtime, err := Build(context.Background(), Options{ConfigPath: *realCodexConfig, Version: "codex-real-e2e"})
	if err != nil {
		t.Fatalf("build production runtime: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start production runtime: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "codex-real-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, snapshot.Identity.LocalTokenPath),
	}, nil)
	if err != nil {
		t.Fatalf("connect official MCP client: %v", err)
	}
	defer session.Close()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatalf("generate real E2E nonce: %v", err)
	}
	suffix := hex.EncodeToString(nonce[:])
	requestRef := "request:codex-real-e2e:" + suffix
	marker := "ORQUESTA_CODEX_E2E_OK_" + suffix
	t.Logf("real Codex evidence request_ref=%s marker=%s", requestRef, marker)
	createdResult := callMCPTool(t, ctx, session, mcpiface.ToolGoalsCreate, map[string]any{
		"request_ref": requestRef,
		"statement":   "Produce un artefacto de texto que contenga exactamente el marcador " + marker + ".",
		"confirm":     true,
	})
	var created mcpiface.CreateGoalOutput
	decodeMCPOutput(t, createdResult, &created)
	if createdResult.IsError || !created.Created || created.Goal == nil {
		t.Fatalf("create real Codex Goal: %+v result=%+v", created, createdResult)
	}

	var closed mcpiface.GoalView
	for closed.GoalRef == "" {
		select {
		case <-ctx.Done():
			t.Fatalf("wait real Codex closure: %v", ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
		result := callMCPTool(t, ctx, session, mcpiface.ToolGoalsGet, map[string]any{"goal_ref": created.Goal.GoalRef})
		var output mcpiface.GetGoalOutput
		decodeMCPOutput(t, result, &output)
		if result.IsError || output.Goal == nil {
			t.Fatalf("get real Codex Goal: %+v result=%+v", output, result)
		}
		if output.Goal.State == string(goal.GoalStateFailed) {
			t.Fatalf("real Codex Goal failed: %+v", output.Goal.Executions)
		}
		if output.Goal.State == string(goal.GoalStateSucceeded) {
			closed = *output.Goal
		}
	}
	if len(closed.Artifacts) != 1 || len(closed.Attestations) != 1 {
		t.Fatalf("real closure lacks evidence: %+v", closed)
	}
	artifactResult := callMCPTool(t, ctx, session, mcpiface.ToolArtifactsRead, map[string]any{
		"goal_ref": closed.GoalRef, "artifact_ref": closed.Artifacts[0].ArtifactRef,
	})
	var artifact mcpiface.ReadArtifactOutput
	decodeMCPOutput(t, artifactResult, &artifact)
	if artifactResult.IsError || artifact.Artifact == nil {
		t.Fatalf("read real artifact: %+v result=%+v", artifact, artifactResult)
	}
	content, err := base64.StdEncoding.DecodeString(artifact.Artifact.ContentBase64)
	if err != nil || !strings.Contains(string(content), marker) {
		t.Fatalf("real artifact missing marker: content=%q err=%v", content, err)
	}
}
