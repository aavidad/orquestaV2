package mcpinterface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
