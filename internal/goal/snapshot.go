package goal

import (
	"time"

	"orquesta/internal/governance"
)

const GoalSnapshotSchemaVersion uint32 = 6
const governanceCompatibleSnapshotSchemaVersion uint32 = 5

// IntentManifestSnapshot is a persistence-neutral representation. Primitive
// ref values keep adapters independent from domain internals.
type IntentManifestSnapshot struct {
	Ref         string
	ActorRef    string
	ProjectRef  string
	Statement   string
	SubmittedAt time.Time
	Hash        string
}

// AppSpecSnapshot nests its unique IntentManifest and complete causal binding.
type AppSpecSnapshot struct {
	Ref         string
	Generation  AppSpecGeneration
	Intent      IntentManifestSnapshot
	ParentRef   string
	ParentHash  string
	Objective   string
	Reason      string
	ConfirmedBy string
	ConfirmedAt time.Time
	Hash        string
}

// PhaseInstanceSnapshot contains immutable phase metadata only.
type PhaseInstanceSnapshot struct {
	Ref           string
	Key           string
	TemplateRef   string
	InputRefs     []string
	CriterionRefs []string
}

// WorkItemSnapshot is the complete immutable state required to rehydrate a
// WorkItem as part of its Goal aggregate.
type WorkItemSnapshot struct {
	Ref                 string
	GoalRef             string
	ActorRef            string
	ProjectRef          string
	Objective           string
	PhaseKey            string
	RoleKey             string
	ParentRef           string
	HandoffRequired     *bool
	DependencyRefs      []string
	WriteSet            []string
	SkillRefs           []string
	ToolRefs            []string
	CapabilityRefs      []string
	OutputContract      OutputContractKind
	BudgetDemand        governance.BudgetDemand
	SecurityCriticality governance.SecurityCriticality
	ReasoningEffort     governance.ReasoningEffort
	SkipReason          WorkItemSkipReason
	InterruptCause      WorkItemInterruptCause
	ReworkOf            string
	State               WorkItemState
	Revision            Revision
	Paused              bool
	CancelRequested     bool
	ControlSequence     uint64
	CreatedAt           time.Time
	StartedAt           time.Time
	InterruptedAt       time.Time
	FinishedAt          time.Time
	ExecutionRef        string
	ArtifactRefs        []string
	AttestationRefs     []string
}

// ChildHandoffResolutionSnapshot persists only the closure-relevant fact.
// Mailbox delivery attempts, claim tokens and leases never enter Goal.
type ChildHandoffResolutionSnapshot struct {
	ParentRef  string
	ChildRef   string
	MessageRef string
	Outcome    ChildHandoffOutcome
	ReceiptRef string
	ResolvedAt time.Time
}

// GoalSnapshot contains the intent and ordered WorkItem snapshots needed for
// a lossless aggregate round trip.
type GoalSnapshot struct {
	SchemaVersion           uint32
	Ref                     string
	ActorRef                string
	ProjectRef              string
	AppSpec                 AppSpecSnapshot
	State                   GoalState
	Revision                Revision
	Paused                  bool
	CancelRequested         bool
	ControlSequence         uint64
	CreatedAt               time.Time
	StartedAt               time.Time
	ClosedAt                time.Time
	PlanGeneration          PlanGeneration
	Phases                  []PhaseInstanceSnapshot
	WorkItems               []WorkItemSnapshot
	ChildHandoffResolutions []ChildHandoffResolutionSnapshot
}

func (manifest IntentManifest) Snapshot() IntentManifestSnapshot {
	return IntentManifestSnapshot{
		Ref:         manifest.ref.String(),
		ActorRef:    manifest.actor.String(),
		ProjectRef:  manifest.project.String(),
		Statement:   manifest.statement,
		SubmittedAt: manifest.submittedAt,
		Hash:        manifest.hash,
	}
}

func (spec AppSpec) Snapshot() AppSpecSnapshot {
	return AppSpecSnapshot{
		Ref: spec.ref.String(), Generation: spec.generation,
		Intent: spec.intent.Snapshot(), ParentRef: spec.parentRef.String(), ParentHash: spec.parentHash,
		Objective: spec.objective, Reason: spec.reason, ConfirmedBy: spec.confirmedBy.String(),
		ConfirmedAt: spec.confirmedAt, Hash: spec.hash,
	}
}

