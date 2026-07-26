package catalog

import (
	"reflect"
	"testing"
)

func TestBuiltInCatalogIsVersionedAndTracesRelatedRoadmapCapabilities(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	if got, want := value.Version().String(), currentVersionValue; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
	if got, want := len(value.Packs()), 1+len(domainKeys); got != want {
		t.Fatalf("packs = %d, want %d", got, want)
	}

	selection, err := value.Compose()
	if err != nil {
		t.Fatalf("compose foundation: %v", err)
	}
	wantRoadmapRefs := []RoadmapCapabilityRef{
		mustRoadmapCapabilityRef("WIZ-01"),
		mustRoadmapCapabilityRef("WIZ-06"),
		mustRoadmapCapabilityRef("WIZ-11"),
		mustRoadmapCapabilityRef("WIZ-16"),
		mustRoadmapCapabilityRef("WIZ-17"),
		mustRoadmapCapabilityRef("WIZ-19"),
		mustRoadmapCapabilityRef("WIZ-25"),
	}
	if got := selection.RoadmapCapabilityRefs(); !reflect.DeepEqual(got, wantRoadmapRefs) {
		t.Fatalf("roadmap refs = %v, want %v", got, wantRoadmapRefs)
	}

	assertQuestionKind(t, selection, "question:work_mode", AnswerChoice)
	assertQuestionKind(t, selection, "question:objective", AnswerFreeText)
	assertQuestionKind(t, selection, "question:app_template", AnswerChoice)
	assertQuestionKind(t, selection, "question:quality_profile", AnswerChoice)
	assertQuestionKind(t, selection, "question:assistance_level", AnswerChoice)
	assertQuestionKind(t, selection, "question:inference_mode", AnswerChoice)

	if len(selection.Defaults()) == 0 {
		t.Fatal("foundation has no disclosed technical defaults")
	}
	for _, technicalDefault := range selection.Defaults() {
		if technicalDefault.DecisionKind() != DecisionTechnicalDefault {
			t.Fatalf(
				"default %q kind = %q",
				technicalDefault.Ref().String(),
				technicalDefault.DecisionKind(),
			)
		}
	}
	for _, question := range selection.Questions() {
		if question.DecisionKind() != DecisionProduct {
			t.Fatalf(
				"question %q kind = %q",
				question.Ref().String(),
				question.DecisionKind(),
			)
		}
	}
}

