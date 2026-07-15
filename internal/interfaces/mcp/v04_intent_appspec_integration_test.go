package mcpinterface

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

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
		"confirm", "normalized_objective", "plan", "project_ref", "request_ref", "statement",
	})
	assertClosedWorldProperties(t, byName[ToolGoalsAmend], []string{
		"confirm", "expected_source_revision", "expected_source_spec_hash", "normalized_objective",
		"project_ref", "reason", "request_ref", "source_goal_ref", "statement",
	})
	assertClosedWorldProperties(t, byName[ToolGoalsGet], []string{"goal_ref", "project_ref"})
	assertClosedWorldProperties(t, byName[ToolGoalsList], []string{"limit", "project_ref"})
	assertClosedWorldProperties(t, byName[ToolArtifactsRead], []string{"artifact_ref", "goal_ref", "project_ref"})
	assertClosedWorldProperties(t, byName[ToolSystemStatus], []string{"project_ref"})
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

func TestV10OfficialMCPRequiresExplicitProjectAndExposesNoPrincipalSpoofFields(t *testing.T) {
	server, _, _ := newTestInterface(t, 16*1024)
	session := serveOfficialClient(t, server)
	validHash := strings.Repeat("a", 64)
	valid := map[string]map[string]any{
		ToolGoalsCreate: {
			"request_ref": "request:missing-project", "statement": "exact", "confirm": true,
		},
		ToolGoalsAmend: {
			"request_ref": "request:missing-project", "source_goal_ref": "goal:source",
			"expected_source_revision": uint64(1), "expected_source_spec_hash": validHash,
			"statement": "exact", "reason": "operator.amendment", "confirm": true,
		},
		ToolGoalsGet:      {"goal_ref": "goal:source"},
		ToolGoalsList:     {"limit": 1},
		ToolArtifactsRead: {"goal_ref": "goal:source", "artifact_ref": "artifact:source"},
		ToolSystemStatus:  {},
	}
	for name, arguments := range valid {
		name, arguments := name, arguments
		t.Run(name+"_missing", func(t *testing.T) {
			result := callTool(t, session, name, arguments)
			if !result.IsError {
				t.Fatalf("missing required project unexpectedly succeeded: %+v", result)
			}
		})
		t.Run(name+"_malformed", func(t *testing.T) {
			malformed := cloneArguments(arguments)
			malformed["project_ref"] = " project:local"
			assertPublicToolError(t, callTool(t, session, name, malformed), publicInvalidRequest)
		})
	}

	listed, err := session.ListTools(testContext(t), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	for _, tool := range listed.Tools {
		payload, marshalErr := json.Marshal(tool.InputSchema)
		if marshalErr != nil {
			t.Fatalf("Marshal(%s schema): %v", tool.Name, marshalErr)
		}
		for _, forbidden := range []string{
			"actor", "actor_ref", "confirmed_by", "default_project_ref", "kind",
			"principal_kind", "principal_ref",
		} {
			if strings.Contains(string(payload), `"`+forbidden+`"`) {
				t.Errorf("%s schema exposes trusted identity field %q: %s", tool.Name, forbidden, payload)
			}
		}
	}
}

func TestV04OfficialMCPCreateAndAmendAreCausalIdempotentAndServerOwned(t *testing.T) {
	server, state, _ := newTestInterface(t, 16*1024)
	session := serveOfficialClient(t, server)
	statement := "  Preserve exact operator intent.  "
	createArguments := map[string]any{
		"project_ref": "project:local",
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
	getResult := callTool(t, session, ToolGoalsGet, map[string]any{
		"project_ref": "project:local", "goal_ref": amended.Goal.GoalRef,
	})
	var got GetGoalOutput
	decodeStructured(t, getResult, &got)
	if getResult.IsError || got.Goal == nil || !reflect.DeepEqual(got.Goal.AppSpec, amended.Goal.AppSpec) ||
		got.Goal.IntentRef != amended.Goal.IntentRef || got.Goal.IntentHash != amended.Goal.IntentHash ||
		got.Goal.Statement != amended.Goal.Statement || got.Goal.ActorRef != amended.Goal.ActorRef ||
		got.Goal.ProjectRef != amended.Goal.ProjectRef {
		t.Fatalf("get successor projection = %+v result=%+v", got, getResult)
	}
	listResult := callTool(t, session, ToolGoalsList, map[string]any{"project_ref": "project:local", "limit": 10})
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
			"project_ref": "project:local", "request_ref": "request:unconfirmed",
			"statement": "must not exist", "confirm": false,
		})
	}, publicInvalidRequest)
	assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
		return callTool(t, session, ToolGoalsAmend, map[string]any{"project_ref": "project:local", "confirm": false})
	}, publicInvalidRequest)

	spoofs := map[string]any{
		"actor_ref": "actor:spoof", "confirmed_by": "actor:spoof", "principal_ref": "principal:spoof",
		"principal_kind": "service",
		"submitted_at":   "1999-01-01T00:00:00Z", "confirmed_at": "1999-01-01T00:00:00Z",
		"intent_ref": "intent:spoof", "app_spec_ref": "app-spec:spoof", "goal_ref": "goal:spoof",
	}
	for field, value := range spoofs {
		field, value := field, value
		t.Run("create_"+field, func(t *testing.T) {
			assertNoStateOrGeneratorChange(t, state, func() *sdkmcp.CallToolResult {
				arguments := map[string]any{
					"project_ref": "project:local", "request_ref": "request:spoof-create-" + field,
					"statement": "trusted input", "confirm": true,
					field: value,
				}
				return callTool(t, session, ToolGoalsCreate, arguments)
			}, "")
		})
	}

	created := callCreateGoal(t, session, map[string]any{
		"project_ref": "project:local", "request_ref": "request:spoof-source",
		"statement": "terminal source", "confirm": true,
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
			"project_ref": "project:local", "request_ref": "request:active-source",
			"statement": "active source", "confirm": true,
		})
		sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
		source, _ := state.GetGoal(context.Background(), sourceRef)
		assertNoDurableStateChange(t, state, func() *sdkmcp.CallToolResult {
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
				"project_ref": "project:local", "request_ref": "request:" + testCase.name + "-source",
				"statement": "terminal source", "confirm": true,
			})
			sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
			state.failGoal(t, sourceRef, "test.terminal_source")
			source, _ := state.GetGoal(context.Background(), sourceRef)
			input := amendmentToolArguments(source, "request:"+testCase.name+"-amend")
			testCase.mutate(input)
			assertNoDurableStateChange(t, state, func() *sdkmcp.CallToolResult {
				return callTool(t, session, ToolGoalsAmend, input)
			}, publicConflict)
		})
	}

	t.Run("foreign", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		localSession := serveOfficialClient(t, server)
		created := callCreateGoal(t, localSession, map[string]any{
			"project_ref": "project:local", "request_ref": "request:foreign-source",
			"statement": "terminal source", "confirm": true,
		})
		sourceRef, _ := goal.NewGoalRef(created.Goal.GoalRef)
		state.failGoal(t, sourceRef, "test.terminal_source")
		source, _ := state.GetGoal(context.Background(), sourceRef)
		foreignServer := newTestInterfaceForPrincipal(t, state, "actor:foreign", "project:foreign", 16*1024)
		foreignSession := serveOfficialClient(t, foreignServer)
		assertNoDurableStateChange(t, state, func() *sdkmcp.CallToolResult {
			return callTool(t, foreignSession, ToolGoalsAmend, amendmentToolArguments(source, "request:foreign-amend"))
		}, publicForbidden)
	})
}

