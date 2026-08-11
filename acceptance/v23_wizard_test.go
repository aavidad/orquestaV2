package acceptance_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/commands"
	"orquesta/internal/config"
	"orquesta/internal/intake"
)

const v23WizardFixturePath = "acceptance/fixtures/v23_wizard.json"

type v23WizardFixture struct {
	SchemaVersion        int                     `json:"schema_version"`
	ContractID           string                  `json:"contract_id"`
	ImplementationStatus string                  `json:"implementation_status"`
	TaskContext          v23WizardTaskContext    `json:"task_context"`
	Classification       v23WizardClassification `json:"capability_classification"`
	ScopeTransfers       []v23ScopeTransfer      `json:"scope_transfers"`
	PhaseCriteria        []string                `json:"phase_criteria"`
	PackagePath          string                  `json:"package_path"`
	StateSchema          string                  `json:"state_schema"`
	AllowedOrigins       []string                `json:"allowed_origins"`
	PublicBindings       []v23WizardBinding      `json:"public_command_bindings"`
	TypedRoundPolicy     v23WizardRoundPolicy    `json:"typed_round_policy"`
	StableErrorCodes     []string                `json:"stable_error_codes"`
	RequiredAssertions   []string                `json:"required_assertions"`
	RequiredTest         v23WizardRequiredTest   `json:"required_test"`
	IntegrationGate      v23WizardRequiredTest   `json:"integration_gate"`
	CandidateFiles       []string                `json:"candidate_files"`
	IntegrationFiles     []string                `json:"integration_files"`
	ForbiddenImports     []string                `json:"forbidden_import_boundaries"`
	CompletedScopes      []string                `json:"completed_integration_scopes"`
	DeferredScopes       []string                `json:"deferred_scopes"`
	SealStatus           string                  `json:"seal_status"`
	ReceiptPath          string                  `json:"receipt_path"`
	NextDependency       string                  `json:"next_causal_dependency"`
}

type v23WizardTaskContext struct {
	ProjectRef        string `json:"project_ref"`
	GoalRef           string `json:"goal_ref"`
	WorkItemRef       string `json:"work_item_ref"`
	ExecutionRef      string `json:"execution_ref"`
	PlanGeneration    uint64 `json:"plan_generation"`
	AppSpecGeneration uint64 `json:"app_spec_generation"`
}

type v23WizardRoundPolicy struct {
	Field                  string `json:"field"`
	ExampleValue           uint32 `json:"example_value"`
	ZeroIsInvalid          bool   `json:"zero_is_invalid"`
	PackageDefault         string `json:"package_default"`
	CanonicalConfiguration string `json:"canonical_configuration"`
	CanonicalDefault       uint32 `json:"canonical_default"`
}

type v23WizardBinding struct {
	ID                     string   `json:"id"`
	Handler                string   `json:"handler"`
	Permission             string   `json:"permission"`
	Kind                   string   `json:"kind"`
	ReplayMode             string   `json:"replay_mode"`
	ForbiddenPayloadFields []string `json:"forbidden_payload_fields"`
	OutputKey              string   `json:"output_key"`
	RequiredEnvelopeFields []string `json:"required_envelope_fields"`
	RequiredOutputFields   []string `json:"required_output_fields"`
}

type v23WizardRequiredTest struct {
	Command            string   `json:"command"`
	TestNames          []string `json:"test_names"`
	RejectNoTestsToRun bool     `json:"reject_no_tests_to_run"`
}

type v23WizardClassification struct {
	Candidate []string `json:"candidate"`
	Partial   []string `json:"partial"`
	Pending   []string `json:"pending"`
	Rejected  []string `json:"rejected"`
}

type v23ScopeTransfer struct {
	CapabilityID       string `json:"capability_id"`
	ToVertical         string `json:"to_vertical"`
	AcceptanceContract string `json:"acceptance_contract"`
}

