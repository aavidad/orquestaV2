package mcpinterface

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/i18n"
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

	statusResult := callTool(t, session, ToolSystemStatus, map[string]any{})
	var status SystemStatusOutput
	decodeStructured(t, statusResult, &status)
	if statusResult.IsError || !status.Ready || status.Version != "test-version" || status.Goals != 0 {
		t.Fatalf("initial status = %+v result=%+v", status, statusResult)
	}

	createArguments := map[string]any{
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
		created.Goal.WorkItems[0].DependencyRefs == nil || created.Goal.WorkItems[0].WriteSet == nil {
		t.Fatalf("evidence arrays are not typed empty arrays: %+v", created.Goal)
	}

	replayedResult := callTool(t, session, ToolGoalsCreate, createArguments)
	var replayed CreateGoalOutput
	decodeStructured(t, replayedResult, &replayed)
	if replayedResult.IsError || replayed.Created || replayed.Goal == nil || replayed.Goal.GoalRef != created.Goal.GoalRef {
		t.Fatalf("idempotent create = %+v result=%+v", replayed, replayedResult)
	}

	getResult := callTool(t, session, ToolGoalsGet, map[string]any{"goal_ref": created.Goal.GoalRef})
	var got GetGoalOutput
	decodeStructured(t, getResult, &got)
	if getResult.IsError || got.Goal == nil || got.Goal.IntentHash == "" || got.Goal.GoalRef != created.Goal.GoalRef {
		t.Fatalf("get output = %+v result=%+v", got, getResult)
	}

	listResult := callTool(t, session, ToolGoalsList, map[string]any{"limit": 1})
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
		"goal_ref": created.Goal.GoalRef, "artifact_ref": artifactRef.String(),
	})
	var artifact ReadArtifactOutput
	decodeStructured(t, artifactResult, &artifact)
	if artifactResult.IsError || artifact.Artifact == nil || artifact.Artifact.MediaType != "text/plain" ||
		artifact.Artifact.ContentBase64 != base64.StdEncoding.EncodeToString(content) {
		t.Fatalf("artifact output = %+v result=%+v", artifact, artifactResult)
	}

	finalStatusResult := callTool(t, session, ToolSystemStatus, map[string]any{})
	var finalStatus SystemStatusOutput
	decodeStructured(t, finalStatusResult, &finalStatus)
	if finalStatus.Goals != 1 || finalStatus.RunningGoals != 1 || finalStatus.PendingActions != 1 {
		t.Fatalf("final status = %+v", finalStatus)
	}
}

func TestV04OfficialMCPToolsAreExactClosedWorldAndSafelyAnnotated(t *testing.T) {
	server, _, _ := newTestInterface(t, 16*1024)
	session := serveOfficialClient(t, server)
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
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("tools = %v, want %v", gotNames, wantNames)
	}
	assertClosedWorldProperties(t, byName[ToolGoalsCreate], []string{
		"confirm", "normalized_objective", "plan", "request_ref", "statement",
	})
	assertClosedWorldProperties(t, byName[ToolGoalsAmend], []string{
		"confirm", "expected_source_revision", "expected_source_spec_hash", "normalized_objective",
		"reason", "request_ref", "source_goal_ref", "statement",
	})
	assertClosedWorldProperties(t, byName[ToolGoalsGet], []string{"goal_ref"})
	assertClosedWorldProperties(t, byName[ToolGoalsList], []string{"limit"})
	assertClosedWorldProperties(t, byName[ToolArtifactsRead], []string{"artifact_ref", "goal_ref"})
	assertClosedWorldProperties(t, byName[ToolSystemStatus], []string{})
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
}

