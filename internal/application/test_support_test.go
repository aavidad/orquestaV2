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
	mu         sync.Mutex
	records    map[goal.GoalRef]GoalRecord
	requests   map[string]goal.GoalRef
	successors map[goal.AppSpecRef]goal.GoalRef
	actions    map[string]memoryAction
	events     []EventRecord
	now        func() time.Time
}

type memoryAction struct {
	record          ActionRecord
	token           string
	workerRef       string
	deliveryAttempt uint64
	fence           uint64
	lease           time.Time
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		records:    make(map[goal.GoalRef]GoalRecord),
		requests:   make(map[string]goal.GoalRef),
		successors: make(map[goal.AppSpecRef]goal.GoalRef),
		actions:    make(map[string]memoryAction),
		now:        time.Now,
	}
}

func (repository *memoryRepository) CreateGoal(_ context.Context, state CreateGoalState) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	intent := state.Goal.AppSpec().Intent()
	requestKey := state.Goal.Actor().String() + "\x00" + state.Goal.Project().String() + "\x00" + state.RequestRef
	if ref, ok := repository.requests[requestKey]; ok {
		record := repository.records[ref]
		if record.RequestFingerprint != state.RequestFingerprint ||
			record.Goal.Actor() != state.Goal.Actor() ||
			record.Goal.Project() != state.Goal.Project() ||
			record.Goal.AppSpec().Intent().Statement() != intent.Statement() ||
			record.Goal.AppSpec().Objective() != state.Goal.AppSpec().Objective() {
			return GoalRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneGoalRecord(record), false, nil
	}
	record := GoalRecord{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Goal: state.Goal, Executions: append([]ExecutionRecord(nil), state.Executions...),
	}
	repository.requests[requestKey] = state.Goal.Ref()
	repository.records[state.Goal.Ref()] = record
	for _, action := range state.Actions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return cloneGoalRecord(record), true, nil
}

func (repository *memoryRepository) AmendGoal(_ context.Context, state AmendGoalState) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	requestKey := state.ActorRef.String() + "\x00" + state.ProjectRef.String() + "\x00" + state.RequestRef
	if ref, ok := repository.requests[requestKey]; ok {
		record := repository.records[ref]
		if record.RequestFingerprint != state.RequestFingerprint {
			return GoalRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneGoalRecord(record), false, nil
	}
	source, ok := repository.records[state.SourceGoalRef]
	if !ok {
		return GoalRecord{}, false, &StateError{Code: StateNotFound}
	}
	if source.Goal.Actor() != state.ActorRef || source.Goal.Project() != state.ProjectRef ||
		source.Goal.Revision() != state.ExpectedSourceRevision ||
		source.Goal.SpecHash() != state.ExpectedSourceSpecHash || !source.Goal.IsTerminal() {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	if _, exists := repository.successors[source.Goal.AppSpec().Ref()]; exists {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	successorSpec := state.Successor.AppSpec()
	parentRef, hasParent := successorSpec.ParentRef()
	if state.Successor.Actor() != state.ActorRef || state.Successor.Project() != state.ProjectRef ||
		state.Successor.State() != goal.GoalStatePending || state.Successor.WorkItemCount() != 0 ||
		!hasParent || parentRef != source.Goal.AppSpec().Ref() ||
		successorSpec.ParentHash() != source.Goal.SpecHash() ||
		successorSpec.Generation() != source.Goal.AppSpec().Generation()+1 {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	record := GoalRecord{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Goal: state.Successor,
	}
	repository.requests[requestKey] = state.Successor.Ref()
	repository.records[state.Successor.Ref()] = record
	repository.successors[source.Goal.AppSpec().Ref()] = state.Successor.Ref()
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
		appSpec := record.Goal.AppSpec()
		intent := appSpec.Intent()
		result = append(result, GoalSummary{
			Ref: record.Goal.Ref(), IntentRef: intent.Ref(), AppSpecRef: appSpec.Ref(),
			AppSpecGeneration: appSpec.Generation(), SpecHash: appSpec.Hash(),
			ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(),
			Statement: intent.Statement(), State: record.Goal.State(),
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
	now := repository.now().UTC()
	refs := make([]string, 0, len(repository.actions))
	for ref := range repository.actions {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		action := repository.actions[ref]
		if action.record.AvailableAt.After(now) || (action.token != "" && action.lease.After(now)) {
			continue
		}
		record := repository.records[action.record.GoalRef]
		item, found := record.Goal.WorkItem(action.record.WorkItemRef)
		if !found || !ports.MatchAgentCapabilities(request.Capabilities, ports.AgentRequirements{
			RoleKey: item.Role().String(), SkillRefs: workItemRefs(item.SkillRefs()),
			ToolRefs: workItemRefs(item.ToolRefs()), CapabilityRefs: workItemRefs(item.CapabilityRefs()),
		}) {
			continue
		}
		action.token = request.Token
		action.workerRef = request.WorkerRef
		action.deliveryAttempt++
		action.fence++
		action.lease = now.Add(request.LeaseDuration)
		repository.actions[ref] = action
		return ActionClaim{
			Action: action.record, Token: action.token, WorkerRef: action.workerRef,
			DeliveryAttempt: action.deliveryAttempt, Fence: action.fence, LeaseUntil: action.lease,
		}, true, nil
	}
	return ActionClaim{}, false, nil
}

func (repository *memoryRepository) RecordLaunchPrepared(_ context.Context, state LaunchPreparedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
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
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
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
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
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
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedQuarantined, state.ErrorCode, state.OperationAt))
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordExecutionReplaced(_ context.Context, state ExecutionReplacedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.FailedExecution)
	record.Executions = append(record.Executions, state.ReplacementExecution)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, state.ErrorCode, state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Events...)
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
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
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
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, state.Execution.FailureCode, state.OperationAt))
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
	if !ok || !memoryClaimMatches(action, claim, operationAt) {
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
	record.ConsumptionReceipts = append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...)
	return record
}

