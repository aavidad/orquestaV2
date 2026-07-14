package goal

import (
	"strings"
	"time"
)

// Revision is an aggregate snapshot revision. The first persisted snapshot is 1.
type Revision uint64

type WorkItemState string

const (
	WorkItemStatePending   WorkItemState = "pending"
	WorkItemStateRunning   WorkItemState = "running"
	WorkItemStateSucceeded WorkItemState = "succeeded"
	WorkItemStateFailed    WorkItemState = "failed"
	WorkItemStateSkipped   WorkItemState = "skipped"
)

func (state WorkItemState) Terminal() bool {
	return state == WorkItemStateSucceeded || state == WorkItemStateFailed || state == WorkItemStateSkipped
}

type WorkItemSkipReason string

const WorkItemSkipReasonDependencyFailed WorkItemSkipReason = "dependency_failed"

type NewWorkItemInput struct {
	Ref            WorkItemRef
	Goal           GoalRef
	Actor          ActorRef
	Project        ProjectRef
	Objective      string
	CreatedAt      time.Time
	Phase          PhaseKey
	Role           RoleKey
	Dependencies   []WorkItemRef
	WriteSet       []WriteScope
	OutputContract OutputContract
}

// WorkItem is an immutable execution-unit snapshot.
type WorkItem struct {
	ref            WorkItemRef
	goal           GoalRef
	actor          ActorRef
	project        ProjectRef
	objective      string
	phase          PhaseKey
	role           RoleKey
	dependencies   []WorkItemRef
	writeSet       []WriteScope
	outputContract OutputContract
	skipReason     WorkItemSkipReason
	state          WorkItemState
	revision       Revision
	createdAt      time.Time
	startedAt      time.Time
	finishedAt     time.Time
	execution      ExecutionRef
	artifacts      []ArtifactRef
	attestations   []AttestationRef
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
		ref:            input.Ref,
		goal:           input.Goal,
		actor:          input.Actor,
		project:        input.Project,
		objective:      input.Objective,
		phase:          phase,
		role:           role,
		dependencies:   append([]WorkItemRef(nil), input.Dependencies...),
		writeSet:       append([]WriteScope(nil), input.WriteSet...),
		outputContract: outputContract,
		state:          WorkItemStatePending,
		revision:       1,
		createdAt:      canonicalTime(input.CreatedAt),
	}
	if err := validateWorkItemPlanMetadata(item); err != nil {
		return WorkItem{}, err
	}
	return item, nil
}

func (item WorkItem) Ref() WorkItemRef               { return item.ref }
func (item WorkItem) Goal() GoalRef                  { return item.goal }
func (item WorkItem) Actor() ActorRef                { return item.actor }
func (item WorkItem) Project() ProjectRef            { return item.project }
func (item WorkItem) Objective() string              { return item.objective }
func (item WorkItem) Phase() PhaseKey                { return item.phase }
func (item WorkItem) Role() RoleKey                  { return item.role }
func (item WorkItem) Dependencies() []WorkItemRef    { return cloneDependencies(item.dependencies) }
func (item WorkItem) WriteSet() []WriteScope         { return cloneWriteSet(item.writeSet) }
func (item WorkItem) OutputContract() OutputContract { return item.outputContract }
func (item WorkItem) State() WorkItemState           { return item.state }
func (item WorkItem) Revision() Revision             { return item.revision }
func (item WorkItem) CreatedAt() time.Time           { return item.createdAt }
func (item WorkItem) IsTerminal() bool               { return item.state.Terminal() }
func (item WorkItem) Artifacts() []ArtifactRef       { return cloneArtifacts(item.artifacts) }
func (item WorkItem) Attestations() []AttestationRef { return cloneAttestations(item.attestations) }

func (item WorkItem) StartedAt() (time.Time, bool) {
	return item.startedAt, !item.startedAt.IsZero()
}

func (item WorkItem) FinishedAt() (time.Time, bool) {
	return item.finishedAt, !item.finishedAt.IsZero()
}

func (item WorkItem) Execution() (ExecutionRef, bool) {
	return item.execution, validExecutionRef(item.execution)
}

func (item WorkItem) SkipReason() (WorkItemSkipReason, bool) {
	return item.skipReason, item.state == WorkItemStateSkipped
}

func (item WorkItem) Start(expected Revision, execution ExecutionRef, at time.Time) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStatePending {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
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
	if !validTransitionTime(at, item.startedAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "finished_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateFailed
	updated.revision++
	updated.finishedAt = canonicalTime(at)
	return updated, nil
}

func (item WorkItem) skipDependencyFailed(expected Revision, at time.Time) (WorkItem, error) {
	if err := item.expectRevision(expected); err != nil {
		return WorkItem{}, err
	}
	if item.state != WorkItemStatePending {
		return WorkItem{}, domainError(ErrorInvalidTransition, "work_item_state")
	}
	if !validTransitionTime(at, item.createdAt) {
		return WorkItem{}, domainError(ErrorInvalidArgument, "finished_at")
	}

	updated := item.clone()
	updated.state = WorkItemStateSkipped
	updated.revision++
	updated.finishedAt = canonicalTime(at)
	updated.skipReason = WorkItemSkipReasonDependencyFailed
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
	item.artifacts = cloneArtifacts(item.artifacts)
	item.attestations = cloneAttestations(item.attestations)
	return item
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