func (goal Goal) Snapshot() GoalSnapshot {
	var phases []PhaseInstanceSnapshot
	if len(goal.phases) > 0 {
		phases = make([]PhaseInstanceSnapshot, len(goal.phases))
		for index, phase := range goal.phases {
			phases[index] = PhaseInstanceSnapshot{
				Ref: phase.ref.String(), Key: phase.key.String(), TemplateRef: phase.templateRef.String(),
				InputRefs: stringsFromRefs(phase.inputRefs), CriterionRefs: stringsFromRefs(phase.criterionRefs),
			}
		}
	}
	var items []WorkItemSnapshot
	if len(goal.itemOrder) > 0 {
		items = make([]WorkItemSnapshot, 0, len(goal.itemOrder))
		for _, ref := range goal.itemOrder {
			items = append(items, snapshotWorkItem(goal.items[ref]))
		}
	}
	var childHandoffs []ChildHandoffResolutionSnapshot
	if len(goal.childHandoffs) > 0 {
		childHandoffs = make([]ChildHandoffResolutionSnapshot, len(goal.childHandoffs))
		for index, resolution := range goal.childHandoffs {
			childHandoffs[index] = ChildHandoffResolutionSnapshot{
				ParentRef: resolution.parentRef.String(), ChildRef: resolution.childRef.String(),
				MessageRef: resolution.messageRef, Outcome: resolution.outcome,
				ReceiptRef: resolution.receiptRef, ResolvedAt: resolution.resolvedAt,
			}
		}
	}
	return GoalSnapshot{
		SchemaVersion:           GoalSnapshotSchemaVersion,
		Ref:                     goal.ref.String(),
		ActorRef:                goal.actor.String(),
		ProjectRef:              goal.project.String(),
		AppSpec:                 goal.appSpec.Snapshot(),
		State:                   goal.state,
		Revision:                goal.revision,
		Paused:                  goal.paused,
		CancelRequested:         goal.cancelRequested,
		ControlSequence:         goal.controlSequence,
		CreatedAt:               goal.createdAt,
		StartedAt:               goal.startedAt,
		ClosedAt:                goal.closedAt,
		PlanGeneration:          goal.planGeneration,
		Phases:                  phases,
		WorkItems:               items,
		ChildHandoffResolutions: childHandoffs,
	}
}

func snapshotWorkItem(item WorkItem) WorkItemSnapshot {
	handoffRequired := item.handoffRequired
	var dependencies []string
	if len(item.dependencies) > 0 {
		dependencies = make([]string, len(item.dependencies))
		for index, ref := range item.dependencies {
			dependencies[index] = ref.String()
		}
	}
	var writeSet []string
	if len(item.writeSet) > 0 {
		writeSet = make([]string, len(item.writeSet))
		for index, scope := range item.writeSet {
			writeSet[index] = scope.String()
		}
	}
	var artifacts []string
	if len(item.artifacts) > 0 {
		artifacts = make([]string, len(item.artifacts))
		for index, ref := range item.artifacts {
			artifacts[index] = ref.String()
		}
	}
	var attestations []string
	if len(item.attestations) > 0 {
		attestations = make([]string, len(item.attestations))
		for index, ref := range item.attestations {
			attestations[index] = ref.String()
		}
	}
	return WorkItemSnapshot{
		Ref:             item.ref.String(),
		GoalRef:         item.goal.String(),
		ActorRef:        item.actor.String(),
		ProjectRef:      item.project.String(),
		Objective:       item.objective,
		PhaseKey:        item.phase.String(),
		RoleKey:         item.role.String(),
		ParentRef:       item.parent.String(),
		HandoffRequired: &handoffRequired,
		DependencyRefs:  dependencies,
		WriteSet:        writeSet,
		SkillRefs:       stringsFromRefs(item.skillRefs),
		ToolRefs:        stringsFromRefs(item.toolRefs),
		CapabilityRefs:  stringsFromRefs(item.capabilityRefs),
		OutputContract:  item.outputContract.kind,
		BudgetDemand:    item.budgetDemand, SecurityCriticality: item.securityCriticality,
		ReasoningEffort: item.reasoningEffort,
		SkipReason:      item.skipReason,
		InterruptCause:  item.interruptCause,
		ReworkOf:        item.reworkOf.String(),
		State:           item.state,
		Revision:        item.revision,
		Paused:          item.paused,
		CancelRequested: item.cancelRequested,
		ControlSequence: item.controlSequence,
		CreatedAt:       item.createdAt,
		StartedAt:       item.startedAt,
		InterruptedAt:   item.interruptedAt,
		FinishedAt:      item.finishedAt,
		ExecutionRef:    item.execution.String(),
		ArtifactRefs:    artifacts,
		AttestationRefs: attestations,
	}
}

func stringsFromRefs[T interface{ String() string }](refs []T) []string {
	if len(refs) == 0 {
		return nil
	}
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}