func TestV04OfficialMCPCreateAndAmendAreCausalIdempotentAndServerOwned(t *testing.T) {
	server, state, _ := newTestInterface(t, 16*1024)
	session := serveOfficialClient(t, server)
	statement := "  Preserve exact operator intent.  "
	createArguments := map[string]any{
		"request_ref": "request:v04-create", "statement": statement,
		"normalized_objective": "Build maintained output", "confirm": true,
	}
	created := callCreateGoal(t, session, createArguments)
	_, appSpecRefErr := goal.NewAppSpecRef(created.Goal.AppSpec.Ref)
	_, intentRefErr := goal.NewIntentRef(created.Goal.AppSpec.Intent.Ref)
	if !created.Created || created.Goal == nil || created.Goal.AppSpec.Generation != 1 ||
		appSpecRefErr != nil || intentRefErr != nil || !goal.IsCanonicalAppSpecHash(created.Goal.AppSpec.Hash) ||
		!canonicalSHA256(created.Goal.AppSpec.Intent.Hash) ||
		created.Goal.AppSpec.Intent.Statement != statement || created.Goal.AppSpec.Objective != "Build maintained output" ||
		created.Goal.AppSpec.Reason != "operator.initial_confirmation" ||
		created.Goal.ActorRef != "actor:local" || created.Goal.ProjectRef != "project:local" ||
		created.Goal.AppSpec.ConfirmedBy != "actor:local" || created.Goal.AppSpec.ConfirmedAt != testNow() ||
		created.Goal.AppSpec.Intent.ActorRef != created.Goal.ActorRef ||
		created.Goal.AppSpec.Intent.ProjectRef != created.Goal.ProjectRef ||
		created.Goal.AppSpec.Intent.SubmittedAt != testNow() || created.Goal.IntentRef != created.Goal.AppSpec.Intent.Ref ||
		created.Goal.IntentHash != created.Goal.AppSpec.Intent.Hash || created.Goal.Statement != statement {
		t.Fatalf("created Goal = %+v", created)
	}
	replayedCreate := callCreateGoal(t, session, createArguments)
	if replayedCreate.Created || replayedCreate.Goal == nil || replayedCreate.Goal.GoalRef != created.Goal.GoalRef {
		t.Fatalf("create replay = %+v", replayedCreate)
	}
	createConflict := cloneArguments(createArguments)
	createConflict["statement"] = "different exact intent"
	assertPublicToolError(t, callTool(t, session, ToolGoalsCreate, createConflict), publicConflict)

	sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
	state.failGoal(t, sourceRef, "test.terminal_source")
	sourceBefore, err := state.GetGoal(context.Background(), sourceRef)
	if err != nil {
		t.Fatalf("GetGoal(source) error = %v", err)
	}
	amendAt := state.clock.Now()
	amendArguments := amendmentToolArguments(sourceBefore, "request:v04-amend")
	amendArguments["statement"] = "  Exact amended intent.  "
	amendArguments["normalized_objective"] = "  Add export safely  "
	amendArguments["reason"] = "operator.scope_changed"
	amended := callAmendGoal(t, session, amendArguments)
	if !amended.Created || amended.Goal == nil || amended.Goal.State != string(goal.GoalStatePending) ||
		amended.Goal.Revision != 1 || amended.Goal.PlanGeneration != 0 || len(amended.Goal.Phases) != 0 ||
		len(amended.Goal.WorkItems) != 0 || len(amended.Goal.Executions) != 0 ||
		len(amended.Goal.Artifacts) != 0 || len(amended.Goal.Attestations) != 0 {
		t.Fatalf("amended Goal shell = %+v", amended)
	}
	appSpec := amended.Goal.AppSpec
	_, amendedSpecRefErr := goal.NewAppSpecRef(appSpec.Ref)
	_, amendedIntentRefErr := goal.NewIntentRef(appSpec.Intent.Ref)
	if amendedSpecRefErr != nil || amendedIntentRefErr != nil || !goal.IsCanonicalAppSpecHash(appSpec.Hash) ||
		!canonicalSHA256(appSpec.Intent.Hash) || appSpec.Ref == sourceBefore.Goal.AppSpec().Ref().String() ||
		appSpec.Intent.Ref == sourceBefore.Goal.Intent().String() || appSpec.Generation != 2 ||
		appSpec.ParentRef != sourceBefore.Goal.AppSpec().Ref().String() ||
		appSpec.ParentHash != sourceBefore.Goal.SpecHash() || appSpec.Hash == sourceBefore.Goal.SpecHash() ||
		appSpec.Objective != "Add export safely" || appSpec.Reason != "operator.scope_changed" ||
		appSpec.ConfirmedBy != "actor:local" || appSpec.ConfirmedAt != amendAt ||
		appSpec.Intent.ActorRef != "actor:local" || appSpec.Intent.ProjectRef != "project:local" ||
		appSpec.Intent.Statement != "  Exact amended intent.  " || appSpec.Intent.SubmittedAt != amendAt ||
		amended.Goal.CreatedAt != amendAt || amended.Goal.IntentRef != appSpec.Intent.Ref ||
		amended.Goal.IntentHash != appSpec.Intent.Hash || amended.Goal.Statement != appSpec.Intent.Statement {
		t.Fatalf("amended AppSpec = %+v goal=%+v", appSpec, amended.Goal)
	}
	if amended.Goal.Phases == nil || amended.Goal.WorkItems == nil || amended.Goal.Executions == nil ||
		amended.Goal.Artifacts == nil || amended.Goal.Attestations == nil {
		t.Fatalf("amended arrays must be typed empty arrays: %+v", amended.Goal)
	}
	sourceAfter, err := state.GetGoal(context.Background(), sourceRef)
	if err != nil || !reflect.DeepEqual(sourceAfter.Goal.Snapshot(), sourceBefore.Goal.Snapshot()) ||
		!reflect.DeepEqual(sourceAfter.Artifacts, sourceBefore.Artifacts) ||
		!reflect.DeepEqual(sourceAfter.Attestations, sourceBefore.Attestations) {
		t.Fatalf("source mutated: before=%+v after=%+v err=%v", sourceBefore, sourceAfter, err)
	}
	getResult := callTool(t, session, ToolGoalsGet, map[string]any{"goal_ref": amended.Goal.GoalRef})
	var got GetGoalOutput
	decodeStructured(t, getResult, &got)
	if getResult.IsError || got.Goal == nil || !reflect.DeepEqual(got.Goal.AppSpec, amended.Goal.AppSpec) ||
		got.Goal.IntentRef != amended.Goal.IntentRef || got.Goal.IntentHash != amended.Goal.IntentHash ||
		got.Goal.Statement != amended.Goal.Statement || got.Goal.ActorRef != amended.Goal.ActorRef ||
		got.Goal.ProjectRef != amended.Goal.ProjectRef {
		t.Fatalf("get successor projection = %+v result=%+v", got, getResult)
	}
	listResult := callTool(t, session, ToolGoalsList, map[string]any{"limit": 10})
	var listed ListGoalsOutput
	decodeStructured(t, listResult, &listed)
	var successorSummary *GoalSummaryView
	for index := range listed.Goals {
		if listed.Goals[index].GoalRef == amended.Goal.GoalRef {
			successorSummary = &listed.Goals[index]
			break
		}
	}
	if listResult.IsError || listed.Count != 2 || successorSummary == nil ||
		successorSummary.AppSpecRef != amended.Goal.AppSpec.Ref ||
		successorSummary.AppSpecGeneration != amended.Goal.AppSpec.Generation ||
		successorSummary.SpecHash != amended.Goal.AppSpec.Hash || successorSummary.IntentRef != amended.Goal.IntentRef {
		t.Fatalf("list successor projection = %+v result=%+v", listed, listResult)
	}
	replayedAmend := callAmendGoal(t, session, amendArguments)
	if replayedAmend.Created || replayedAmend.Goal == nil || replayedAmend.Goal.GoalRef != amended.Goal.GoalRef {
		t.Fatalf("amend replay = %+v", replayedAmend)
	}
	amendConflict := cloneArguments(amendArguments)
	amendConflict["statement"] = "different amendment"
	assertPublicToolError(t, callTool(t, session, ToolGoalsAmend, amendConflict), publicConflict)
	secondBranch := cloneArguments(amendArguments)
	secondBranch["request_ref"] = "request:v04-second-branch"
	assertPublicToolError(t, callTool(t, session, ToolGoalsAmend, secondBranch), publicConflict)
	goals, requests, successors, _ := state.counts()
	if goals != 2 || requests != 2 || successors != 1 {
		t.Fatalf("amend durable counts = goals:%d requests:%d successors:%d", goals, requests, successors)
	}
}

