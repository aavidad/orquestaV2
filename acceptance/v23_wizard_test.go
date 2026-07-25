package acceptance_test

import (
	"encoding/json"
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

	"orquesta/internal/intake"
)

const v23WizardFixturePath = "acceptance/fixtures/v23_wizard.json"

type v23WizardFixture struct {
	SchemaVersion        int                   `json:"schema_version"`
	ContractID           string                `json:"contract_id"`
	ImplementationStatus string                `json:"implementation_status"`
	TaskContext          v23WizardTaskContext  `json:"task_context"`
	OwnedCapabilityIDs   []string              `json:"owned_capability_ids"`
	PhaseCriteria        []string              `json:"phase_criteria"`
	PackagePath          string                `json:"package_path"`
	StateSchema          string                `json:"state_schema"`
	AllowedOrigins       []string              `json:"allowed_origins"`
	TypedRoundPolicy     v23WizardRoundPolicy  `json:"typed_round_policy"`
	StableErrorCodes     []string              `json:"stable_error_codes"`
	RequiredAssertions   []string              `json:"required_assertions"`
	RequiredTest         v23WizardRequiredTest `json:"required_test"`
	CandidateFiles       []string              `json:"candidate_files"`
	ForbiddenImports     []string              `json:"forbidden_import_boundaries"`
	RemainingWIZ         []v23WizardCapability `json:"remaining_wiz"`
	DeferredScopes       []string              `json:"deferred_scopes"`
	SealStatus           string                `json:"seal_status"`
	ReceiptPath          string                `json:"receipt_path"`
	NextDependency       string                `json:"next_causal_dependency"`
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
}

type v23WizardRequiredTest struct {
	Command            string   `json:"command"`
	TestNames          []string `json:"test_names"`
	RejectNoTestsToRun bool     `json:"reject_no_tests_to_run"`
}