func TestComposeCombinedPacksIsOrderIndependentAndDeduplicated(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	calendar := mustPackRef("pack:calendar")
	commerce := mustPackRef("pack:commerce")

	left, err := value.Compose(calendar, commerce, calendar)
	if err != nil {
		t.Fatalf("compose left: %v", err)
	}
	right, err := value.Compose(commerce, calendar)
	if err != nil {
		t.Fatalf("compose right: %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("composition depends on input order or duplicate refs\nleft: %#v\nright: %#v", left, right)
	}

	assertUniqueRefs(t, "packs", packRefStrings(left.PackRefs()))
	assertUniqueRefs(t, "taxonomies", taxonomyRefStrings(left.Taxonomies()))
	assertUniqueRefs(t, "questions", questionRefStrings(left.Questions()))
	assertUniqueRefs(t, "defaults", defaultRefStrings(left.Defaults()))
	if got := countString(questionRefStrings(left.Questions()), "question:data_sensitivity"); got != 1 {
		t.Fatalf("shared data sensitivity question appears %d times, want 1", got)
	}
	if got := countString(taxonomyRefStrings(left.Taxonomies()), "taxonomy:data_sensitivity"); got != 1 {
		t.Fatalf("shared data sensitivity taxonomy appears %d times, want 1", got)
	}
	if !containsString(
		questionRefStrings(left.Questions()),
		"question:domain_calendar_integration_mode",
	) {
		t.Fatal("calendar question missing")
	}
	if !containsString(
		questionRefStrings(left.Questions()),
		"question:domain_commerce_integration_mode",
	) {
		t.Fatal("commerce question missing")
	}
}

func TestEveryBuiltInOptionHasHelpExampleAndRationaleKeys(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	for _, pack := range value.Packs() {
		assertMessageKey(t, pack.Ref().String()+".label", pack.LabelKey())
		assertMessageKey(t, pack.Ref().String()+".help", pack.HelpKey())
		assertMessageKey(t, pack.Ref().String()+".example", pack.ExampleKey())
		for _, taxonomy := range pack.Taxonomies() {
			assertMessageKey(t, taxonomy.Ref().String()+".label", taxonomy.LabelKey())
			assertMessageKey(t, taxonomy.Ref().String()+".help", taxonomy.HelpKey())
			assertMessageKey(t, taxonomy.Ref().String()+".example", taxonomy.ExampleKey())
			for _, term := range taxonomy.Terms() {
				assertMessageKey(t, term.Ref().String()+".label", term.LabelKey())
				assertMessageKey(t, term.Ref().String()+".help", term.HelpKey())
				assertMessageKey(t, term.Ref().String()+".example", term.ExampleKey())
			}
		}
		for _, question := range pack.Questions() {
			assertMessageKey(t, question.Ref().String()+".prompt", question.PromptKey())
			assertMessageKey(t, question.Ref().String()+".why", question.WhyKey())
			assertMessageKey(t, question.Ref().String()+".help", question.HelpKey())
			assertMessageKey(t, question.Ref().String()+".example", question.ExampleKey())
			recommended := 0
			for _, option := range question.Options() {
				assertMessageKey(t, option.Ref().String()+".label", option.LabelKey())
				assertMessageKey(t, option.Ref().String()+".help", option.HelpKey())
				assertMessageKey(t, option.Ref().String()+".example", option.ExampleKey())
				assertMessageKey(t, option.Ref().String()+".rationale", option.RationaleKey())
				if option.Recommended() {
					recommended++
				}
			}
			if question.AnswerKind() == AnswerChoice && recommended != 1 {
				t.Fatalf(
					"question %q has %d recommendations, want 1",
					question.Ref().String(),
					recommended,
				)
			}
		}
		for _, technicalDefault := range pack.Defaults() {
			assertMessageKey(t, technicalDefault.Ref().String()+".label", technicalDefault.LabelKey())
			assertMessageKey(t, technicalDefault.Ref().String()+".help", technicalDefault.HelpKey())
			assertMessageKey(t, technicalDefault.Ref().String()+".example", technicalDefault.ExampleKey())
			assertMessageKey(t, technicalDefault.Ref().String()+".rationale", technicalDefault.RationaleKey())
		}
	}
}

func TestPackSelectionIsExplicitAndNeverInferredFromText(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	foundation, err := value.Compose()
	if err != nil {
		t.Fatalf("compose foundation: %v", err)
	}
	if got := domainQuestionCount(foundation.Questions()); got != 0 {
		t.Fatalf("foundation domain questions = %d, want 0", got)
	}

	calendar, err := value.Compose(mustPackRef("pack:calendar"))
	if err != nil {
		t.Fatalf("compose calendar: %v", err)
	}
	refs := questionRefStrings(calendar.Questions())
	if !containsString(refs, "question:domain_calendar_integration_mode") {
		t.Fatal("explicit calendar pack did not add calendar question")
	}
	if containsString(refs, "question:domain_commerce_integration_mode") {
		t.Fatal("unselected commerce pack leaked into composition")
	}
}

func TestCatalogAndSelectionsReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	first := value.Packs()
	originalPackRef := first[0].Ref()
	first[0] = Pack{}
	if got := value.Packs()[0].Ref(); got != originalPackRef {
		t.Fatalf("catalog packs mutated through accessor: %q", got.String())
	}

	selection := value.ComposeAll()
	questions := selection.Questions()
	if len(questions) == 0 {
		t.Fatal("compose all returned no questions")
	}
	originalQuestionRef := questions[0].Ref()
	options := questions[0].Options()
	questions[0] = Question{}
	if len(options) > 0 {
		options[0] = Option{}
	}
	freshQuestions := selection.Questions()
	if freshQuestions[0].Ref() != originalQuestionRef {
		t.Fatal("selection questions mutated through accessor")
	}
	if len(freshQuestions[0].Options()) > 0 &&
		freshQuestions[0].Options()[0].Ref().String() == "" {
		t.Fatal("question options mutated through accessor")
	}
}

func TestCatalogRejectsConflictingDefinitionsAcrossPacks(t *testing.T) {
	t.Parallel()

	value := BuiltIn()
	packs := value.Packs()
	changed := false
	for packIndex := range packs {
		if packs[packIndex].Ref().String() != "pack:commerce" {
			continue
		}
		for questionIndex := range packs[packIndex].questions {
			if packs[packIndex].questions[questionIndex].Ref().String() !=
				"question:data_sensitivity" {
				continue
			}
			packs[packIndex].questions[questionIndex].promptKey =
				key("wizard.question.data_sensitivity.alternate_prompt")
			changed = true
		}
	}
	if !changed {
		t.Fatal("test did not find shared question")
	}
	_, err := NewCatalog(CatalogInput{
		Version: value.Version(), FoundationRef: value.FoundationRef(), Packs: packs,
	})
	if got := ErrorCodeOf(err); got != ErrorConflictingDefinition {
		t.Fatalf("error code = %q, want %q; err=%v", got, ErrorConflictingDefinition, err)
	}
}