func memoryClaimMatches(action memoryAction, claim ActionClaim, operationAt time.Time) bool {
	return action.token == claim.Token && action.workerRef == claim.WorkerRef &&
		action.deliveryAttempt == claim.DeliveryAttempt && action.fence == claim.Fence &&
		action.lease.Equal(claim.LeaseUntil) && operationAt.Before(action.lease)
}

func consumptionReceipt(claim ActionClaim, outcome ActionConsumptionOutcome, code string, at time.Time) ActionConsumptionReceipt {
	return ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind,
		GoalRef: claim.Action.GoalRef, WorkItemRef: claim.Action.WorkItemRef,
		ExecutionRef: claim.Action.ExecutionRef, PlanGeneration: claim.Action.PlanGeneration,
		WorkItemGeneration: claim.Action.WorkItemGeneration, Fence: claim.Fence,
		DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: outcome, ErrorCode: code, ConsumedAt: at.UTC(),
	}
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
	mu                               sync.Mutex
	now                              func() time.Time
	launches                         int
	launchRequests                   []ports.AgentLaunchRequest
	launchSpecHashes                 map[goal.ExecutionRef]string
	receiptSpecHashOverride          string
	preserveEmptyReceiptSpecHash     bool
	preserveEmptyObservationSpecHash bool
	observations                     []ports.AgentObservation
	launchErr                        error
	launchEntered                    chan struct{}
	launchRelease                    <-chan struct{}
}

func (agent *scriptedAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return testAgentCapabilities(), nil
}

func testAgentCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true,
	}
}

func (agent *scriptedAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.launches++
	agent.launchRequests = append(agent.launchRequests, request)
	if agent.launchSpecHashes == nil {
		agent.launchSpecHashes = make(map[goal.ExecutionRef]string)
	}
	agent.launchSpecHashes[request.ExecutionRef] = request.SpecHash
	entered := agent.launchEntered
	release := agent.launchRelease
	now := agent.now
	launchErr := agent.launchErr
	receiptSpecHash := agent.receiptSpecHashOverride
	if receiptSpecHash == "" && !agent.preserveEmptyReceiptSpecHash {
		receiptSpecHash = request.SpecHash
	}
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
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: receiptSpecHash,
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
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
	if observation.SpecHash == "" && !agent.preserveEmptyObservationSpecHash {
		observation.SpecHash = agent.launchSpecHashes[execution]
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = agent.now()
	}
	return observation, nil
}

func newTestOrchestrator(t interface{ Fatalf(string, ...any) }, repository *memoryRepository, clock *mutableClock, agent *scriptedAgent) (*Orchestrator, *memoryArtifactStore) {
	artifacts := newMemoryArtifactStore()
	repository.now = clock.Now
	capabilities, err := agent.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("agent capabilities: %v", err)
	}
	orchestrator, err := New(Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxExecutionAttempts: 3, ClaimLease: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capabilities,
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
