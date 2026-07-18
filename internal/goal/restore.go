package goal

import (
	"strings"
	"time"

	"orquesta/internal/governance"
)

// RestoreIntentManifest validates persisted data and recomputes its hash.
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
		Ref: ref, Actor: actor, Project: project,
		Statement: snapshot.Statement, SubmittedAt: snapshot.SubmittedAt,
	})
	if err != nil {
		return IntentManifest{}, err
	}
	if snapshot.Hash != manifest.Hash() {
		return IntentManifest{}, domainError(ErrorIntentHashMismatch, "intent_hash")
	}
	return manifest, nil
}

// RestoreAppSpec accepts a complete immutable generation, recomputes both the
// nested Intent hash and the AppSpec hash, and rejects altered persistence.
func RestoreAppSpec(snapshot AppSpecSnapshot) (AppSpec, error) {
	intent, err := RestoreIntentManifest(snapshot.Intent)
	if err != nil {
		return AppSpec{}, err
	}
	ref, err := NewAppSpecRef(snapshot.Ref)
	if err != nil {
		return AppSpec{}, err
	}
	confirmedBy, err := NewActorRef(snapshot.ConfirmedBy)
	if err != nil {
		return AppSpec{}, err
	}
	var parentRef AppSpecRef
	if snapshot.ParentRef != "" {
		parentRef, err = NewAppSpecRef(snapshot.ParentRef)
		if err != nil {
			return AppSpec{}, err
		}
	}
	if snapshot.Generation == 0 {
		return AppSpec{}, domainError(ErrorSnapshotInvalid, "app_spec_generation")
	}
	spec := AppSpec{
		ref: ref, generation: snapshot.Generation, intent: intent,
		parentRef: parentRef, parentHash: snapshot.ParentHash,
		objective: snapshot.Objective, reason: snapshot.Reason,
		confirmedBy: confirmedBy, confirmedAt: canonicalTime(snapshot.ConfirmedAt),
	}
	if err := validateAppSpecInput(AppSpecInput{
		Ref: ref, Intent: intent, Objective: spec.objective, Reason: spec.reason,
		ConfirmedBy: confirmedBy, ConfirmedAt: spec.confirmedAt,
	}); err != nil {
		return AppSpec{}, err
	}
	if spec.isRoot() {
		if validAppSpecRef(parentRef) || snapshot.ParentHash != "" {
			return AppSpec{}, domainError(ErrorSnapshotInvalid, "app_spec_parent")
		}
	} else if !validAppSpecRef(parentRef) || parentRef == ref || !validCanonicalSHA256(snapshot.ParentHash) {
		return AppSpec{}, domainError(ErrorSnapshotInvalid, "app_spec_parent")
	}
	expectedHash := hashAppSpec(spec)
	if snapshot.Hash != expectedHash {
		return AppSpec{}, domainError(ErrorAppSpecHashMismatch, "app_spec_hash")
	}
	spec.hash = expectedHash
	return spec, nil
}