func TestV04MemoryStateScopesRequestRefsAndSerializesConcurrentSuccessors(t *testing.T) {
	t.Run("request scope", func(t *testing.T) {
		server, state, _ := newTestInterface(t, 16*1024)
		local := serveOfficialClient(t, server)
		foreignServer := newTestInterfaceForPrincipal(t, state, "actor:other", "project:other", 16*1024)
		foreign := serveOfficialClient(t, foreignServer)
		arguments := map[string]any{
			"project_ref": "project:local", "request_ref": "request:shared",
			"statement": "same request ref", "confirm": true,
		}
		first := callCreateGoal(t, local, arguments)
		foreignArguments := cloneArguments(arguments)
		foreignArguments["project_ref"] = "project:other"
		second := callCreateGoal(t, foreign, foreignArguments)
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
			"project_ref": "project:local", "request_ref": "request:concurrent-source",
			"statement": "terminal source", "confirm": true,
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
	requiredValues, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("input schema required fields missing: %s", payload)
	}
	projectRequired := false
	for _, value := range requiredValues {
		if value == "project_ref" {
			projectRequired = true
			break
		}
	}
	if !projectRequired {
		t.Fatalf("project_ref is not required: %s", payload)
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
		"project_ref": source.Goal.Project().String(), "request_ref": requestRef,
		"source_goal_ref":           source.Goal.Ref().String(),
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

func assertNoDurableStateChange(
	t *testing.T,
	state *memoryState,
	operation func() *sdkmcp.CallToolResult,
	wantCode string,
) {
	t.Helper()
	beforeGoals, beforeRequests, beforeSuccessors, beforePending := state.counts()
	result := operation()
	assertPublicToolError(t, result, wantCode)
	afterGoals, afterRequests, afterSuccessors, afterPending := state.counts()
	if afterGoals != beforeGoals || afterRequests != beforeRequests || afterSuccessors != beforeSuccessors ||
		afterPending != beforePending {
		t.Fatalf("rejected request changed durable state: %d/%d/%d/%d -> %d/%d/%d/%d",
			beforeGoals, beforeRequests, beforeSuccessors, beforePending,
			afterGoals, afterRequests, afterSuccessors, afterPending)
	}
}
