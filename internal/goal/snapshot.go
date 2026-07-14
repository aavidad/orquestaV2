package goal

import "time"

const GoalSnapshotSchemaVersion uint32 = 3

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
	Ref             string
	GoalRef         string
	ActorRef        string
	ProjectRef      string
	Objective       string
	PhaseKey        string
	RoleKey         string
	ParentRef       string
	DependencyRefs  []string
	WriteSet        []string
	SkillRefs       []string
	ToolRefs        []string
	CapabilityRefs  []string
	OutputContract  OutputContractKind
	SkipReason      WorkItemSkipReason
	State           WorkItemState
	Revision        Revision
	CreatedAt       time.Time
	StartedAt       time.Time
	FinishedAt      time.Time
	ExecutionRef    string
	ArtifactRefs    []string
	AttestationRefs []string
}

// GoalSnapshot contains the intent and ordered WorkItem snapshots needed for
// a lossless aggregate round trip.
type GoalSnapshot struct {
	SchemaVersion  uint32
	Ref            string
	ActorRef       string
	ProjectRef     string
	AppSpec        AppSpecSnapshot
	State          GoalState
	Revision       Revision
	CreatedAt      time.Time
	StartedAt      time.Time
	ClosedAt       time.Time
	PlanGeneration PlanGeneration
	Phases         []PhaseInstanceSnapshot
	WorkItems      []WorkItemSnapshot
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
	return GoalSnapshot{
		SchemaVersion:  GoalSnapshotSchemaVersion,
		Ref:            goal.ref.String(),
		ActorRef:       goal.actor.String(),
		ProjectRef:     goal.project.String(),
		AppSpec:        goal.appSpec.Snapshot(),
		State:          goal.state,
		Revision:       goal.revision,
		CreatedAt:      goal.createdAt,
		StartedAt:      goal.startedAt,
		ClosedAt:       goal.closedAt,
		PlanGeneration: goal.planGeneration,
		Phases:         phases,
		WorkItems:      items,
	}
}

func snapshotWorkItem(item WorkItem) WorkItemSnapshot {
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
		DependencyRefs:  dependencies,
		WriteSet:        writeSet,
		SkillRefs:       stringsFromRefs(item.skillRefs),
		ToolRefs:        stringsFromRefs(item.toolRefs),
		CapabilityRefs:  stringsFromRefs(item.capabilityRefs),
		OutputContract:  item.outputContract.kind,
		SkipReason:      item.skipReason,
		State:           item.state,
		Revision:        item.revision,
		CreatedAt:       item.createdAt,
		StartedAt:       item.startedAt,
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
