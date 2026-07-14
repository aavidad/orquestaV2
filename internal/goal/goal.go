package goal

import "time"

type GoalState string

const (
	GoalStatePending   GoalState = "pending"
	GoalStateRunning   GoalState = "running"
	GoalStateSucceeded GoalState = "succeeded"
	GoalStateFailed    GoalState = "failed"
)

func (state GoalState) Terminal() bool {
	return state == GoalStateSucceeded || state == GoalStateFailed
}

type GoalOutcome string

const (
	GoalOutcomeSucceeded GoalOutcome = "succeeded"
	GoalOutcomeFailed    GoalOutcome = "failed"
)

// Goal is an immutable aggregate snapshot and the consistency boundary for its
// WorkItems. All aggregate mutations require the caller's expected revision.
type Goal struct {
	ref            GoalRef
	actor          ActorRef
	project        ProjectRef
	intentManifest IntentManifest
	state          GoalState
	revision       Revision
	createdAt      time.Time
	startedAt      time.Time
	closedAt       time.Time
	items          map[WorkItemRef]WorkItem
	itemOrder      []WorkItemRef
}

func NewGoal(ref GoalRef, intent IntentManifest, createdAt time.Time) (Goal, error) {
	if !validGoalRef(ref) {
		return Goal{}, domainError(ErrorInvalidRef, "goal_ref")
	}
	if !validIntentManifest(intent) {
		return Goal{}, domainError(ErrorInvalidArgument, "intent_manifest")
	}
	if !validTransitionTime(createdAt, intent.SubmittedAt()) {
		return Goal{}, domainError(ErrorInvalidArgument, "created_at")
	}

	return Goal{
		ref:            ref,
		actor:          intent.Actor(),
		project:        intent.Project(),
		intentManifest: intent,
		state:          GoalStatePending,
		revision:       1,
		createdAt:      canonicalTime(createdAt),
		items:          make(map[WorkItemRef]WorkItem),
	}, nil
}

func (goal Goal) Ref() GoalRef         { return goal.ref }
func (goal Goal) Actor() ActorRef      { return goal.actor }
func (goal Goal) Project() ProjectRef  { return goal.project }
func (goal Goal) Intent() IntentRef    { return goal.intentManifest.Ref() }
func (goal Goal) IntentHash() string   { return goal.intentManifest.Hash() }
func (goal Goal) State() GoalState     { return goal.state }
func (goal Goal) Revision() Revision   { return goal.revision }
func (goal Goal) CreatedAt() time.Time { return goal.createdAt }
func (goal Goal) IsTerminal() bool     { return goal.state.Terminal() }
func (goal Goal) WorkItemCount() int   { return len(goal.itemOrder) }

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

func (goal Goal) AddWorkItem(expected Revision, item WorkItem) (Goal, error) {
	if err := goal.expectRevision(expected); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStatePending {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
	}
	if !validWorkItemRef(item.ref) {
		return Goal{}, domainError(ErrorInvalidRef, "work_item_ref")
	}
	if item.goal != goal.ref || item.actor != goal.actor || item.project != goal.project {
		return Goal{}, domainError(ErrorScopeConflict, "work_item_scope")
	}
	if item.state != WorkItemStatePending || item.revision != 1 {
		return Goal{}, domainError(ErrorInvalidArgument, "work_item_snapshot")
	}
	if item.createdAt.Before(goal.createdAt) {
		return Goal{}, domainError(ErrorInvalidArgument, "work_item_created_at")
	}
	if _, exists := goal.items[item.ref]; exists {
		return Goal{}, domainError(ErrorDuplicateWorkItem, "work_item_ref")
	}

	updated := goal.clone()
	updated.items[item.ref] = item.clone()
	updated.itemOrder = append(updated.itemOrder, item.ref)
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
	item, err = item.Start(expectedItem, execution, at)
	if err != nil {
		return Goal{}, err
	}
	return goal.withUpdatedWorkItem(item), nil
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
	return goal.withUpdatedWorkItem(item), nil
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

	allSucceeded := true
	hasFailed := false
	latestFinishedAt := goal.startedAt
	for _, item := range goal.items {
		if !item.IsTerminal() {
			return Goal{}, domainError(ErrorWorkItemsNotTerminal, "work_items")
		}
		if item.state != WorkItemStateSucceeded {
			allSucceeded = false
		}
		if item.state == WorkItemStateFailed {
			hasFailed = true
		}
		if item.finishedAt.After(latestFinishedAt) {
			latestFinishedAt = item.finishedAt
		}
	}
	if outcome == GoalOutcomeSucceeded && !allSucceeded {
		return Goal{}, domainError(ErrorOutcomeConflict, "goal_outcome")
	}
	if outcome == GoalOutcomeFailed && !hasFailed {
		return Goal{}, domainError(ErrorOutcomeConflict, "goal_outcome")
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
	updated.revision++
	updated.closedAt = canonicalTime(at)
	return updated, nil
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

func (goal Goal) clone() Goal {
	items := make(map[WorkItemRef]WorkItem, len(goal.items))
	for ref, item := range goal.items {
		items[ref] = item.clone()
	}
	goal.items = items
	goal.itemOrder = append([]WorkItemRef(nil), goal.itemOrder...)
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
