package goal

import "time"

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

// WorkItemSnapshot is the complete immutable state required to rehydrate a
// WorkItem as part of its Goal aggregate.
type WorkItemSnapshot struct {
	Ref             string
	GoalRef         string
	ActorRef        string
	ProjectRef      string
	Objective       string
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
	Ref        string
	ActorRef   string
	ProjectRef string
	Intent     IntentManifestSnapshot
	State      GoalState
	Revision   Revision
	CreatedAt  time.Time
	StartedAt  time.Time
	ClosedAt   time.Time
	WorkItems  []WorkItemSnapshot
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

func (goal Goal) Snapshot() GoalSnapshot {
	var items []WorkItemSnapshot
	if len(goal.itemOrder) > 0 {
		items = make([]WorkItemSnapshot, 0, len(goal.itemOrder))
		for _, ref := range goal.itemOrder {
			items = append(items, snapshotWorkItem(goal.items[ref]))
		}
	}
	return GoalSnapshot{
		Ref:        goal.ref.String(),
		ActorRef:   goal.actor.String(),
		ProjectRef: goal.project.String(),
		Intent:     goal.intentManifest.Snapshot(),
		State:      goal.state,
		Revision:   goal.revision,
		CreatedAt:  goal.createdAt,
		StartedAt:  goal.startedAt,
		ClosedAt:   goal.closedAt,
		WorkItems:  items,
	}
}

func snapshotWorkItem(item WorkItem) WorkItemSnapshot {
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
