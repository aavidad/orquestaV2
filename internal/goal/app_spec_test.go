package goal_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestAppSpecRequiresConfirmationAndBindsIntent(t *testing.T) {
	intent := newIntent(t)
	confirmedAt := baseTime().Add(time.Minute).In(time.FixedZone("confirm-zone", 2*60*60))
	input := domain.AppSpecInput{
		Ref:         mustRef(t, "app-spec:confirmation", domain.NewAppSpecRef),
		Intent:      intent,
		Objective:   "  Preserve this confirmed objective exactly!  ",
		Reason:      " initial operator confirmation ",
		ConfirmedBy: mustRef(t, "actor:reviewer", domain.NewActorRef),
		ConfirmedAt: confirmedAt,
	}
	spec, err := domain.NewInitialAppSpec(input)
	if err != nil {
		t.Fatalf("NewInitialAppSpec() error = %v", err)
	}
	if spec.Ref() != input.Ref || spec.Generation() != 1 || spec.Intent().Hash() != intent.Hash() ||
		spec.Objective() != input.Objective || spec.Reason() != input.Reason ||
		spec.ConfirmedBy() != input.ConfirmedBy || !spec.ConfirmedAt().Equal(confirmedAt) ||
		spec.ConfirmedAt().Location() != time.UTC {
		t.Fatalf("confirmed AppSpec fields = %+v", spec.Snapshot())
	}
	if parent, ok := spec.ParentRef(); ok || parent.String() != "" || spec.ParentHash() != "" {
		t.Fatalf("root parent = %q/%q/%v", parent.String(), spec.ParentHash(), ok)
	}
	if len(spec.Hash()) != 64 || strings.ToLower(spec.Hash()) != spec.Hash() {
		t.Fatalf("Hash() = %q", spec.Hash())
	}

	equivalent := input
	equivalent.ConfirmedAt = confirmedAt.UTC()
	same, err := domain.NewInitialAppSpec(equivalent)
	if err != nil || same.Hash() != spec.Hash() {
		t.Fatalf("equivalent confirmation hash = %q, want %q; err=%v", same.Hash(), spec.Hash(), err)
	}
	framed := input
	framed.Objective, framed.Reason = "  Preserve this confirmed objective exactly!  initial", " operator confirmation "
	other, err := domain.NewInitialAppSpec(framed)
	if err != nil {
		t.Fatalf("framed AppSpec: %v", err)
	}
	if other.Hash() == spec.Hash() {
		t.Fatal("framed fields produced ambiguous hash")
	}

	invalid := []struct {
		name  string
		input domain.AppSpecInput
		code  domain.ErrorCode
	}{
		{name: "ref", input: withAppSpecRef(input, domain.AppSpecRef{}), code: domain.ErrorInvalidRef},
		{name: "objective", input: withAppSpecObjective(input, " \n\t"), code: domain.ErrorInvalidArgument},
		{name: "reason", input: withAppSpecReason(input, " "), code: domain.ErrorInvalidArgument},
		{name: "principal", input: withAppSpecConfirmedBy(input, domain.ActorRef{}), code: domain.ErrorInvalidRef},
		{name: "confirmation before intent", input: withAppSpecConfirmedAt(input, intent.SubmittedAt().Add(-time.Nanosecond)), code: domain.ErrorInvalidArgument},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			_, createErr := domain.NewInitialAppSpec(test.input)
			requireCode(t, createErr, test.code)
		})
	}
}

