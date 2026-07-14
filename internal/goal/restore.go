package goal

import (
	"strings"
	"time"
)

// RestoreIntentManifest validates persisted data and recomputes its hash before
// returning a domain value. Persisted hashes are never trusted as constructors.
func RestoreIntentManifest(snapshot IntentManifestSnapshot) (IntentManifest, error) {
	ref, err := NewIntentRef(snapshot.Ref)
	if err != nil {
		return IntentManifest{}, err
	}
	actor, err := NewActorRef(snapshot.ActorRef)
	if err != nil {
		return IntentManifest{}, err
	}
	project, err := NewProjectRef(snapshot.ProjectRef)
	if err != nil {
		return IntentManifest{}, err
	}
	manifest, err := NewIntentManifest(IntentManifestInput{
		Ref:         ref,
		Actor:       actor,
		Project:     project,
		Statement:   snapshot.Statement,
		SubmittedAt: snapshot.SubmittedAt,
	})
	if err != nil {
		return IntentManifest{}, err
	}
	if snapshot.Hash != manifest.Hash() {
		return IntentManifest{}, domainError(ErrorIntentHashMismatch, "intent_hash")
	}
	return manifest, nil
}

// RestoreGoal rehydrates a complete aggregate only after every persisted
// invariant has been checked. It performs no I/O and depends on no adapter.
func RestoreGoal(snapshot GoalSnapshot) (Goal, error) {
	intent, err := RestoreIntentManifest(snapshot.Intent)
	if err != nil {
		return Goal{}, err
	}
	ref, err := NewGoalRef(snapshot.Ref)
	if err != nil {
		return Goal{}, err
	}
	actor, err := NewActorRef(snapshot.ActorRef)
	if err != nil {
		return Goal{}, err
	}
	project, err := NewProjectRef(snapshot.ProjectRef)
	if err != nil {
		return Goal{}, err
	}
	if actor != intent.Actor() || project != intent.Project() {
		return Goal{}, domainError(ErrorScopeConflict, "goal_scope")
	}
	if snapshot.Revision < 1 {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_revision")
	}
	if !validGoalState(snapshot.State) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_state")
	}
	if snapshot.CreatedAt.IsZero() || canonicalTime(snapshot.CreatedAt).Before(intent.SubmittedAt()) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_created_at")
	}

	restored := Goal{
		ref:            ref,
		actor:          actor,
		project:        project,
		intentManifest: intent,
		state:          snapshot.State,
		revision:       snapshot.Revision,
		createdAt:      canonicalTime(snapshot.CreatedAt),
		startedAt:      canonicalOptionalTime(snapshot.StartedAt),
		closedAt:       canonicalOptionalTime(snapshot.ClosedAt),
		items:          make(map[WorkItemRef]WorkItem, len(snapshot.WorkItems)),
		itemOrder:      make([]WorkItemRef, 0, len(snapshot.WorkItems)),
	}
	for _, itemSnapshot := range snapshot.WorkItems {
		item, restoreErr := restoreWorkItem(itemSnapshot)
		if restoreErr != nil {
			return Goal{}, restoreErr
		}
		if _, duplicate := restored.items[item.ref]; duplicate {
			return Goal{}, domainError(ErrorDuplicateWorkItem, "work_item_ref")
		}
		if item.goal != restored.ref || item.actor != restored.actor || item.project != restored.project {
			return Goal{}, domainError(ErrorScopeConflict, "work_item_scope")
		}
		if item.createdAt.Before(restored.createdAt) {
			return Goal{}, domainError(ErrorSnapshotInvalid, "work_item_created_at")
		}
		restored.items[item.ref] = item
		restored.itemOrder = append(restored.itemOrder, item.ref)
	}
	if err := validateRestoredGoal(restored); err != nil {
		return Goal{}, err
	}
	if restored.revision != revisionForGoalSnapshot(restored) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_revision")
	}
	return restored, nil
}

