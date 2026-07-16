package goal

import (
	"strings"
	"time"
)

// Revision is an aggregate snapshot revision. The first persisted snapshot is 1.
type Revision uint64

type WorkItemState string

const (
	WorkItemStatePending     WorkItemState = "pending"
	WorkItemStateRunning     WorkItemState = "running"
	WorkItemStateSucceeded   WorkItemState = "succeeded"
	WorkItemStateFailed      WorkItemState = "failed"
	WorkItemStateSkipped     WorkItemState = "skipped"
	WorkItemStateInterrupted WorkItemState = "interrupted"
	WorkItemStateCanceled    WorkItemState = "canceled"
	WorkItemStateSuperseded  WorkItemState = "superseded"
)

func (state WorkItemState) Terminal() bool {
	return state == WorkItemStateSucceeded || state == WorkItemStateFailed || state == WorkItemStateSkipped ||
		state == WorkItemStateCanceled || state == WorkItemStateSuperseded
}

type WorkItemSkipReason string

const (
	WorkItemSkipReasonDependencyFailed   WorkItemSkipReason = "dependency_failed"
	WorkItemSkipReasonDependencyCanceled WorkItemSkipReason = "dependency_canceled"
)

type WorkItemInterruptCause string

const (
	WorkItemInterruptExecutionStopped WorkItemInterruptCause = "execution_stopped"
	WorkItemInterruptExecutionFailed  WorkItemInterruptCause = "execution_failed"
)

type NewWorkItemInput struct {
	Ref             WorkItemRef
	Goal            GoalRef
	Actor           ActorRef
	Project         ProjectRef
	Objective       string
	CreatedAt       time.Time
	Phase           PhaseKey
	Role            RoleKey
	Parent          WorkItemRef
	HandoffRequired bool
	Dependencies    []WorkItemRef
	WriteSet        []WriteScope
	SkillRefs       []SkillRef
	ToolRefs        []ToolRef
	CapabilityRefs  []CapabilityRef
	OutputContract  OutputContract
}

// WorkItem is an immutable execution-unit snapshot.
type WorkItem struct {
	ref             WorkItemRef
	goal            GoalRef
	actor           ActorRef
	project         ProjectRef
	objective       string
	phase           PhaseKey
	role            RoleKey
	parent          WorkItemRef
	handoffRequired bool
	dependencies    []WorkItemRef
	writeSet        []WriteScope
	skillRefs       []SkillRef
	toolRefs        []ToolRef
	capabilityRefs  []CapabilityRef
	outputContract  OutputContract
	skipReason      WorkItemSkipReason
	interruptCause  WorkItemInterruptCause
	reworkOf        WorkItemRef
	state           WorkItemState
	revision        Revision
	paused          bool
	cancelRequested bool
	controlSequence uint64
	createdAt       time.Time
	startedAt       time.Time
	interruptedAt   time.Time
	finishedAt      time.Time
	execution       ExecutionRef
	artifacts       []ArtifactRef
	attestations    []AttestationRef
}

