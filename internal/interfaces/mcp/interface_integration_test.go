package mcpinterface

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestOfficialClientListsExactToolsAndCallsGoalArtifactAndStatus(t *testing.T) {
	server, state, artifacts := newTestInterface(t, 16*1024)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	session := connectOfficialClient(t, httpServer.URL)

	listed, err := session.ListTools(testContext(t), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	wantNames := []string{ToolArtifactsRead, ToolGoalsAmend, ToolGoalsCreate, ToolGoalsGet, ToolGoalsList, ToolSystemStatus}
	gotNames := make([]string, 0, len(listed.Tools))
	byName := make(map[string]*sdkmcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		gotNames = append(gotNames, tool.Name)
		byName[tool.Name] = tool
	}
	sort.Strings(gotNames)
	if fmt.Sprint(gotNames) != fmt.Sprint(wantNames) {
		t.Fatalf("tools = %v, want %v", gotNames, wantNames)
	}
	for _, name := range []string{ToolGoalsAmend, ToolGoalsCreate} {
		annotations := byName[name].Annotations
		if annotations == nil || annotations.ReadOnlyHint || !annotations.IdempotentHint ||
			annotations.DestructiveHint == nil || *annotations.DestructiveHint ||
			annotations.OpenWorldHint == nil || *annotations.OpenWorldHint {
			t.Fatalf("%s annotations = %+v", name, annotations)
		}
	}
	for _, name := range []string{ToolArtifactsRead, ToolGoalsGet, ToolGoalsList, ToolSystemStatus} {
		annotations := byName[name].Annotations
		if annotations == nil || !annotations.ReadOnlyHint || !annotations.IdempotentHint ||
			annotations.OpenWorldHint == nil || *annotations.OpenWorldHint {
			t.Fatalf("%s annotations = %+v", name, annotations)
		}
	}

	statusResult := callTool(t, session, ToolSystemStatus, map[string]any{"project_ref": "project:local"})
	var status SystemStatusOutput
	decodeStructured(t, statusResult, &status)
	if statusResult.IsError || !status.Ready || status.Version != "test-version" || status.Goals != 0 {
		t.Fatalf("initial status = %+v result=%+v", status, statusResult)
	}

	createArguments := map[string]any{
		"project_ref": "project:local",
		"request_ref": "request:test-1", "statement": "Produce a durable artifact",
		"normalized_objective": "Produce durable artifact", "confirm": true,
	}
	createdResult := callTool(t, session, ToolGoalsCreate, createArguments)
	var created CreateGoalOutput
	decodeStructured(t, createdResult, &created)
	if createdResult.IsError || !created.Created || created.Goal == nil {
		t.Fatalf("create output = %+v result=%+v", created, createdResult)
	}
	if created.Goal.ProjectRef != "project:local" || created.Goal.ActorRef != "actor:local" ||
		created.Goal.State != string(goal.GoalStateRunning) || created.Goal.Revision == 0 ||
		created.Goal.PlanGeneration != 1 || created.Goal.CreatedAt.IsZero() || created.Goal.StartedAt == nil ||
		len(created.Goal.Phases) != 1 || created.Goal.Phases[0].PhaseKey != goal.DefaultPhaseKey().String() ||
		len(created.Goal.WorkItems) != 1 || len(created.Goal.Executions) != 1 || created.Goal.Executions[0].DeadlineAt != nil {
		t.Fatalf("created Goal view = %+v", created.Goal)
	}
	if created.Goal.AppSpec.Ref == "" || created.Goal.AppSpec.Generation != 1 || created.Goal.AppSpec.Hash == "" ||
		created.Goal.AppSpec.ParentRef != "" || created.Goal.AppSpec.ParentHash != "" ||
		created.Goal.AppSpec.Objective != "Produce durable artifact" ||
		created.Goal.AppSpec.ConfirmedBy != "actor:local" || created.Goal.AppSpec.ConfirmedAt != testNow() ||
		created.Goal.AppSpec.Intent.Ref != created.Goal.IntentRef ||
		created.Goal.AppSpec.Intent.Hash != created.Goal.IntentHash ||
		created.Goal.AppSpec.Intent.Statement != created.Goal.Statement ||
		created.Goal.AppSpec.Intent.SubmittedAt != testNow() {
		t.Fatalf("created AppSpec projection = %+v", created.Goal.AppSpec)
	}
	if created.Goal.Phases == nil || created.Goal.Executions == nil || created.Goal.Artifacts == nil ||
		created.Goal.Attestations == nil || created.Goal.WorkItems[0].ArtifactRefs == nil ||
		created.Goal.WorkItems[0].DependencyRefs == nil || created.Goal.WorkItems[0].WriteSet == nil ||
		created.Goal.WorkItems[0].RequiredTests == nil {
		t.Fatalf("evidence arrays are not typed empty arrays: %+v", created.Goal)
	}

	replayedResult := callTool(t, session, ToolGoalsCreate, createArguments)
	var replayed CreateGoalOutput
	decodeStructured(t, replayedResult, &replayed)
	if replayedResult.IsError || replayed.Created || replayed.Goal == nil || replayed.Goal.GoalRef != created.Goal.GoalRef {
		t.Fatalf("idempotent create = %+v result=%+v", replayed, replayedResult)
	}

	getResult := callTool(t, session, ToolGoalsGet, map[string]any{
		"project_ref": "project:local", "goal_ref": created.Goal.GoalRef,
	})
	var got GetGoalOutput
	decodeStructured(t, getResult, &got)
	if getResult.IsError || got.Goal == nil || got.Goal.IntentHash == "" || got.Goal.GoalRef != created.Goal.GoalRef {
		t.Fatalf("get output = %+v result=%+v", got, getResult)
	}

	listResult := callTool(t, session, ToolGoalsList, map[string]any{"project_ref": "project:local", "limit": 1})
	var list ListGoalsOutput
	decodeStructured(t, listResult, &list)
	if listResult.IsError || list.Count != 1 || len(list.Goals) != 1 || list.Goals[0].GoalRef != created.Goal.GoalRef ||
		list.Goals[0].AppSpecRef != created.Goal.AppSpec.Ref || list.Goals[0].AppSpecGeneration != 1 ||
		list.Goals[0].SpecHash != created.Goal.AppSpec.Hash {
		t.Fatalf("list output = %+v result=%+v", list, listResult)
	}

	goalRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
	content := []byte("artifact body")
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	artifactRef, _ := goal.NewArtifactRef("artifact:sha256:" + digestText)
	state.attachArtifact(t, goalRef, application.ArtifactRecord{
		Stored: ports.StoredArtifact{
			Ref: artifactRef, Digest: digestText, MediaType: "text/plain", Size: int64(len(content)),
		},
		CreatedAt: testNow().Add(time.Second),
	})
	artifacts.set(ports.ArtifactContent{
		Ref: artifactRef, Digest: digestText, Size: int64(len(content)), Content: content,
	})
	artifactResult := callTool(t, session, ToolArtifactsRead, map[string]any{
		"project_ref": "project:local", "goal_ref": created.Goal.GoalRef, "artifact_ref": artifactRef.String(),
	})
	var artifact ReadArtifactOutput
	decodeStructured(t, artifactResult, &artifact)
	if artifactResult.IsError || artifact.Artifact == nil || artifact.Artifact.MediaType != "text/plain" ||
		artifact.Artifact.ContentBase64 != base64.StdEncoding.EncodeToString(content) {
		t.Fatalf("artifact output = %+v result=%+v", artifact, artifactResult)
	}

	finalStatusResult := callTool(t, session, ToolSystemStatus, map[string]any{"project_ref": "project:local"})
	var finalStatus SystemStatusOutput
	decodeStructured(t, finalStatusResult, &finalStatus)
	if finalStatus.Goals != 1 || finalStatus.RunningGoals != 1 || finalStatus.PendingActions != 1 {
		t.Fatalf("final status = %+v", finalStatus)
	}
}

func TestToolErrorsAreTypedLocalizedAndDoNotLeakApplicationErrors(t *testing.T) {
	server, _, _ := newTestInterface(t, 16*1024)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	session := connectOfficialClient(t, httpServer.URL)

	result := callTool(t, session, ToolGoalsGet, map[string]any{
		"project_ref": "project:local", "goal_ref": "goal:missing",
	})
	var output GetGoalOutput
	decodeStructured(t, result, &output)
	if !result.IsError || output.Error == nil || output.Error.Code != publicNotFound {
		t.Fatalf("error output = %+v result=%+v", output, result)
	}
	if output.Error.Message != "No se encontró el recurso solicitado." {
		t.Fatalf("localized message = %q", output.Error.Message)
	}
	contentJSON, err := json.Marshal(result.Content)
	if err != nil {
		t.Fatalf("Marshal(content) error = %v", err)
	}
	if strings.Contains(string(contentJSON), "state.not_found") || strings.Contains(string(contentJSON), "application.") {
		t.Fatalf("internal error leaked: %s", contentJSON)
	}

	invalidLimit := callTool(t, session, ToolGoalsList, map[string]any{"project_ref": "project:local", "limit": 999})
	var invalid ListGoalsOutput
	decodeStructured(t, invalidLimit, &invalid)
	if !invalidLimit.IsError || invalid.Error == nil || invalid.Error.Code != publicInvalidRequest {
		t.Fatalf("invalid limit output = %+v result=%+v", invalid, invalidLimit)
	}

	request := map[string]any{
		"project_ref": "project:local", "request_ref": "request:conflict",
		"statement": "first statement", "confirm": true,
	}
	if result := callTool(t, session, ToolGoalsCreate, request); result.IsError {
		t.Fatalf("initial conflict fixture failed: %+v", result)
	}
	request["statement"] = "different statement"
	conflictResult := callTool(t, session, ToolGoalsCreate, request)
	var conflict CreateGoalOutput
	decodeStructured(t, conflictResult, &conflict)
	if !conflictResult.IsError || conflict.Error == nil || conflict.Error.Code != publicConflict {
		t.Fatalf("conflict output = %+v result=%+v", conflict, conflictResult)
	}
}

func TestV10MissingOrInvalidRequestPrincipalFailsBeforeProjectState(t *testing.T) {
	_, state, _ := newTestInterface(t, 16*1024)
	tests := []struct {
		name     string
		provider identity.Provider
	}{
		{name: "missing context principal", provider: identity.ContextProvider{}},
		{name: "invalid provider principal", provider: staticIdentityProvider{principal: identity.Principal{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newTestInterfaceForIdentity(t, state.orchestrator, test.provider, 16*1024)
			session := serveOfficialClient(t, server)
			beforeGoals, beforeRequests, beforeSuccessors, beforePending := state.counts()
			result := callTool(t, session, ToolGoalsList, map[string]any{
				"project_ref": "project:local", "limit": 1,
			})
			var output ListGoalsOutput
			decodeStructured(t, result, &output)
			if !result.IsError || output.Error == nil || output.Error.Code != publicInternal {
				t.Fatalf("missing principal output=%+v result=%+v", output, result)
			}
			afterGoals, afterRequests, afterSuccessors, afterPending := state.counts()
			if afterGoals != beforeGoals || afterRequests != beforeRequests ||
				afterSuccessors != beforeSuccessors || afterPending != beforePending {
				t.Fatal("missing principal changed project state")
			}
		})
	}
}

func TestV10MCPProjectIsolationAndForbiddenAreStable(t *testing.T) {
	localServer, state, _ := newTestInterface(t, 16*1024)
	local := serveOfficialClient(t, localServer)
	createdResult := callTool(t, local, ToolGoalsCreate, map[string]any{
		"project_ref": "project:local", "request_ref": "request:v10-isolation",
		"statement": "private project work", "confirm": true,
	})
	var created CreateGoalOutput
	decodeStructured(t, createdResult, &created)
	if createdResult.IsError || created.Goal == nil {
		t.Fatalf("isolation fixture = %+v result=%+v", created, createdResult)
	}

	t.Run("same principal cannot cross an explicit project scope", func(t *testing.T) {
		otherProjectServer := newTestInterfaceForPrincipal(
			t, state, "actor:local", "project:other", 16*1024,
		)
		otherProject := serveOfficialClient(t, otherProjectServer)
		assertPublicToolError(t, callTool(t, otherProject, ToolGoalsGet, map[string]any{
			"project_ref": "project:other", "goal_ref": created.Goal.GoalRef,
		}), publicNotFound)
		assertPublicToolError(t, callTool(t, otherProject, ToolArtifactsRead, map[string]any{
			"project_ref": "project:other", "goal_ref": created.Goal.GoalRef,
			"artifact_ref": "artifact:hidden",
		}), publicNotFound)
		listResult := callTool(t, otherProject, ToolGoalsList, map[string]any{
			"project_ref": "project:other", "limit": 10,
		})
		var list ListGoalsOutput
		decodeStructured(t, listResult, &list)
		if listResult.IsError || list.Count != 0 || len(list.Goals) != 0 {
			t.Fatalf("other project leaked goals: %+v result=%+v", list, listResult)
		}
		statusResult := callTool(t, otherProject, ToolSystemStatus, map[string]any{
			"project_ref": "project:other",
		})
		var status SystemStatusOutput
		decodeStructured(t, statusResult, &status)
		if statusResult.IsError || status.Goals != 0 || status.PendingActions != 0 {
			t.Fatalf("other project leaked status: %+v result=%+v", status, statusResult)
		}
	})

	t.Run("principal without project membership sees not found", func(t *testing.T) {
		outsiderServer := newTestInterfaceForPrincipal(
			t, state, "actor:outsider", "project:other", 16*1024,
		)
		outsider := serveOfficialClient(t, outsiderServer)
		beforeGoals, beforeRequests, beforeSuccessors, beforePending := state.counts()
		assertPublicToolError(t, callTool(t, outsider, ToolGoalsGet, map[string]any{
			"project_ref": "project:local", "goal_ref": created.Goal.GoalRef,
		}), publicNotFound)
		assertPublicToolError(t, callTool(t, outsider, ToolArtifactsRead, map[string]any{
			"project_ref": "project:local", "goal_ref": created.Goal.GoalRef,
			"artifact_ref": "artifact:hidden",
		}), publicNotFound)
		assertPublicToolError(t, callTool(t, outsider, ToolGoalsCreate, map[string]any{
			"project_ref": "project:local", "request_ref": "request:v10-cross-project-write",
			"statement": "must not be created", "confirm": true,
		}), publicForbidden)
		afterGoals, afterRequests, afterSuccessors, afterPending := state.counts()
		if afterGoals != beforeGoals || afterRequests != beforeRequests ||
			afterSuccessors != beforeSuccessors || afterPending != beforePending {
			t.Fatal("cross-project request changed durable state")
		}
	})

	t.Run("known member without write permission receives localized forbidden", func(t *testing.T) {
		viewerServer := newTestInterfaceForPrincipalRole(
			t, state, "actor:viewer", "project:local", identity.RoleViewer, 16*1024,
		)
		viewer := serveOfficialClient(t, viewerServer)
		result := callTool(t, viewer, ToolGoalsCreate, map[string]any{
			"project_ref": "project:local", "request_ref": "request:v10-viewer-write",
			"statement": "must be forbidden", "confirm": true,
		})
		var output CreateGoalOutput
		decodeStructured(t, result, &output)
		if !result.IsError || output.Error == nil || output.Error.Code != publicForbidden ||
			output.Error.Message != "No tiene permiso para realizar esta operación." {
			t.Fatalf("forbidden output=%+v result=%+v", output, result)
		}
	})
}

func TestFailedGoalProjectsStableFailureCodeWithoutProviderDiagnostic(t *testing.T) {
	server, state, _ := newTestInterface(t, 16*1024)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	session := connectOfficialClient(t, httpServer.URL)

	createdResult := callTool(t, session, ToolGoalsCreate, map[string]any{
		"project_ref": "project:local", "request_ref": "request:failed-view",
		"statement": "fail safely", "confirm": true,
	})
	var created CreateGoalOutput
	decodeStructured(t, createdResult, &created)
	if createdResult.IsError || created.Goal == nil {
		t.Fatalf("create = %+v result=%+v", created, createdResult)
	}
	goalRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
	state.failGoal(t, goalRef, "provider.stable_failure")

	result := callTool(t, session, ToolGoalsGet, map[string]any{
		"project_ref": "project:local", "goal_ref": goalRef.String(),
	})
	var output GetGoalOutput
	decodeStructured(t, result, &output)
	if result.IsError || output.Goal == nil || output.Goal.State != string(goal.GoalStateFailed) ||
		output.Goal.Executions[0].State != string(application.ExecutionFailed) ||
		output.Goal.Executions[0].FailureCode != "provider.stable_failure" ||
		len(output.Goal.Artifacts) != 0 || len(output.Goal.Attestations) != 0 {
		t.Fatalf("failed Goal view = %+v result=%+v", output, result)
	}
	payload, err := json.Marshal(output)
	if err != nil || strings.Contains(string(payload), "stderr") {
		t.Fatalf("failed Goal leaked diagnostic: %s err=%v", payload, err)
	}
}

func TestHTTPHandlerEnforcesCrossOriginLocalhostAndBodyProtections(t *testing.T) {
	server, _, _ := newTestInterface(t, 512)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`

	t.Run("cross origin", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, httpServer.URL, strings.NewReader(initialize))
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://evil.example")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", response.StatusCode)
		}
	})

	t.Run("localhost rebinding", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, httpServer.URL, strings.NewReader(initialize))
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		request.Host = "evil.example"
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", response.StatusCode)
		}
	})

	t.Run("body limit", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, httpServer.URL, strings.NewReader(strings.Repeat("x", 513)))
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", response.StatusCode)
		}
	})
}