func restoreWorkItem(snapshot WorkItemSnapshot) (WorkItem, error) {
	ref, err := NewWorkItemRef(snapshot.Ref)
	if err != nil {
		return WorkItem{}, err
	}
	goalRef, err := NewGoalRef(snapshot.GoalRef)
	if err != nil {
		return WorkItem{}, err
	}
	actor, err := NewActorRef(snapshot.ActorRef)
	if err != nil {
		return WorkItem{}, err
	}
	project, err := NewProjectRef(snapshot.ProjectRef)
	if err != nil {
		return WorkItem{}, err
	}
	if strings.TrimSpace(snapshot.Objective) == "" {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_objective")
	}
	if !validWorkItemState(snapshot.State) {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_state")
	}
	if snapshot.Revision != revisionForWorkItemState(snapshot.State) {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_revision")
	}
	if snapshot.CreatedAt.IsZero() {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_created_at")
	}

	var execution ExecutionRef
	if snapshot.ExecutionRef != "" {
		execution, err = NewExecutionRef(snapshot.ExecutionRef)
		if err != nil {
			return WorkItem{}, err
		}
	}
	artifacts := make([]ArtifactRef, 0, len(snapshot.ArtifactRefs))
	for _, value := range snapshot.ArtifactRefs {
		artifact, artifactErr := NewArtifactRef(value)
		if artifactErr != nil {
			return WorkItem{}, artifactErr
		}
		artifacts = append(artifacts, artifact)
	}
	if err := validateArtifactRefs(artifacts); err != nil {
		return WorkItem{}, err
	}
	attestations := make([]AttestationRef, 0, len(snapshot.AttestationRefs))
	for _, value := range snapshot.AttestationRefs {
		attestation, attestationErr := NewAttestationRef(value)
		if attestationErr != nil {
			return WorkItem{}, attestationErr
		}
		attestations = append(attestations, attestation)
	}
	if err := validateAttestationRefs(attestations); err != nil {
		return WorkItem{}, err
	}

	restored := WorkItem{
		ref:          ref,
		goal:         goalRef,
		actor:        actor,
		project:      project,
		objective:    snapshot.Objective,
		state:        snapshot.State,
		revision:     snapshot.Revision,
		createdAt:    canonicalTime(snapshot.CreatedAt),
		startedAt:    canonicalOptionalTime(snapshot.StartedAt),
		finishedAt:   canonicalOptionalTime(snapshot.FinishedAt),
		execution:    execution,
		artifacts:    artifacts,
		attestations: attestations,
	}
	if err := validateRestoredWorkItem(restored); err != nil {
		return WorkItem{}, err
	}
	return restored, nil
}

func validateRestoredWorkItem(item WorkItem) error {
	hasExecution := validExecutionRef(item.execution)
	hasStarted := !item.startedAt.IsZero()
	hasFinished := !item.finishedAt.IsZero()
	hasArtifacts := len(item.artifacts) > 0
	hasAttestations := len(item.attestations) > 0

	switch item.state {
	case WorkItemStatePending:
		if hasExecution || hasStarted || hasFinished || hasArtifacts || hasAttestations {
			return domainError(ErrorSnapshotInvalid, "pending_work_item")
		}
	case WorkItemStateRunning:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			hasFinished || hasArtifacts || hasAttestations {
			return domainError(ErrorSnapshotInvalid, "running_work_item")
		}
	case WorkItemStateSucceeded:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			!validTransitionTime(item.finishedAt, item.startedAt) || !hasArtifacts || !hasAttestations {
			return domainError(ErrorSnapshotInvalid, "succeeded_work_item")
		}
	case WorkItemStateFailed:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			!validTransitionTime(item.finishedAt, item.startedAt) || hasArtifacts || hasAttestations {
			return domainError(ErrorSnapshotInvalid, "failed_work_item")
		}
	}
	return nil
}