func TestV04OfficialMCPRejectsConfirmationAndAuthoritySpoofWithoutEffects(t *testing.T) {
	server, state, _ := newTestInterface(t, 16*1024)
	session := serveOfficialClient(t, server)
	assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
		return callTool(t, session, ToolGoalsCreate, map[string]any{
			"request_ref": "request:unconfirmed", "statement": "must not exist", "confirm": false,
		})
	}, publicInvalidRequest)
	assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
		return callTool(t, session, ToolGoalsAmend, map[string]any{"confirm": false})
	}, publicInvalidRequest)

	spoofs := map[string]any{
		"actor_ref": "actor:spoof", "project_ref": "project:spoof", "confirmed_by": "actor:spoof",
		"submitted_at": "1999-01-01T00:00:00Z", "confirmed_at": "1999-01-01T00:00:00Z",
		"intent_ref": "intent:spoof", "app_spec_ref": "app-spec:spoof", "goal_ref": "goal:spoof",
	}
	for field, value := range spoofs {
		field, value := field, value
		t.Run("create_"+field, func(t *testing.T) {
			assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
				arguments := map[string]any{
					"request_ref": "request:spoof-create-" + field, "statement": "trusted input", "confirm": true,
					field: value,
				}
				return callTool(t, session, ToolGoalsCreate, arguments)
			}, "")
		})
	}

	created := callCreateGoal(t, session, map[string]any{
		"request_ref": "request:spoof-source", "statement": "terminal source", "confirm": true,
	})
	sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
	state.failGoal(t, sourceRef, "test.terminal_source")
	source, _ := state.GetGoal(context.Background(), sourceRef)
	for field, value := range spoofs {
		field, value := field, value
		t.Run("amend_"+field, func(t *testing.T) {
			assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
				arguments := amendmentToolArguments(source, "request:spoof-amend-"+field)
				arguments[field] = value
				return callTool(t, session, ToolGoalsAmend, arguments)
			}, "")
		})
	}
}

