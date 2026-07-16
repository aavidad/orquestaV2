package goal

import (
	"strings"
	"time"
)

type GoalState string

const (
	GoalStatePending   GoalState = "pending"
	GoalStateRunning   GoalState = "running"
	GoalStateSucceeded GoalState = "succeeded"
	GoalStateFailed    GoalState = "failed"
	GoalStateCanceled  GoalState = "canceled"
)

func (state GoalState) Terminal() bool {
	return state == GoalStateSucceeded || state == GoalStateFailed || state == GoalStateCanceled
}

type GoalOutcome string

const (
	GoalOutcomeSucceeded GoalOutcome = "succeeded"
	GoalOutcomeFailed    GoalOutcome = "failed"
)

// ChildHandoffOutcome is the terminal causal observation required from every
// contractual child before its parent and Goal may close successfully.
type ChildHandoffOutcome string

const (
	ChildHandoffAcknowledged ChildHandoffOutcome = "acknowledged"
	ChildHandoffBlocked      ChildHandoffOutcome = "blocked"
)

// ChildHandoffResolution is immutable evidence that one direct contractual
// child has either delivered a handoff acknowledged by its exact recipient or
// has an explicit blockage receipt. Mailbox claims and leases stay outside
// Goal; only this closure-relevant fact belongs to the aggregate.
type ChildHandoffResolution struct {
	parentRef  WorkItemRef
	childRef   WorkItemRef
	messageRef string
	outcome    ChildHandoffOutcome
	receiptRef string
	resolvedAt time.Time
}

func (resolution ChildHandoffResolution) ParentRef() WorkItemRef       { return resolution.parentRef }
func (resolution ChildHandoffResolution) ChildRef() WorkItemRef        { return resolution.childRef }
func (resolution ChildHandoffResolution) MessageRef() string           { return resolution.messageRef }
func (resolution ChildHandoffResolution) Outcome() ChildHandoffOutcome { return resolution.outcome }
func (resolution ChildHandoffResolution) ReceiptRef() string           { return resolution.receiptRef }
func (resolution ChildHandoffResolution) ResolvedAt() time.Time        { return resolution.resolvedAt }

// Goal is an immutable aggregate snapshot and the consistency boundary for its
// WorkItems. All aggregate mutations require the caller's expected revision.
type Goal struct {
	ref             GoalRef
	actor           ActorRef
	project         ProjectRef
	appSpec         AppSpec
	state           GoalState
	revision        Revision
	paused          bool
	cancelRequested bool
	controlSequence uint64
	createdAt       time.Time
	startedAt       time.Time
	closedAt        time.Time
	planGeneration  PlanGeneration
	phases          []PhaseInstance
	items           map[WorkItemRef]WorkItem
	itemOrder       []WorkItemRef
	childHandoffs   []ChildHandoffResolution
}

func NewGoal(ref GoalRef, spec AppSpec, createdAt time.Time) (Goal, error) {
	if !validGoalRef(ref) {
		return Goal{}, domainError(ErrorInvalidRef, "goal_ref")
	}
	if !validAppSpec(spec) || !spec.isRoot() {
		return Goal{}, domainError(ErrorInvalidArgument, "app_spec")
	}
	if !validTransitionTime(createdAt, spec.ConfirmedAt()) {
		return Goal{}, domainError(ErrorInvalidArgument, "created_at")
	}

	return Goal{
		ref:       ref,
		actor:     spec.Intent().Actor(),
		project:   spec.Intent().Project(),
		appSpec:   spec,
		state:     GoalStatePending,
		revision:  1,
		createdAt: canonicalTime(createdAt),
		items:     make(map[WorkItemRef]WorkItem),
	}, nil
}