// RestoreGoal accepts the current complete schema and the immediately previous
// schema whose WorkItems predate immutable governance metadata. Every emitted
// snapshot uses the current schema, so compatibility is one-way and bounded.
func RestoreGoal(snapshot GoalSnapshot) (Goal, error) {
	if snapshot.SchemaVersion != GoalSnapshotSchemaVersion &&
		snapshot.SchemaVersion != governanceCompatibleSnapshotSchemaVersion {
		return Goal{}, domainError(ErrorSnapshotInvalid, "schema_version")
	}
	spec, specErr := RestoreAppSpec(snapshot.AppSpec)
	intent := spec.Intent()
	ref, refErr := NewGoalRef(snapshot.Ref)
	actor, actorErr := NewActorRef(snapshot.ActorRef)
	project, projectErr := NewProjectRef(snapshot.ProjectRef)
	if err := firstError(specErr, refErr, actorErr, projectErr); err != nil {
		return Goal{}, err
	}
	if actor != intent.Actor() || project != intent.Project() {
		return Goal{}, domainError(ErrorScopeConflict, "goal_scope")
	}
	if snapshot.Revision < 1 || !validGoalState(snapshot.State) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_header")
	}
	if snapshot.CreatedAt.IsZero() || canonicalTime(snapshot.CreatedAt).Before(spec.ConfirmedAt()) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_created_at")
	}

	phases, err := restorePhases(snapshot.Phases)
	if err != nil {
		return Goal{}, err
	}
	if snapshot.PlanGeneration == 0 {
		if len(phases) != 0 || len(snapshot.WorkItems) != 0 {
			return Goal{}, domainError(ErrorSnapshotInvalid, "plan")
		}
	} else if len(phases) == 0 || len(snapshot.WorkItems) == 0 {
		return Goal{}, domainError(ErrorSnapshotInvalid, "plan")
	}

	restored := Goal{
		ref: ref, actor: actor, project: project, appSpec: spec,
		state: snapshot.State, revision: snapshot.Revision,
		paused: snapshot.Paused, cancelRequested: snapshot.CancelRequested,
		controlSequence: snapshot.ControlSequence,
		createdAt:       canonicalTime(snapshot.CreatedAt),
		startedAt:       canonicalOptionalTime(snapshot.StartedAt),
		closedAt:        canonicalOptionalTime(snapshot.ClosedAt),
		planGeneration:  snapshot.PlanGeneration, phases: phases,
		items:     make(map[WorkItemRef]WorkItem, len(snapshot.WorkItems)),
		itemOrder: make([]WorkItemRef, 0, len(snapshot.WorkItems)),
	}
	if err := restoreGoalItems(snapshot, &restored); err != nil {
		return Goal{}, err
	}
	childHandoffs, err := restoreChildHandoffResolutions(snapshot.ChildHandoffResolutions)
	if err != nil {
		return Goal{}, err
	}
	restored.childHandoffs = childHandoffs
	if restored.planGeneration > 0 {
		if err := validateRestoredPlan(Plan{
			generation: restored.planGeneration,
			phases:     restored.phases,
			items:      restored.WorkItems(),
		}); err != nil {
			return Goal{}, err
		}
	}
	if err := validateRestoredGoal(restored); err != nil {
		return Goal{}, err
	}
	if restored.revision != revisionForGoalSnapshot(restored) {
		return Goal{}, domainError(ErrorSnapshotInvalid, "goal_revision")
	}
	return restored, nil
}

func restoreGoalItems(snapshot GoalSnapshot, restored *Goal) error {
	for _, itemSnapshot := range snapshot.WorkItems {
		item, err := restoreWorkItem(itemSnapshot, snapshot.SchemaVersion)
		if err != nil {
			return err
		}
		if _, duplicate := restored.items[item.ref]; duplicate {
			return domainError(ErrorDuplicateWorkItem, "work_item_ref")
		}
		if item.goal != restored.ref || item.actor != restored.actor || item.project != restored.project {
			return domainError(ErrorScopeConflict, "work_item_scope")
		}
		if item.createdAt.Before(restored.createdAt) {
			return domainError(ErrorSnapshotInvalid, "work_item_created_at")
		}
		restored.items[item.ref] = item
		restored.itemOrder = append(restored.itemOrder, item.ref)
	}
	return nil
}

