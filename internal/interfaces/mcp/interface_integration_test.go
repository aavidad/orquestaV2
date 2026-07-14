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
	wantNames := []string{ToolArtifactsRead, ToolGoalsCreate, ToolGoalsGet, ToolGoalsList, ToolSystemStatus}
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
	createAnnotations := byName[ToolGoalsCreate].Annotations
	if createAnnotations == nil || createAnnotations.ReadOnlyHint || !createAnnotations.IdempotentHint ||
		createAnnotations.DestructiveHint == nil || *createAnnotations.DestructiveHint ||
		createAnnotations.OpenWorldHint == nil || *createAnnotations.OpenWorldHint {
		t.Fatalf("create annotations = %+v", createAnnotations)
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

	createArguments := map[string]any{"request_ref": "request:test-1", "statement": "Produce a durable artifact"}
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
	if listResult.IsError || list.Count != 1 || len(list.Goals) != 1 || list.Goals[0].GoalRef != created.Goal.GoalRef {
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

	request := map[string]any{"request_ref": "request:conflict", "statement": "first statement"}
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
		"request_ref": "request:failed-view", "statement": "fail safely",
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
		byRef:     make(map[goal.GoalRef]application.GoalRecord),
		byRequest: make(map[string]goal.GoalRef),
	}
	artifacts := &memoryArtifacts{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
	orchestrator, err := application.New(application.Dependencies{
		State: state, Launcher: inertAgent{}, Observer: inertAgent{}, Artifacts: artifacts,
		Clock: fixedClock{}, IDs: &sequentialIDs{}, MaxOutputBytes: 4096,
		MaxActionAttempts: 3, ClaimLease: time.Minute, ObservationDelay: time.Second,
		ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("application.New() error = %v", err)
	}
	actorRef, _ := goal.NewActorRef("actor:local")
	projectRef, _ := goal.NewProjectRef("project:local")
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
	return server, state, artifacts
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow() }

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
	mu        sync.Mutex
	byRef     map[goal.GoalRef]application.GoalRecord
	byRequest map[string]goal.GoalRef
}

func (state *memoryState) CreateGoal(_ context.Context, input application.CreateGoalState) (application.GoalRecord, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if existingRef, found := state.byRequest[input.RequestRef]; found {
		existing := state.byRef[existingRef]
		if existing.RequestFingerprint != input.RequestFingerprint {
			return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
		}
		return cloneGoalRecord(existing), false, nil
	}
	record := application.GoalRecord{
		RequestRef: input.RequestRef, RequestFingerprint: input.RequestFingerprint,
		Intent: input.Intent, Goal: input.Goal,
		Executions: append([]application.ExecutionRecord(nil), input.Executions...),
	}
	state.byRef[input.Goal.Ref()] = record
	state.byRequest[input.RequestRef] = input.Goal.Ref()
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
		result = append(result, application.GoalSummary{
			Ref: record.Goal.Ref(), IntentRef: record.Goal.Intent(), ActorRef: record.Goal.Actor(),
			ProjectRef: record.Goal.Project(), Statement: record.Intent.Statement(),
			State: record.Goal.State(), Revision: record.Goal.Revision(), CreatedAt: record.Goal.CreatedAt(),
			ArtifactCount: len(record.Artifacts),
		})
	}
	return result, nil
}

func (state *memoryState) Status(context.Context) (application.RepositoryStatus, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	status := application.RepositoryStatus{Goals: int64(len(state.byRef)), PendingActions: int64(len(state.byRef))}
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
}

func cloneGoalRecord(record application.GoalRecord) application.GoalRecord {
	record.Executions = append([]application.ExecutionRecord(nil), record.Executions...)
	record.Artifacts = append([]application.ArtifactRecord(nil), record.Artifacts...)
	record.Attestations = append([]application.AttestationRecord(nil), record.Attestations...)
	return record
}
