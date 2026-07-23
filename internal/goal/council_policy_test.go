package goal_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/council"
	domain "orquesta/internal/goal"
)

func TestWriterRequiresCouncilPolicyWhileReadOnlyMayOmitIt(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:council-policy")
	writerRef := mustRef(t, "work-item:council-policy-writer", domain.NewWorkItemRef)
	_, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref: writerRef, Goal: fixture.goal.Ref(), Actor: fixture.actor, Project: fixture.project,
		Objective: "write policy", CreatedAt: baseTime().Add(2 * time.Minute), Phase: phase.Key(),
		WriteSet:      []domain.WriteScope{mustScope(t, "internal/council")},
		RequiredTests: []domain.RequiredTestSpec{requiredTestSpec(t, "required-test:council-policy", "tool:test", []string{"./..."}, ".")},
	})
	if domain.ErrorCodeOf(err) != domain.ErrorInvalidPlan {
		t.Fatalf("writer without council policy error=%v", err)
	}

	readOnly, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:council-policy-read-only", domain.NewWorkItemRef), Phase: phase.Key(),
	})
	if err != nil {
		t.Fatalf("read-only item rejected: %v", err)
	}
	if policy, found := readOnly.CouncilPolicy(); found || policy != "" {
		t.Fatalf("read-only policy=%q/%v, want empty/false", policy, found)
	}
}

func TestCouncilPolicyRoundTripCloneEqualityDigestAndReplan(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:council-policy-roundtrip")
	sourceRef := mustRef(t, "work-item:council-policy-source", domain.NewWorkItemRef)
	source, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: sourceRef, Phase: phase.Key(), WriteSet: []domain.WriteScope{mustScope(t, "internal/source")},
		CouncilPolicy: council.PolicyRequired,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{source},
	})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := planned.Snapshot()
	if snapshot.WorkItems[0].CouncilPolicy != council.PolicyRequired {
		t.Fatalf("snapshot policy=%q", snapshot.WorkItems[0].CouncilPolicy)
	}
	clone := cloneGoalSnapshot(snapshot)
	clone.WorkItems[0].CouncilPolicy = council.PolicyAuto
	if snapshot.WorkItems[0].CouncilPolicy != council.PolicyRequired || reflect.DeepEqual(snapshot, clone) {
		t.Fatal("policy clone/equality lost immutable snapshot value")
	}
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Fatal("policy changed across snapshot restore")
	}
	restoredSource, _ := restored.WorkItem(sourceRef)
	if policy, found := restoredSource.CouncilPolicy(); !found || policy != council.PolicyRequired {
		t.Fatalf("restored policy=%q/%v", policy, found)
	}

	requiredSubject, err := council.NewSubject(council.Subject{
		ProjectRef: fixture.project.String(), GoalRef: fixture.goal.Ref().String(), WorkItemRef: sourceRef.String(),
		ChangeSetRef: "change-set:council-policy", SpecHash: strings.Repeat("a", 64),
		ReviewSubjectDigest: "sha256:" + strings.Repeat("b", 64), ReviewGateDigest: "sha256:" + strings.Repeat("c", 64),
		Policy: council.PolicyRequired, PlanGeneration: 1, WorkItemGeneration: 1, AppSpecGeneration: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	autoSubject, err := council.NewSubject(council.Subject{
		ProjectRef: fixture.project.String(), GoalRef: fixture.goal.Ref().String(), WorkItemRef: sourceRef.String(),
		ChangeSetRef: "change-set:council-policy", SpecHash: strings.Repeat("a", 64),
		ReviewSubjectDigest: "sha256:" + strings.Repeat("b", 64), ReviewGateDigest: "sha256:" + strings.Repeat("c", 64),
		Policy: council.PolicyAuto, PlanGeneration: 1, WorkItemGeneration: 1, AppSpecGeneration: 1,
	})
	if err != nil || requiredSubject.Digest() == autoSubject.Digest() {
		t.Fatalf("policy did not bind subject digest: err=%v", err)
	}

	running, err := restored.Start(restored.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	successor, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:council-policy-successor", domain.NewWorkItemRef), Phase: phase.Key(),
		WriteSet: []domain.WriteScope{mustScope(t, "internal/successor")}, CouncilPolicy: council.PolicyAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	replanned, err := running.ApplyReplan(running.Revision(), domain.ReplanInput{
		ExpectedPlanGeneration: running.PlanGeneration(), Source: sourceRef, ExpectedSourceRevision: source.Revision(),
		Cause: domain.ReplanCauseSplitPending, Successors: []domain.WorkItem{successor}, At: baseTime().Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	successor, _ = replanned.WorkItem(successor.Ref())
	if policy, found := successor.CouncilPolicy(); !found || policy != council.PolicyAuto {
		t.Fatalf("replan successor policy=%q/%v", policy, found)
	}
}

func TestCouncilPolicyLegacyTerminalPreservedButLiveAndCurrentMissingFail(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:council-policy-legacy")
	ref := mustRef(t, "work-item:council-policy-legacy", domain.NewWorkItemRef)
	item := fixture.item(t, ref, phase.Key(), nil, []domain.WriteScope{mustScope(t, "internal/legacy")})
	plan := mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{item},
	})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	currentMissing := planned.Snapshot()
	currentMissing.WorkItems[0].CouncilPolicy = ""
	if _, err := domain.RestoreGoal(currentMissing); domain.ErrorCodeOf(err) != domain.ErrorSnapshotInvalid {
		t.Fatalf("current live writer without policy error=%v", err)
	}

	running, err := planned.Start(planned.Revision(), baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	completed := startGoalItem(t, running, ref, "execution:council-policy-legacy", baseTime().Add(4*time.Minute))
	completed = succeedGoalItem(t, completed, ref, "council-policy-legacy", baseTime().Add(5*time.Minute))
	legacy := completed.Snapshot()
	legacy.SchemaVersion = domain.GoalSnapshotSchemaVersion - 1
	legacy.WorkItems[0].CouncilPolicy = ""
	restored, err := domain.RestoreGoal(legacy)
	if err != nil {
		t.Fatalf("legacy terminal policy restore: %v", err)
	}
	restoredItem, _ := restored.WorkItem(ref)
	if policy, found := restoredItem.CouncilPolicy(); found || policy != "" {
		t.Fatalf("legacy terminal policy=%q/%v", policy, found)
	}
	expected := cloneGoalSnapshot(legacy)
	expected.SchemaVersion = domain.GoalSnapshotSchemaVersion
	reemitted := restored.Snapshot()
	if !reflect.DeepEqual(reemitted, expected) {
		t.Fatalf("legacy terminal preservation changed snapshot:\n got: %#v\nwant: %#v", reemitted, expected)
	}
	restarted, err := domain.RestoreGoal(reemitted)
	if err != nil {
		t.Fatalf("reemitted terminal policy restart: %v", err)
	}
	if got := restarted.Snapshot(); !reflect.DeepEqual(got, reemitted) {
		t.Fatalf("terminal policy restart changed snapshot:\n got: %#v\nwant: %#v", got, reemitted)
	}
}