func restoreWorkItem(snapshot WorkItemSnapshot, schemaVersion uint32) (WorkItem, error) {
	ref, refErr := NewWorkItemRef(snapshot.Ref)
	goalRef, goalErr := NewGoalRef(snapshot.GoalRef)
	actor, actorErr := NewActorRef(snapshot.ActorRef)
	project, projectErr := NewProjectRef(snapshot.ProjectRef)
	if err := firstError(refErr, goalErr, actorErr, projectErr); err != nil {
		return WorkItem{}, err
	}
	if strings.TrimSpace(snapshot.Objective) == "" || !validWorkItemState(snapshot.State) {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_header")
	}
	if !validRestoredWorkItemRevision(snapshot.State, snapshot.Revision, snapshot.ControlSequence) || snapshot.CreatedAt.IsZero() {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "work_item_revision")
	}
	phase, phaseErr := NewPhaseKey(snapshot.PhaseKey)
	role, roleErr := NewRoleKey(snapshot.RoleKey)
	outputContract, outputErr := NewOutputContract(snapshot.OutputContract)
	if err := firstError(phaseErr, roleErr, outputErr); err != nil {
		return WorkItem{}, err
	}
	dependencies, dependenciesErr := restoreWorkItemRefs(snapshot.DependencyRefs)
	writeSet, writeSetErr := restoreWriteSet(snapshot.WriteSet)
	parent, parentErr := restoreOptionalWorkItemRef(snapshot.ParentRef)
	if err := firstError(dependenciesErr, writeSetErr, parentErr); err != nil {
		return WorkItem{}, err
	}
	if snapshot.HandoffRequired == nil {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "handoff_required")
	}
	handoffRequired := *snapshot.HandoffRequired
	if handoffRequired && !validWorkItemRef(parent) {
		return WorkItem{}, domainError(ErrorSnapshotInvalid, "handoff_parent")
	}
	skillRefs, skillErr := restoreSkillRefs(snapshot.SkillRefs)
	toolRefs, toolErr := restoreToolRefs(snapshot.ToolRefs)
	capabilityRefs, capabilityErr := restoreCapabilityRefs(snapshot.CapabilityRefs)
	if err := firstError(skillErr, toolErr, capabilityErr); err != nil {
		return WorkItem{}, err
	}
	budgetDemand, criticality, effort, governanceErr := restoreWorkItemGovernance(snapshot, schemaVersion, ref)
	if governanceErr != nil {
		return WorkItem{}, governanceErr
	}
	execution, executionErr := restoreExecutionRef(snapshot.ExecutionRef)
	reworkOf, reworkErr := restoreOptionalWorkItemRef(snapshot.ReworkOf)
	artifacts, artifactsErr := restoreArtifactRefs(snapshot.ArtifactRefs)
	attestations, attestationsErr := restoreAttestationRefs(snapshot.AttestationRefs)
	if err := firstError(executionErr, reworkErr, artifactsErr, attestationsErr); err != nil {
		return WorkItem{}, err
	}

	restored := WorkItem{
		ref: ref, goal: goalRef, actor: actor, project: project,
		objective: snapshot.Objective, phase: phase, role: role,
		parent: parent, handoffRequired: handoffRequired,
		dependencies: dependencies, writeSet: writeSet,
		skillRefs: skillRefs, toolRefs: toolRefs, capabilityRefs: capabilityRefs,
		outputContract: outputContract, skipReason: snapshot.SkipReason,
		budgetDemand: budgetDemand, securityCriticality: criticality, reasoningEffort: effort,
		interruptCause: snapshot.InterruptCause, reworkOf: reworkOf,
		state: snapshot.State, revision: snapshot.Revision,
		paused: snapshot.Paused, cancelRequested: snapshot.CancelRequested,
		controlSequence: snapshot.ControlSequence,
		createdAt:       canonicalTime(snapshot.CreatedAt),
		startedAt:       canonicalOptionalTime(snapshot.StartedAt),
		interruptedAt:   canonicalOptionalTime(snapshot.InterruptedAt),
		finishedAt:      canonicalOptionalTime(snapshot.FinishedAt),
		execution:       execution, artifacts: artifacts, attestations: attestations,
	}
	if err := validateWorkItemPlanMetadata(restored); err != nil {
		return WorkItem{}, err
	}
	if err := validateRestoredWorkItem(restored); err != nil {
		return WorkItem{}, err
	}
	return restored, nil
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func restoreWorkItemGovernance(
	snapshot WorkItemSnapshot,
	schemaVersion uint32,
	ref WorkItemRef,
) (governance.BudgetDemand, governance.SecurityCriticality, governance.ReasoningEffort, error) {
	if schemaVersion == governanceCompatibleSnapshotSchemaVersion {
		if snapshot.BudgetDemand != (governance.BudgetDemand{}) || snapshot.SecurityCriticality != "" ||
			snapshot.ReasoningEffort != "" {
			return governance.BudgetDemand{}, "", "", domainError(ErrorSnapshotInvalid, "work_item_governance_schema")
		}
		return governance.BudgetDemand{Ref: "budget-demand:" + ref.String()},
			governance.SecurityCriticalityNormal, governance.ReasoningEffortMedium, nil
	}
	if governance.ValidateBudgetDemand(snapshot.BudgetDemand) != nil {
		return governance.BudgetDemand{}, "", "", domainError(ErrorSnapshotInvalid, "budget_demand")
	}
	if governance.ValidateSecurityCriticality(snapshot.SecurityCriticality) != nil {
		return governance.BudgetDemand{}, "", "", domainError(ErrorSnapshotInvalid, "security_criticality")
	}
	if governance.ValidateReasoningEffort(snapshot.ReasoningEffort) != nil {
		return governance.BudgetDemand{}, "", "", domainError(ErrorSnapshotInvalid, "reasoning_effort")
	}
	return snapshot.BudgetDemand, snapshot.SecurityCriticality, snapshot.ReasoningEffort, nil
}