func TestAppSpecAmendmentRequiresExactNextGeneration(t *testing.T) {
	initial := newInitialAppSpec(t, newIntent(t))
	original := initial.Snapshot()
	secondIntent := newIntentForSpec(t, initial.Intent(), "intent:amendment-2", "second exact intent", baseTime().Add(time.Minute))
	second, err := initial.Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:generation-2", domain.NewAppSpecRef), Intent: secondIntent,
		Objective: "second confirmed objective", Reason: "operator corrected scope",
		ConfirmedBy: mustRef(t, "actor:reviewer", domain.NewActorRef), ConfirmedAt: baseTime().Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Amend(generation 2) error = %v", err)
	}
	parent, ok := second.ParentRef()
	if !ok || second.Generation() != 2 || parent != initial.Ref() || second.ParentHash() != initial.Hash() ||
		second.Intent().Ref() != secondIntent.Ref() || second.Hash() == initial.Hash() {
		t.Fatalf("generation 2 = %+v", second.Snapshot())
	}
	if !reflect.DeepEqual(initial.Snapshot(), original) {
		t.Fatal("Amend mutated parent AppSpec")
	}
	thirdIntent := newIntentForSpec(t, initial.Intent(), "intent:amendment-3", "third exact intent", baseTime().Add(3*time.Minute))
	third, err := second.Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:generation-3", domain.NewAppSpecRef), Intent: thirdIntent,
		Objective: "third confirmed objective", Reason: "second correction",
		ConfirmedBy: initial.Intent().Actor(), ConfirmedAt: baseTime().Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Amend(generation 3) error = %v", err)
	}
	thirdParent, ok := third.ParentRef()
	if !ok || third.Generation() != 3 || thirdParent != second.Ref() || third.ParentHash() != second.Hash() {
		t.Fatalf("generation 3 = %+v", third.Snapshot())
	}
}

func TestAppSpecAmendmentRejectsWrongScopeTimeAndReusedIdentity(t *testing.T) {
	initial := newInitialAppSpec(t, newIntent(t))
	validIntent := newIntentForSpec(t, initial.Intent(), "intent:valid-amendment", "valid amendment", baseTime().Add(time.Minute))
	valid := domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:valid-amendment", domain.NewAppSpecRef), Intent: validIntent,
		Objective: "valid objective", Reason: "valid reason", ConfirmedBy: initial.Intent().Actor(),
		ConfirmedAt: baseTime().Add(2 * time.Minute),
	}
	wrongScopeIntent, err := domain.NewIntentManifest(domain.IntentManifestInput{
		Ref:   mustRef(t, "intent:wrong-scope", domain.NewIntentRef),
		Actor: mustRef(t, "actor:other", domain.NewActorRef), Project: initial.Intent().Project(),
		Statement: "wrong scope", SubmittedAt: baseTime().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("wrong-scope fixture: %v", err)
	}
	earlyIntent := newIntentForSpec(t, initial.Intent(), "intent:too-early", "too early", baseTime())
	tests := []struct {
		name  string
		input domain.AppSpecInput
		code  domain.ErrorCode
	}{
		{name: "scope", input: withAppSpecIntent(valid, wrongScopeIntent), code: domain.ErrorScopeConflict},
		{name: "old intent time", input: withAppSpecIntent(valid, earlyIntent), code: domain.ErrorInvalidArgument},
		{name: "confirmation before new intent", input: withAppSpecConfirmedAt(valid, validIntent.SubmittedAt().Add(-time.Nanosecond)), code: domain.ErrorInvalidArgument},
		{name: "reused intent", input: withAppSpecIntent(valid, initial.Intent()), code: domain.ErrorInvalidArgument},
		{name: "reused spec ref", input: withAppSpecRef(valid, initial.Ref()), code: domain.ErrorInvalidArgument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, amendErr := initial.Amend(test.input)
			requireCode(t, amendErr, test.code)
		})
	}
}