func NewWorkItem(input NewWorkItemInput) (WorkItem, error) {
	if !validWorkItemRef(input.Ref) {
		return WorkItem{}, domainError(ErrorInvalidRef, "work_item_ref")
	}
	if !validGoalRef(input.Goal) {
		return WorkItem{}, domainError(ErrorInvalidRef, "goal_ref")
	}
	if !validActorRef(input.Actor) {
		return WorkItem{}, domainError(ErrorInvalidRef, "actor_ref")
	}
	if !validProjectRef(input.Project) {
		return WorkItem{}, domainError(ErrorInvalidRef, "project_ref")
	}
	if strings.TrimSpace(input.Objective) == "" {
		return WorkItem{}, domainError(ErrorInvalidArgument, "objective")
	}
	if input.CreatedAt.IsZero() {
		return WorkItem{}, domainError(ErrorInvalidArgument, "created_at")
	}
	if input.HandoffRequired && !validWorkItemRef(input.Parent) {
		return WorkItem{}, domainError(ErrorInvalidPlan, "handoff_parent")
	}

	phase := input.Phase
	if !validPhaseKey(phase) {
		phase = defaultPhaseKey()
	}
	role := input.Role
	if !validRoleKey(role) {
		role = defaultRoleKey()
	}
	outputContract := input.OutputContract
	if outputContract.kind == "" {
		outputContract = EvidenceBundleOutputContract()
	}
	if !validOutputContractKind(outputContract.kind) {
		return WorkItem{}, domainError(ErrorInvalidPlan, "output_contract")
	}

	item := WorkItem{
		ref:             input.Ref,
		goal:            input.Goal,
		actor:           input.Actor,
		project:         input.Project,
		objective:       input.Objective,
		phase:           phase,
		role:            role,
		parent:          input.Parent,
		handoffRequired: input.HandoffRequired,
		dependencies:    append([]WorkItemRef(nil), input.Dependencies...),
		writeSet:        append([]WriteScope(nil), input.WriteSet...),
		skillRefs:       cloneRefs(input.SkillRefs),
		toolRefs:        cloneRefs(input.ToolRefs),
		capabilityRefs:  cloneRefs(input.CapabilityRefs),
		outputContract:  outputContract,
		state:           WorkItemStatePending,
		revision:        1,
		createdAt:       canonicalTime(input.CreatedAt),
	}
	if err := validateWorkItemPlanMetadata(item); err != nil {
		return WorkItem{}, err
	}
	return item, nil
}

func (item WorkItem) Ref() WorkItemRef                { return item.ref }
func (item WorkItem) Goal() GoalRef                   { return item.goal }
func (item WorkItem) Actor() ActorRef                 { return item.actor }
func (item WorkItem) Project() ProjectRef             { return item.project }
func (item WorkItem) Objective() string               { return item.objective }
func (item WorkItem) Phase() PhaseKey                 { return item.phase }
func (item WorkItem) Role() RoleKey                   { return item.role }
func (item WorkItem) Parent() (WorkItemRef, bool)     { return item.parent, validWorkItemRef(item.parent) }
func (item WorkItem) HandoffRequired() bool           { return item.handoffRequired }
func (item WorkItem) Dependencies() []WorkItemRef     { return cloneDependencies(item.dependencies) }
func (item WorkItem) WriteSet() []WriteScope          { return cloneWriteSet(item.writeSet) }
func (item WorkItem) SkillRefs() []SkillRef           { return cloneRefs(item.skillRefs) }
func (item WorkItem) ToolRefs() []ToolRef             { return cloneRefs(item.toolRefs) }
func (item WorkItem) CapabilityRefs() []CapabilityRef { return cloneRefs(item.capabilityRefs) }
func (item WorkItem) OutputContract() OutputContract  { return item.outputContract }
func (item WorkItem) State() WorkItemState            { return item.state }
func (item WorkItem) Revision() Revision              { return item.revision }
func (item WorkItem) CreatedAt() time.Time            { return item.createdAt }
func (item WorkItem) IsTerminal() bool                { return item.state.Terminal() }
func (item WorkItem) Artifacts() []ArtifactRef        { return cloneArtifacts(item.artifacts) }
func (item WorkItem) Attestations() []AttestationRef  { return cloneAttestations(item.attestations) }
func (item WorkItem) Paused() bool                    { return item.paused }
func (item WorkItem) CancelRequested() bool           { return item.cancelRequested }
func (item WorkItem) ControlSequence() uint64         { return item.controlSequence }

func (item WorkItem) InterruptCause() (WorkItemInterruptCause, bool) {
	return item.interruptCause, item.interruptCause != ""
}

func (item WorkItem) ReworkOf() (WorkItemRef, bool) {
	return item.reworkOf, validWorkItemRef(item.reworkOf)
}

func (item WorkItem) StartedAt() (time.Time, bool) {
	return item.startedAt, !item.startedAt.IsZero()
}