// NewSuccessorGoal creates a fresh pending aggregate from an exact amendment.
// Source is immutable and must already be terminal; no plan or evidence moves.
func NewSuccessorGoal(ref GoalRef, source Goal, amended AppSpec, createdAt time.Time) (Goal, error) {
	if !validGoalRef(ref) {
		return Goal{}, domainError(ErrorInvalidRef, "goal_ref")
	}
	if ref == source.ref {
		return Goal{}, domainError(ErrorInvalidArgument, "goal_ref")
	}
	if !validAppSpec(source.appSpec) || !validGoalRef(source.ref) || !source.IsTerminal() || source.closedAt.IsZero() {
		return Goal{}, domainError(ErrorInvalidTransition, "source_goal")
	}
	if !validAppSpec(amended) || amended.isRoot() {
		return Goal{}, domainError(ErrorInvalidArgument, "app_spec")
	}
	parentRef, hasParent := amended.ParentRef()
	expectedGeneration, generationErr := nextAppSpecGeneration(source.appSpec.Generation())
	if generationErr != nil {
		return Goal{}, generationErr
	}
	if !hasParent || parentRef != source.appSpec.Ref() || amended.ParentHash() != source.SpecHash() ||
		amended.Generation() != expectedGeneration {
		return Goal{}, domainError(ErrorInvalidArgument, "app_spec_parent")
	}
	intent := amended.Intent()
	if intent.Actor() != source.actor || intent.Project() != source.project {
		return Goal{}, domainError(ErrorScopeConflict, "goal_scope")
	}
	if intent.SubmittedAt().Before(source.closedAt) || amended.ConfirmedAt().Before(source.closedAt) ||
		!validTransitionTime(createdAt, amended.ConfirmedAt()) || canonicalTime(createdAt).Before(source.closedAt) {
		return Goal{}, domainError(ErrorInvalidArgument, "created_at")
	}
	return Goal{
		ref: ref, actor: intent.Actor(), project: intent.Project(), appSpec: amended,
		state: GoalStatePending, revision: 1, createdAt: canonicalTime(createdAt),
		items: make(map[WorkItemRef]WorkItem),
	}, nil
}

func (goal Goal) Ref() GoalRef                   { return goal.ref }
func (goal Goal) Actor() ActorRef                { return goal.actor }
func (goal Goal) Project() ProjectRef            { return goal.project }
func (goal Goal) AppSpec() AppSpec               { return goal.appSpec }
func (goal Goal) SpecHash() string               { return goal.appSpec.Hash() }
func (goal Goal) Intent() IntentRef              { return goal.appSpec.Intent().Ref() }
func (goal Goal) IntentHash() string             { return goal.appSpec.Intent().Hash() }
func (goal Goal) State() GoalState               { return goal.state }
func (goal Goal) Revision() Revision             { return goal.revision }
func (goal Goal) CreatedAt() time.Time           { return goal.createdAt }
func (goal Goal) IsTerminal() bool               { return goal.state.Terminal() }
func (goal Goal) WorkItemCount() int             { return len(goal.itemOrder) }
func (goal Goal) PlanGeneration() PlanGeneration { return goal.planGeneration }

func (goal Goal) Phases() []PhaseInstance {
	return clonePhases(goal.phases)
}

func (goal Goal) StartedAt() (time.Time, bool) {
	return goal.startedAt, !goal.startedAt.IsZero()
}

func (goal Goal) ClosedAt() (time.Time, bool) {
	return goal.closedAt, !goal.closedAt.IsZero()
}

func (goal Goal) WorkItem(ref WorkItemRef) (WorkItem, bool) {
	item, ok := goal.items[ref]
	return item.clone(), ok
}

func (goal Goal) WorkItems() []WorkItem {
	items := make([]WorkItem, 0, len(goal.itemOrder))
	for _, ref := range goal.itemOrder {
		items = append(items, goal.items[ref].clone())
	}
	return items
}

func (goal Goal) ChildHandoffResolutions() []ChildHandoffResolution {
	return cloneChildHandoffResolutions(goal.childHandoffs)
}