func TestGoalSuccessorLeavesTerminalParentUnchangedAndStartsEmpty(t *testing.T) {
	source := succeededGoalSnapshotFixture(t)
	original := source.Snapshot()
	amended := amendTerminalGoalSpec(t, source)
	createdAt := baseTime().Add(11 * time.Minute)
	successor, err := domain.NewSuccessorGoal(
		mustRef(t, "goal:successor", domain.NewGoalRef), source, amended, createdAt,
	)
	if err != nil {
		t.Fatalf("NewSuccessorGoal() error = %v", err)
	}
	if !reflect.DeepEqual(source.Snapshot(), original) {
		t.Fatal("NewSuccessorGoal mutated terminal parent")
	}
	if successor.State() != domain.GoalStatePending || successor.Revision() != 1 ||
		successor.PlanGeneration() != 0 || successor.WorkItemCount() != 0 || len(successor.Phases()) != 0 ||
		successor.AppSpec().Hash() != amended.Hash() || successor.SpecHash() != amended.Hash() ||
		successor.Intent() != amended.Intent().Ref() || successor.IntentHash() != amended.Intent().Hash() ||
		!successor.CreatedAt().Equal(createdAt) {
		t.Fatalf("successor = %+v", successor.Snapshot())
	}
	if _, started := successor.StartedAt(); started {
		t.Fatal("successor unexpectedly started")
	}
	if _, closed := successor.ClosedAt(); closed {
		t.Fatal("successor unexpectedly closed")
	}
	_, err = domain.NewGoal(mustRef(t, "goal:not-root", domain.NewGoalRef), amended, createdAt)
	requireCode(t, err, domain.ErrorInvalidArgument)
}

func TestGoalSuccessorRejectsNonterminalSourceAndWrongParent(t *testing.T) {
	nonterminal, _ := newGoalWithItems(t, "not terminal")
	newIntent := newIntentForSpec(t, nonterminal.AppSpec().Intent(), "intent:nonterminal-amend", "amend too early", baseTime().Add(9*time.Minute))
	amended, err := nonterminal.AppSpec().Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:nonterminal-amend", domain.NewAppSpecRef), Intent: newIntent,
		Objective: "amend too early", Reason: "test nonterminal guard",
		ConfirmedBy: newIntent.Actor(), ConfirmedAt: baseTime().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("nonterminal amendment fixture: %v", err)
	}
	_, err = domain.NewSuccessorGoal(
		mustRef(t, "goal:nonterminal-successor", domain.NewGoalRef), nonterminal, amended, baseTime().Add(11*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidTransition)

	terminal := succeededGoalSnapshotFixture(t)
	otherIntent := newIntentForSpec(t, terminal.AppSpec().Intent(), "intent:other-root", "other root", baseTime().Add(9*time.Minute))
	otherRoot, err := domain.NewInitialAppSpec(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:other-root", domain.NewAppSpecRef), Intent: otherIntent,
		Objective: "other root", Reason: "independent root", ConfirmedBy: otherIntent.Actor(),
		ConfirmedAt: baseTime().Add(9*time.Minute + time.Second),
	})
	if err != nil {
		t.Fatalf("other root: %v", err)
	}
	otherNextIntent := newIntentForSpec(t, otherIntent, "intent:other-next", "other next", baseTime().Add(10*time.Minute))
	wrongParent, err := otherRoot.Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:other-next", domain.NewAppSpecRef), Intent: otherNextIntent,
		Objective: "other next", Reason: "wrong parent fixture", ConfirmedBy: otherIntent.Actor(),
		ConfirmedAt: baseTime().Add(10*time.Minute + time.Second),
	})
	if err != nil {
		t.Fatalf("wrong parent fixture: %v", err)
	}
	_, err = domain.NewSuccessorGoal(
		mustRef(t, "goal:wrong-parent", domain.NewGoalRef), terminal, wrongParent, baseTime().Add(11*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidArgument)
}