func (item WorkItem) FinishedAt() (time.Time, bool) {
	return item.finishedAt, !item.finishedAt.IsZero()
}

func (item WorkItem) InterruptedAt() (time.Time, bool) {
	return item.interruptedAt, !item.interruptedAt.IsZero()
}

func (item WorkItem) Execution() (ExecutionRef, bool) {
	return item.execution, validExecutionRef(item.execution)
}

func (item WorkItem) SkipReason() (WorkItemSkipReason, bool) {
	return item.skipReason, item.state == WorkItemStateSkipped
}

func (item WorkItem) handoffBlockedByOwnOutcome() bool {
	return item.state == WorkItemStateFailed || item.state == WorkItemStateSkipped ||
		item.state == WorkItemStateCanceled || item.state == WorkItemStateSuperseded
}

func (item WorkItem) Start(expected Revision, execution ExecutionRef, at time.Time) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStatePending {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if item.paused || item.cancelRequested {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_control")
	}
	if !validExecutionRef(execution) {
		return WorkItem{}, domainError(ErrorInvalidRef, "execution_ref")
	}
	if !validTransitionTime(at, item.createdAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "started_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateRunning
	updated.revision++
	updated.execution = execution
	updated.startedAt = canonicalTime(at)
	return updated, nil
}

// replaceExecution changes only the replaceable execution attempt bound to a
// running WorkItem. WorkItem remains the lifecycle authority: replacement does
// not restart work, move timestamps, or change state.
func (item WorkItem) replaceExecution(
	expected Revision,
	current ExecutionRef,
	replacement ExecutionRef,
	at time.Time,
) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStateRunning {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if item.cancelRequested {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_cancel_requested")
	}
	if !validExecutionRef(current) {
		return WorkItem{}, domainError(ErrorInvalidRef, "current_execution_ref")
	}
	if !validExecutionRef(replacement) {
		return WorkItem{}, domainError(ErrorInvalidRef, "replacement_execution_ref")
	}
	if current != item.execution {
		return WorkItem{}, domainError(ErrorRevisionConflict, "current_execution_ref")
	}
	if replacement == current {
		return WorkItem{}, domainError(ErrorInvalidArgument, "replacement_execution_ref")
	}
	if !validTransitionTime(at, item.startedAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "replacement_at")
	}

	updated := item.clone()
	updated.revision++
	updated.execution = replacement
	return updated, nil
}

