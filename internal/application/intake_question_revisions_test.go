package application

import (
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/intake"
)

func TestQuestionRevisionSnapshotReplayRejectsTamperedVersionChain(t *testing.T) {
	state, err := intake.NewState(
		"intake:question-versions",
		intake.Policy{MaxQuestionRounds: 3},
	)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := intake.NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("c", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	initialChange := intakeQuestionChange(1, intake.OriginChat)
	initialChange.StateRef = state.Ref()
	initialChange.Derivation = identity
	initial, err := intake.Apply(state, initialChange)
	if err != nil {
		t.Fatal(err)
	}
	revisedQuestion := initial.Questions()[0]
	revisedQuestion.Options[0].Recommended = false
	revisedQuestion.Options[1].Recommended = true
	revised, err := intake.Apply(initial, intake.Change{
		StateRef: initial.Ref(), ExpectedRevision: initial.Revision(),
		Origin: intake.OriginForm,
		QuestionRevisions: []intake.Question{
			revisedQuestion,
		},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntake(revised)
	restored, err := RestoreIntake(snapshot)
	if err != nil ||
		!reflect.DeepEqual(SnapshotIntake(restored), snapshot) {
		t.Fatalf("snapshot replay err=%v restored=%+v", err, SnapshotIntake(restored))
	}

	tampered := snapshot
	tampered.QuestionVersions = append(
		[]intake.QuestionVersion(nil),
		snapshot.QuestionVersions...,
	)
	tampered.QuestionVersions[1].ReplacesRevision++
	if _, err = RestoreIntake(tampered); !IsStateError(err, StateInvalid) {
		t.Fatalf("tampered replaces_revision err=%v", err)
	}
}