func TestV04OfficialMCPRejectsActiveStaleAndForeignAmendmentBeforeGenerators(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		session := serveOfficialClient(t, server)
		created := callCreateGoal(t, session, map[string]any{
			"request_ref": "request:active-source", "statement": "active source", "confirm": true,
		})
		sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
		source, _ := state.GetGoal(context.Background(), sourceRef)
		assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
			return callTool(t, session, ToolGoalsAmend, amendmentToolArguments(source, "request:active-amend"))
		}, publicConflict)
	})

	for _, testCase := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "stale_revision", mutate: func(input map[string]any) {
			input["expected_source_revision"] = input["expected_source_revision"].(uint64) + 1
		}},
		{name: "stale_hash", mutate: func(input map[string]any) {
			replacement := strings.Repeat("a", 64)
			if input["expected_source_spec_hash"] == replacement {
				replacement = strings.Repeat("b", 64)
			}
			input["expected_source_spec_hash"] = replacement
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, state, _ := newTestInterface(t, 16*1024)
			session := serveOfficialClient(t, server)
			created := callCreateGoal(t, session, map[string]any{
				"request_ref": "request:" + testCase.name + "-source", "statement": "terminal source", "confirm": true,
			})
			sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
			state.failGoal(t, sourceRef, "test.terminal_source")
			source, _ := state.GetGoal(context.Background(), sourceRef)
			input := amendmentToolArguments(source, "request:"+testCase.name+"-amend")
			testCase.mutate(input)
			assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
				return callTool(t, session, ToolGoalsAmend, input)
			}, publicConflict)
		})
	}

	t.Run("foreign", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		localSession := serveOfficialClient(t, server)
		created := callCreateGoal(t, localSession, map[string]any{
			"request_ref": "request:foreign-source", "statement": "terminal source", "confirm": true,
		})
		sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
		state.failGoal(t, sourceRef, "test.terminal_source")
		source, _ := state.GetGoal(context.Background(), sourceRef)
		foreignServer := newTestInterfaceForPrincipal(t, state.orchestrator, "actor:foreign", "project:foreign", 16*1024)
		foreignSession := serveOfficialClient(t, foreignServer)
		assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
			return callTool(t, foreignSession, ToolGoalsAmend, amendmentToolArguments(source, "request:foreign-amend"))
		}, publicNotFound)
	})
}

func TestV04MemoryStateScopesRequestRefsAndSerializesConcurrentSuccessors(t *testing.T) {
	t.Run("request scope", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		local := serveOfficialClient(t, server)
		foreignServer := newTestInterfaceForPrincipal(t, state.orchestrator, "actor:other", "project:other", 16*1024)
		foreign := serveOfficialClient(t, foreignServer)
		arguments := map[string]any{"request_ref": "request:shared", "statement": "same request ref", "confirm": true}
		first := callCreateGoal(t, local, arguments)
		second := callCreateGoal(t, foreign, arguments)
		if !first.Created || !second.Created || first.Goal == nil || second.Goal == nil ||
			first.Goal.GoalRef == second.Goal.GoalRef || first.Goal.ActorRef != "actor:local" ||
			second.Goal.ActorRef != "actor:other" || first.Goal.ProjectRef != "project:local" ||
			second.Goal.ProjectRef != "project:other" {
			t.Fatalf("scoped create: first=%+v second=%+v", first, second)
		}
		goals, requests, _, _ := state.counts()
		if goals != 2 || requests != 2 {
			t.Fatalf("scoped counts = goals:%d requests:%d", goals, requests)
		}
	})

	t.Run("concurrent successor", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		session := serveOfficialClient(t, server)
		created := callCreateGoal(t, session, map[string]any{
			"request_ref": "request:concurrent-source", "statement": "terminal source", "confirm": true,
		})
		sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
		state.failGoal(t, sourceRef, "test.terminal_source")
		source, _ := state.GetGoal(context.Background(), sourceRef)
		secondSession := serveOfficialClient(t, server)
		type outcome struct {
			result *sdkmcp.CallToolResult
			err    error
		}
		start := make(chan struct{})
		outcomes := make(chan outcome, 2)
		calls := []struct {
			session    *sdkmcp.ClientSession
			requestRef string
		}{
			{session: session, requestRef: "request:concurrent-a"},
			{session: secondSession, requestRef: "request:concurrent-b"},
		}
		for _, request := range calls {
			request := request
			go func() {
				<-start
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				arguments := amendmentToolArguments(source, request.requestRef)
				result, err := request.session.CallTool(ctx, &sdkmcp.CallToolParams{
					Name: ToolGoalsAmend, Arguments: arguments,
				})
				outcomes <- outcome{result: result, err: err}
			}()
		}
		close(start)
		createdCount, conflictCount := 0, 0
		for index := 0; index < 2; index++ {
			outcome := <-outcomes
			if outcome.err != nil || outcome.result == nil {
				t.Fatalf("concurrent MCP call error: result=%+v err=%v", outcome.result, outcome.err)
			}
			payload, err := json.Marshal(outcome.result.StructuredContent)
			if err != nil {
				t.Fatalf("marshal concurrent output: %v", err)
			}
			if !outcome.result.IsError {
				var output AmendGoalOutput
				if err := json.Unmarshal(payload, &output); err != nil || !output.Created || output.Goal == nil {
					t.Fatalf("concurrent success output=%+v payload=%s err=%v", output, payload, err)
				}
				createdCount++
				continue
			}
			var output AmendGoalOutput
			if err := json.Unmarshal(payload, &output); err != nil || output.Error == nil || output.Error.Code != publicConflict {
				t.Fatalf("concurrent conflict output=%+v payload=%s err=%v", output, payload, err)
			}
			conflictCount++
		}
		goals, requests, successors, _ := state.counts()
		if createdCount != 1 || conflictCount != 1 || goals != 2 || requests != 2 || successors != 1 {
			t.Fatalf("concurrent successor = created:%d conflicts:%d goals:%d requests:%d successors:%d",
				createdCount, conflictCount, goals, requests, successors)
		}
	})
}

