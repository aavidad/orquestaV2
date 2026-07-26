package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/goal"
	mcpiface "orquesta/internal/interfaces/mcp"
)

var realCodexConfig = flag.String(
	"orquesta-real-codex-config",
	"",
	"opt-in TOML config for the real Codex MCP end-to-end test",
)

type realCodexGoalSnapshot struct {
	Goal struct {
		State string `json:"state"`
	} `json:"goal"`
	WorkItems []struct {
		Ref           string `json:"work_item_ref"`
		State         string `json:"state"`
		ExecutionRef  string `json:"execution_ref"`
		InterruptCode string `json:"interrupt_code"`
	} `json:"work_items"`
	Executions []struct {
		Ref         string `json:"execution_ref"`
		AttemptNo   int    `json:"attempt_no"`
		MaxAttempts int    `json:"max_attempts"`
		State       string `json:"state"`
		FailureCode string `json:"failure_code"`
	} `json:"executions"`
}

func TestRealCodexAdapterClosesGoalThroughProductionMCPServer(t *testing.T) {
	if *realCodexConfig == "" {
		t.Skip("real Codex E2E is opt-in")
	}
	snapshot, err := loadConfigSnapshot(context.Background(), *realCodexConfig)
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
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, snapshot.IdentityLocalTokenPath()),
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
	createdResult := callMCPTool(t, ctx, session, "orquesta.goals.create", map[string]any{
		"version": "1", "project_ref": snapshot.ProjectDefault(), "request_ref": requestRef,
		"payload": map[string]any{
			"statement": "Produce un artefacto de texto que contenga exactamente el marcador " + marker + ".",
			"confirm":   true,
		},
	})
	var created mcpiface.CommandToolOutput
	decodeMCPOutput(t, createdResult, &created)
	var createdData struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	decodeCommandData(t, created.Result, &createdData)
	if createdResult.IsError || created.Result.Failure != nil || createdData.Goal.GoalRef == "" {
		t.Fatalf("create real Codex Goal: %+v result=%+v", created, createdResult)
	}

	closed := false
	poll := 0
	var lastSnapshot realCodexGoalSnapshot
	for !closed {
		select {
		case <-ctx.Done():
			t.Fatalf("wait real Codex closure: %v snapshot=%+v", ctx.Err(), lastSnapshot)
		case <-time.After(250 * time.Millisecond):
		}
		poll++
		result := callMCPTool(t, ctx, session, "orquesta.goals.get", map[string]any{
			"version": "1", "project_ref": snapshot.ProjectDefault(),
			"request_ref": fmt.Sprintf("%s:get:%d", requestRef, poll),
			"payload":     map[string]any{"goal_ref": createdData.Goal.GoalRef},
		})
		var output mcpiface.CommandToolOutput
		decodeMCPOutput(t, result, &output)
		decodeCommandData(t, output.Result, &lastSnapshot)
		if result.IsError || output.Result.Failure != nil {
			t.Fatalf("get real Codex Goal: %+v result=%+v", output, result)
		}
		switch lastSnapshot.Goal.State {
		case string(goal.GoalStateFailed), string(goal.GoalStateCanceled):
			t.Fatalf("real Codex Goal terminated without success: %+v", lastSnapshot)
		}
		closed = lastSnapshot.Goal.State == string(goal.GoalStateSucceeded)
	}
	goalRef, err := goal.NewGoalRef(createdData.Goal.GoalRef)
	if err != nil {
		t.Fatalf("parse real Codex Goal ref: %v", err)
	}
	record, err := runtime.Orchestrator().GetGoal(ctx, testRuntimeAccess(t, runtime), goalRef)
	if err != nil || len(record.Artifacts) != 1 || len(record.Attestations) != 1 {
		t.Fatalf("real closure lacks evidence: record=%+v err=%v", record, err)
	}
	artifactResult := callMCPTool(t, ctx, session, "orquesta.artifacts.read", map[string]any{
		"version": "1", "project_ref": snapshot.ProjectDefault(), "request_ref": requestRef + ":artifact",
		"payload": map[string]any{
			"goal_ref": goalRef.String(), "artifact_ref": record.Artifacts[0].Stored.Ref.String(),
		},
	})
	var artifact mcpiface.CommandToolOutput
	decodeMCPOutput(t, artifactResult, &artifact)
	var artifactData struct {
		ContentBase64 string `json:"content_base64"`
	}
	decodeCommandData(t, artifact.Result, &artifactData)
	if artifactResult.IsError || artifact.Result.Failure != nil || artifactData.ContentBase64 == "" {
		t.Fatalf("read real artifact: %+v result=%+v", artifact, artifactResult)
	}
	content, err := base64.StdEncoding.DecodeString(artifactData.ContentBase64)
	if err != nil || !strings.Contains(string(content), marker) {
		t.Fatalf("real artifact missing marker: content=%q err=%v", content, err)
	}
}