// ApplyPlan evolves a pending or running Goal monotonically. Goal revision is
// the CAS; the proposal generation must be exactly current+1. Existing phases
// and WorkItems remain an exact ordered prefix and only pending work may be
// appended. It never changes lifecycle by itself.
func (goal Goal) ApplyPlan(expectedGoal Revision, plan Plan) (Goal, error) {
	if err := goal.expectRevision(expectedGoal); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStatePending && goal.state != GoalStateRunning {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if goal.cancelRequested {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_cancel_requested")
	}
	nextGeneration, err := nextPlanGeneration(goal.planGeneration)
	if err != nil {
		return Goal{}, err
	}
	if plan.generation != nextGeneration {
		return Goal{}, domainError(ErrorInvalidPlan, "plan_generation")
	}
	if err := validatePlan(plan); err != nil {
		return Goal{}, err
	}
	if len(plan.phases) < len(goal.phases) || len(plan.items) < len(goal.itemOrder) {
		return Goal{}, domainError(ErrorInvalidPlan, "non_monotonic_plan")
	}
	for index, existing := range goal.phases {
		if !equalPhaseInstances(existing, plan.phases[index]) {
			return Goal{}, domainError(ErrorInvalidPlan, "phase_changed")
		}
	}
	for index, ref := range goal.itemOrder {
		candidate := plan.items[index]
		if candidate.ref != ref || !equalWorkItems(goal.items[ref], candidate) {
			return Goal{}, domainError(ErrorInvalidPlan, "work_item_changed")
		}
	}
	for _, item := range plan.items[len(goal.itemOrder):] {
		if item.state != WorkItemStatePending || item.revision != 1 ||
			!item.startedAt.IsZero() || !item.finishedAt.IsZero() || validExecutionRef(item.execution) ||
			len(item.artifacts) != 0 || len(item.attestations) != 0 || item.paused || item.cancelRequested ||
			item.controlSequence != 0 || item.interruptCause != "" || !item.interruptedAt.IsZero() ||
			validWorkItemRef(item.reworkOf) {
			return Goal{}, domainError(ErrorInvalidPlan, "new_work_item_snapshot")
		}
		if validWorkItemRef(item.parent) {
			if parent, existed := goal.items[item.parent]; existed && parent.IsTerminal() {
				return Goal{}, domainError(ErrorInvalidPlan, "terminal_parent")
			}
		}
	}

	items := make(map[WorkItemRef]WorkItem, len(plan.items))
	order := make([]WorkItemRef, 0, len(plan.items))
	for _, item := range plan.items {
		if item.goal != goal.ref || item.actor != goal.actor || item.project != goal.project {
			return Goal{}, domainError(ErrorScopeConflict, "work_item_scope")
		}
		if item.createdAt.Before(goal.createdAt) {
			return Goal{}, domainError(ErrorInvalidArgument, "work_item_created_at")
		}
		items[item.ref] = item.clone()
		order = append(order, item.ref)
	}

	updated := goal.clone()
	updated.planGeneration = plan.generation
	updated.phases = clonePhases(plan.phases)
	updated.items = items
	updated.itemOrder = order
	updated.revision++
	return updated, nil
}

func (goal Goal) Start(expected Revision, at time.Time) (Goal, error) {
	if err := goal.expectRevision(expected); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStatePending {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if goal.cancelRequested {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_cancel_requested")
	}
	if len(goal.itemOrder) == 0 {
		return Goal{}, domainError(ErrorWorkItemsRequired, "work_items")
	}
	if !validTransitionTime(at, goal.createdAt) {
		return Goal{}, domainError(ErrorInvalidArgument, "started_at")
	}
	startedAt := canonicalTime(at)
	for _, item := range goal.items {
		if item.createdAt.After(startedAt) {
			return Goal{}, domainError(ErrorInvalidArgument, "started_at")
		}
	}

	updated := goal.clone()
	updated.state = GoalStateRunning
	updated.revision++
	updated.startedAt = startedAt
	return updated, nil
}

func (goal Goal) StartWorkItem(
	expectedGoal Revision,
	expectedItem Revision,
	ref WorkItemRef,
	execution ExecutionRef,
	at time.Time,
) (Goal, error) {
	item, err := goal.workItemForTransition(expectedGoal, ref, at)
	if err != nil {
		return Goal{}, err
	}
	if !goal.workItemReady(ref) {
		return Goal{}, domainError(ErrorWorkItemNotReady, "work_item_ref")
	}
	item, err = item.Start(expectedItem, execution, at)
	if err != nil {
		return Goal{}, err
	}
	return goal.withUpdatedWorkItem(item), nil
}

// ReplaceWorkItemExecution atomically replaces the attempt reference of one
// running WorkItem under both Goal and WorkItem CAS. It never creates another
// lifecycle: state and started_at remain owned by the existing WorkItem.
func (goal Goal) ReplaceWorkItemExecution(
	expectedGoal Revision,
	expectedItem Revision,
	ref WorkItemRef,
	currentExecution ExecutionRef,
	replacementExecution ExecutionRef,
	at time.Time,
) (Goal, error) {
	item, err := goal.workItemForTransition(expectedGoal, ref, at)
	if err != nil {
		return Goal{}, err
	}
	item, err = item.replaceExecution(expectedItem, currentExecution, replacementExecution, at)
	if err != nil {
		return Goal{}, err
	}
	return goal.withUpdatedWorkItem(item), nil
}

// RunnableWorkItems returns dependency-ready pending work that does not
// conflict with any running item. It intentionally does not choose a cohort.
func (goal Goal) RunnableWorkItems() []WorkItem {
	if goal.state != GoalStateRunning || goal.paused || goal.cancelRequested {
		return nil
	}

	runningWriteSets := make([][]WriteScope, 0)
	for _, ref := range goal.itemOrder {
		item := goal.items[ref]
		if item.state == WorkItemStateRunning {
			runningWriteSets = append(runningWriteSets, item.writeSet)
		}
	}

	runnable := make([]WorkItem, 0)
	for _, ref := range goal.itemOrder {
		item := goal.items[ref]
		if item.state != WorkItemStatePending || item.paused || item.cancelRequested || !goal.dependenciesSucceeded(item) {
			continue
		}
		if conflictsWithWriteSets(item.writeSet, runningWriteSets) {
			continue
		}
		runnable = append(runnable, item.clone())
	}
	return runnable
}

// ReadyWorkItems greedily derives one deterministic maximal conflict-free
// cohort in plan order, accounting for both running and already selected work.
func (goal Goal) ReadyWorkItems() []WorkItem {
	runnable := goal.RunnableWorkItems()
	ready := make([]WorkItem, 0, len(runnable))
	selectedWriteSets := make([][]WriteScope, 0, len(runnable))
	for _, item := range runnable {
		if conflictsWithWriteSets(item.writeSet, selectedWriteSets) {
			continue
		}
		ready = append(ready, item.clone())
		selectedWriteSets = append(selectedWriteSets, item.writeSet)
	}
	return ready
}

// ChildWorkItems derives parent lineage independently from dependency edges.
// HandoffRequired is a separate immutable policy. Order follows the plan.
func (goal Goal) ChildWorkItems(parent WorkItemRef) []WorkItem {
	children := make([]WorkItem, 0)
	for _, ref := range goal.itemOrder {
		item := goal.items[ref]
		if item.parent == parent {
			children = append(children, item.clone())
		}
	}
	return children
}

// ChildHandoffsResolved reports whether every direct contractual child has a
// terminal handoff fact. A leaf is resolved; an unknown parent is not.
func (goal Goal) ChildHandoffsResolved(parent WorkItemRef) bool {
	if _, exists := goal.items[parent]; !exists {
		return false
	}
	return goal.childHandoffsResolvedForParent(parent)
}

// ResolveChildHandoff records the only closure-relevant mailbox fact in Goal.
// An exact replay is idempotent even when it carries the pre-write expected
// revision; a different fact for the same parent/child pair is contradictory.
func (goal Goal) ResolveChildHandoff(
	expected Revision,
	parentRef WorkItemRef,
	childRef WorkItemRef,
	messageRef string,
	outcome ChildHandoffOutcome,
	receiptRef string,
	at time.Time,
) (Goal, error) {
	if !validWorkItemRef(parentRef) || !validWorkItemRef(childRef) ||
		!validChildHandoffRecordRef(messageRef) || !validChildHandoffOutcome(outcome) ||
		!validChildHandoffRecordRef(receiptRef) || at.IsZero() {
		return Goal{}, domainError(ErrorChildHandoffInvalid, "child_handoff")
	}
	resolution := ChildHandoffResolution{
		parentRef: parentRef, childRef: childRef, messageRef: messageRef,
		outcome: outcome, receiptRef: receiptRef, resolvedAt: canonicalTime(at),
	}
	for _, existing := range goal.childHandoffs {
		if existing.parentRef != parentRef || existing.childRef != childRef {
			continue
		}
		if equalChildHandoffResolutions(existing, resolution) {
			return goal, nil
		}
		return Goal{}, domainError(ErrorChildHandoffConflict, "child_handoff")
	}
	if err := goal.expectRevision(expected); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStateRunning {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if _, exists := goal.items[parentRef]; !exists {
		return Goal{}, domainError(ErrorChildHandoffInvalid, "parent_ref")
	}
	child, exists := goal.items[childRef]
	if !exists || child.parent != parentRef {
		return Goal{}, domainError(ErrorChildHandoffInvalid, "child_ref")
	}
	if !child.handoffRequired || child.state != WorkItemStateSucceeded ||
		!validTransitionTime(resolution.resolvedAt, child.finishedAt) {
		return Goal{}, domainError(ErrorChildHandoffInvalid, "resolved_at")
	}

	updated := goal.clone()
	updated.childHandoffs = append(updated.childHandoffs, resolution)
	updated.revision++
	return updated, nil
}

func (goal Goal) SucceedWorkItem(
	expectedGoal Revision,
	expectedItem Revision,
	ref WorkItemRef,
	artifacts []ArtifactRef,
	attestations []AttestationRef,
	at time.Time,
) (Goal, error) {
	item, err := goal.workItemForTransition(expectedGoal, ref, at)
	if err != nil {
		return Goal{}, err
	}
	if err := item.expectRevision(expectedItem); err != nil {
		return Goal{}, err
	}
	if item.state != WorkItemStateRunning {
		return Goal{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if !goal.childHandoffsResolvedForParent(ref) {
		return Goal{}, domainError(ErrorChildHandoffsPending, "child_handoffs")
	}
	item, err = item.Succeed(expectedItem, artifacts, attestations, at)
	if err != nil {
		return Goal{}, err
	}
	return goal.withUpdatedWorkItem(item), nil
}

func (goal Goal) FailWorkItem(
	expectedGoal Revision,
	expectedItem Revision,
	ref WorkItemRef,
	at time.Time,
) (Goal, error) {
	item, err := goal.workItemForTransition(expectedGoal, ref, at)
	if err != nil {
		return Goal{}, err
	}
	item, err = item.Fail(expectedItem, at)
	if err != nil {
		return Goal{}, err
	}
	updated := goal.clone()
	updated.items[item.ref] = item
	if err := updated.cascadeDependencySkips(at); err != nil {
		return Goal{}, err
	}
	updated.revision++
	return updated, nil
}

func (goal Goal) Close(expected Revision, outcome GoalOutcome, at time.Time) (Goal, error) {
	if err := goal.expectRevision(expected); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStateRunning {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if outcome != GoalOutcomeSucceeded && outcome != GoalOutcomeFailed {
		return Goal{}, domainError(ErrorInvalidArgument, "goal_outcome")
	}
	if goal.cancelRequested {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_cancel_requested")
	}
	if !goal.allChildHandoffsResolved() {
		return Goal{}, domainError(ErrorChildHandoffsPending, "child_handoffs")
	}

	closableOutcome, closable := goal.ClosableOutcome()
	if !closable {
		for _, item := range goal.items {
			if !item.IsTerminal() {
				return Goal{}, domainError(ErrorWorkItemsNotTerminal, "work_items")
			}
		}
		return Goal{}, domainError(ErrorOutcomeConflict, "goal_outcome")
	}
	if outcome != closableOutcome {
		return Goal{}, domainError(ErrorOutcomeConflict, "goal_outcome")
	}
	latestFinishedAt := goal.startedAt
	for _, item := range goal.items {
		if item.finishedAt.After(latestFinishedAt) {
			latestFinishedAt = item.finishedAt
		}
	}
	if !validTransitionTime(at, latestFinishedAt) {
		return Goal{}, domainError(ErrorInvalidArgument, "closed_at")
	}

	updated := goal.clone()
	if outcome == GoalOutcomeSucceeded {
		updated.state = GoalStateSucceeded
	} else {
		updated.state = GoalStateFailed
	}
	updated.paused = false
	updated.revision++
	updated.closedAt = canonicalTime(at)
	return updated, nil
}

// ClosableOutcome derives closure eligibility without duplicating lifecycle
// rules in application or schedulers.
func (goal Goal) ClosableOutcome() (GoalOutcome, bool) {
	if goal.state != GoalStateRunning || len(goal.items) == 0 || !goal.allChildHandoffsResolved() {
		return "", false
	}
	allSucceeded := true
	hasFailed := false
	for ref := range goal.items {
		logical, resolved := goal.LogicalWorkItemOutcome(ref)
		if !resolved {
			return "", false
		}
		allSucceeded = allSucceeded && logical == WorkItemLogicalSucceeded
		hasFailed = hasFailed || logical == WorkItemLogicalFailed
	}
	if allSucceeded {
		return GoalOutcomeSucceeded, true
	}
	if hasFailed {
		return GoalOutcomeFailed, true
	}
	return "", false
}

func (goal Goal) expectRevision(expected Revision) error {
	if expected != goal.revision {
		return domainError(ErrorRevisionConflict, "goal_revision")
	}
	return nil
}

func (goal Goal) workItemForTransition(
	expected Revision,
	ref WorkItemRef,
	at time.Time,
) (WorkItem, error) {
	if err := goal.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if goal.state != GoalStateRunning {
		return WorkItem{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if !validWorkItemRef(ref) {
		return WorkItem{}, domainError(ErrorInvalidRef, "work_item_ref")
	}
	if !validTransitionTime(at, goal.startedAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "transition_at")
	}
	item, ok := goal.items[ref]
	if !ok {
		return WorkItem{}, domainError(ErrorWorkItemNotFound, "work_item_ref")
	}
	return item.clone(), nil
}

func (goal Goal) withUpdatedWorkItem(item WorkItem) Goal {
	updated := goal.clone()
	updated.items[item.ref] = item.clone()
	updated.revision++
	return updated
}

func (goal Goal) workItemReady(ref WorkItemRef) bool {
	for _, item := range goal.ReadyWorkItems() {
		if item.ref == ref {
			return true
		}
	}
	return false
}

func (goal Goal) dependenciesSucceeded(item WorkItem) bool {
	for _, dependency := range item.dependencies {
		outcome, resolved := goal.LogicalWorkItemOutcome(dependency)
		if !resolved || outcome != WorkItemLogicalSucceeded {
			return false
		}
	}
	return true
}

func (goal Goal) hasFailedDependency(item WorkItem) bool {
	for _, dependency := range item.dependencies {
		outcome, resolved := goal.LogicalWorkItemOutcome(dependency)
		if resolved && outcome == WorkItemLogicalFailed {
			return true
		}
	}
	return false
}

func (goal Goal) childHandoffsResolvedForParent(parentRef WorkItemRef) bool {
	parent, exists := goal.items[parentRef]
	if !exists {
		return false
	}
	if parent.handoffBlockedByOwnOutcome() {
		return true
	}
	for _, childRef := range goal.itemOrder {
		child := goal.items[childRef]
		if child.parent == parentRef && child.handoffRequired && !child.handoffBlockedByOwnOutcome() &&
			(child.state != WorkItemStateSucceeded || !goal.childHandoffResolved(parentRef, childRef)) {
			return false
		}
	}
	return true
}

func (goal Goal) allChildHandoffsResolved() bool {
	for _, parentRef := range goal.itemOrder {
		if !goal.childHandoffsResolvedForParent(parentRef) {
			return false
		}
	}
	return true
}

func (goal Goal) childHandoffResolved(parentRef, childRef WorkItemRef) bool {
	for _, resolution := range goal.childHandoffs {
		if resolution.parentRef == parentRef && resolution.childRef == childRef {
			return true
		}
	}
	return false
}

func validChildHandoffOutcome(outcome ChildHandoffOutcome) bool {
	return outcome == ChildHandoffAcknowledged || outcome == ChildHandoffBlocked
}

func validChildHandoffRecordRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

func equalChildHandoffResolutions(left, right ChildHandoffResolution) bool {
	return left.parentRef == right.parentRef && left.childRef == right.childRef &&
		left.messageRef == right.messageRef && left.outcome == right.outcome &&
		left.receiptRef == right.receiptRef && left.resolvedAt.Equal(right.resolvedAt)
}

func cloneChildHandoffResolutions(resolutions []ChildHandoffResolution) []ChildHandoffResolution {
	return append([]ChildHandoffResolution(nil), resolutions...)
}

func conflictsWithWriteSets(candidate []WriteScope, others [][]WriteScope) bool {
	for _, other := range others {
		if writeSetsOverlap(candidate, other) {
			return true
		}
	}
	return false
}

func (goal Goal) clone() Goal {
	items := make(map[WorkItemRef]WorkItem, len(goal.items))
	for ref, item := range goal.items {
		items[ref] = item.clone()
	}
	goal.items = items
	goal.phases = clonePhases(goal.phases)
	goal.itemOrder = append([]WorkItemRef(nil), goal.itemOrder...)
	goal.childHandoffs = cloneChildHandoffResolutions(goal.childHandoffs)
	return goal
}

func validIntentManifest(intent IntentManifest) bool {
	return validIntentRef(intent.ref) &&
		validActorRef(intent.actor) &&
		validProjectRef(intent.project) &&
		intent.statement != "" &&
		!intent.submittedAt.IsZero() &&
		intent.hash == hashIntentManifest(intent)
}