func serveOfficialClient(t *testing.T, server *Interface) *sdkmcp.ClientSession {
	t.Helper()
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return connectOfficialClient(t, httpServer.URL)
}

func assertClosedWorldProperties(t *testing.T, tool *sdkmcp.Tool, expected []string) {
	t.Helper()
	if tool == nil {
		t.Fatal("tool missing")
	}
	payload, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("Marshal(input schema) error = %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(payload, &schema); err != nil {
		t.Fatalf("Unmarshal(input schema=%s) error = %v", payload, err)
	}
	additional, ok := schema["additionalProperties"].(bool)
	if !ok || additional {
		t.Fatalf("input schema is not closed-world: %s", payload)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		if len(expected) == 0 {
			return
		}
		t.Fatalf("input schema properties missing: %s", payload)
	}
	actual := make([]string, 0, len(properties))
	for name := range properties {
		actual = append(actual, name)
	}
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("input properties = %v, want %v", actual, expected)
	}
}

func callCreateGoal(t *testing.T, session *sdkmcp.ClientSession, arguments map[string]any) CreateGoalOutput {
	t.Helper()
	result := callTool(t, session, ToolGoalsCreate, arguments)
	var output CreateGoalOutput
	decodeStructured(t, result, &output)
	if result.IsError || output.Error != nil || output.Goal == nil {
		t.Fatalf("create output=%+v result=%+v", output, result)
	}
	return output
}

func callAmendGoal(t *testing.T, session *sdkmcp.ClientSession, arguments map[string]any) AmendGoalOutput {
	t.Helper()
	result := callTool(t, session, ToolGoalsAmend, arguments)
	var output AmendGoalOutput
	decodeStructured(t, result, &output)
	if result.IsError || output.Error != nil || output.Goal == nil {
		t.Fatalf("amend output=%+v result=%+v", output, result)
	}
	return output
}

func assertPublicToolError(t *testing.T, result *sdkmcp.CallToolResult, code string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("tool result unexpectedly succeeded: %+v", result)
	}
	if code == "" {
		return
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(error output) error = %v", err)
	}
	var output struct {
		Error *ToolError `json:"error"`
	}
	if err := json.Unmarshal(payload, &output); err != nil || output.Error == nil || output.Error.Code != code {
		t.Fatalf("public error = %+v payload=%s err=%v, want %s", output.Error, payload, err, code)
	}
}

func amendmentToolArguments(source application.GoalRecord, requestRef string) map[string]any {
	return map[string]any{
		"request_ref": requestRef, "source_goal_ref": source.Goal.Ref().String(),
		"expected_source_revision":  uint64(source.Goal.Revision()),
		"expected_source_spec_hash": source.Goal.SpecHash(),
		"statement":                 "amended exact intent", "normalized_objective": "amended objective",
		"reason": "operator.amendment", "confirm": true,
	}
}

func cloneArguments(arguments map[string]any) map[string]any {
	clone := make(map[string]any, len(arguments))
	for key, value := range arguments {
		clone[key] = value
	}
	return clone
}

func canonicalSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func assertNoStateOrGeneratorChange(
	t *testing.T,
	state *memoryState,
	operation func() *sdkmcp.CallToolResult,
	wantCode string,
) {
	t.Helper()
	beforeGoals, beforeRequests, beforeSuccessors, beforePending := state.counts()
	beforeIDs := state.ids.Count()
	beforeClock := state.clock.Calls()
	result := operation()
	assertPublicToolError(t, result, wantCode)
	afterGoals, afterRequests, afterSuccessors, afterPending := state.counts()
	if afterGoals != beforeGoals || afterRequests != beforeRequests || afterSuccessors != beforeSuccessors ||
		afterPending != beforePending || state.ids.Count() != beforeIDs || state.clock.Calls() != beforeClock {
		t.Fatalf("rejected request caused effects: state %d/%d/%d/%d -> %d/%d/%d/%d ids %d->%d clock %d->%d",
			beforeGoals, beforeRequests, beforeSuccessors, beforePending,
			afterGoals, afterRequests, afterSuccessors, afterPending,
			beforeIDs, state.ids.Count(), beforeClock, state.clock.Calls())
	}
}