type v23WizardCapability struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func TestAcceptanceV23WizardIntakeContract(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := loadV23WizardFixture(t, root)
	assertV23WizardFixture(t, root, fixture)
	assertV23IntakeImportBoundary(t, root, fixture.ForbiddenImports)

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
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V23-WIZARD" ||
		fixture.ImplementationStatus != "partial_green_unsealed" ||
		fixture.PackagePath != "internal/intake" || fixture.StateSchema != intake.StateSchema ||
		fixture.SealStatus != "not_sealed" || fixture.ReceiptPath != "" ||
		fixture.NextDependency == "" {
		t.Fatalf("invalid V23 partial envelope: %+v", fixture)
	}
	if fixture.TaskContext != (v23WizardTaskContext{
		ProjectRef: "project:v23", GoalRef: "goal:a7b8704cc9b368213fd7c201dfc8e6c9",
		WorkItemRef:    "work-item:7e8333c215d71ea3199bb4b48fea7a15",
		ExecutionRef:   "execution:1e8b0c8164d7dda5e626f43e108996f8",
		PlanGeneration: 1, AppSpecGeneration: 1,
	}) {
		t.Fatalf("invalid task context: %+v", fixture.TaskContext)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"WIZ-03", "WIZ-04", "WIZ-15"}) ||
		!reflect.DeepEqual(fixture.PhaseCriteria, []string{
			"criterion:v23-one-versioned-intake",
			"criterion:v23-gap-derived-questions",
			"criterion:v23-visible-recommendation",
			"criterion:v23-nonempty-required-test",
		}) ||
		!reflect.DeepEqual(fixture.AllowedOrigins, []string{"chat", "form"}) {
		t.Fatalf("invalid capability/criterion scope: %+v", fixture)
	}
	if fixture.TypedRoundPolicy != (v23WizardRoundPolicy{
		Field: "max_question_rounds", ExampleValue: 2, ZeroIsInvalid: true,
		PackageDefault: "none", CanonicalConfiguration: "pending_goal_with_lease_L-CONFIG",
	}) {
		t.Fatalf("invalid round policy boundary: %+v", fixture.TypedRoundPolicy)
	}
	wantErrors := []string{
		string(intake.ErrorChannelStateCreation), string(intake.ErrorChoiceConflict),
		string(intake.ErrorDuplicateRef), string(intake.ErrorInvalidArgument),
		string(intake.ErrorInvalidOrigin), string(intake.ErrorInvalidRef),
		string(intake.ErrorIssueNotFound), string(intake.ErrorMessageKeyInvalid),
		string(intake.ErrorOptionNotFound), string(intake.ErrorQuestionNotFound),
		string(intake.ErrorRecommendationCount), string(intake.ErrorRevisionConflict),
		string(intake.ErrorRoundLimit), string(intake.ErrorStateMismatch),
	}
	wantAssertions := []string{
		"chat_and_form_advance_one_ref_and_revision_sequence",
		"failed_change_does_not_mutate_the_current_snapshot",
		"question_references_at_least_one_recorded_gap_or_contradiction",
		"required_test_runs_named_contract_and_two_negatives",
		"selected_and_recommended_options_coexist_when_they_differ",
		"typed_policy_is_shared_across_origins_and_has_no_package_default",
	}
	if !reflect.DeepEqual(fixture.StableErrorCodes, wantErrors) ||
		!reflect.DeepEqual(fixture.RequiredAssertions, wantAssertions) {
		t.Fatalf("invalid stable codes/assertions: %+v", fixture)
	}
	wantRequiredTest := v23WizardRequiredTest{
		Command: "go test -mod=vendor -race -count=1 -v ./acceptance -run '^(TestAcceptanceV23WizardIntakeContract|TestV23WizardIntakeNegativeAndAtomicContract|TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset)$'",
		TestNames: []string{
			"TestAcceptanceV23WizardIntakeContract",
			"TestV23WizardIntakeNegativeAndAtomicContract",
			"TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset",
		},
		RejectNoTestsToRun: true,
	}
	if !reflect.DeepEqual(fixture.RequiredTest, wantRequiredTest) {
		t.Fatalf("invalid required test: %+v", fixture.RequiredTest)
	}
	if !reflect.DeepEqual(fixture.ForbiddenImports, []string{
		"adapters", "bootstrap", "database", "filesystem", "interfaces",
		"legacy", "orquesta/modulos", "provider",
	}) {
		t.Fatalf("invalid import boundary: %+v", fixture.ForbiddenImports)
	}
	assertV23RemainingCapabilities(t, fixture.RemainingWIZ)
	wantDeferred := []string{
		"application_idempotency", "canonical_round_default_under_L-CONFIG",
		"causal_plan_creation", "command_registry_binding", "dossier_generation",
		"durable_cas_persistence_and_restart", "explicit_confirmation",
		"freeze_after_confirmation", "full_wizard_i18n_catalog",
		"roadmap_promotion", "seal_and_receipt", "templates_and_domain_packs",
		"web_surface",
	}
	if !reflect.DeepEqual(fixture.DeferredScopes, wantDeferred) {
		t.Fatalf("invalid deferred scope: %+v", fixture.DeferredScopes)
	}
	wantCandidateFiles := []string{
		"acceptance/fixtures/v23_wizard.json",
		"acceptance/v23_wizard_test.go",
		"docs/reconstruccion/analisis_y_contrato_v23_wizard.md",
		"internal/intake/errors.go",
		"internal/intake/intake_test.go",
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
}

func assertV23RemainingCapabilities(t *testing.T, remaining []v23WizardCapability) {
	t.Helper()
	want := make(map[string]string, 22)
	for number := 1; number <= 25; number++ {
		id := "WIZ-" + twoDigits(number)
		if id == "WIZ-03" || id == "WIZ-04" || id == "WIZ-15" {
			continue
		}
		want[id] = "pending"
	}
	want["WIZ-12"], want["WIZ-14"] = "roadmap_rejected", "roadmap_rejected"
	got := make(map[string]string, len(remaining))
	for _, capability := range remaining {
		if _, duplicate := got[capability.ID]; duplicate {
			t.Fatalf("duplicate remaining capability %q", capability.ID)
		}
		got[capability.ID] = capability.Status
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining capabilities = %+v want=%+v", got, want)
	}
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

func twoDigits(number int) string {
	if number < 10 {
		return "0" + strconv.Itoa(number)
	}
	return strconv.Itoa(number)
}
