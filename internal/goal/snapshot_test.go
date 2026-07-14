package goal_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestIntentManifestSnapshotRoundTripAndTamperDetection(t *testing.T) {
	manifest := newIntent(t)
	snapshot := manifest.Snapshot()

	restored, err := domain.RestoreIntentManifest(snapshot)
	if err != nil {
		t.Fatalf("RestoreIntentManifest() error = %v", err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("snapshot round trip differs:\n got: %#v\nwant: %#v", got, snapshot)
	}

	tamperedContent := snapshot
	tamperedContent.Statement += " altered"
	_, err = domain.RestoreIntentManifest(tamperedContent)
	requireCode(t, err, domain.ErrorIntentHashMismatch)

	tamperedHash := snapshot
	tamperedHash.Hash = strings.Repeat("0", 64)
	_, err = domain.RestoreIntentManifest(tamperedHash)
	requireCode(t, err, domain.ErrorIntentHashMismatch)

	invalidRef := snapshot
	invalidRef.ActorRef = ""
	_, err = domain.RestoreIntentManifest(invalidRef)
	requireCode(t, err, domain.ErrorInvalidRef)
}

func TestGoalSnapshotRoundTripPreservesStatesEvidenceAndOrder(t *testing.T) {
	tests := []struct {
		name  string
		build func(*testing.T) domain.Goal
	}{
		{name: "pending", build: pendingGoalSnapshotFixture},
		{name: "running", build: runningGoalSnapshotFixture},
		{name: "succeeded", build: succeededGoalSnapshotFixture},
		{name: "failed", build: failedGoalSnapshotFixture},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := test.build(t)
			snapshot := aggregate.Snapshot()
			restored, err := domain.RestoreGoal(snapshot)
			if err != nil {
				t.Fatalf("RestoreGoal() error = %v", err)
			}
			if got := restored.Snapshot(); !reflect.DeepEqual(got, snapshot) {
				t.Fatalf("snapshot round trip differs:\n got: %#v\nwant: %#v", got, snapshot)
			}
			if restored.IntentHash() != snapshot.AppSpec.Intent.Hash {
				t.Fatalf("IntentHash() = %q, want %q", restored.IntentHash(), snapshot.AppSpec.Intent.Hash)
			}
			items := restored.WorkItems()
			for index, item := range items {
				if item.Ref().String() != snapshot.WorkItems[index].Ref {
					t.Fatalf("work-item order[%d] = %q, want %q", index, item.Ref(), snapshot.WorkItems[index].Ref)
				}
			}
		})
	}
}

func TestGoalSnapshotIsDetachedFromAggregate(t *testing.T) {
	aggregate := succeededGoalSnapshotFixture(t)
	snapshot := aggregate.Snapshot()
	original := aggregate.Snapshot()

	snapshot.AppSpec.Intent.Statement = "mutated outside domain"
	snapshot.AppSpec.Objective = "mutated confirmed objective"
	snapshot.WorkItems[0].Objective = "mutated outside domain"
	snapshot.WorkItems[0].ArtifactRefs[0] = "artifact:mutated"
	snapshot.WorkItems = append(snapshot.WorkItems, snapshot.WorkItems[0])

	if got := aggregate.Snapshot(); !reflect.DeepEqual(got, original) {
		t.Fatalf("mutating DTO changed aggregate:\n got: %#v\nwant: %#v", got, original)
	}
}

func TestRestoreGoalRejectsManipulatedOrIncoherentPayloads(t *testing.T) {
	valid := succeededGoalSnapshotFixture(t).Snapshot()
	tests := []struct {
		name   string
		mutate func(*domain.GoalSnapshot)
		code   domain.ErrorCode
	}{
		{
			name: "tampered intent",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.AppSpec.Intent.Statement += " tampered"
			},
			code: domain.ErrorIntentHashMismatch,
		},
		{
			name: "invalid goal ref",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.Ref = ""
			},
			code: domain.ErrorInvalidRef,
		},
		{
			name: "goal scope differs from intent",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.ActorRef = "actor:other"
			},
			code: domain.ErrorScopeConflict,
		},
		{
			name: "zero goal revision",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.Revision = 0
			},
			code: domain.ErrorSnapshotInvalid,
		},
		{
			name: "unknown goal state",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.State = domain.GoalState("paused")
			},
			code: domain.ErrorSnapshotInvalid,
		},
		{
			name: "duplicate work item",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems = append(snapshot.WorkItems, cloneWorkItemSnapshot(snapshot.WorkItems[0]))
			},
			code: domain.ErrorDuplicateWorkItem,
		},
		{
			name: "work item outside goal scope",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].ProjectRef = "project:other"
			},
			code: domain.ErrorScopeConflict,
		},
		{
			name: "zero work item revision",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].Revision = 0
			},
			code: domain.ErrorSnapshotInvalid,
		},
		{
			name: "unknown work item state",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].State = domain.WorkItemState("waiting")
			},
			code: domain.ErrorSnapshotInvalid,
		},
		{
			name: "finish before start",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].FinishedAt = snapshot.WorkItems[0].StartedAt.Add(-time.Second)
			},
			code: domain.ErrorSnapshotInvalid,
		},
		{
			name: "duplicate artifact evidence",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].ArtifactRefs = append(
					snapshot.WorkItems[0].ArtifactRefs,
					snapshot.WorkItems[0].ArtifactRefs[0],
				)
			},
			code: domain.ErrorDuplicateEvidence,
		},
		{
			name: "successful goal contains valid failed item",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.WorkItems[0].State = domain.WorkItemStateFailed
				snapshot.WorkItems[0].ArtifactRefs = nil
				snapshot.WorkItems[0].AttestationRefs = nil
			},
			code: domain.ErrorOutcomeConflict,
		},
		{
			name: "close before work item finish",
			mutate: func(snapshot *domain.GoalSnapshot) {
				snapshot.ClosedAt = snapshot.WorkItems[1].FinishedAt.Add(-time.Second)
			},
			code: domain.ErrorSnapshotInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := cloneGoalSnapshot(valid)
			test.mutate(&snapshot)
			_, err := domain.RestoreGoal(snapshot)
			requireCode(t, err, test.code)
		})
	}
}