func validateRestoredWorkItem(item WorkItem) error {
	hasExecution := validExecutionRef(item.execution)
	hasStarted := !item.startedAt.IsZero()
	hasFinished := !item.finishedAt.IsZero()
	hasInterrupted := !item.interruptedAt.IsZero()
	hasArtifacts := len(item.artifacts) > 0
	hasAttestations := len(item.attestations) > 0
	if item.state != WorkItemStateSkipped && item.skipReason != "" {
		return domainError(ErrorSnapshotInvalid, "skip_reason")
	}
	if (item.paused || item.cancelRequested) && item.controlSequence == 0 ||
		item.paused && (item.cancelRequested || item.IsTerminal()) {
		return domainError(ErrorSnapshotInvalid, "work_item_control")
	}
	if item.state != WorkItemStateInterrupted && item.state != WorkItemStateSuperseded &&
		item.state != WorkItemStateCanceled && (item.interruptCause != "" || hasInterrupted) {
		return domainError(ErrorSnapshotInvalid, "interrupt_cause")
	}

	switch item.state {
	case WorkItemStatePending:
		if hasExecution || hasStarted || hasInterrupted || hasFinished || hasArtifacts || hasAttestations ||
			item.cancelRequested {
			return domainError(ErrorSnapshotInvalid, "pending_work_item")
		}
	case WorkItemStateRunning:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) || hasInterrupted ||
			hasFinished || hasArtifacts || hasAttestations {
			return domainError(ErrorSnapshotInvalid, "running_work_item")
		}
	case WorkItemStateSucceeded:
		requiresArtifacts := item.outputContract.kind != OutputContractAttestation
		requiresAttestations := item.outputContract.kind != OutputContractArtifact
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			!validTransitionTime(item.finishedAt, item.startedAt) ||
			(requiresArtifacts && !hasArtifacts) || (requiresAttestations && !hasAttestations) ||
			item.cancelRequested {
			return domainError(ErrorSnapshotInvalid, "succeeded_work_item")
		}
	case WorkItemStateFailed:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			!validTransitionTime(item.finishedAt, item.startedAt) || hasArtifacts || hasAttestations ||
			item.cancelRequested {
			return domainError(ErrorSnapshotInvalid, "failed_work_item")
		}
	case WorkItemStateSkipped:
		if (item.skipReason != WorkItemSkipReasonDependencyFailed && item.skipReason != WorkItemSkipReasonDependencyCanceled) ||
			hasExecution || hasStarted || hasInterrupted || !validTransitionTime(item.finishedAt, item.createdAt) ||
			hasArtifacts || hasAttestations || item.cancelRequested {
			return domainError(ErrorSnapshotInvalid, "skipped_work_item")
		}
	case WorkItemStateInterrupted:
		if !hasExecution || !validTransitionTime(item.startedAt, item.createdAt) ||
			!validTransitionTime(item.interruptedAt, item.startedAt) || hasFinished || hasArtifacts || hasAttestations ||
			!validInterruptCause(item.interruptCause) || item.cancelRequested ||
			item.controlSequence == 0 {
			return domainError(ErrorSnapshotInvalid, "interrupted_work_item")
		}
	case WorkItemStateCanceled:
		if !item.cancelRequested || !hasFinished || hasArtifacts || hasAttestations ||
			(hasStarted != hasExecution) || (hasStarted && !validTransitionTime(item.startedAt, item.createdAt)) ||
			!validTransitionTime(item.finishedAt, chooseWorkItemNotBefore(item)) ||
			(hasInterrupted != (item.interruptCause != "")) || (hasInterrupted && !validInterruptCause(item.interruptCause)) ||
			item.controlSequence == 0 {
			return domainError(ErrorSnapshotInvalid, "canceled_work_item")
		}
	case WorkItemStateSuperseded:
		if !hasFinished || item.cancelRequested || hasArtifacts || hasAttestations ||
			(hasStarted != hasExecution) || (hasStarted && !validTransitionTime(item.startedAt, item.createdAt)) ||
			!validTransitionTime(item.finishedAt, chooseWorkItemNotBefore(item)) ||
			(hasInterrupted != (item.interruptCause != "")) || (hasInterrupted && !validInterruptCause(item.interruptCause)) ||
			item.controlSequence == 0 {
			return domainError(ErrorSnapshotInvalid, "superseded_work_item")
		}
	}
	return nil
}

