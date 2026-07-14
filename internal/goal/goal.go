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
	appSpec        AppSpec
	state          GoalState
	revision       Revision
	createdAt      time.Time
	startedAt      time.Time
	closedAt       time.Time
	planGeneration PlanGeneration
	phases         []PhaseInstance
	items          map[WorkItemRef]WorkItem
	itemOrder      []WorkItemRef
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

// ApplyPlan atomically replaces a pending Goal plan. Goal revision is the CAS;
// the proposal generation must be exactly current+1. It never changes
// lifecycle by itself.
func (goal Goal) ApplyPlan(expectedGoal Revision, plan Plan) (Goal, error) {
	if err := goal.expectRevision(expectedGoal); err != nil {
		return Goal{}, err
	}
	if goal.state != GoalStatePending {
		return Goal{}, domainError(ErrorInvalidTransition, "goal_state")
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

// ReadyWorkItems derives every currently eligible item. It does not choose a
// cohort: overlapping pending candidates remain visible until one is started
// through Goal CAS, after which conflicts with that running item are excluded.
func (goal Goal) ReadyWorkItems() []WorkItem {
	if goal.state != GoalStateRunning {
		return nil
	}

	runningWriteSets := make([][]WriteScope, 0)
	for _, ref := range goal.itemOrder {
		item := goal.items[ref]
		if item.state == WorkItemStateRunning {
			runningWriteSets = append(runningWriteSets, item.writeSet)
		}
	}

	ready := make([]WorkItem, 0)
	for _, ref := range goal.itemOrder {
		item := goal.items[ref]
		if item.state != WorkItemStatePending || !goal.dependenciesSucceeded(item) {
			continue
		}
		if conflictsWithWriteSets(item.writeSet, runningWriteSets) {
			continue
		}
		ready = append(ready, item.clone())
	}
	return ready
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
	updated := goal.clone()
	updated.items[item.ref] = item
	for {
		changed := false
		for _, candidateRef := range updated.itemOrder {
			candidate := updated.items[candidateRef]
			if candidate.state != WorkItemStatePending || !updated.hasFailedDependency(candidate) {
				continue
			}
			candidate, err = candidate.skipDependencyFailed(candidate.revision, at)
			if err != nil {
				return Goal{}, err
			}
			updated.items[candidateRef] = candidate
			changed = true
		}
		if !changed {
			break
		}
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
	updated.revision++
	updated.closedAt = canonicalTime(at)
	return updated, nil
}

// ClosableOutcome derives closure eligibility without duplicating lifecycle
// rules in application or schedulers.
func (goal Goal) ClosableOutcome() (GoalOutcome, bool) {
	if goal.state != GoalStateRunning || len(goal.items) == 0 {
		return "", false
	}
	allSucceeded := true
	hasFailed := false
	for _, item := range goal.items {
		if !item.IsTerminal() {
			return "", false
		}
		allSucceeded = allSucceeded && item.state == WorkItemStateSucceeded
		hasFailed = hasFailed || item.state == WorkItemStateFailed
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
	item, exists := goal.items[ref]
	if !exists || item.state != WorkItemStatePending || !goal.dependenciesSucceeded(item) {
		return false
	}
	for _, runningRef := range goal.itemOrder {
		running := goal.items[runningRef]
		if running.state == WorkItemStateRunning && writeSetsOverlap(item.writeSet, running.writeSet) {
			return false
		}
	}
	return true
}

func (goal Goal) dependenciesSucceeded(item WorkItem) bool {
	for _, dependency := range item.dependencies {
		dependencyItem, exists := goal.items[dependency]
		if !exists || dependencyItem.state != WorkItemStateSucceeded {
			return false
		}
	}
	return true
}

func (goal Goal) hasFailedDependency(item WorkItem) bool {
	for _, dependency := range item.dependencies {
		dependencyItem, exists := goal.items[dependency]
		if exists && (dependencyItem.state == WorkItemStateFailed || dependencyItem.state == WorkItemStateSkipped) {
			return true
		}
	}
	return false
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