func TestRestoreGoalRejectsPositiveButUnderivableRevisions(t *testing.T) {
	fixtures := []struct {
		name  string
		build func(*testing.T) domain.Goal
	}{
		{name: "pending", build: pendingGoalSnapshotFixture},
		{name: "running", build: runningGoalSnapshotFixture},
		{name: "succeeded", build: succeededGoalSnapshotFixture},
		{name: "failed", build: failedGoalSnapshotFixture},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name+" goal revision", func(t *testing.T) {
			snapshot := fixture.build(t).Snapshot()
			snapshot.Revision++
			_, err := domain.RestoreGoal(snapshot)
			requireCode(t, err, domain.ErrorSnapshotInvalid)
		})
		t.Run(fixture.name+" work item revision", func(t *testing.T) {
			snapshot := fixture.build(t).Snapshot()
			snapshot.WorkItems[0].Revision++
			_, err := domain.RestoreGoal(snapshot)
			requireCode(t, err, domain.ErrorSnapshotInvalid)
		})
	}
}

func TestWorkItemTransitionRejectsDuplicateEvidence(t *testing.T) {
	item := newWorkItem(t, "produce unique evidence")
	running, err := item.Start(
		item.Revision(),
		mustRef(t, "execution:duplicate-evidence", domain.NewExecutionRef),
		baseTime().Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	artifact := mustRef(t, "artifact:duplicate", domain.NewArtifactRef)
	attestation := mustRef(t, "attestation:unique", domain.NewAttestationRef)
	_, err = running.Succeed(
		running.Revision(),
		[]domain.ArtifactRef{artifact, artifact},
		[]domain.AttestationRef{attestation},
		baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorDuplicateEvidence)
}

func pendingGoalSnapshotFixture(t *testing.T) domain.Goal {
	t.Helper()
	aggregate, _ := newGoalWithItems(t, "first", "second")
	return aggregate
}

func runningGoalSnapshotFixture(t *testing.T) domain.Goal {
	t.Helper()
	aggregate, refs := newGoalWithItems(t, "first", "second")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return startGoalItem(t, running, refs[0], "execution:running-round-trip", baseTime().Add(4*time.Minute))
}

func succeededGoalSnapshotFixture(t *testing.T) domain.Goal {
	t.Helper()
	aggregate, refs := newGoalWithItems(t, "first", "second")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running = startGoalItem(t, running, refs[0], "execution:snapshot-first", baseTime().Add(4*time.Minute))
	running = succeedGoalItem(t, running, refs[0], "snapshot-first", baseTime().Add(5*time.Minute))
	running = startGoalItem(t, running, refs[1], "execution:snapshot-second", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[1], "snapshot-second", baseTime().Add(7*time.Minute))
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeSucceeded, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return closed
}

func failedGoalSnapshotFixture(t *testing.T) domain.Goal {
	t.Helper()
	aggregate, refs := newGoalWithItems(t, "first", "second")
	running, err := aggregate.Start(aggregate.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running = startGoalItem(t, running, refs[0], "execution:snapshot-failed", baseTime().Add(4*time.Minute))
	first, _ := running.WorkItem(refs[0])
	running, err = running.FailWorkItem(running.Revision(), first.Revision(), refs[0], baseTime().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("FailWorkItem() error = %v", err)
	}
	running = startGoalItem(t, running, refs[1], "execution:snapshot-success", baseTime().Add(6*time.Minute))
	running = succeedGoalItem(t, running, refs[1], "snapshot-success", baseTime().Add(7*time.Minute))
	closed, err := running.Close(running.Revision(), domain.GoalOutcomeFailed, baseTime().Add(8*time.Minute))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return closed
}

func cloneGoalSnapshot(snapshot domain.GoalSnapshot) domain.GoalSnapshot {
	cloned := snapshot
	cloned.Phases = make([]domain.PhaseInstanceSnapshot, len(snapshot.Phases))
	for index, phase := range snapshot.Phases {
		cloned.Phases[index] = phase
		cloned.Phases[index].InputRefs = append([]string(nil), phase.InputRefs...)
		cloned.Phases[index].CriterionRefs = append([]string(nil), phase.CriterionRefs...)
	}
	cloned.WorkItems = make([]domain.WorkItemSnapshot, len(snapshot.WorkItems))
	for index, item := range snapshot.WorkItems {
		cloned.WorkItems[index] = cloneWorkItemSnapshot(item)
	}
	return cloned
}

func cloneWorkItemSnapshot(snapshot domain.WorkItemSnapshot) domain.WorkItemSnapshot {
	cloned := snapshot
	cloned.DependencyRefs = append([]string(nil), snapshot.DependencyRefs...)
	cloned.WriteSet = append([]string(nil), snapshot.WriteSet...)
	cloned.SkillRefs = append([]string(nil), snapshot.SkillRefs...)
	cloned.ToolRefs = append([]string(nil), snapshot.ToolRefs...)
	cloned.CapabilityRefs = append([]string(nil), snapshot.CapabilityRefs...)
	cloned.ArtifactRefs = append([]string(nil), snapshot.ArtifactRefs...)
	cloned.AttestationRefs = append([]string(nil), snapshot.AttestationRefs...)
	return cloned
}