func TestToolErrorsAreTypedLocalizedAndDoNotLeakApplicationErrors(t *testing.T) {
	server, _, _ := newTestInterface(t, 16*1024)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	session := connectOfficialClient(t, httpServer.URL)

	result := callTool(t, session, ToolGoalsGet, map[string]any{"goal_ref": "goal:missing"})
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

	invalidLimit := callTool(t, session, ToolGoalsList, map[string]any{"limit": 999})
	var invalid ListGoalsOutput
	decodeStructured(t, invalidLimit, &invalid)
	if !invalidLimit.IsError || invalid.Error == nil || invalid.Error.Code != publicInvalidRequest {
		t.Fatalf("invalid limit output = %+v result=%+v", invalid, invalidLimit)
	}

	request := map[string]any{"request_ref": "request:conflict", "statement": "first statement", "confirm": true}
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

func TestFailedGoalProjectsStableFailureCodeWithoutProviderDiagnostic(t *testing.T) {
	server, state, _ := newTestInterface(t, 16*1024)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	session := connectOfficialClient(t, httpServer.URL)

	createdResult := callTool(t, session, ToolGoalsCreate, map[string]any{
		"request_ref": "request:failed-view", "statement": "fail safely", "confirm": true,
	})
	var created CreateGoalOutput
	decodeStructured(t, createdResult, &created)
	if createdResult.IsError || created.Goal == nil {
		t.Fatalf("create = %+v result=%+v", created, createdResult)
	}
	goalRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
	state.failGoal(t, goalRef, "provider.stable_failure")

	result := callTool(t, session, ToolGoalsGet, map[string]any{"goal_ref": goalRef.String()})
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

func connectOfficialClient(t *testing.T, endpoint string) *sdkmcp.ClientSession {
	t.Helper()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "1"}, nil)
	session, err := client.Connect(testContext(t), &sdkmcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return session
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, arguments map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(testContext(t), &sdkmcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	return result
}

func decodeStructured(t *testing.T, result *sdkmcp.CallToolResult, target any) {
	t.Helper()
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(structured) error = %v", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("Unmarshal(structured=%s) error = %v", payload, err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func newTestInterface(t *testing.T, maxRequestBytes int64) (*Interface, *memoryState, *memoryArtifacts) {
	t.Helper()
	state := &memoryState{
		byRef:        make(map[goal.GoalRef]application.GoalRecord),
		byRequest:    make(map[memoryRequestKey]goal.GoalRef),
		byParentSpec: make(map[goal.AppSpecRef]goal.GoalRef),
	}
	clock := &testClock{now: testNow()}
	ids := &sequentialIDs{}
	state.clock = clock
	state.ids = ids
	artifacts := &memoryArtifacts{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
	orchestrator, err := application.New(application.Dependencies{
		State: state, Launcher: inertAgent{}, Observer: inertAgent{}, Artifacts: artifacts,
		Clock: clock, IDs: ids, MaxOutputBytes: 4096,
		MaxActionAttempts: 3, ClaimLease: time.Minute, ObservationDelay: time.Second,
		ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("application.New() error = %v", err)
	}
	state.orchestrator = orchestrator
	server := newTestInterfaceForPrincipal(t, orchestrator, "actor:local", "project:local", maxRequestBytes)
	return server, state, artifacts
}

func newTestInterfaceForPrincipal(
	t *testing.T,
	orchestrator *application.Orchestrator,
	actor string,
	project string,
	maxRequestBytes int64,
) *Interface {
	t.Helper()
	actorRef, _ := goal.NewActorRef(actor)
	projectRef, _ := goal.NewProjectRef(project)
	provider, err := identity.NewLocalOwnerProvider(actorRef, projectRef)
	if err != nil {
		t.Fatalf("NewLocalOwnerProvider() error = %v", err)
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v", err)
	}
	server, err := New(Config{
		Orchestrator: orchestrator, Identity: provider, Catalog: catalog, Locale: "es",
		MaxListLimit: 10, MaxRequestBytes: maxRequestBytes, Version: "test-version",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server
}

type testClock struct {
	mu    sync.Mutex
	now   time.Time
	calls uint64
}

func (clock *testClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.calls++
	return clock.now
}

func (clock *testClock) Set(now time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = now.UTC()
}

func (clock *testClock) Calls() uint64 {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.calls
}

func testNow() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) }

type sequentialIDs struct {
	mu   sync.Mutex
	next uint64
}

func (ids *sequentialIDs) NewID(_ context.Context, namespace string) (string, error) {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:test-%d", namespace, ids.next), nil
}

func (ids *sequentialIDs) Count() uint64 {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	return ids.next
}

type inertAgent struct{}

func (inertAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:test"}, nil
}

func (inertAgent) Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return ports.AgentLaunchReceipt{}, errors.New("test.agent_not_used")
}

func (inertAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, errors.New("test.agent_not_used")
}

type memoryArtifacts struct {
	mu      sync.Mutex
	content map[goal.ArtifactRef]ports.ArtifactContent
}

func (store *memoryArtifacts) Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	return ports.StoredArtifact{}, errors.New("test.artifact_put_not_used")
}

func (store *memoryArtifacts) Get(_ context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content, found := store.content[ref]
	if !found {
		return ports.ArtifactContent{}, errors.New("artifact.not_found")
	}
	if content.Size != expectedSize {
		return ports.ArtifactContent{}, errors.New("artifact.size_mismatch")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}

func (store *memoryArtifacts) set(content ports.ArtifactContent) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content.Content = append([]byte(nil), content.Content...)
	store.content[content.Ref] = content
}

type memoryState struct {
	mu             sync.Mutex
	byRef          map[goal.GoalRef]application.GoalRecord
	byRequest      map[memoryRequestKey]goal.GoalRef
	byParentSpec   map[goal.AppSpecRef]goal.GoalRef
	pendingActions int64
	orchestrator   *application.Orchestrator
	clock          *testClock
	ids            *sequentialIDs
}

type memoryRequestKey struct {
	actor   goal.ActorRef
	project goal.ProjectRef
	ref     string
}

func requestKey(actor goal.ActorRef, project goal.ProjectRef, ref string) memoryRequestKey {
	return memoryRequestKey{actor: actor, project: project, ref: ref}
}

func (state *memoryState) CreateGoal(_ context.Context, input application.CreateGoalState) (application.GoalRecord, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	key := requestKey(input.Goal.Actor(), input.Goal.Project(), input.RequestRef)
	if existingRef, found := state.byRequest[key]; found {
		existing := state.byRef[existingRef]
		if existing.RequestFingerprint != input.RequestFingerprint {
			return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
		}
		return cloneGoalRecord(existing), false, nil
	}
	record := application.GoalRecord{
		RequestRef: input.RequestRef, RequestFingerprint: input.RequestFingerprint,
		Goal:       input.Goal,
		Executions: append([]application.ExecutionRecord(nil), input.Executions...),
	}
	if _, exists := state.byRef[input.Goal.Ref()]; exists {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
	}
	state.byRef[input.Goal.Ref()] = record
	state.byRequest[key] = input.Goal.Ref()
	state.pendingActions += int64(len(input.Actions))
	return cloneGoalRecord(record), true, nil
}

func (state *memoryState) AmendGoal(_ context.Context, input application.AmendGoalState) (application.GoalRecord, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	key := requestKey(input.ActorRef, input.ProjectRef, input.RequestRef)
	if existingRef, found := state.byRequest[key]; found {
		existing, exists := state.byRef[existingRef]
		if !exists {
			return application.GoalRecord{}, false, &application.StateError{Code: application.StateInvalid}
		}
		if existing.RequestFingerprint != input.RequestFingerprint {
			return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
		}
		return cloneGoalRecord(existing), false, nil
	}
	source, found := state.byRef[input.SourceGoalRef]
	if !found || source.Goal.Actor() != input.ActorRef || source.Goal.Project() != input.ProjectRef {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateNotFound}
	}
	if source.Goal.Revision() != input.ExpectedSourceRevision || source.Goal.SpecHash() != input.ExpectedSourceSpecHash {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
	}
	if !source.Goal.IsTerminal() {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
	}
	successor := input.Successor
	spec := successor.AppSpec()
	parentRef, hasParent := spec.ParentRef()
	if successor.Actor() != input.ActorRef || successor.Project() != input.ProjectRef ||
		successor.State() != goal.GoalStatePending || successor.WorkItemCount() != 0 || !hasParent ||
		parentRef != source.Goal.AppSpec().Ref() || spec.ParentHash() != source.Goal.SpecHash() ||
		spec.Generation() != source.Goal.AppSpec().Generation()+1 {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateInvalid}
	}
	if _, exists := state.byParentSpec[source.Goal.AppSpec().Ref()]; exists {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
	}
	if _, exists := state.byRef[successor.Ref()]; exists {
		return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
	}
	record := application.GoalRecord{
		RequestRef: input.RequestRef, RequestFingerprint: input.RequestFingerprint, Goal: successor,
	}
	state.byRef[successor.Ref()] = record
	state.byRequest[key] = successor.Ref()
	state.byParentSpec[source.Goal.AppSpec().Ref()] = successor.Ref()
	return cloneGoalRecord(record), true, nil
}

