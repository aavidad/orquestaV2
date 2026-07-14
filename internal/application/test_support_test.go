package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type memoryRepository struct {
	mu       sync.Mutex
	records  map[goal.GoalRef]GoalRecord
	requests map[string]goal.GoalRef
	actions  map[string]memoryAction
	events   []EventRecord
}

type memoryAction struct {
	record    ActionRecord
	token     string
	workerRef string
	attempt   uint64
	lease     time.Time
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		records:  make(map[goal.GoalRef]GoalRecord),
		requests: make(map[string]goal.GoalRef),
		actions:  make(map[string]memoryAction),
	}
}

func (repository *memoryRepository) CreateGoal(_ context.Context, state CreateGoalState) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	requestKey := state.Intent.Actor().String() + "\x00" + state.Intent.Project().String() + "\x00" + state.RequestRef
	if ref, ok := repository.requests[requestKey]; ok {
		record := repository.records[ref]
		if record.RequestFingerprint != state.RequestFingerprint ||
			record.Intent.Actor() != state.Intent.Actor() ||
			record.Intent.Project() != state.Intent.Project() ||
			record.Intent.Statement() != state.Intent.Statement() {
			return GoalRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneGoalRecord(record), false, nil
	}
	record := GoalRecord{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Intent: state.Intent,
		Goal:   state.Goal, Executions: append([]ExecutionRecord(nil), state.Executions...),
	}
	repository.requests[requestKey] = state.Goal.Ref()
	repository.records[state.Goal.Ref()] = record
	for _, action := range state.Actions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return cloneGoalRecord(record), true, nil
}

func (repository *memoryRepository) GetGoal(_ context.Context, ref goal.GoalRef) (GoalRecord, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, ok := repository.records[ref]
	if !ok {
		return GoalRecord{}, &StateError{Code: StateNotFound}
	}
	return cloneGoalRecord(record), nil
}