func validateRestoredGoal(goal Goal) error {
	hasStarted := !goal.startedAt.IsZero()
	hasClosed := !goal.closedAt.IsZero()

	switch goal.state {
	case GoalStatePending:
		if hasStarted || hasClosed {
			return domainError(ErrorSnapshotInvalid, "pending_goal")
		}
		for _, item := range goal.items {
			if item.state != WorkItemStatePending {
				return domainError(ErrorSnapshotInvalid, "pending_goal_work_items")
			}
		}
	case GoalStateRunning:
		if len(goal.items) == 0 || !validTransitionTime(goal.startedAt, goal.createdAt) || hasClosed {
			return domainError(ErrorSnapshotInvalid, "running_goal")
		}
		if err := validateStartedGoalWorkItems(goal, false); err != nil {
			return err
		}
	case GoalStateSucceeded:
		if len(goal.items) == 0 || !validTransitionTime(goal.startedAt, goal.createdAt) ||
			!validTransitionTime(goal.closedAt, goal.startedAt) {
			return domainError(ErrorSnapshotInvalid, "succeeded_goal")
		}
		if err := validateStartedGoalWorkItems(goal, true); err != nil {
			return err
		}
		for _, item := range goal.items {
			if item.state != WorkItemStateSucceeded {
				return domainError(ErrorOutcomeConflict, "succeeded_goal_work_items")
			}
		}
	case GoalStateFailed:
		if len(goal.items) == 0 || !validTransitionTime(goal.startedAt, goal.createdAt) ||
			!validTransitionTime(goal.closedAt, goal.startedAt) {
			return domainError(ErrorSnapshotInvalid, "failed_goal")
		}
		if err := validateStartedGoalWorkItems(goal, true); err != nil {
			return err
		}
		hasFailed := false
		for _, item := range goal.items {
			if item.state == WorkItemStateFailed {
				hasFailed = true
			}
		}
		if !hasFailed {
			return domainError(ErrorOutcomeConflict, "failed_goal_work_items")
		}
	}
	return nil
}

func validateStartedGoalWorkItems(goal Goal, requireTerminal bool) error {
	for _, item := range goal.items {
		if item.createdAt.After(goal.startedAt) {
			return domainError(ErrorSnapshotInvalid, "goal_started_at")
		}
		if item.state != WorkItemStatePending && item.startedAt.Before(goal.startedAt) {
			return domainError(ErrorSnapshotInvalid, "work_item_started_at")
		}
		if requireTerminal {
			if !item.IsTerminal() {
				return domainError(ErrorWorkItemsNotTerminal, "work_items")
			}
			if item.finishedAt.After(goal.closedAt) {
				return domainError(ErrorSnapshotInvalid, "goal_closed_at")
			}
		}
	}
	return nil
}

func validGoalState(state GoalState) bool {
	return state == GoalStatePending || state == GoalStateRunning || state == GoalStateSucceeded || state == GoalStateFailed
}

func validWorkItemState(state WorkItemState) bool {
	return state == WorkItemStatePending || state == WorkItemStateRunning ||
		state == WorkItemStateSucceeded || state == WorkItemStateFailed
}

func revisionForWorkItemState(state WorkItemState) Revision {
	switch state {
	case WorkItemStatePending:
		return 1
	case WorkItemStateRunning:
		return 2
	case WorkItemStateSucceeded, WorkItemStateFailed:
		return 3
	default:
		return 0
	}
}

func revisionForGoalSnapshot(goal Goal) Revision {
	revision := Revision(1 + len(goal.items))
	if goal.state != GoalStatePending {
		revision++
	}
	for _, item := range goal.items {
		switch item.state {
		case WorkItemStateRunning:
			revision++
		case WorkItemStateSucceeded, WorkItemStateFailed:
			revision += 2
		}
	}
	if goal.state.Terminal() {
		revision++
	}
	return revision
}

func canonicalOptionalTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return canonicalTime(value)
}