func TestAcceptanceV23WizardIntakeContract(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := loadV23WizardFixture(t, root)
	assertV23WizardFixture(t, root, fixture)
	assertV23IntakeImportBoundary(t, root, fixture.ForbiddenImports)
	assertV23PublicCommandBindings(t, fixture.PublicBindings)
	assertV23NamedGatesResolveExactlyOneTest(t, root, fixture)

	const stateRef intake.Ref = "intake:v23-acceptance"
	state, err := intake.NewState(stateRef, intake.Policy{
		MaxQuestionRounds: fixture.TypedRoundPolicy.ExampleValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	chat, err := intake.Apply(state, intake.Change{
		StateRef: stateRef, ExpectedRevision: 1, Origin: intake.OriginChat,
		Issues:    []intake.Issue{v23AudienceGap()},
		Questions: []intake.Question{v23AudienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	form, err := intake.Apply(chat, intake.Change{
		StateRef: stateRef, ExpectedRevision: 2, Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if state.Ref() != chat.Ref() || chat.Ref() != form.Ref() ||
		state.Revision() != 1 || chat.Revision() != 2 || form.Revision() != 3 ||
		form.Schema() != fixture.StateSchema || form.QuestionRounds() != 1 {
		t.Fatalf("shared state contract failed: state=%q revisions=%d/%d/%d rounds=%d schema=%q",
			form.Ref(), state.Revision(), chat.Revision(), form.Revision(), form.QuestionRounds(), form.Schema())
	}
	history := form.History()
	if len(history) != 2 || history[0].Origin != intake.OriginChat ||
		history[1].Origin != intake.OriginForm {
		t.Fatalf("origins created divergent history: %+v", history)
	}
	decision, ok := form.CurrentDecision("intake-question:audience")
	if !ok || decision.Choice != "intake-option:audience-personal" ||
		decision.Recommendation != "intake-option:audience-team" ||
		decision.RecommendationRationale != "intake.option.audience.team.rationale" {
		t.Fatalf("choice/recommendation contract failed: %+v found=%v", decision, ok)
	}
}

func TestV23WizardIntakeNegativeAndAtomicContract(t *testing.T) {
	const stateRef intake.Ref = "intake:v23-negative"
	base, err := intake.NewState(stateRef, intake.Policy{MaxQuestionRounds: 2})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name              string
		recommendTeam     bool
		recommendPersonal bool
		code              intake.ErrorCode
	}{
		{name: "zero recommendation", code: intake.ErrorRecommendationCount},
		{name: "multiple recommendations", recommendTeam: true, recommendPersonal: true, code: intake.ErrorRecommendationCount},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, applyErr := intake.Apply(base, intake.Change{
				StateRef: stateRef, ExpectedRevision: 1, Origin: intake.OriginChat,
				Issues: []intake.Issue{v23AudienceGap()},
				Questions: []intake.Question{
					v23AudienceQuestion(test.recommendTeam, test.recommendPersonal),
				},
			})
			if intake.ErrorCodeOf(applyErr) != test.code {
				t.Fatalf("error=%v code=%q want=%q", applyErr, intake.ErrorCodeOf(applyErr), test.code)
			}
			assertV23StateUnchanged(t, base, 1, 0, 0, 0)
		})
	}

	current, err := intake.Apply(base, intake.Change{
		StateRef: stateRef, ExpectedRevision: 1, Origin: intake.OriginChat,
		Issues:    []intake.Issue{v23AudienceGap()},
		Questions: []intake.Question{v23AudienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = intake.Apply(current, intake.Change{
		StateRef: stateRef, ExpectedRevision: 1, Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
	})
	if intake.ErrorCodeOf(err) != intake.ErrorRevisionConflict {
		t.Fatalf("stale revision error = %v", err)
	}
	assertV23StateUnchanged(t, current, 2, 1, 1, 1)

	invalidRef := intake.Change{
		StateRef: "form:private-state", ExpectedRevision: 1, Origin: intake.OriginForm,
		Issues: []intake.Issue{v23AudienceGap()},
	}
	if _, err = intake.Apply(base, invalidRef); intake.ErrorCodeOf(err) != intake.ErrorInvalidRef {
		t.Fatalf("invalid ref error = %v", err)
	}
	if _, err = intake.Apply(intake.State{}, invalidRef); intake.ErrorCodeOf(err) != intake.ErrorChannelStateCreation {
		t.Fatalf("channel state creation error = %v", err)
	}
}

func TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.IntakeMaxQuestionRounds() != 6 {
		t.Fatalf("canonical intake round default = %d", snapshot.IntakeMaxQuestionRounds())
	}
	if _, err := intake.NewState("intake:v23-policy-zero", intake.Policy{}); intake.ErrorCodeOf(err) != intake.ErrorInvalidArgument {
		t.Fatalf("zero policy accepted: %v", err)
	}
	state, err := intake.NewState("intake:v23-policy-one", intake.Policy{MaxQuestionRounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	state, err = intake.Apply(state, intake.Change{
		StateRef: "intake:v23-policy-one", ExpectedRevision: 1, Origin: intake.OriginChat,
		Issues:    []intake.Issue{v23AudienceGap()},
		Questions: []intake.Question{v23AudienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = intake.Apply(state, intake.Change{
		StateRef: "intake:v23-policy-one", ExpectedRevision: 2, Origin: intake.OriginForm,
		Issues: []intake.Issue{{
			Ref: "intake-issue:delivery-conflict", Kind: intake.IssueContradiction,
			Field: "delivery", DetailKey: "intake.issue.delivery.contradiction",
		}},
		Questions: []intake.Question{{
			Ref:         "intake-question:delivery",
			DerivedFrom: []intake.IssueRef{"intake-issue:delivery-conflict"},
			PromptKey:   "intake.question.delivery.prompt",
			WhyKey:      "intake.question.delivery.why",
			Options: []intake.Option{
				{Ref: "intake-option:delivery-local", LabelKey: "intake.option.delivery.local.label", RationaleKey: "intake.option.delivery.local.rationale", Recommended: true},
				{Ref: "intake-option:delivery-cloud", LabelKey: "intake.option.delivery.cloud.label", RationaleKey: "intake.option.delivery.cloud.rationale"},
			},
		}},
	})
	if intake.ErrorCodeOf(err) != intake.ErrorRoundLimit {
		t.Fatalf("form reset shared round policy: %v", err)
	}
	assertV23StateUnchanged(t, state, 2, 1, 1, 1)
}

func TestV23WizardRecommendationContextAndDependencyPrimitives(t *testing.T) {
	const stateRef intake.Ref = "intake:v23-interactions"
	state, err := intake.NewState(stateRef, intake.Policy{MaxQuestionRounds: 2})
	if err != nil {
		t.Fatal(err)
	}
	audience := v23AudienceQuestion(true, false)
	delivery := v23DeliveryQuestion(audience.Ref)
	state, err = intake.Apply(state, intake.Change{
		StateRef: stateRef, ExpectedRevision: 1, Origin: intake.OriginChat,
		Issues: []intake.Issue{
			v23AudienceGap(),
			{
				Ref: "intake-issue:delivery-gap", Kind: intake.IssueGap,
				Field: "delivery", DetailKey: "intake.issue.delivery.missing",
			},
		},
		Questions: []intake.Question{audience, delivery},
	})
	if err != nil {
		t.Fatal(err)
	}

	change, err := intake.BuildAcceptRecommendationsChange(
		state,
		intake.AcceptRecommendationsRequest{
			StateRef: stateRef, ExpectedRevision: 2,
			Origin: intake.OriginForm, QuestionRound: 1,
		},
	)
	if err != nil || len(change.Choices) != 2 {
		t.Fatalf("recommendation change=%+v err=%v", change, err)
	}
	state, err = intake.Apply(state, change)
	if err != nil {
		t.Fatal(err)
	}
	beforeContext := state
	contextView, err := intake.ReemitContext(state, intake.ContextRequest{
		StateRef: stateRef, ExpectedRevision: 3,
		Origin: intake.OriginChat, Kind: intake.ContextHelp,
		QuestionRefs: []intake.QuestionRef{delivery.Ref},
	})
	if err != nil || len(contextView.Questions) != 1 ||
		contextView.Questions[0].CurrentDecision == nil ||
		!reflect.DeepEqual(state, beforeContext) {
		t.Fatalf("pure context=%+v err=%v", contextView, err)
	}

	state, err = intake.Apply(state, intake.Change{
		StateRef: stateRef, ExpectedRevision: 3, Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: audience.Ref,
			OptionRef:   audience.Options[1].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened := state.ReopenedDecisions()
	if len(reopened) != 1 || reopened[0].QuestionRef != delivery.Ref ||
		len(reopened[0].InvalidatedBy) != 1 ||
		reopened[0].InvalidatedBy[0].QuestionRef != audience.Ref {
		t.Fatalf("reopened decisions=%+v", reopened)
	}
}

func loadV23WizardFixture(t *testing.T, root string) v23WizardFixture {
	t.Helper()
	file, err := os.Open(filepath.Join(root, v23WizardFixturePath))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var fixture v23WizardFixture
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("fixture contains trailing JSON: %v", err)
	}
	return fixture
}

func assertV23WizardFixture(t *testing.T, root string, fixture v23WizardFixture) {
	t.Helper()
	if fixture.SchemaVersion != 2 || fixture.ContractID != "AC-V23-WIZARD" ||
		fixture.ImplementationStatus != "partial_green_unsealed" ||
		fixture.PackagePath != "internal/intake" || fixture.StateSchema != intake.StateSchema ||
		fixture.SealStatus != "not_sealed" || fixture.ReceiptPath != "" ||
		fixture.NextDependency == "" {
		t.Fatalf("invalid V23 partial envelope: %+v", fixture)
	}
	if fixture.TaskContext != (v23WizardTaskContext{
		ProjectRef: "project:default", GoalRef: "goal:48a624ca7849e44622705b9d3e4ab0a4",
		WorkItemRef:    "work-item:397631cf19b01d9b9c90c2a49633ac84",
		ExecutionRef:   "execution:0421bce3564edff3e62364aae6bbc590",
		PlanGeneration: 1, AppSpecGeneration: 1,
	}) {
		t.Fatalf("invalid task context: %+v", fixture.TaskContext)
	}
	assertV23CapabilityClassification(t, fixture.Classification)
	if !reflect.DeepEqual(fixture.ScopeTransfers, []v23ScopeTransfer{
		{CapabilityID: "UI-05", ToVertical: "web_admin", AcceptanceContract: "AC-V24-WEB-ADMIN"},
		{CapabilityID: "WIZ-13", ToVertical: "web_admin", AcceptanceContract: "AC-V24-WEB-ADMIN"},
		{CapabilityID: "WIZ-10", ToVertical: "domain_plugins", AcceptanceContract: "AC-V28-DOMAIN-PLUGINS"},
	}) {
		t.Fatalf("invalid V23 scope transfers: %+v", fixture.ScopeTransfers)
	}
	if !reflect.DeepEqual(fixture.PhaseCriteria, []string{
		"criterion:v23-one-versioned-intake",
		"criterion:v23-gap-derived-questions",
		"criterion:v23-visible-recommendation",
		"criterion:v23-confirm-exact-dossier-into-immutable-appspec",
		"criterion:v23-nonempty-required-test",
	}) ||
		!reflect.DeepEqual(fixture.AllowedOrigins, []string{"chat", "form"}) {
		t.Fatalf("invalid capability/criterion scope: %+v", fixture)
	}
	wantForbidden := []string{"actor_ref", "project_ref", "request_ref", "request_fingerprint"}
	dossierOutput := []string{
		"schema", "dossier_ref", "actor_ref", "project_ref", "intake_ref", "intake_revision",
		"intake_digest", "source_intake_receipt_ref", "statement", "objective", "sections",
		"diagrams", "decisions", "risk_refs", "plan", "plan_digest", "digest", "generation_receipt",
	}
	wantBindings := []v23WizardBinding{
		{
			ID: "orquesta.intakes.create", Handler: "CreateIntake",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "intake",
			RequiredEnvelopeFields: []string{"intake"},
			RequiredOutputFields:   []string{"intake_ref", "project_ref", "revision", "receipt_ref"},
		},
		{
			ID: "orquesta.intakes.get", Handler: "GetIntake",
			Permission: "goals.get", Kind: "query", ReplayMode: "read_reexecute",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "intake",
			RequiredEnvelopeFields: []string{"intake"},
			RequiredOutputFields: []string{
				"state_schema", "intake_ref", "actor_ref", "project_ref", "revision",
				"max_question_rounds", "question_rounds", "issues", "questions",
				"decisions", "history",
			},
		},
		{
			ID: "orquesta.intakes.apply", Handler: "ApplyIntake",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "intake",
			RequiredEnvelopeFields: []string{"intake"},
			RequiredOutputFields:   []string{"intake_ref", "project_ref", "revision", "receipt_ref"},
		},
		{
			ID:         "orquesta.intakes.recommendations.accept",
			Handler:    "AcceptIntakeRecommendations",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "intake",
			RequiredEnvelopeFields: []string{"intake"},
			RequiredOutputFields:   []string{"intake_ref", "project_ref", "revision", "receipt_ref"},
		},
		{
			ID: "orquesta.intakes.context.get", Handler: "GetIntakeContext",
			Permission: "goals.get", Kind: "query", ReplayMode: "read_reexecute",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "context",
			RequiredEnvelopeFields: []string{"context"},
			RequiredOutputFields: []string{
				"state_ref", "revision", "origin", "kind", "issues", "questions",
			},
		},
		{
			ID: "orquesta.intakes.wizard.gaps.apply", Handler: "ApplyWizardGaps",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "intake",
			RequiredEnvelopeFields: []string{
				"intake", "evaluation", "evaluation_replay_exact", "request_ref_reserved",
				"request_outcome", "input_durability", "evaluator_identity",
			},
			RequiredOutputFields: []string{"intake_ref", "project_ref", "revision", "receipt_ref"},
		},
		{
			ID:         "orquesta.intakes.wizard.dossier.prepare",
			Handler:    "PrepareWizardDossier",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: append(
				append([]string(nil), wantForbidden...),
				"authorization_receipt_ref",
			),
			OutputKey:              "dossier",
			RequiredEnvelopeFields: []string{"dossier", "identity", "preview"},
			RequiredOutputFields:   dossierOutput,
		},
		{
			ID: "orquesta.intakes.dossier.prepare", Handler: "PrepareIntakeDossier",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: []string{"actor_ref", "project_ref", "request_ref", "request_fingerprint", "authorization_receipt_ref"},
			OutputKey:              "dossier",
			RequiredEnvelopeFields: []string{"dossier"},
			RequiredOutputFields:   dossierOutput,
		},
		{
			ID: "orquesta.intakes.dossier.get", Handler: "GetIntakeDossier",
			Permission: "goals.get", Kind: "query", ReplayMode: "read_reexecute",
			ForbiddenPayloadFields: wantForbidden,
			OutputKey:              "dossier",
			RequiredEnvelopeFields: []string{"dossier"},
			RequiredOutputFields:   dossierOutput,
		},
		{
			ID: "orquesta.intakes.dossier.confirm", Handler: "ConfirmIntakeDossier",
			Permission: "goals.create", Kind: "command", ReplayMode: "application_receipt",
			ForbiddenPayloadFields: append(
				append([]string(nil), wantForbidden...),
				"authorization_receipt_ref", "goal_ref", "app_spec_ref",
			),
			OutputKey:              "confirmation",
			RequiredEnvelopeFields: []string{"goal", "confirmation"},
			RequiredOutputFields: []string{
				"receipt_ref", "request_ref", "request_fingerprint", "principal_ref",
				"actor_ref", "project_ref", "intake_ref", "intake_revision",
				"intake_digest", "source_intake_receipt_ref", "dossier_ref",
				"dossier_digest", "plan_digest", "goal_ref", "app_spec_ref",
				"spec_hash", "authorization_receipt_ref", "confirmed_at",
			},
		},
	}
	if !reflect.DeepEqual(fixture.PublicBindings, wantBindings) {
		t.Fatalf("invalid public bindings: %+v", fixture.PublicBindings)
	}
	if fixture.TypedRoundPolicy != (v23WizardRoundPolicy{
		Field: "max_question_rounds", ExampleValue: 2, ZeroIsInvalid: true,
		PackageDefault: "none", CanonicalConfiguration: "intake.max_question_rounds",
		CanonicalDefault: 6,
	}) {
		t.Fatalf("invalid round policy boundary: %+v", fixture.TypedRoundPolicy)
	}
	wantErrors := []string{
		string(intake.ErrorAnswerTextForbidden), string(intake.ErrorAnswerTextRequired),
		string(intake.ErrorChannelStateCreation), string(intake.ErrorChoiceConflict),
		string(intake.ErrorDependencyCycle), string(intake.ErrorDependencyPending),
		string(intake.ErrorDuplicateRef), string(intake.ErrorInvalidArgument),
		string(intake.ErrorInvalidOrigin), string(intake.ErrorInvalidRef),
		string(intake.ErrorIssueNotFound), string(intake.ErrorMessageKeyInvalid),
		string(intake.ErrorOptionNotFound), string(intake.ErrorQuestionNotFound),
		string(intake.ErrorQuestionRound),
		string(intake.ErrorRecommendationCount), string(intake.ErrorRevisionConflict),
		string(intake.ErrorRecommendationsDone),
		string(intake.ErrorRoundLimit), string(intake.ErrorStateMismatch),
	}
	wantAssertions := []string{
		"accept_all_recommendations_compiles_one_atomic_change",
		"chat_and_form_advance_one_ref_and_revision_sequence",
		"dependency_changes_reopen_transitive_decisions",
		"failed_change_does_not_mutate_the_current_snapshot",
		"help_and_clarification_do_not_consume_round_or_revision",
		"question_references_at_least_one_recorded_gap_or_contradiction",
		"public_wizard_gap_application_exposes_durable_canonical_input_receipt",
		"public_wizard_gap_application_exposes_typed_request_outcome_receipt",
		"exact_dossier_confirmation_freezes_it_and_creates_one_causal_goal",
		"required_test_runs_named_contract_and_two_negatives",
		"selected_and_recommended_options_coexist_when_they_differ",
		"typed_policy_is_shared_across_origins_with_canonical_default_six",
	}
	if !reflect.DeepEqual(fixture.StableErrorCodes, wantErrors) ||
		!reflect.DeepEqual(fixture.RequiredAssertions, wantAssertions) {
		t.Fatalf("invalid stable codes/assertions: %+v", fixture)
	}
	wantRequiredTest := v23WizardRequiredTest{
		Command: "go test -mod=vendor -race -count=1 -v ./acceptance -run '^(TestAcceptanceV23WizardIntakeContract|TestV23WizardIntakeNegativeAndAtomicContract|TestV23WizardRecommendationContextAndDependencyPrimitives|TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset)$'",
		TestNames: []string{
			"TestAcceptanceV23WizardIntakeContract",
			"TestV23WizardIntakeNegativeAndAtomicContract",
			"TestV23WizardRecommendationContextAndDependencyPrimitives",
			"TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset",
		},
		RejectNoTestsToRun: true,
	}
	if !reflect.DeepEqual(fixture.RequiredTest, wantRequiredTest) {
		t.Fatalf("invalid required test: %+v", fixture.RequiredTest)
	}
	wantIntegrationGate := v23WizardRequiredTest{
		Command: "go test -mod=vendor -count=1 ./internal/intake ./internal/wizard/catalog ./internal/wizard/gaps ./internal/wizard/stages ./internal/application ./internal/adapters/state/sqlite ./internal/commands ./internal/bootstrap -run '^(TestBuildIntakeDossierBindsVerifiedRecordPlanAndCompleteDecisions|TestIntakeDossierHashProjectsEveryCurrentDecisionField|TestBuildIntakeDossierRejectsTamperedRecordUnresolvedStateAndInvalidPlan|TestBuildIntakeDossierRejectsEveryCompilerInvalidPlanMetadata|TestIntakeDossierServiceExactReplaySurvivesLaterIntakeWithoutReread|TestIntakeDossierServiceRejectsCoherentlyRewrittenAdapterFingerprint|TestIntakeDossierSnapshotRoundTripLosesNoDataAndRecomputesIdentity|TestIntakeDossierSQLiteRestartAndHistoricalReplay|TestIntakeDossierSQLiteReusesContentAndRecordsDistinctRequests|TestV23DossierRecoveryRejectsTamperedCanonicalSnapshot|TestIntakeServiceReplayReturnsExactReceiptAfterLaterMutation|TestIntakeServiceRejectsStaleAndDivergentRequestsWithoutWrite|TestIntakeSQLiteRestartAndHistoricalReplay|TestIntakeSQLiteConcurrentCASAdmitsOneReceipt|TestV23RecoveryRejectsDivergentIntakeBranch|TestOrchestratorIntakeUsesAuthenticatedScopeInsteadOfSpoofedRequestFields|TestIntakeCommandsBindAuthorityOutsidePayloadAndProjectPublicState|TestIntakeDossierCommandsBindAuthorityRejectSpoofAndProjectCompletePlan|TestConfirmIntakeDossierCommandBindsEnvelopeAndExposesExactReceipt|TestConfirmIntakeDossierCreatesRunningGoalFromExactDossier|TestConfirmIntakeDossierExactReplayAndSingleGoalPerDossier|TestConfirmIntakeDossierExactReplayAcceptsLiveGoalProgress|TestV23ConfirmIntakeDossierRejectsSubstitutedPersistedBindings|TestV23ConfirmIntakeDossierReplayRejectsIncompleteLiveRecord|TestIntakeDossierConfirmationSQLiteAtomicReplayRestartAndFreeze|TestIntakeDossierConfirmationSQLiteConcurrentExactReplayAndDivergence|TestIntakeDossierConfirmationSQLiteRollsBackGoalAndOutboxOnReceiptFailure|TestV23DossierConfirmationRecoveryRejectsTamperedBinding|TestV23DossierConfirmationRecoveryAcceptsLivePlanGeneration|TestHistoricalGlobalRegistryDigestReplaysOnlyUnchangedDefinitionAfterAdditiveUpgrade|TestHistoricalRegistryAdmissionReplaysThroughCurrentDispatcherAndSQLite|TestV23IntakeDispatcherPersistsCASAndReplayAcrossRestart|TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart|TestDerivationIdentityIsValidatedAndRecordedInSingleMutationHistory|TestBuiltInV1SemanticDigestGolden|TestEvaluatorV1SourceDigestGolden|TestEvaluatorV1SemanticDigestGolden|TestWizardGapsFirstEvaluationPersistsThroughIntakeWriter|TestWizardGapsRejectsAlteredCanonicalAnsweredQuestionPayload|TestIntakeSQLiteDerivationIdentitySurvivesRestartAndReplay)$'",
		TestNames: []string{
			"TestBuildIntakeDossierBindsVerifiedRecordPlanAndCompleteDecisions",
			"TestIntakeDossierHashProjectsEveryCurrentDecisionField",
			"TestBuildIntakeDossierRejectsTamperedRecordUnresolvedStateAndInvalidPlan",
			"TestBuildIntakeDossierRejectsEveryCompilerInvalidPlanMetadata",
			"TestIntakeDossierServiceExactReplaySurvivesLaterIntakeWithoutReread",
			"TestIntakeDossierServiceRejectsCoherentlyRewrittenAdapterFingerprint",
			"TestIntakeDossierSnapshotRoundTripLosesNoDataAndRecomputesIdentity",
			"TestIntakeDossierSQLiteRestartAndHistoricalReplay",
			"TestIntakeDossierSQLiteReusesContentAndRecordsDistinctRequests",
			"TestV23DossierRecoveryRejectsTamperedCanonicalSnapshot",
			"TestIntakeServiceReplayReturnsExactReceiptAfterLaterMutation",
			"TestIntakeServiceRejectsStaleAndDivergentRequestsWithoutWrite",
			"TestIntakeSQLiteRestartAndHistoricalReplay",
			"TestIntakeSQLiteConcurrentCASAdmitsOneReceipt",
			"TestV23RecoveryRejectsDivergentIntakeBranch",
			"TestOrchestratorIntakeUsesAuthenticatedScopeInsteadOfSpoofedRequestFields",
			"TestIntakeCommandsBindAuthorityOutsidePayloadAndProjectPublicState",
			"TestIntakeDossierCommandsBindAuthorityRejectSpoofAndProjectCompletePlan",
			"TestConfirmIntakeDossierCommandBindsEnvelopeAndExposesExactReceipt",
			"TestConfirmIntakeDossierCreatesRunningGoalFromExactDossier",
			"TestConfirmIntakeDossierExactReplayAndSingleGoalPerDossier",
			"TestConfirmIntakeDossierExactReplayAcceptsLiveGoalProgress",
			"TestV23ConfirmIntakeDossierRejectsSubstitutedPersistedBindings",
			"TestV23ConfirmIntakeDossierReplayRejectsIncompleteLiveRecord",
			"TestIntakeDossierConfirmationSQLiteAtomicReplayRestartAndFreeze",
			"TestIntakeDossierConfirmationSQLiteConcurrentExactReplayAndDivergence",
			"TestIntakeDossierConfirmationSQLiteRollsBackGoalAndOutboxOnReceiptFailure",
			"TestV23DossierConfirmationRecoveryRejectsTamperedBinding",
			"TestV23DossierConfirmationRecoveryAcceptsLivePlanGeneration",
			"TestHistoricalGlobalRegistryDigestReplaysOnlyUnchangedDefinitionAfterAdditiveUpgrade",
			"TestHistoricalRegistryAdmissionReplaysThroughCurrentDispatcherAndSQLite",
			"TestV23IntakeDispatcherPersistsCASAndReplayAcrossRestart",
			"TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart",
			"TestDerivationIdentityIsValidatedAndRecordedInSingleMutationHistory",
			"TestBuiltInV1SemanticDigestGolden",
			"TestEvaluatorV1SourceDigestGolden",
			"TestEvaluatorV1SemanticDigestGolden",
			"TestWizardGapsFirstEvaluationPersistsThroughIntakeWriter",
			"TestWizardGapsRejectsAlteredCanonicalAnsweredQuestionPayload",
			"TestIntakeSQLiteDerivationIdentitySurvivesRestartAndReplay",
		},
		RejectNoTestsToRun: true,
	}
	newWizardGapTests := []string{
		"TestOrchestratorWizardGapsBindsAuthenticatedScopeAndSharedWriter",
		"TestOrchestratorDeniedWizardGapsDoesNotReachSharedWriter",
		"TestOrchestratorWithoutIntakeStoreRejectsWizardGaps",
		"TestWizardGapsCommandBindsAuthorityAndProjectsCompleteEvaluation",
		"TestWizardGapsCommandRejectsUnknownPackAndSpoofedAuthorityBeforeUseCase",
		"TestWizardGapsCommandHasCanonicalHTTPMCPAndCLIBindings",
		"TestV23WizardGapsCLIAndMCPUseSharedSQLiteIntakeWriter",
		"TestWizardGapsNoOpCreatesNoIntakeMutationAndReservesOutcome",
		"TestWizardGapsNoOpReservationConflictWithoutReplayFailsClosed",
		"TestWizardGapsNoOpReservesRequestRefAndReplaysHistoricalResult",
		"TestWizardGapsSQLiteNoOpOutcomeReplaysHistoricalStateAfterRestart",
		"TestV23WizardGapsNoOpReplayIsExactAfterLaterMutationAndRestart",
		"TestOrchestratorRejectsPartialWizardGapsStoreComposition",
		"TestWizardGapsCommandRejectsMissingApplicationRequestOutcome",
		"TestWizardGapsSQLiteConcurrentExactNoOpReservesOneOutcome",
		"TestWizardGapsSQLiteConcurrentDivergentNoOpAdmitsOnePayload",
		"TestWizardGapsSQLiteConcurrentNoOpAndMutationReserveOneEffect",
		"TestV23WizardGapsRecoveryRejectsCorruptedNoOpOutcomes",
		"TestWizardGapsInputReceiptPersistsCanonicalContextAndSelections",
		"TestWizardGapsInputValidationRejectsCoherentContextTampering",
		"TestWizardGapsInputEvaluationRejectsCoherentSelectionTampering",
		"TestWizardGapsInputEvaluationRejectsCoherentMutationReceiptTampering",
		"TestWizardGapsInputEvaluationRejectsCoherentNoOpOutcomeTampering",
		"TestWizardGapsSQLiteMutationInputReplaysExactlyAfterRestart",
		"TestWizardGapsSQLiteConcurrentExactMutationCommitsOneInput",
		"TestWizardGapsSQLiteInputFailureRollsBackWholeRequest",
		"TestV23WizardGapsInputRecoveryRejectsColumnCorruptions",
		"TestV23WizardGapsInputRecoveryRejectsCoherentMutationSelectionTamper",
		"TestV23WizardGapsInputRejectsCoherentMutationFingerprintTamper",
		"TestV23WizardGapsInputRecoveryRejectsCoherentNoOpOutcomeTamper",
		"TestV23WizardGapsInputUpgradeFromSchema20DoesNotInventLegacyInput",
	}
	wantIntegrationGate.TestNames = append(wantIntegrationGate.TestNames, newWizardGapTests...)
	wantIntegrationGate.Command = strings.TrimSuffix(wantIntegrationGate.Command, ")$'") +
		"|" + strings.Join(newWizardGapTests, "|") + ")$'"
	newWizardDossierTests := []string{
		"TestBuiltInCatalogContainsExactlySixCompleteTemplates",
		"TestBuiltInTemplatesExposeExplicitGovernedEffects",
		"TestCatalogAndNestedAccessorsReturnDefensiveCopies",
		"TestCatalogSupportsConcurrentReadAndCopyMutation",
		"TestTemplateConstructionHasDeterministicOrder",
		"TestTemplateRejectsMissingDependencyAndCycles",
		"TestTemplateRejectsUnorderedWriteSetConflict",
		"TestUnitRejectsEmptyTestsAndUnresolvedCriterion",
		"TestUnitRejectsUnsafeWritePathAndUnguardedExternalMutation",
		"TestCatalogRejectsMissingAndConflictingTemplateDefinitions",
		"TestPackageImportsStayPureAndDoNotReachForbiddenLayers",
		"TestBuiltInV1CatalogDigestIsFrozen",
		"TestTemplateDigestCoversPreviewOnlySemantics",
		"TestBuiltInVersionAndDigestFailClosed",
		"TestCompileWizardStagePlanCoversEveryBuiltInTemplate",
		"TestCompileWizardStagePlanIsDeterministicAcrossRefInputOrder",
		"TestCompileWizardStagePlanRejectsInvalidInput",
		"TestCompileWizardStagePlanMaterializesStageDAGAsDependencyKeys",
		"TestCompileWizardStagePlanProjectsUnrepresentableSemanticsExactly",
		"TestCompileWizardStagePlanMapsSecurityAndEffortAxesIndependently",
		"TestIntakeDossierProjectionGeneratesCanonicalMinimumContentAndDiagrams",
		"TestIntakeDossierProjectionIsStableAcrossEquivalentReordering",
		"TestIntakeDossierProjectionPreservesExactOrderSensitivePlan",
		"TestIntakeDossierProjectionCopiesInputsOutputsAndSupportsConcurrentReads",
		"TestIntakeDossierProjectionRejectsStructurallyIncompleteOrDivergentInput",
		"TestIntakeDossierProjectionEscapesMarkdownAndMermaidControlSyntax",
		"TestIntakeDossierProjectionDiagramGrowthIsLinearAndIndicesAreUnbounded",
		"TestIntakeDossierProjectionBindsDecisionViewsToDurableIntakeRecord",
		"TestIntakeDossierProjectionPreservesDurableDecisionOrderWhileViewsReorder",
		"TestIntakeDossierProjectionRecordsUnjustifiedDeviationWithoutBlocking",
		"TestPrepareWizardDossierCompilesBuiltInPlanAndPersistsOneDossier",
		"TestPrepareWizardDossierReplaysExactHistoricalProjectionAfterIntakeAdvance",
		"TestPrepareWizardDossierRejectsDivergenceAndStaleFirstCreation",
		"TestPrepareWizardDossierRejectsUnknownTemplateCallerPlanAndBadAuthority",
		"TestPrepareWizardDossierKeepsV1PreviewWhenAdditiveV2ChangesOnlyRoadmap",
		"TestWizardDossierCommandResolvesIdentityBindsAuthorityAndExposesSafePreview",
		"TestWizardDossierCommandRejectsCallerPlanAndDigestSpoofBeforeAdmission",
		"TestWizardDossierCommandRejectsUnknownCatalogAndTemplateWithoutUseCase",
		"TestV23WizardDossierPublicReplaySurvivesSQLiteRestart",
	}
	wantIntegrationGate.TestNames = append(wantIntegrationGate.TestNames, newWizardDossierTests...)
	wantIntegrationGate.Command = strings.TrimSuffix(wantIntegrationGate.Command, ")$'") +
		"|" + strings.Join(newWizardDossierTests, "|") + ")$'"
	if !reflect.DeepEqual(fixture.IntegrationGate, wantIntegrationGate) {
		t.Fatalf("invalid integration gate: %+v", fixture.IntegrationGate)
	}
	if !reflect.DeepEqual(fixture.ForbiddenImports, []string{
		"adapters", "bootstrap", "database", "filesystem", "interfaces",
		"legacy", "orquesta/modulos", "provider",
	}) {
		t.Fatalf("invalid import boundary: %+v", fixture.ForbiddenImports)
	}
	if !reflect.DeepEqual(fixture.CompletedScopes, []string{
		"application_idempotency",
		"application_dossier_builder",
		"command_registry_binding",
		"durable_cas_persistence_and_restart",
		"durable_dossier_persistence",
		"dossier_generation",
		"public_dossier_commands",
		"causal_plan_creation",
		"canonical_round_default",
		"explicit_confirmation",
		"freeze_after_confirmation",
		"intake_dependency_reopen_projection",
		"intake_pure_context_reemission",
		"intake_recommendation_batch_compiler",
		"public_pure_context_command",
		"public_recommendation_batch_command",
		"public_wizard_dossier_prepare",
		"public_wizard_gap_application",
		"versioned_wizard_gaps_evaluator_identity",
		"wizard_gap_application_compiler_fail_closed",
		"wizard_catalog_foundation_and_15_domain_packs",
		"wizard_gap_explicit_outcome_receipt",
		"wizard_gap_input_receipt_durability",
		"wizard_gap_exact_evaluation_snapshot_and_replay",
		"wizard_gap_noop_request_reservation_and_historical_replay",
		"wizard_gap_reconciliation",
		"wizard_help_surface",
	}) {
		t.Fatalf("invalid completed integration scope: %+v", fixture.CompletedScopes)
	}
	wantDeferred := []string{
		"roadmap_promotion", "seal_and_receipt",
	}
	if !reflect.DeepEqual(fixture.DeferredScopes, wantDeferred) {
		t.Fatalf("invalid deferred scope: %+v", fixture.DeferredScopes)
	}
	if fixture.NextDependency != "v23_10_candidate_gate" {
		t.Fatalf("invalid next dependency: %q", fixture.NextDependency)
	}
	wantCandidateFiles := []string{
		"acceptance/fixtures/v23_wizard.json",
		"acceptance/v23_wizard_test.go",
		"docs/reconstruccion/analisis_y_contrato_v23_wizard.md",
		"internal/intake/errors.go",
		"internal/intake/dependencies.go",
		"internal/intake/dependencies_test.go",
		"internal/intake/intake_test.go",
		"internal/intake/interactions.go",
		"internal/intake/interactions_test.go",
		"internal/intake/answer_text.go",
		"internal/intake/derivation_test.go",
		"internal/intake/model.go",
		"internal/intake/state.go",
	}
	if !reflect.DeepEqual(fixture.CandidateFiles, wantCandidateFiles) {
		t.Fatalf("invalid candidate files: %+v", fixture.CandidateFiles)
	}
	for _, name := range wantCandidateFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatalf("candidate %q: %v", name, err)
		}
	}
	wantIntegrationFiles := []string{
		"acceptance/v23_wizard_dossier_generation_test.go",
		"acceptance/v23_wizard_help_test.go",
		"acceptance/v23_wizard_snapshot_test.go",
		"docs/reconstruccion/corte_v23_comandos_dossier_2026-07-26.md",
		"docs/reconstruccion/corte_v23_dossier_durable_2026-07-26.md",
		"docs/reconstruccion/corte_v23_inputs_wizard_durables_2026-07-28.md",
		"docs/reconstruccion/corte_v23_intake_durable_2026-07-26.md",
		"internal/application/intake_dossier.go",
		"internal/application/intake_dossier_confirmation.go",
		"internal/application/intake_dossier_confirmation_adversarial_test.go",
		"internal/application/intake_dossier_confirmation_test.go",
		"internal/application/intake_dossier_plan_binding.go",
		"internal/application/intake_dossier_projection.go",
		"internal/application/intake_dossier_projection_render.go",
		"internal/application/intake_dossier_projection_test.go",
		"internal/application/intake_dossier_service.go",
		"internal/application/intake_dossier_service_test.go",
		"internal/application/intake_dossier_snapshot.go",
		"internal/application/intake_dossier_snapshot_test.go",
		"internal/application/intake_dossier_test.go",
		"internal/application/intake_dossier_orchestrator_test.go",
		"internal/application/intake_chain.go",
		"internal/application/intake_derivation_test.go",
		"internal/application/intake_orchestrator.go",
		"internal/application/intake_service.go",
		"internal/application/intake_service_test.go",
		"internal/application/orchestrator.go",
		"internal/application/wizard_dossier_preparation.go",
		"internal/application/wizard_dossier_preparation_test.go",
		"internal/application/wizard_gaps.go",
		"internal/application/wizard_gaps_identity_test.go",
		"internal/application/wizard_gaps_orchestrator_test.go",
		"internal/application/wizard_gaps_outcomes.go",
		"internal/application/wizard_gaps_inputs.go",
		"internal/application/wizard_gaps_inputs_test.go",
		"internal/application/wizard_gaps_preflight.go",
		"internal/application/wizard_gaps_snapshot.go",
		"internal/application/wizard_gaps_snapshot_test.go",
		"internal/application/wizard_gaps_test.go",
		"internal/application/wizard_stage_plan.go",
		"internal/application/wizard_stage_plan_test.go",
		"internal/adapters/state/sqlite/intake_dossier.go",
		"internal/adapters/state/sqlite/intake_dossier_confirmation.go",
		"internal/adapters/state/sqlite/intake_dossier_confirmation_test.go",
		"internal/adapters/state/sqlite/intake_dossier_test.go",
		"internal/adapters/state/sqlite/intake_dependencies_test.go",
		"internal/adapters/state/sqlite/intake_derivation_test.go",
		"internal/adapters/state/sqlite/intake.go",
		"internal/adapters/state/sqlite/migrations/020_wizard_gaps_outcomes.sql",
		"internal/adapters/state/sqlite/migrations/021_wizard_gaps_inputs.sql",
		"internal/adapters/state/sqlite/migrations/036_wizard_gaps_result_snapshots.sql",
		"internal/adapters/state/sqlite/migrations/018_intake_dossiers.sql",
		"internal/adapters/state/sqlite/migrations/019_intake_dossier_confirmations.sql",
		"internal/adapters/state/sqlite/migrations/017_intake.sql",
		"internal/adapters/state/sqlite/recovery_validation.go",
		"internal/adapters/state/sqlite/recovery_validation_versions.go",
		"internal/adapters/state/sqlite/recovery_validation_v23_wizard_gaps.go",
		"internal/adapters/state/sqlite/recovery_validation_v23_wizard_gaps_inputs.go",
		"internal/adapters/state/sqlite/recovery_validation_v23_wizard_gaps_test.go",
		"internal/adapters/state/sqlite/recovery_validation_v23_dossier.go",
		"internal/adapters/state/sqlite/recovery_validation_v23.go",
		"internal/adapters/state/sqlite/repository_test.go",
		"internal/adapters/state/sqlite/wizard_gaps_outcomes.go",
		"internal/adapters/state/sqlite/wizard_gaps_outcomes_test.go",
		"internal/adapters/state/sqlite/wizard_gaps_inputs.go",
		"internal/adapters/state/sqlite/wizard_gaps_inputs_test.go",
		"internal/adapters/state/sqlite/wizard_gaps_snapshot_validation.go",
		"internal/adapters/state/sqlite/wizard_gaps_snapshot_validation_test.go",
		"internal/adapters/state/sqlite/wizard_gaps_snapshots.go",
		"internal/adapters/state/sqlite/wizard_gaps_snapshots_test.go",
		"internal/bootstrap/command_registry_upgrade_e2e_test.go",
		"internal/bootstrap/command_surfaces.go",
		"internal/bootstrap/intake_e2e_test.go",
		"internal/bootstrap/runtime.go",
		"internal/bootstrap/wizard_dossier_e2e_test.go",
		"internal/bootstrap/wizard_gaps_public_e2e_test.go",
		"internal/bootstrap/wizard_gaps_snapshot_e2e_test.go",
		"internal/commands/handlers_intake.go",
		"internal/commands/handlers_wizard_dossier.go",
		"internal/commands/handlers_wizard_dossier_input.go",
		"internal/commands/handlers_wizard_dossier_test.go",
		"internal/commands/handlers_wizard_gaps.go",
		"internal/commands/handlers_wizard_gaps_test.go",
		"internal/commands/application_handlers.go",
		"internal/commands/cmd/commandgen/main.go",
		"internal/commands/definitions_generated.go",
		"internal/commands/dispatcher_contract_test.go",
		"internal/commands/mutation_replay_test.go",
		"internal/commands/registry.json",
		"internal/commands/registry_admission_identity.go",
		"internal/config/integer_test.go",
		"internal/config/keys_generated.go",
		"internal/i18n/catalogs/en.json",
		"internal/i18n/catalogs/es.json",
		"internal/i18n/catalog_test.go",
		"internal/i18n/manifest.json",
		"internal/wizard/catalog/builtin.go",
		"internal/wizard/catalog/catalog.go",
		"internal/wizard/catalog/catalog_test.go",
		"internal/wizard/catalog/errors.go",
		"internal/wizard/catalog/model.go",
		"internal/wizard/catalog/semantic.go",
		"internal/wizard/catalog/semantic_test.go",
		"internal/wizard/catalog/validation.go",
		"internal/wizard/gaps/canonical_context.go",
		"internal/wizard/gaps/catalog.go",
		"internal/wizard/gaps/engine.go",
		"internal/wizard/gaps/evaluator.go",
		"internal/wizard/gaps/evaluator_test.go",
		"internal/wizard/gaps/input.go",
		"internal/wizard/gaps/model.go",
		"internal/wizard/gaps/rules.go",
		"internal/wizard/gaps/semantic.go",
		"internal/wizard/gaps/semantic_corpus.go",
		"internal/wizard/stages/builtin.go",
		"internal/wizard/stages/catalog.go",
		"internal/wizard/stages/catalog_test.go",
		"internal/wizard/stages/digest.go",
		"internal/wizard/stages/digest_test.go",
		"internal/wizard/stages/errors.go",
		"internal/wizard/stages/model.go",
		"internal/wizard/stages/validation.go",
	}
	if !reflect.DeepEqual(fixture.IntegrationFiles, wantIntegrationFiles) {
		t.Fatalf("invalid integration files: %+v", fixture.IntegrationFiles)
	}
	for _, name := range wantIntegrationFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatalf("integration file %q: %v", name, err)
		}
	}
}

func assertV23PublicCommandBindings(t *testing.T, bindings []v23WizardBinding) {
	t.Helper()
	definitions := make(map[string]commands.Definition)
	for _, definition := range commands.CanonicalDefinitions() {
		definitions[definition.ID] = definition
	}
	declared := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		declared[binding.ID] = struct{}{}
		definition, found := definitions[binding.ID]
		if !found || definition.Handler != binding.Handler ||
			definition.Permission != binding.Permission ||
			string(definition.Kind) != binding.Kind ||
			string(definition.ReplayMode) != binding.ReplayMode {
			t.Fatalf("binding %q mismatch: %+v found=%v", binding.ID, definition, found)
		}
		var input struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(definition.InputSchema, &input); err != nil {
			t.Fatalf("%s input schema: %v", binding.ID, err)
		}
		for _, forbidden := range binding.ForbiddenPayloadFields {
			if _, exists := input.Properties[forbidden]; exists {
				t.Fatalf("%s payload exposes authority field %q", binding.ID, forbidden)
			}
		}
		var output struct {
			Required   []string `json:"required"`
			Properties map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(definition.OutputSchema, &output); err != nil {
			t.Fatalf("%s output schema: %v", binding.ID, err)
		}
		if !reflect.DeepEqual(output.Required, binding.RequiredEnvelopeFields) {
			t.Fatalf("%s envelope fields=%v want=%v", binding.ID,
				output.Required, binding.RequiredEnvelopeFields)
		}
		boundOutput, exists := output.Properties[binding.OutputKey]
		if !exists || !reflect.DeepEqual(boundOutput.Required, binding.RequiredOutputFields) {
			t.Fatalf("%s output fields=%v want=%v", binding.ID,
				boundOutput.Required, binding.RequiredOutputFields)
		}
	}
	for id := range definitions {
		if !strings.HasPrefix(id, "orquesta.intakes.") {
			continue
		}
		if _, found := declared[id]; !found {
			t.Fatalf("public intake command %q is missing from the V23 acceptance fixture", id)
		}
	}
}

func TestAcceptanceV23WizardCapabilityClassification(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := loadV23WizardFixture(t, root)
	assertV23CapabilityClassification(t, fixture.Classification)
	if fixture.SealStatus != "not_sealed" || fixture.ReceiptPath != "" {
		t.Fatalf("classification task attempted to seal V23: %+v", fixture)
	}
	integrated := make(map[string]struct{}, len(fixture.IntegrationFiles))
	for _, path := range fixture.IntegrationFiles {
		integrated[path] = struct{}{}
	}
	candidates := make(map[string]struct{}, len(fixture.Classification.Candidate))
	for _, id := range fixture.Classification.Candidate {
		candidates[id] = struct{}{}
	}
	declarations := v23PackageTestDeclarations(t, root, "acceptance")
	for _, evidence := range []struct {
		capabilities []string
		path, test   string
	}{
		{[]string{"WIZ-03", "WIZ-04", "WIZ-05", "WIZ-07"},
			"acceptance/v23_wizard_dossier_generation_test.go",
			"TestV23DossierGenerationContractUsesCanonicalProjection"},
		{[]string{"WIZ-06", "WIZ-08", "WIZ-09"},
			"acceptance/v23_wizard_help_test.go",
			"TestV23WizardHelpSurfaceIsCompleteAndReadOnly"},
		{[]string{"WIZ-22"}, "acceptance/v23_wizard_snapshot_test.go",
			"TestV23WizardGapExactReplay"},
	} {
		if _, found := integrated[evidence.path]; !found || declarations[evidence.test] != 1 {
			t.Fatalf("classification evidence missing file=%s test=%s", evidence.path, evidence.test)
		}
		for _, id := range evidence.capabilities {
			if _, promoted := candidates[id]; !promoted {
				t.Fatalf("evidence-backed capability %s is not candidate", id)
			}
		}
	}

	roadmapFile, err := os.Open(filepath.Join(root, "product", "roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer roadmapFile.Close()
	var roadmap struct {
		Capabilities []struct {
			ID                  string   `json:"id"`
			OwnerContext        string   `json:"owner_context"`
			AcceptanceContracts []string `json:"acceptance_contracts"`
		} `json:"capability_entries"`
	}
	if err := json.NewDecoder(roadmapFile).Decode(&roadmap); err != nil {
		t.Fatal(err)
	}
	classified := make(map[string]struct{}, 26)
	for _, values := range [][]string{fixture.Classification.Candidate, fixture.Classification.Partial,
		fixture.Classification.Pending, fixture.Classification.Rejected} {
		for _, id := range values {
			classified[id] = struct{}{}
		}
	}
	formalOwners := 0
	for _, capability := range roadmap.Capabilities {
		ownsV23 := false
		if capability.OwnerContext == "wizard" {
			for _, contract := range capability.AcceptanceContracts {
				if contract == "AC-V23-WIZARD" {
					ownsV23 = true
					break
				}
			}
		}
		if !ownsV23 {
			continue
		}
		formalOwners++
		if _, found := classified[capability.ID]; !found {
			t.Fatalf("formal V23 capability %s is not classified", capability.ID)
		}
	}
	if formalOwners != len(classified) {
		t.Fatalf("V23 formal owners=%d classified=%d", formalOwners, len(classified))
	}
	for _, transfer := range fixture.ScopeTransfers {
		if _, counted := classified[transfer.CapabilityID]; counted {
			t.Fatalf("transferred capability %s counted in V23", transfer.CapabilityID)
		}
	}
}

func assertV23CapabilityClassification(t *testing.T, classification v23WizardClassification) {
	t.Helper()
	want := v23WizardClassification{
		Candidate: []string{
			"WIZ-03", "WIZ-04", "WIZ-05", "WIZ-06", "WIZ-07", "WIZ-08",
			"WIZ-09", "WIZ-15", "WIZ-18", "WIZ-22", "WIZ-23",
		},
		Partial: []string{
			"WIZ-01", "WIZ-02", "WIZ-11",
			"WIZ-16", "WIZ-17", "WIZ-19", "WIZ-20", "WIZ-21",
			"WIZ-24", "WIZ-25", "STG-01", "STG-03", "STG-07",
		},
		Pending:  []string{},
		Rejected: []string{"WIZ-12", "WIZ-14"},
	}
	if !reflect.DeepEqual(classification, want) {
		t.Fatalf("V23 capability classification = %+v want=%+v", classification, want)
	}
	seen := make(map[string]string, 26)
	for class, ids := range map[string][]string{
		"candidate": classification.Candidate,
		"partial":   classification.Partial,
		"pending":   classification.Pending,
		"rejected":  classification.Rejected,
	} {
		for _, id := range ids {
			if previous, duplicate := seen[id]; duplicate {
				t.Fatalf("V23 capability %q is classified as both %s and %s", id, previous, class)
			}
			seen[id] = class
		}
	}
	if len(seen) != 26 {
		t.Fatalf("V23 classification covers %d unique capabilities, want 26", len(seen))
	}
}

func assertV23NamedGatesResolveExactlyOneTest(
	t *testing.T,
	root string,
	fixture v23WizardFixture,
) {
	t.Helper()
	gates := []struct {
		name     string
		value    v23WizardRequiredTest
		packages []string
	}{
		{
			name: "required_test", value: fixture.RequiredTest,
			packages: []string{"acceptance"},
		},
		{
			name: "integration_gate", value: fixture.IntegrationGate,
			packages: []string{
				"internal/intake",
				"internal/wizard/catalog",
				"internal/wizard/gaps",
				"internal/wizard/stages",
				"internal/application",
				"internal/adapters/state/sqlite",
				"internal/commands",
				"internal/bootstrap",
			},
		},
	}
	declaredByPackage := make(map[string]map[string]int, len(gates[1].packages)+1)
	claimed := make(map[string]string)
	for _, gate := range gates {
		if !gate.value.RejectNoTestsToRun || len(gate.value.TestNames) == 0 {
			t.Fatalf("%s does not fail closed on an empty test selection: %+v", gate.name, gate.value)
		}
		runNames := v23GateRunNames(t, gate.name, gate.value.Command)
		if !reflect.DeepEqual(runNames, gate.value.TestNames) {
			t.Fatalf("%s command names=%v want fixture names=%v",
				gate.name, runNames, gate.value.TestNames)
		}
		for _, name := range gate.value.TestNames {
			if previous, duplicate := claimed[name]; duplicate {
				t.Fatalf("test gate %q is claimed by both %s and %s", name, previous, gate.name)
			}
			claimed[name] = gate.name
			count := 0
			for _, packagePath := range gate.packages {
				declared, found := declaredByPackage[packagePath]
				if !found {
					declared = v23PackageTestDeclarations(t, root, packagePath)
					declaredByPackage[packagePath] = declared
				}
				count += declared[name]
			}
			if count != 1 {
				t.Fatalf("%s named gate %q resolves to %d Test functions, want exactly one",
					gate.name, name, count)
			}
		}
	}
}

func v23GateRunNames(t *testing.T, gateName, command string) []string {
	t.Helper()
	const prefix = "-run '^("
	start := strings.Index(command, prefix)
	end := strings.LastIndex(command, ")$'")
	if start < 0 || end < 0 || end <= start+len(prefix) {
		t.Fatalf("%s has no exact anchored -run selector: %q", gateName, command)
	}
	if strings.Count(command, prefix) != 1 || strings.Count(command, ")$'") != 1 {
		t.Fatalf("%s has an ambiguous -run selector: %q", gateName, command)
	}
	return strings.Split(command[start+len(prefix):end], "|")
}

func v23PackageTestDeclarations(t *testing.T, root, packagePath string) map[string]int {
	t.Helper()
	pattern := filepath.Join(root, filepath.FromSlash(packagePath), "*_test.go")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %s: %v", packagePath, err)
	}
	if len(paths) == 0 {
		t.Fatalf("gate package %q has no test files", packagePath)
	}
	declared := make(map[string]int)
	for _, path := range paths {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "Test") {
				continue
			}
			declared[function.Name.Name]++
		}
	}
	return declared
}

func assertV23IntakeImportBoundary(t *testing.T, root string, forbidden []string) {
	t.Helper()
	intakeRoot := filepath.Join(root, "internal", "intake")
	var goFiles []string
	err := filepath.WalkDir(intakeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		goFiles = append(goFiles, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(goFiles) == 0 {
		t.Fatal("internal/intake contains no Go files")
	}
	sort.Strings(goFiles)
	for _, path := range goFiles {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(importPath, "orquesta/") ||
				v23ImportMatchesBoundary(importPath, forbidden) {
				t.Fatalf("internal/intake imports forbidden boundary %q", importPath)
			}
		}
	}
}

func v23ImportMatchesBoundary(importPath string, forbidden []string) bool {
	segments := strings.Split(importPath, "/")
	for _, boundary := range forbidden {
		if strings.Contains(boundary, "/") {
			if importPath == boundary || strings.HasPrefix(importPath, boundary+"/") ||
				strings.Contains(importPath, "/"+boundary+"/") ||
				strings.HasSuffix(importPath, "/"+boundary) {
				return true
			}
			continue
		}
		for _, segment := range segments {
			if segment == boundary {
				return true
			}
		}
	}
	return false
}

func assertV23StateUnchanged(t *testing.T, state intake.State, revision intake.Revision, issues, questions, history int) {
	t.Helper()
	if state.Revision() != revision || len(state.Issues()) != issues ||
		len(state.Questions()) != questions || len(state.History()) != history {
		t.Fatalf("state mutated after rejection: revision=%d issues=%d questions=%d history=%d",
			state.Revision(), len(state.Issues()), len(state.Questions()), len(state.History()))
	}
}

func v23AudienceGap() intake.Issue {
	return intake.Issue{
		Ref: "intake-issue:audience-gap", Kind: intake.IssueGap,
		Field: "audience", DetailKey: "intake.issue.audience.missing",
	}
}

func v23AudienceQuestion(recommendTeam, recommendPersonal bool) intake.Question {
	return intake.Question{
		Ref:         "intake-question:audience",
		DerivedFrom: []intake.IssueRef{"intake-issue:audience-gap"},
		PromptKey:   "intake.question.audience.prompt",
		WhyKey:      "intake.question.audience.why",
		Options: []intake.Option{
			{
				Ref: "intake-option:audience-team", LabelKey: "intake.option.audience.team.label",
				RationaleKey: "intake.option.audience.team.rationale", Recommended: recommendTeam,
			},
			{
				Ref: "intake-option:audience-personal", LabelKey: "intake.option.audience.personal.label",
				RationaleKey: "intake.option.audience.personal.rationale", Recommended: recommendPersonal,
			},
		},
	}
}

func v23DeliveryQuestion(dependency intake.QuestionRef) intake.Question {
	return intake.Question{
		Ref:         "intake-question:delivery",
		DerivedFrom: []intake.IssueRef{"intake-issue:delivery-gap"},
		DependsOn:   []intake.QuestionRef{dependency},
		PromptKey:   "intake.question.delivery.prompt",
		WhyKey:      "intake.question.delivery.why",
		Options: []intake.Option{
			{
				Ref: "intake-option:delivery-local", LabelKey: "intake.option.delivery.local.label",
				RationaleKey: "intake.option.delivery.local.rationale", Recommended: true,
			},
			{
				Ref: "intake-option:delivery-cloud", LabelKey: "intake.option.delivery.cloud.label",
				RationaleKey: "intake.option.delivery.cloud.rationale",
			},
		},
	}
}