func (item WorkItem) Succeed(
	expected Revision,
	artifacts []ArtifactRef,
	attestations []AttestationRef,
	at time.Time,
) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStateRunning {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if item.cancelRequested {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_cancel_requested")
	}
	if item.outputContract.kind != OutputContractAttestation && len(artifacts) == 0 {
		return WorkItem{}, domainError(ErrorEvidenceRequired, "artifact_refs")
	}
	if item.outputContract.kind != OutputContractArtifact && len(attestations) == 0 {
		return WorkItem{}, domainError(ErrorEvidenceRequired, "attestation_refs")
	}
	if err := validateArtifactRefs(artifacts); err != nil {
		return WorkItem{}, err
	}
	if err := validateAttestationRefs(attestations); err != nil {
		return WorkItem{}, err
	}
	if !validTransitionTime(at, item.startedAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "finished_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateSucceeded
	updated.paused = false
	updated.revision++
	updated.finishedAt = canonicalTime(at)
	updated.artifacts = cloneArtifacts(artifacts)
	updated.attestations = cloneAttestations(attestations)
	return updated, nil
}

// Fail is an explicit typed transition. Objective or execution text never
// changes state and is never interpreted as failure evidence.
func (item WorkItem) Fail(expected Revision, at time.Time) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStateRunning {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if item.cancelRequested {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_cancel_requested")
	}
	if !validTransitionTime(at, item.startedAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "finished_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateFailed
	updated.paused = false
	updated.revision++
	updated.finishedAt = canonicalTime(at)
	return updated, nil
}

func (item WorkItem) skipDependency(expected Revision, reason WorkItemSkipReason, at time.Time) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStatePending {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if reason != WorkItemSkipReasonDependencyFailed && reason != WorkItemSkipReasonDependencyCanceled {
		return WorkItem{}, domainError(ErrorInvalidArgument, "skip_reason")
	}
	if !validTransitionTime(at, item.createdAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "finished_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateSkipped
	updated.paused = false
	updated.revision++
	updated.finishedAt = canonicalTime(at)
	updated.skipReason = reason
	return updated, nil
}

func (item WorkItem) expectRevision(expected Revision) error {
	if expected != item.revision {
		return domainError(ErrorRevisionConflict, "work_item_revision")
	}
	return nil
}

func (item WorkItem) clone() WorkItem {
	item.dependencies = cloneDependencies(item.dependencies)
	item.writeSet = cloneWriteSet(item.writeSet)
	item.skillRefs = cloneRefs(item.skillRefs)
	item.toolRefs = cloneRefs(item.toolRefs)
	item.capabilityRefs = cloneRefs(item.capabilityRefs)
	item.artifacts = cloneArtifacts(item.artifacts)
	item.attestations = cloneAttestations(item.attestations)
	return item
}

func equalWorkItems(left, right WorkItem) bool {
	return left.ref == right.ref && left.goal == right.goal && left.actor == right.actor &&
		left.project == right.project && left.objective == right.objective && left.phase == right.phase &&
		left.role == right.role && left.parent == right.parent && left.handoffRequired == right.handoffRequired &&
		refsEqual(left.dependencies, right.dependencies) && refsEqual(left.writeSet, right.writeSet) &&
		refsEqual(left.skillRefs, right.skillRefs) && refsEqual(left.toolRefs, right.toolRefs) &&
		refsEqual(left.capabilityRefs, right.capabilityRefs) && left.outputContract == right.outputContract &&
		left.skipReason == right.skipReason && left.state == right.state && left.revision == right.revision &&
		left.interruptCause == right.interruptCause && left.reworkOf == right.reworkOf &&
		left.paused == right.paused && left.cancelRequested == right.cancelRequested &&
		left.controlSequence == right.controlSequence &&
		left.createdAt.Equal(right.createdAt) && left.startedAt.Equal(right.startedAt) &&
		left.interruptedAt.Equal(right.interruptedAt) && left.finishedAt.Equal(right.finishedAt) && left.execution == right.execution &&
		refsEqual(left.artifacts, right.artifacts) && refsEqual(left.attestations, right.attestations)
}

func cloneDependencies(refs []WorkItemRef) []WorkItemRef {
	return append([]WorkItemRef(nil), refs...)
}

func cloneWriteSet(scopes []WriteScope) []WriteScope {
	return append([]WriteScope(nil), scopes...)
}

func validateArtifactRefs(refs []ArtifactRef) error {
	seen := make(map[ArtifactRef]struct{}, len(refs))
	for _, ref := range refs {
		if !validArtifactRef(ref) {
			return domainError(ErrorInvalidRef, "artifact_ref")
		}
		if _, exists := seen[ref]; exists {
			return domainError(ErrorDuplicateEvidence, "artifact_ref")
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func validateAttestationRefs(refs []AttestationRef) error {
	seen := make(map[AttestationRef]struct{}, len(refs))
	for _, ref := range refs {
		if !validAttestationRef(ref) {
			return domainError(ErrorInvalidRef, "attestation_ref")
		}
		if _, exists := seen[ref]; exists {
			return domainError(ErrorDuplicateEvidence, "attestation_ref")
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func cloneArtifacts(refs []ArtifactRef) []ArtifactRef {
	return append([]ArtifactRef(nil), refs...)
}

func cloneAttestations(refs []AttestationRef) []AttestationRef {
	return append([]AttestationRef(nil), refs...)
}

func validTransitionTime(value, notBefore time.Time) bool {
	return !value.IsZero() && !canonicalTime(value).Before(notBefore)
}