func validateRestoredGoal(goal Goal) error {
	if !validAppSpec(goal.appSpec) || goal.actor != goal.appSpec.Intent().Actor() ||
		goal.project != goal.appSpec.Intent().Project() {
		return domainError(ErrorScopeConflict, "app_spec_scope")
	}
	hasStarted := !goal.startedAt.IsZero()
	hasClosed := !goal.closedAt.IsZero()
	if (goal.paused || goal.cancelRequested) && goal.controlSequence == 0 || goal.paused && goal.cancelRequested {
		return domainError(ErrorSnapshotInvalid, "goal_control")
	}
	switch goal.state {
	case GoalStatePending:
		if hasStarted || hasClosed {
			return domainError(ErrorSnapshotInvalid, "pending_goal")
		}
		for _, item := range goal.items {
			if item.state != WorkItemStatePending && !(goal.cancelRequested && item.state == WorkItemStateCanceled) {
				return domainError(ErrorSnapshotInvalid, "pending_goal_work_items")
			}
		}
	case GoalStateRunning:
		if len(goal.items) == 0 || !validTransitionTime(goal.startedAt, goal.createdAt) || hasClosed {
			return domainError(ErrorSnapshotInvalid, "running_goal")
		}
	case GoalStateSucceeded, GoalStateFailed:
		if len(goal.items) == 0 || !validTransitionTime(goal.startedAt, goal.createdAt) ||
			!validTransitionTime(goal.closedAt, goal.startedAt) || goal.paused || goal.cancelRequested {
			return domainError(ErrorSnapshotInvalid, "closed_goal")
		}
	case GoalStateCanceled:
		if !goal.cancelRequested || goal.controlSequence < 2 || !validTransitionTime(goal.closedAt, goal.createdAt) {
			return domainError(ErrorSnapshotInvalid, "canceled_goal")
		}
	}
	if goal.state != GoalStatePending && (goal.state != GoalStateCanceled || hasStarted) {
		if err := validateStartedGoalWorkItems(goal, goal.state.Terminal()); err != nil {
			return err
		}
	}
	if err := validateRestoredChildHandoffs(goal); err != nil {
		return err
	}
	if goal.state == GoalStateSucceeded {
		for ref := range goal.items {
			outcome, resolved := goal.LogicalWorkItemOutcome(ref)
			if !resolved || outcome != WorkItemLogicalSucceeded {
				return domainError(ErrorOutcomeConflict, "succeeded_goal_work_items")
			}
		}
	}
	if goal.state == GoalStateFailed {
		hasFailed := false
		for ref := range goal.items {
			outcome, resolved := goal.LogicalWorkItemOutcome(ref)
			hasFailed = hasFailed || (resolved && outcome == WorkItemLogicalFailed)
		}
		if !hasFailed {
			return domainError(ErrorOutcomeConflict, "failed_goal_work_items")
		}
	}
	if goal.state == GoalStateCanceled {
		for _, item := range goal.items {
			if !item.IsTerminal() || item.finishedAt.After(goal.closedAt) {
				return domainError(ErrorWorkItemsNotTerminal, "work_items")
			}
		}
	}
	if goal.controlSequence > uint64(goal.revision) {
		return domainError(ErrorSnapshotInvalid, "control_sequence")
	}
	for _, item := range goal.items {
		if item.controlSequence > goal.controlSequence {
			return domainError(ErrorSnapshotInvalid, "work_item_control_sequence")
		}
		if goal.cancelRequested && !item.IsTerminal() &&
			(item.state != WorkItemStateRunning || !item.cancelRequested) {
			return domainError(ErrorSnapshotInvalid, "uncascaded_cancel")
		}
		failedDependency := goal.hasFailedDependency(item)
		if item.state == WorkItemStateSkipped && !failedDependency {
			return domainError(ErrorSnapshotInvalid, "skipped_dependency")
		}
		if item.state == WorkItemStatePending && failedDependency {
			return domainError(ErrorSnapshotInvalid, "uncascaded_dependency")
		}
		if item.state == WorkItemStateSkipped {
			reason, blocked := dependencySkipReasonIn(goal.items, item)
			if !blocked || reason != item.skipReason {
				return domainError(ErrorSnapshotInvalid, "skip_reason")
			}
		}
	}
	return nil
}