func TestConstructorsRejectIncompleteGuidanceAndInvalidRecommendationCount(t *testing.T) {
	t.Parallel()

	termRef := mustTermRef("term:test.one")
	_, err := NewOption(OptionInput{
		Ref:          mustOptionRef("option:test.one"),
		TermRef:      termRef,
		LabelKey:     key("wizard.test.one.label"),
		HelpKey:      key("wizard.test.one.help"),
		RationaleKey: key("wizard.test.one.rationale"),
	})
	if got := ErrorCodeOf(err); got != ErrorInvalidMessageKey {
		t.Fatalf("missing example error = %q, want %q", got, ErrorInvalidMessageKey)
	}

	options := []Option{
		testOption("one", false),
		testOption("two", false),
	}
	_, err = NewQuestion(QuestionInput{
		Ref:          mustQuestionRef("question:test"),
		Slot:         mustSlotKey("test.slot"),
		AnswerKind:   AnswerChoice,
		DecisionKind: DecisionProduct,
		TaxonomyRef:  mustTaxonomyRef("taxonomy:test"),
		PromptKey:    key("wizard.test.prompt"),
		WhyKey:       key("wizard.test.why"),
		HelpKey:      key("wizard.test.help"),
		ExampleKey:   key("wizard.test.example"),
		Options:      options,
	})
	if got := ErrorCodeOf(err); got != ErrorRecommendationCount {
		t.Fatalf("recommendation error = %q, want %q", got, ErrorRecommendationCount)
	}
}

func TestComposeRejectsUnknownPackRef(t *testing.T) {
	t.Parallel()

	_, err := BuiltIn().Compose(mustPackRef("pack:unknown"))
	if got := ErrorCodeOf(err); got != ErrorPackNotFound {
		t.Fatalf("error code = %q, want %q; err=%v", got, ErrorPackNotFound, err)
	}
}

func assertQuestionKind(t *testing.T, selection Selection, ref string, kind AnswerKind) {
	t.Helper()
	for _, question := range selection.Questions() {
		if question.Ref().String() != ref {
			continue
		}
		if question.AnswerKind() != kind {
			t.Fatalf("%s kind = %q, want %q", ref, question.AnswerKind(), kind)
		}
		return
	}
	t.Fatalf("question %q not found", ref)
}

func assertMessageKey(t *testing.T, field string, value MessageKey) {
	t.Helper()
	if value.String() == "" {
		t.Fatalf("%s is empty", field)
	}
	if _, err := NewMessageKey(value.String()); err != nil {
		t.Fatalf("%s = %q is invalid: %v", field, value.String(), err)
	}
}

func assertUniqueRefs(t *testing.T, field string, refs []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, ok := seen[ref]; ok {
			t.Fatalf("%s contains duplicate ref %q", field, ref)
		}
		seen[ref] = struct{}{}
	}
}

func packRefStrings(values []PackRef) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.String()
	}
	return out
}

func taxonomyRefStrings(values []Taxonomy) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.Ref().String()
	}
	return out
}

func questionRefStrings(values []Question) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.Ref().String()
	}
	return out
}

func defaultRefStrings(values []TechnicalDefault) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.Ref().String()
	}
	return out
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}

func containsString(values []string, target string) bool {
	return countString(values, target) > 0
}

func domainQuestionCount(values []Question) int {
	count := 0
	for _, value := range values {
		if value.Slot().String() == "data.sensitivity" {
			count++
			continue
		}
		for _, domain := range domainKeys {
			if value.Slot().String() == "domains."+domain+".integration_mode" {
				count++
			}
		}
	}
	return count
}

func testOption(name string, recommended bool) Option {
	prefix := "wizard.test." + name
	return mustOption(OptionInput{
		Ref:          mustOptionRef("option:test." + name),
		TermRef:      mustTermRef("term:test." + name),
		LabelKey:     key(prefix + ".label"),
		HelpKey:      key(prefix + ".help"),
		ExampleKey:   key(prefix + ".example"),
		RationaleKey: key(prefix + ".rationale"),
		Recommended:  recommended,
	})
}
