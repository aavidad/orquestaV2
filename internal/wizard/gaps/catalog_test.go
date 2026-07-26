package gaps

import (
	"reflect"
	"testing"
)

func TestInventoryCoversUniversalTechnicalAndCrossRules(t *testing.T) {
	t.Parallel()

	inventory := BuiltIn()
	if inventory.SchemaVersion() != SchemaVersion {
		t.Fatalf("schema = %q", inventory.SchemaVersion())
	}
	wantDimensions := []DimensionRef{
		DimensionU1, DimensionU2, DimensionU3, DimensionU4,
		DimensionU5, DimensionU6, DimensionU7, DimensionU8,
		DimensionU9, DimensionU10, DimensionU11, DimensionU12,
		DimensionT1, DimensionT2, DimensionT3, DimensionT4,
		DimensionT5, DimensionT6, DimensionT7, DimensionT8,
	}
	dimensions := inventory.Dimensions()
	if len(dimensions) != len(wantDimensions) {
		t.Fatalf("dimensions = %d, want %d", len(dimensions), len(wantDimensions))
	}
	for index, want := range wantDimensions {
		got := dimensions[index]
		if got.Ref() != want || got.Ordinal() != index+1 {
			t.Fatalf("dimension[%d] = %s/%d", index, got.Ref(), got.Ordinal())
		}
		wantLayer := LayerUniversal
		wantKind := DecisionProduct
		if index >= 12 {
			wantLayer = LayerTechnical
			wantKind = DecisionTechnical
			if got.DefaultOption() == "" {
				t.Fatalf("%s has no disclosed technical default", got.Ref())
			}
		}
		if got.Layer() != wantLayer || got.DecisionKind() != wantKind {
			t.Fatalf("%s layer/kind = %s/%s", got.Ref(), got.Layer(), got.DecisionKind())
		}
		assertPresentationKeys(t, string(got.Ref()), got.PromptKey(), got.WhyKey(), got.HelpKey(), got.ExampleKey())
		assertOptionCatalog(t, string(got.Ref()), got.Options())
	}

	wantRules := []RuleRef{
		RuleR1, RuleR2, RuleR3, RuleR4, RuleR5, RuleR6, RuleR7, RuleR8,
	}
	rules := inventory.Rules()
	if len(rules) != len(wantRules) {
		t.Fatalf("rules = %d, want %d", len(rules), len(wantRules))
	}
	for index, want := range wantRules {
		if rules[index].Ref() != want || rules[index].Ordinal() != index+1 {
			t.Fatalf("rule[%d] = %s/%d", index, rules[index].Ref(), rules[index].Ordinal())
		}
		if rules[index].DetailKey() == "" {
			t.Fatalf("%s has no detail key", want)
		}
	}
}

func TestEmptyInputEmitsTwelveUniversalGapsAndEightDisclosedDefaults(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{})
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion() != SchemaVersion {
		t.Fatalf("schema = %q", result.SchemaVersion())
	}
	questions := result.Questions()
	if len(questions) != 12 {
		t.Fatalf("questions = %d, want 12", len(questions))
	}
	issues := issueIndex(result.Issues())
	for index, question := range questions {
		want := DimensionRef("U" + string(rune('1'+index)))
		if index >= 9 {
			want = []DimensionRef{DimensionU10, DimensionU11, DimensionU12}[index-9]
		}
		if question.Dimension() != want {
			t.Fatalf("question[%d] dimension = %s, want %s", index, question.Dimension(), want)
		}
		assertQuestionCausality(t, question, issues)
		assertOptionCatalog(t, string(question.Ref()), question.Options())
	}
	defaults := result.DefaultProposals()
	if len(defaults) != 8 {
		t.Fatalf("defaults = %d, want 8", len(defaults))
	}
	for index, item := range defaults {
		want := DimensionRef("T" + string(rune('1'+index)))
		if item.Dimension() != want {
			t.Fatalf("default[%d] = %s, want %s", index, item.Dimension(), want)
		}
		if item.DecisionKind() != DecisionTechnicalDefault || item.ImplicitlyApplied() {
			t.Fatalf("%s default is not disclosed/unapplied", item.Dimension())
		}
		assertPresentationKeys(
			t,
			string(item.Dimension()),
			item.LabelKey(),
			item.HelpKey(),
			item.ExampleKey(),
			item.RationaleKey(),
		)
	}
}

func TestInventoryAndResultsReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	inventory := BuiltIn()
	dimensions := inventory.Dimensions()
	dimensions[0].dependsOn = []DimensionRef{DimensionT8}
	dimensions[0].options[1].recommended = false
	again := inventory.Dimensions()
	if len(again[0].DependsOn()) != 0 || !again[0].Options()[1].Recommended() {
		t.Fatal("inventory aliases caller-owned slices")
	}

	result, err := Evaluate(Input{})
	if err != nil {
		t.Fatal(err)
	}
	questions := result.Questions()
	questions[0].derivedFrom[0] = "mutated"
	questions[0].options[1].recommended = false
	questions[2].dependsOn[0] = "mutated"
	issues := result.Issues()
	issues[2].dependsOn[0] = DimensionT8
	defaults := result.DefaultProposals()
	defaults[0].dimension = DimensionU1

	freshQuestions := result.Questions()
	if freshQuestions[0].DerivedFrom()[0] == "mutated" ||
		!freshQuestions[0].Options()[1].Recommended() ||
		freshQuestions[2].DependsOn()[0] == "mutated" {
		t.Fatal("question result aliases caller-owned slices")
	}
	if result.Issues()[2].DependsOn()[0] == DimensionT8 {
		t.Fatal("issue result aliases caller-owned slices")
	}
	if result.DefaultProposals()[0].Dimension() == DimensionU1 {
		t.Fatal("default result aliases caller-owned slice")
	}
}

func TestSemanticQuestionDependenciesAreExplicit(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{})
	if err != nil {
		t.Fatal(err)
	}
	questions := result.Questions()
	tests := []struct {
		ref  DimensionRef
		want QuestionRef
	}{
		{DimensionU3, questionRef(DimensionU1)},
		{DimensionU6, questionRef(DimensionU1)},
		{DimensionU11, questionRef(DimensionU1)},
		{DimensionU12, questionRef(DimensionU4)},
	}
	for _, test := range tests {
		question, found := findDimensionQuestion(questions, test.ref)
		if !found {
			t.Fatalf("missing %s", test.ref)
		}
		if !containsQuestionRef(question.DependsOn(), test.want) {
			t.Fatalf("%s dependencies = %#v, want %s", test.ref, question.DependsOn(), test.want)
		}
	}
}

func assertOptionCatalog(t *testing.T, owner string, options []Option) {
	t.Helper()
	recommendations := 0
	freeText := 0
	refs := make(map[OptionRef]struct{}, len(options))
	for _, option := range options {
		if _, duplicate := refs[option.Ref()]; duplicate {
			t.Fatalf("%s duplicate option %s", owner, option.Ref())
		}
		refs[option.Ref()] = struct{}{}
		if option.Recommended() {
			recommendations++
		}
		if option.Kind() == OptionFreeText {
			freeText++
		}
		assertPresentationKeys(
			t,
			owner+"/"+string(option.Ref()),
			option.LabelKey(),
			option.HelpKey(),
			option.ExampleKey(),
			option.RationaleKey(),
		)
	}
	if recommendations != 1 {
		t.Fatalf("%s recommendations = %d", owner, recommendations)
	}
	if freeText != 1 {
		t.Fatalf("%s real free-text options = %d", owner, freeText)
	}
}

func assertPresentationKeys(t *testing.T, owner string, keys ...MessageKey) {
	t.Helper()
	for index, key := range keys {
		if key == "" {
			t.Fatalf("%s presentation key[%d] empty", owner, index)
		}
	}
}

func issueIndex(values []Issue) map[IssueRef]Issue {
	result := make(map[IssueRef]Issue, len(values))
	for _, value := range values {
		result[value.Ref()] = value
	}
	return result
}

func assertQuestionCausality(
	t *testing.T,
	question Question,
	issues map[IssueRef]Issue,
) {
	t.Helper()
	if len(question.DerivedFrom()) == 0 {
		t.Fatalf("%s has no issue refs", question.Ref())
	}
	for _, ref := range question.DerivedFrom() {
		if _, found := issues[ref]; !found {
			t.Fatalf("%s derives from absent issue %s", question.Ref(), ref)
		}
	}
}

func containsQuestionRef(values []QuestionRef, want QuestionRef) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBuiltInIsStableAcrossCalls(t *testing.T) {
	t.Parallel()
	if !reflect.DeepEqual(BuiltIn(), BuiltIn()) {
		t.Fatal("built-in inventory changed across calls")
	}
}