func validateStartedGoalWorkItems(goal Goal, requireTerminal bool) error {
	for _, item := range goal.items {
		if item.state == WorkItemStateSkipped ||
			((item.state == WorkItemStateCanceled || item.state == WorkItemStateSuperseded) && item.startedAt.IsZero()) {
			if item.finishedAt.Before(goal.startedAt) {
				return domainError(ErrorSnapshotInvalid, "work_item_finished_at")
			}
		} else if item.state != WorkItemStatePending && item.startedAt.Before(goal.startedAt) {
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

func chooseWorkItemNotBefore(item WorkItem) time.Time {
	if !item.interruptedAt.IsZero() {
		return item.interruptedAt
	}
	if !item.startedAt.IsZero() {
		return item.startedAt
	}
	return item.createdAt
}

func validGoalState(state GoalState) bool {
	return state == GoalStatePending || state == GoalStateRunning || state == GoalStateSucceeded ||
		state == GoalStateFailed || state == GoalStateCanceled
}

func validWorkItemState(state WorkItemState) bool {
	return state == WorkItemStatePending || state == WorkItemStateRunning ||
		state == WorkItemStateSucceeded || state == WorkItemStateFailed || state == WorkItemStateSkipped ||
		state == WorkItemStateInterrupted || state == WorkItemStateCanceled || state == WorkItemStateSuperseded
}

func revisionForWorkItemState(state WorkItemState) Revision {
	switch state {
	case WorkItemStatePending:
		return 1
	case WorkItemStateRunning, WorkItemStateSkipped:
		return 2
	case WorkItemStateSucceeded, WorkItemStateFailed:
		return 3
	default:
		return 0
	}
}

func validRestoredWorkItemRevision(state WorkItemState, revision Revision, controlSequence ...uint64) bool {
	sequence := uint64(0)
	if len(controlSequence) > 0 {
		sequence = controlSequence[0]
	}
	if revision < 1 || sequence > uint64(revision-1) {
		return false
	}
	effective := revision - Revision(sequence)
	minimum := revisionForWorkItemState(state)
	switch state {
	case WorkItemStatePending, WorkItemStateSkipped:
		return minimum > 0 && effective == minimum
	case WorkItemStateRunning, WorkItemStateSucceeded, WorkItemStateFailed:
		return minimum > 0 && effective >= minimum
	case WorkItemStateInterrupted:
		return effective >= 2
	case WorkItemStateCanceled, WorkItemStateSuperseded:
		return effective >= 1
	default:
		return false
	}
}

func revisionForGoalSnapshot(goal Goal) Revision {
	revision := Revision(1) + Revision(goal.planGeneration)
	if !goal.startedAt.IsZero() {
		revision++
	}
	revision += Revision(goal.controlSequence)
	superseded := Revision(0)
	for _, item := range goal.items {
		effectiveRevision := item.revision - Revision(item.controlSequence)
		switch item.state {
		case WorkItemStateRunning, WorkItemStateSucceeded, WorkItemStateFailed,
			WorkItemStateInterrupted, WorkItemStateCanceled, WorkItemStateSuperseded:
			if effectiveRevision > 1 {
				revision += effectiveRevision - 1
			}
		}
		if item.state == WorkItemStateSuperseded {
			superseded++
		}
	}
	revision -= superseded
	if goal.state == GoalStateSucceeded || goal.state == GoalStateFailed {
		revision++
	}
	revision += Revision(len(goal.childHandoffs))
	return revision
}

func restoreChildHandoffResolutions(
	snapshots []ChildHandoffResolutionSnapshot,
) ([]ChildHandoffResolution, error) {
	resolutions := make([]ChildHandoffResolution, 0, len(snapshots))
	for _, snapshot := range snapshots {
		parentRef, err := NewWorkItemRef(snapshot.ParentRef)
		if err != nil {
			return nil, err
		}
		childRef, err := NewWorkItemRef(snapshot.ChildRef)
		if err != nil {
			return nil, err
		}
		if !validChildHandoffRecordRef(snapshot.MessageRef) ||
			!validChildHandoffOutcome(snapshot.Outcome) ||
			!validChildHandoffRecordRef(snapshot.ReceiptRef) || snapshot.ResolvedAt.IsZero() {
			return nil, domainError(ErrorSnapshotInvalid, "child_handoff")
		}
		resolutions = append(resolutions, ChildHandoffResolution{
			parentRef: parentRef, childRef: childRef, messageRef: snapshot.MessageRef,
			outcome: snapshot.Outcome, receiptRef: snapshot.ReceiptRef,
			resolvedAt: canonicalTime(snapshot.ResolvedAt),
		})
	}
	return resolutions, nil
}

func validateRestoredChildHandoffs(goal Goal) error {
	type relation struct {
		parent WorkItemRef
		child  WorkItemRef
	}
	seen := make(map[relation]struct{}, len(goal.childHandoffs))
	for _, resolution := range goal.childHandoffs {
		key := relation{parent: resolution.parentRef, child: resolution.childRef}
		if _, duplicate := seen[key]; duplicate {
			return domainError(ErrorSnapshotInvalid, "duplicate_child_handoff")
		}
		parent, parentExists := goal.items[resolution.parentRef]
		child, childExists := goal.items[resolution.childRef]
		if !parentExists || !childExists || child.parent != parent.ref || !child.handoffRequired ||
			child.state != WorkItemStateSucceeded ||
			!validTransitionTime(resolution.resolvedAt, child.finishedAt) {
			return domainError(ErrorSnapshotInvalid, "child_handoff_relation")
		}
		if goal.state.Terminal() && resolution.resolvedAt.After(goal.closedAt) {
			return domainError(ErrorSnapshotInvalid, "child_handoff_resolved_at")
		}
		seen[key] = struct{}{}
	}
	for _, parentRef := range goal.itemOrder {
		parent := goal.items[parentRef]
		if parent.state == WorkItemStateSucceeded && !goal.childHandoffsResolvedForParent(parentRef) {
			return domainError(ErrorSnapshotInvalid, "succeeded_parent_child_handoffs")
		}
	}
	if (goal.state == GoalStateSucceeded || goal.state == GoalStateFailed) && !goal.allChildHandoffsResolved() {
		return domainError(ErrorSnapshotInvalid, "closed_goal_child_handoffs")
	}
	return nil
}

func restorePhases(snapshots []PhaseInstanceSnapshot) ([]PhaseInstance, error) {
	phases := make([]PhaseInstance, 0, len(snapshots))
	for _, snapshot := range snapshots {
		ref, err := NewPhaseRef(snapshot.Ref)
		if err != nil {
			return nil, err
		}
		key, err := NewPhaseKey(snapshot.Key)
		if err != nil {
			return nil, err
		}
		templateRef, err := NewPhaseTemplateRef(snapshot.TemplateRef)
		if err != nil {
			return nil, err
		}
		inputRefs, err := restoreInputRefs(snapshot.InputRefs)
		if err != nil {
			return nil, err
		}
		criterionRefs, err := restoreCriterionRefs(snapshot.CriterionRefs)
		if err != nil {
			return nil, err
		}
		phase, err := NewPhaseInstanceWithMetadata(PhaseInstanceInput{
			Ref: ref, Key: key, TemplateRef: templateRef,
			InputRefs: inputRefs, CriterionRefs: criterionRefs,
		})
		if err != nil {
			return nil, err
		}
		phases = append(phases, phase)
	}
	return phases, nil
}

func restoreOptionalWorkItemRef(value string) (WorkItemRef, error) {
	if value == "" {
		return WorkItemRef{}, nil
	}
	return NewWorkItemRef(value)
}

func restoreInputRefs(values []string) ([]InputRef, error) {
	refs := make([]InputRef, 0, len(values))
	for _, value := range values {
		ref, err := NewInputRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreCriterionRefs(values []string) ([]CriterionRef, error) {
	refs := make([]CriterionRef, 0, len(values))
	for _, value := range values {
		ref, err := NewCriterionRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreSkillRefs(values []string) ([]SkillRef, error) {
	refs := make([]SkillRef, 0, len(values))
	for _, value := range values {
		ref, err := NewSkillRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreToolRefs(values []string) ([]ToolRef, error) {
	refs := make([]ToolRef, 0, len(values))
	for _, value := range values {
		ref, err := NewToolRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreCapabilityRefs(values []string) ([]CapabilityRef, error) {
	refs := make([]CapabilityRef, 0, len(values))
	for _, value := range values {
		ref, err := NewCapabilityRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreWorkItemRefs(values []string) ([]WorkItemRef, error) {
	refs := make([]WorkItemRef, 0, len(values))
	for _, value := range values {
		ref, err := NewWorkItemRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func restoreWriteSet(values []string) ([]WriteScope, error) {
	scopes := make([]WriteScope, 0, len(values))
	for _, value := range values {
		scope, err := NewWriteScope(value)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

func restoreExecutionRef(value string) (ExecutionRef, error) {
	if value == "" {
		return ExecutionRef{}, nil
	}
	return NewExecutionRef(value)
}

func restoreArtifactRefs(values []string) ([]ArtifactRef, error) {
	refs := make([]ArtifactRef, 0, len(values))
	for _, value := range values {
		ref, err := NewArtifactRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := validateArtifactRefs(refs); err != nil {
		return nil, err
	}
	return refs, nil
}

func restoreAttestationRefs(values []string) ([]AttestationRef, error) {
	refs := make([]AttestationRef, 0, len(values))
	for _, value := range values {
		ref, err := NewAttestationRef(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := validateAttestationRefs(refs); err != nil {
		return nil, err
	}
	return refs, nil
}

func canonicalOptionalTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return canonicalTime(value)
}