func (state *memoryState) GetGoal(_ context.Context, ref goal.GoalRef) (application.GoalRecord, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	record, found := state.byRef[ref]
	if !found {
		return application.GoalRecord{}, &application.StateError{Code: application.StateNotFound}
	}
	return cloneGoalRecord(record), nil
}

func (state *memoryState) ListGoals(_ context.Context, actorRef goal.ActorRef, projectRef goal.ProjectRef, limit int) ([]application.GoalSummary, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	refs := make([]string, 0, len(state.byRef))
	byString := make(map[string]application.GoalRecord, len(state.byRef))
	for _, record := range state.byRef {
		if record.Goal.Actor() == actorRef && record.Goal.Project() == projectRef {
			ref := record.Goal.Ref().String()
			refs = append(refs, ref)
			byString[ref] = record
		}
	}
	sort.Strings(refs)
	if len(refs) > limit {
		refs = refs[:limit]
	}
	result := make([]application.GoalSummary, 0, len(refs))
	for _, ref := range refs {
		record := byString[ref]
		appSpec := record.Goal.AppSpec()
		result = append(result, application.GoalSummary{
			Ref: record.Goal.Ref(), IntentRef: record.Goal.Intent(), AppSpecRef: appSpec.Ref(),
			AppSpecGeneration: appSpec.Generation(), SpecHash: appSpec.Hash(), ActorRef: record.Goal.Actor(),
			ProjectRef: record.Goal.Project(), Statement: appSpec.Intent().Statement(),
			State: record.Goal.State(), Revision: record.Goal.Revision(), CreatedAt: record.Goal.CreatedAt(),
			ArtifactCount: len(record.Artifacts),
		})
	}
	return result, nil
}