func TestRestoreAppSpecAndGoalRejectTamperedSpecAndNestedIntent(t *testing.T) {
	spec := newInitialAppSpec(t, newIntent(t))
	snapshot := spec.Snapshot()
	restored, err := domain.RestoreAppSpec(snapshot)
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Fatalf("RestoreAppSpec() = %+v, err=%v", restored.Snapshot(), err)
	}
	tamperedSpec := snapshot
	tamperedSpec.Objective += " altered"
	_, err = domain.RestoreAppSpec(tamperedSpec)
	requireCode(t, err, domain.ErrorAppSpecHashMismatch)
	tamperedHash := snapshot
	tamperedHash.Hash = strings.Repeat("0", 64)
	_, err = domain.RestoreAppSpec(tamperedHash)
	requireCode(t, err, domain.ErrorAppSpecHashMismatch)
	tamperedIntent := snapshot
	tamperedIntent.Intent.Statement += " altered"
	_, err = domain.RestoreAppSpec(tamperedIntent)
	requireCode(t, err, domain.ErrorIntentHashMismatch)
	newIntent := newIntentForSpec(t, spec.Intent(), "intent:self-parent", "self parent", baseTime().Add(2*time.Minute))
	amended, amendErr := spec.Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:self-parent", domain.NewAppSpecRef), Intent: newIntent,
		Objective: "self parent", Reason: "negative fixture", ConfirmedBy: newIntent.Actor(),
		ConfirmedAt: baseTime().Add(3 * time.Minute),
	})
	if amendErr != nil {
		t.Fatalf("self-parent fixture: %v", amendErr)
	}
	selfParent := amended.Snapshot()
	selfParent.ParentRef = selfParent.Ref
	_, err = domain.RestoreAppSpec(selfParent)
	requireCode(t, err, domain.ErrorSnapshotInvalid)

	goalSnapshot := succeededGoalSnapshotFixture(t).Snapshot()
	goalSnapshot.AppSpec.Reason += " altered"
	_, err = domain.RestoreGoal(goalSnapshot)
	requireCode(t, err, domain.ErrorAppSpecHashMismatch)
}

func amendTerminalGoalSpec(t *testing.T, source domain.Goal) domain.AppSpec {
	t.Helper()
	intent := newIntentForSpec(t, source.AppSpec().Intent(), "intent:successor", "successor exact intent", baseTime().Add(9*time.Minute))
	amended, err := source.AppSpec().Amend(domain.AppSpecInput{
		Ref: mustRef(t, "app-spec:successor", domain.NewAppSpecRef), Intent: intent,
		Objective: "successor confirmed objective", Reason: "terminal amendment",
		ConfirmedBy: intent.Actor(), ConfirmedAt: baseTime().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("AppSpec.Amend() error = %v", err)
	}
	return amended
}

func newIntentForSpec(
	t *testing.T,
	base domain.IntentManifest,
	ref string,
	statement string,
	submittedAt time.Time,
) domain.IntentManifest {
	t.Helper()
	intent, err := domain.NewIntentManifest(domain.IntentManifestInput{
		Ref: mustRef(t, ref, domain.NewIntentRef), Actor: base.Actor(), Project: base.Project(),
		Statement: statement, SubmittedAt: submittedAt,
	})
	if err != nil {
		t.Fatalf("NewIntentManifest(%q) error = %v", ref, err)
	}
	return intent
}

func withAppSpecRef(input domain.AppSpecInput, ref domain.AppSpecRef) domain.AppSpecInput {
	input.Ref = ref
	return input
}

func withAppSpecIntent(input domain.AppSpecInput, intent domain.IntentManifest) domain.AppSpecInput {
	input.Intent = intent
	return input
}

func withAppSpecObjective(input domain.AppSpecInput, objective string) domain.AppSpecInput {
	input.Objective = objective
	return input
}

func withAppSpecReason(input domain.AppSpecInput, reason string) domain.AppSpecInput {
	input.Reason = reason
	return input
}

func withAppSpecConfirmedBy(input domain.AppSpecInput, actor domain.ActorRef) domain.AppSpecInput {
	input.ConfirmedBy = actor
	return input
}

func withAppSpecConfirmedAt(input domain.AppSpecInput, confirmedAt time.Time) domain.AppSpecInput {
	input.ConfirmedAt = confirmedAt
	return input
}