func (repository *memoryRepository) ListGoals(_ context.Context, actor goal.ActorRef, project goal.ProjectRef, limit int) ([]GoalSummary, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	result := make([]GoalSummary, 0, len(repository.records))
	for _, record := range repository.records {
		if record.Goal.Actor() != actor || record.Goal.Project() != project {
			continue
		}
		closedAt, _ := record.Goal.ClosedAt()
		result = append(result, GoalSummary{
			Ref: record.Goal.Ref(), IntentRef: record.Intent.Ref(),
			ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(),
			Statement: record.Intent.Statement(), State: record.Goal.State(),
			Revision: record.Goal.Revision(), CreatedAt: record.Goal.CreatedAt(),
			ClosedAt: closedAt, ArtifactCount: len(record.Artifacts),
		})
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].CreatedAt.After(result[right].CreatedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (repository *memoryRepository) Status(context.Context) (RepositoryStatus, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	status := RepositoryStatus{Goals: int64(len(repository.records)), PendingActions: int64(len(repository.actions))}
	for _, record := range repository.records {
		if record.Goal.State() == goal.GoalStateRunning {
			status.RunningGoals++
		}
	}
	for _, event := range repository.events {
		if event.Kind == "action.quarantined" {
			status.QuarantinedActions++
		}
	}
	return status, nil
}

func (repository *memoryRepository) ClaimNextAction(_ context.Context, request ClaimRequest) (ActionClaim, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	refs := make([]string, 0, len(repository.actions))
	for ref := range repository.actions {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		action := repository.actions[ref]
		if action.record.AvailableAt.After(request.Now) || (action.token != "" && action.lease.After(request.Now)) {
			continue
		}
		action.token = request.Token
		action.workerRef = request.WorkerRef
		action.attempt++
		action.lease = request.Now.Add(request.LeaseDuration)
		repository.actions[ref] = action
		return ActionClaim{
			Action: action.record, Token: action.token, WorkerRef: action.workerRef,
			Attempt: action.attempt, LeaseUntil: action.lease,
		}, true, nil
	}
	return ActionClaim{}, false, nil
}

func (repository *memoryRepository) RecordLaunchPrepared(_ context.Context, state LaunchPreparedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || action.token != state.Claim.Token || action.workerRef != state.Claim.WorkerRef ||
		!state.OperationAt.Before(action.lease) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Goal.Ref()]
	if record.Goal.Revision() != state.ExpectedGoalRevision {
		return &StateError{Code: StateConflict}
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	repository.records[state.Goal.Ref()] = record
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordLaunchAccepted(_ context.Context, state LaunchAcceptedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || action.token != state.Claim.Token || action.workerRef != state.Claim.WorkerRef ||
		!state.OperationAt.Before(action.lease) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	record.Executions = replaceExecution(record.Executions, state.Execution)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RequeueAction(_ context.Context, state ActionRequeuedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || action.token != state.Claim.Token || action.workerRef != state.Claim.WorkerRef ||
		!state.OperationAt.Before(action.lease) {
		return &StateError{Code: StateConflict}
	}
	record, ok := repository.records[state.Claim.Action.GoalRef]
	current, found := executionForAction(record, state.Claim.Action)
	if !ok || !found || current.Ref != state.Execution.Ref {
		return &StateError{Code: StateConflict}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	repository.records[record.Goal.Ref()] = record
	action.record.AvailableAt = state.AvailableAt
	action.token = ""
	action.workerRef = ""
	action.lease = time.Time{}
	repository.actions[action.record.Ref] = action
	return nil
}

func (repository *memoryRepository) QuarantineAction(_ context.Context, state ActionQuarantinedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || action.token != state.Claim.Token || action.workerRef != state.Claim.WorkerRef ||
		!state.OperationAt.Before(action.lease) {
		return &StateError{Code: StateConflict}
	}
	delete(repository.actions, state.Claim.Action.Ref)
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordGoalSucceeded(_ context.Context, state GoalSucceededState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	record.Artifacts = append(record.Artifacts, state.Artifact)
	record.Attestations = append(record.Attestations, state.Attestation)
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) RecordGoalFailed(_ context.Context, state GoalFailedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) validateMutation(claim ActionClaim, operationAt time.Time, goalRevision, itemRevision goal.Revision) error {
	action, ok := repository.actions[claim.Action.Ref]
	if !ok || action.token != claim.Token || action.workerRef != claim.WorkerRef || !operationAt.Before(action.lease) {
		return &StateError{Code: StateConflict}
	}
	record, ok := repository.records[claim.Action.GoalRef]
	if !ok || record.Goal.Revision() != goalRevision {
		return &StateError{Code: StateConflict}
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.Revision() != itemRevision {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func cloneGoalRecord(record GoalRecord) GoalRecord {
	record.Executions = append([]ExecutionRecord(nil), record.Executions...)
	record.Artifacts = append([]ArtifactRecord(nil), record.Artifacts...)
	record.Attestations = append([]AttestationRecord(nil), record.Attestations...)
	return record
}

func onlyExecution(t *testing.T, record GoalRecord) ExecutionRecord {
	t.Helper()
	if len(record.Executions) != 1 {
		t.Fatalf("execution count = %d, want 1", len(record.Executions))
	}
	return record.Executions[0]
}

type mutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *mutableClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *mutableClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type sequentialIDs struct {
	mu   sync.Mutex
	next int
}

func (ids *sequentialIDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:%03d", prefix, ids.next), nil
}

type memoryArtifactStore struct {
	mu      sync.Mutex
	content map[goal.ArtifactRef]ports.ArtifactContent
}

func newMemoryArtifactStore() *memoryArtifactStore {
	return &memoryArtifactStore{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
}

func (store *memoryArtifactStore) Put(_ context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + digestText)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	stored := ports.StoredArtifact{Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content))}
	store.content[ref] = ports.ArtifactContent{Ref: ref, Digest: stored.Digest, Size: stored.Size, Content: append([]byte(nil), request.Content...)}
	return stored, nil
}

func (store *memoryArtifactStore) Get(_ context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content, ok := store.content[ref]
	if !ok {
		return ports.ArtifactContent{}, errors.New("artifact.not_found")
	}
	if content.Size != expectedSize {
		return ports.ArtifactContent{}, errors.New("artifact.size_mismatch")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}

type scriptedAgent struct {
	mu            sync.Mutex
	now           func() time.Time
	launches      int
	observations  []ports.AgentObservation
	launchErr     error
	launchEntered chan struct{}
	launchRelease <-chan struct{}
}

func (agent *scriptedAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:test"}, nil
}

func (agent *scriptedAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.launches++
	entered := agent.launchEntered
	release := agent.launchRelease
	now := agent.now
	launchErr := agent.launchErr
	agent.mu.Unlock()
	if entered != nil {
		select {
		case entered <- struct{}{}:
		default:
		}
	}
	if release != nil {
		<-release
	}
	if launchErr != nil {
		return ports.AgentLaunchReceipt{}, launchErr
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, ProviderRef: "provider:test",
		ExternalRef:    "external:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: now(),
	}, nil
}

func (agent *scriptedAgent) Observe(_ context.Context, execution goal.ExecutionRef) (ports.AgentObservation, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if len(agent.observations) == 0 {
		return ports.AgentObservation{}, errors.New("test.no_observation")
	}
	observation := agent.observations[0]
	agent.observations = agent.observations[1:]
	observation.ExecutionRef = execution
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = agent.now()
	}
	return observation, nil
}

func newTestOrchestrator(t interface{ Fatalf(string, ...any) }, repository *memoryRepository, clock *mutableClock, agent *scriptedAgent) (*Orchestrator, *memoryArtifactStore) {
	artifacts := newMemoryArtifactStore()
	orchestrator, err := New(Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxActionAttempts: 10, ClaimLease: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator, artifacts
}

func testScope(t interface{ Fatalf(string, ...any) }) (goal.ActorRef, goal.ProjectRef) {
	actor, err := goal.NewActorRef("actor:test")
	if err != nil {
		t.Fatalf("actor ref: %v", err)
	}
	project, err := goal.NewProjectRef("project:test")
	if err != nil {
		t.Fatalf("project ref: %v", err)
	}
	return actor, project
}