func (state *memoryState) Status(context.Context) (application.RepositoryStatus, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	status := application.RepositoryStatus{Goals: int64(len(state.byRef)), PendingActions: state.pendingActions}
	for _, record := range state.byRef {
		if record.Goal.State() == goal.GoalStateRunning {
			status.RunningGoals++
		}
	}
	return status, nil
}

func (state *memoryState) ClaimNextAction(context.Context, application.ClaimRequest) (application.ActionClaim, bool, error) {
	return application.ActionClaim{}, false, nil
}

func (state *memoryState) RecordLaunchAccepted(context.Context, application.LaunchAcceptedState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) RecordLaunchPrepared(context.Context, application.LaunchPreparedState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) RequeueAction(context.Context, application.ActionRequeuedState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) QuarantineAction(context.Context, application.ActionQuarantinedState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) RecordGoalSucceeded(context.Context, application.GoalSucceededState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) RecordGoalFailed(context.Context, application.GoalFailedState) error {
	return errors.New("test.state_write_not_used")
}

func (state *memoryState) attachArtifact(t *testing.T, goalRef goal.GoalRef, artifact application.ArtifactRecord) {
	t.Helper()
	state.mu.Lock()
	defer state.mu.Unlock()
	record, found := state.byRef[goalRef]
	if !found {
		t.Fatalf("Goal %s not found", goalRef.String())
	}
	artifact.GoalRef = goalRef
	artifact.WorkItemRef = record.Executions[0].WorkItemRef
	record.Artifacts = append(record.Artifacts, artifact)
	state.byRef[goalRef] = record
}

func (state *memoryState) failGoal(t *testing.T, goalRef goal.GoalRef, code string) {
	t.Helper()
	state.mu.Lock()
	defer state.mu.Unlock()
	record, found := state.byRef[goalRef]
	if !found {
		t.Fatalf("Goal %s not found", goalRef.String())
	}
	item, found := record.Goal.WorkItem(record.Executions[0].WorkItemRef)
	if !found {
		t.Fatalf("WorkItem %s not found", record.Executions[0].WorkItemRef.String())
	}
	at := testNow().Add(time.Second)
	aggregate, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, at,
	)
	if err != nil {
		t.Fatalf("start WorkItem: %v", err)
	}
	item, _ = aggregate.WorkItem(item.Ref())
	aggregate, err = aggregate.FailWorkItem(aggregate.Revision(), item.Revision(), item.Ref(), at)
	if err != nil {
		t.Fatalf("fail WorkItem: %v", err)
	}
	aggregate, err = aggregate.Close(aggregate.Revision(), goal.GoalOutcomeFailed, at)
	if err != nil {
		t.Fatalf("close Goal: %v", err)
	}
	record.Goal = aggregate
	record.Executions[0].State = application.ExecutionFailed
	record.Executions[0].StartedAt = at
	record.Executions[0].FinishedAt = at
	record.Executions[0].FailureCode = code
	state.byRef[goalRef] = record
	state.clock.Set(at.Add(time.Second))
}

func cloneGoalRecord(record application.GoalRecord) application.GoalRecord {
	record.Executions = append([]application.ExecutionRecord(nil), record.Executions...)
	record.Artifacts = append([]application.ArtifactRecord(nil), record.Artifacts...)
	record.Attestations = append([]application.AttestationRecord(nil), record.Attestations...)
	return record
}

func (state *memoryState) counts() (goals int, requests int, successors int, pendingActions int64) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return len(state.byRef), len(state.byRequest), len(state.byParentSpec), state.pendingActions
}
