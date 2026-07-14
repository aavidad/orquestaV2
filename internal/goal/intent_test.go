package goal_test

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"

	domain "orquesta/internal/goal"
)

func TestIntentManifestIsImmutableAndIntegrityHashed(t *testing.T) {
	at := baseTime()
	input := domain.IntentManifestInput{
		Ref:         mustRef(t, "intent:001", domain.NewIntentRef),
		Actor:       mustRef(t, "actor:local", domain.NewActorRef),
		Project:     mustRef(t, "project:orquesta", domain.NewProjectRef),
		Statement:   "Preserve this exact intent, including  spaces and punctuation!",
		SubmittedAt: at,
	}

	manifest, err := domain.NewIntentManifest(input)
	if err != nil {
		t.Fatalf("NewIntentManifest() error = %v", err)
	}
	sameInput := input
	sameInput.SubmittedAt = at.In(time.FixedZone("equivalent", 2*60*60))
	same, err := domain.NewIntentManifest(sameInput)
	if err != nil {
		t.Fatalf("NewIntentManifest(same) error = %v", err)
	}
	if manifest.Hash() != same.Hash() {
		t.Fatalf("equivalent inputs produced different hashes: %q != %q", manifest.Hash(), same.Hash())
	}
	if len(manifest.Hash()) != 64 {
		t.Fatalf("Hash() length = %d, want 64", len(manifest.Hash()))
	}
	if _, err := hex.DecodeString(manifest.Hash()); err != nil {
		t.Fatalf("Hash() is not hexadecimal: %v", err)
	}
	if manifest.Statement() != input.Statement {
		t.Fatalf("Statement() = %q, want exact %q", manifest.Statement(), input.Statement)
	}
	if manifest.SubmittedAt().Location() != time.UTC {
		t.Fatalf("SubmittedAt() location = %v, want UTC", manifest.SubmittedAt().Location())
	}

	changedInput := input
	changedInput.Statement += " changed"
	changed, err := domain.NewIntentManifest(changedInput)
	if err != nil {
		t.Fatalf("NewIntentManifest(changed) error = %v", err)
	}
	if changed.Hash() == manifest.Hash() {
		t.Fatal("changing manifest content did not change its hash")
	}
	changedIdentity := input
	changedIdentity.Ref = mustRef(t, "intent:002", domain.NewIntentRef)
	withOtherIdentity, err := domain.NewIntentManifest(changedIdentity)
	if err != nil {
		t.Fatalf("NewIntentManifest(changed identity) error = %v", err)
	}
	if withOtherIdentity.Hash() == manifest.Hash() {
		t.Fatal("integrity hash did not bind the opaque manifest identity")
	}
}

func TestIntentManifestRejectsStructurallyInvalidInput(t *testing.T) {
	valid := domain.IntentManifestInput{
		Ref:         mustRef(t, "intent:valid", domain.NewIntentRef),
		Actor:       mustRef(t, "actor:valid", domain.NewActorRef),
		Project:     mustRef(t, "project:valid", domain.NewProjectRef),
		Statement:   "build the requested result",
		SubmittedAt: baseTime(),
	}

	tests := []struct {
		name  string
		input domain.IntentManifestInput
		code  domain.ErrorCode
	}{
		{name: "intent ref", input: withIntentRef(valid, domain.IntentRef{}), code: domain.ErrorInvalidRef},
		{name: "actor ref", input: withActorRef(valid, domain.ActorRef{}), code: domain.ErrorInvalidRef},
		{name: "project ref", input: withProjectRef(valid, domain.ProjectRef{}), code: domain.ErrorInvalidRef},
		{name: "blank statement", input: withStatement(valid, " \n\t"), code: domain.ErrorInvalidArgument},
		{name: "zero submitted time", input: withSubmittedAt(valid, time.Time{}), code: domain.ErrorInvalidArgument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := domain.NewIntentManifest(test.input)
			requireCode(t, err, test.code)
		})
	}
}

func TestOpaqueRefsPreserveValuesWithoutWordClassification(t *testing.T) {
	const value = "failed/garbage/capacity_limited:still-opaque"
	constructors := []struct {
		name string
		new  func(string) (string, error)
	}{
		{name: "actor", new: stringRef(domain.NewActorRef)},
		{name: "project", new: stringRef(domain.NewProjectRef)},
		{name: "intent", new: stringRef(domain.NewIntentRef)},
		{name: "goal", new: stringRef(domain.NewGoalRef)},
		{name: "work item", new: stringRef(domain.NewWorkItemRef)},
		{name: "phase", new: stringRef(domain.NewPhaseRef)},
		{name: "phase template", new: stringRef(domain.NewPhaseTemplateRef)},
		{name: "input", new: stringRef(domain.NewInputRef)},
		{name: "criterion", new: stringRef(domain.NewCriterionRef)},
		{name: "skill", new: stringRef(domain.NewSkillRef)},
		{name: "tool", new: stringRef(domain.NewToolRef)},
		{name: "capability", new: stringRef(domain.NewCapabilityRef)},
		{name: "execution", new: stringRef(domain.NewExecutionRef)},
		{name: "artifact", new: stringRef(domain.NewArtifactRef)},
		{name: "attestation", new: stringRef(domain.NewAttestationRef)},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			got, err := constructor.new(value)
			if err != nil {
				t.Fatalf("constructor(%q) error = %v", value, err)
			}
			if got != value {
				t.Fatalf("constructor(%q) = %q", value, got)
			}
			_, err = constructor.new(" surrounded ")
			requireCode(t, err, domain.ErrorInvalidRef)
		})
	}

	manifest, err := domain.NewIntentManifest(domain.IntentManifestInput{
		Ref:         mustRef(t, "intent:words", domain.NewIntentRef),
		Actor:       mustRef(t, "actor:words", domain.NewActorRef),
		Project:     mustRef(t, "project:words", domain.NewProjectRef),
		Statement:   "failed garbage stop capacity_limited are plain intent text",
		SubmittedAt: baseTime(),
	})
	if err != nil {
		t.Fatalf("plain text was classified as failure: %v", err)
	}
	if manifest.Statement() == "" {
		t.Fatal("manifest lost accepted statement")
	}
}

func TestDomainErrorsExposeStableCodes(t *testing.T) {
	_, err := domain.NewGoalRef("")
	requireCode(t, err, domain.ErrorInvalidRef)
	if got := domain.ErrorCodeOf(errors.New("goal.invalid_ref")); got != "" {
		t.Fatalf("ErrorCodeOf(untyped text) = %q, want empty", got)
	}
}

func baseTime() time.Time {
	return time.Date(2026, time.July, 14, 10, 0, 0, 123456789, time.UTC)
}

func mustRef[T any](t *testing.T, value string, constructor func(string) (T, error)) T {
	t.Helper()
	ref, err := constructor(value)
	if err != nil {
		t.Fatalf("construct ref %q: %v", value, err)
	}
	return ref
}

func requireCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %q", want)
	}
	if got := domain.ErrorCodeOf(err); got != want {
		t.Fatalf("ErrorCodeOf(%v) = %q, want %q", err, got, want)
	}
}

type stringerRef interface {
	String() string
}

func stringRef[T stringerRef](constructor func(string) (T, error)) func(string) (string, error) {
	return func(value string) (string, error) {
		ref, err := constructor(value)
		return ref.String(), err
	}
}

func withIntentRef(input domain.IntentManifestInput, ref domain.IntentRef) domain.IntentManifestInput {
	input.Ref = ref
	return input
}

func withActorRef(input domain.IntentManifestInput, ref domain.ActorRef) domain.IntentManifestInput {
	input.Actor = ref
	return input
}

func withProjectRef(input domain.IntentManifestInput, ref domain.ProjectRef) domain.IntentManifestInput {
	input.Project = ref
	return input
}

func withStatement(input domain.IntentManifestInput, statement string) domain.IntentManifestInput {
	input.Statement = statement
	return input
}

func withSubmittedAt(input domain.IntentManifestInput, submittedAt time.Time) domain.IntentManifestInput {
	input.SubmittedAt = submittedAt
	return input
}
