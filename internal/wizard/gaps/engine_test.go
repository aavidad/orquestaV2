package gaps

import (
	"reflect"
	"sync"
	"testing"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
)

func TestAllTechnicalDimensionsActivateFromExplicitTypedFacts(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{
		Facts: Facts{
			CorporateIdentity:      DeclarationDeclared,
			TargetUsers:            DeclarationDeclared,
			IntegrationAuth:        DeclarationDeclared,
			IntegrationCriticality: DeclarationDeclared,
		},
		Selections: []Selection{
			preset(DimensionU1, "team"),
			preset(DimensionU4, "persisted_user_data"),
			preset(DimensionU5, "sensitive"),
			preset(DimensionU7, "public_api"),
			preset(DimensionU10, "cloud_container"),
			preset(DimensionU11, "realtime"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[DimensionRef]Question)
	for _, question := range result.Questions() {
		if question.DecisionKind() == DecisionTechnical {
			seen[question.Dimension()] = question
		}
	}
	for _, ref := range []DimensionRef{
		DimensionT1, DimensionT2, DimensionT3, DimensionT4,
		DimensionT5, DimensionT6, DimensionT7, DimensionT8,
	} {
		question, found := seen[ref]
		if !found {
			t.Errorf("missing active technical question %s", ref)
			continue
		}
		assertOptionCatalog(t, string(ref), question.Options())
	}
	if len(result.DefaultProposals()) != 0 {
		t.Fatalf("active technical dimensions also defaulted: %#v", result.DefaultProposals())
	}
}

func TestCrossRuleInventoryProducesSemanticIssues(t *testing.T) {
	t.Parallel()

	calendar := mustPackRef(t, "pack:calendar")
	tests := []struct {
		name  string
		input Input
		rule  RuleRef
		kind  IssueKind
	}{
		{
			name: "R1 typed sharing contradiction",
			input: Input{
				Facts:      Facts{SharingIntent: SharingShared},
				Selections: []Selection{preset(DimensionU1, "personal")},
			},
			rule: RuleR1, kind: IssueContradiction,
		},
		{
			name: "R2 typed surface contradiction",
			input: Input{
				Facts:      Facts{Surface: SurfaceServerService},
				Selections: []Selection{preset(DimensionU2, "responsive_web")},
			},
			rule: RuleR2, kind: IssueContradiction,
		},
		{
			name:  "R3 explicit pack dependency",
			input: Input{PackRefs: []catalog.PackRef{calendar}},
			rule:  RuleR3, kind: IssueGap,
		},
		{
			name: "R4 persisted data dependency",
			input: Input{
				Selections: []Selection{preset(DimensionU4, "persisted_user_data")},
			},
			rule: RuleR4, kind: IssueGap,
		},
		{
			name: "R5 missing integration governance",
			input: Input{
				Selections: []Selection{preset(DimensionU7, "external_services")},
			},
			rule: RuleR5, kind: IssueGap,
		},
		{
			name: "R6 native mobile platform contradiction",
			input: Input{
				Facts:      Facts{Surface: SurfaceNativeMobile},
				Selections: []Selection{preset(DimensionU2, "native_desktop")},
			},
			rule: RuleR6, kind: IssueContradiction,
		},
		{
			name: "R7 deployment contradiction",
			input: Input{
				Facts:      Facts{Surface: SurfaceNativeMobile},
				Selections: []Selection{preset(DimensionU10, "cloud_container")},
			},
			rule: RuleR7, kind: IssueContradiction,
		},
		{
			name: "R8 missing target users",
			input: Input{
				Facts:      Facts{TargetUsers: DeclarationMissing},
				Selections: []Selection{preset(DimensionU1, "team")},
			},
			rule: RuleR8, kind: IssueGap,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := Evaluate(test.input)
			if err != nil {
				t.Fatal(err)
			}
			var found *Issue
			for _, issue := range result.Issues() {
				if issue.RuleRef() == test.rule {
					copy := issue
					found = &copy
					break
				}
			}
			if found == nil {
				t.Fatalf("rule %s produced no issue", test.rule)
			}
			if found.Kind() != test.kind {
				t.Fatalf("rule %s kind = %s, want %s", test.rule, found.Kind(), test.kind)
			}
			causal := false
			for _, question := range result.Questions() {
				for _, ref := range question.DerivedFrom() {
					causal = causal || ref == found.Ref()
				}
			}
			if !causal {
				t.Fatalf("rule %s issue has no derived question", test.rule)
			}
		})
	}
}

func TestContradictorySelectionsAreRecoverableAndRequestioned(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{Selections: []Selection{
		preset(DimensionU1, "personal"),
		preset(DimensionU1, "team"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	issue, found := findDimensionIssue(result.Issues(), DimensionU1, IssueContradiction)
	if !found {
		t.Fatal("selection contradiction was not detected")
	}
	question, found := findDimensionQuestion(result.Questions(), DimensionU1)
	if !found {
		t.Fatal("recoverable contradiction did not re-question U1")
	}
	if !containsIssueRef(question.DerivedFrom(), issue.Ref()) {
		t.Fatalf("U1 question does not derive from %s", issue.Ref())
	}
}

func TestSelectionsAndPacksAreOrderIndependentAndDeduplicated(t *testing.T) {
	t.Parallel()

	calendar := mustPackRef(t, "pack:calendar")
	commerce := mustPackRef(t, "pack:commerce")
	left, err := Evaluate(Input{
		Facts: Facts{
			TargetUsers:            DeclarationDeclared,
			IntegrationAuth:        DeclarationDeclared,
			IntegrationCriticality: DeclarationDeclared,
		},
		Selections: []Selection{
			preset(DimensionU1, "team"),
			preset(DimensionU4, "persisted_user_data"),
			preset(DimensionU1, "team"),
		},
		PackRefs: []catalog.PackRef{commerce, calendar, commerce},
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := Evaluate(Input{
		Facts: Facts{
			TargetUsers:            DeclarationDeclared,
			IntegrationAuth:        DeclarationDeclared,
			IntegrationCriticality: DeclarationDeclared,
		},
		Selections: []Selection{
			preset(DimensionU4, "persisted_user_data"),
			preset(DimensionU1, "team"),
		},
		PackRefs: []catalog.PackRef{calendar, commerce},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatal("shuffle/repetition changed deterministic result")
	}
	if got := left.PackRefs(); len(got) != 2 ||
		got[0].String() != "pack:calendar" ||
		got[1].String() != "pack:commerce" {
		t.Fatalf("pack refs = %#v", got)
	}
	seenRefs := make(map[QuestionRef]struct{})
	seenSlots := make(map[SlotKey]struct{})
	packQuestions := 0
	for _, question := range left.Questions() {
		if _, duplicate := seenRefs[question.Ref()]; duplicate {
			t.Fatalf("duplicate question ref %s", question.Ref())
		}
		seenRefs[question.Ref()] = struct{}{}
		if question.PackRef().String() == "" {
			continue
		}
		packQuestions++
		if _, duplicate := seenSlots[question.Slot()]; duplicate {
			t.Fatalf("duplicate pack slot %s", question.Slot())
		}
		seenSlots[question.Slot()] = struct{}{}
	}
	if packQuestions != 2 {
		t.Fatalf("pack questions = %d, want 2", packQuestions)
	}
}

func TestPackAndRuleQuestionsAcceptTypedExplicitSelections(t *testing.T) {
	t.Parallel()

	calendar := mustPackRef(t, "pack:calendar")
	initial, err := Evaluate(Input{
		Selections: []Selection{preset(DimensionU7, "external_services")},
		PackRefs:   []catalog.PackRef{calendar},
	})
	if err != nil {
		t.Fatal(err)
	}
	var packQuestion Question
	var governance Question
	for _, question := range initial.Questions() {
		switch {
		case question.PackRef().String() == "pack:calendar":
			packQuestion = question
		case question.Ref() == QuestionRef("intake-question:wizard.r5"):
			governance = question
		}
	}
	if packQuestion.Ref() == "" || governance.Ref() == "" {
		t.Fatal("expected pack and R5 questions")
	}
	free, found := optionByKind(packQuestion.Options(), OptionFreeText)
	if !found {
		t.Fatal("pack question has no selectable free-text option")
	}
	recommended, found := governance.RecommendedOption()
	if !found {
		t.Fatal("R5 has no unique recommendation")
	}
	result, err := Evaluate(Input{
		Selections: []Selection{preset(DimensionU7, "external_services")},
		QuestionSelections: []QuestionSelection{
			{
				Question: packQuestion.Ref(),
				Option:   free.Ref(),
				FreeText: "connector:calendar-custom",
			},
			{Question: governance.Ref(), Option: recommended.Ref()},
		},
		PackRefs: []catalog.PackRef{calendar},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range result.Questions() {
		if question.Ref() == packQuestion.Ref() || question.Ref() == governance.Ref() {
			t.Fatalf("resolved supplemental question re-emitted: %s", question.Ref())
		}
	}
}

func TestEveryQuestionHasRealIssueRecommendationHelpAndFreeChoice(t *testing.T) {
	t.Parallel()

	calendar := mustPackRef(t, "pack:calendar")
	result, err := Evaluate(Input{
		Facts: Facts{
			CorporateIdentity: DeclarationDeclared,
			Surface:           SurfaceNativeMobile,
			SharingIntent:     SharingShared,
			TargetUsers:       DeclarationMissing,
			IntegrationAuth:   DeclarationMissing,
		},
		Selections: []Selection{
			preset(DimensionU1, "personal"),
			preset(DimensionU2, "native_desktop"),
			preset(DimensionU4, "persisted_user_data"),
			preset(DimensionU5, "regulated"),
			preset(DimensionU7, "external_services"),
			preset(DimensionU10, "cloud_container"),
			preset(DimensionU11, "realtime"),
		},
		PackRefs: []catalog.PackRef{calendar},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues := issueIndex(result.Issues())
	for _, question := range result.Questions() {
		assertQuestionCausality(t, question, issues)
		assertPresentationKeys(
			t,
			string(question.Ref()),
			question.PromptKey(),
			question.WhyKey(),
			question.HelpKey(),
			question.ExampleKey(),
		)
		assertOptionCatalog(t, string(question.Ref()), question.Options())
	}
}

func TestTypedKernelFactsRecommendNoUIAndKeepTechnicalOperations(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{Facts: Facts{Surface: SurfaceKernelModule}})
	if err != nil {
		t.Fatal(err)
	}
	platform, found := findDimensionQuestion(result.Questions(), DimensionU2)
	if !found {
		t.Fatal("missing U2")
	}
	recommended, found := platform.RecommendedOption()
	if !found || recommended.Ref() != optionRef(DimensionU2, "no_ui") {
		t.Fatalf("U2 recommendation = %s", recommended.Ref())
	}
	for _, ref := range []DimensionRef{DimensionT3, DimensionT6} {
		question, found := findDimensionQuestion(result.Questions(), ref)
		if !found {
			t.Fatalf("kernel facts dropped %s", ref)
		}
		recommended, found := question.RecommendedOption()
		if !found {
			t.Fatalf("%s has no recommendation", ref)
		}
		want := optionRef(DimensionT3, "kernel_trace")
		if ref == DimensionT6 {
			want = optionRef(DimensionT6, "kernel_build_ci")
		}
		if recommended.Ref() != want {
			t.Fatalf("%s recommendation = %s, want %s", ref, recommended.Ref(), want)
		}
	}
}

func TestChangedTypedAudienceReevaluatesTechnicalDependencies(t *testing.T) {
	t.Parallel()

	personal, err := Evaluate(Input{
		Facts:      Facts{TargetUsers: DeclarationDeclared},
		Selections: []Selection{preset(DimensionU1, "personal")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findDimensionQuestion(personal.Questions(), DimensionT1); found {
		t.Fatal("personal audience unexpectedly asks T1")
	}
	if !hasDefaultProposal(personal.DefaultProposals(), DimensionT1) {
		t.Fatal("inactive T1 default was not disclosed")
	}

	team, err := Evaluate(Input{
		Facts:      Facts{TargetUsers: DeclarationDeclared},
		Selections: []Selection{preset(DimensionU1, "team")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findDimensionQuestion(team.Questions(), DimensionT1); !found {
		t.Fatal("changed team audience did not activate T1")
	}
	if hasDefaultProposal(team.DefaultProposals(), DimensionT1) {
		t.Fatal("active T1 still emitted a default proposal")
	}
}

func TestQuestionProjectionIsAcceptedByPureIntake(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := intake.NewState(
		"intake:gaps-projection-universal",
		intake.Policy{MaxQuestionRounds: 6},
	)
	if err != nil {
		t.Fatal(err)
	}
	next := applyProjectedQuestions(t, state, result, questionRefs(result.Questions())...)
	if next.QuestionRounds() != 1 ||
		len(next.Questions()) != len(result.Questions()) {
		t.Fatalf(
			"projected state rounds/questions = %d/%d, want 1/%d",
			next.QuestionRounds(),
			len(next.Questions()),
			len(result.Questions()),
		)
	}
}

func TestResolvedParentDependencyReopensTechnicalDecisionInIntake(t *testing.T) {
	t.Parallel()

	initial, err := Evaluate(Input{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := intake.NewState(
		"intake:gaps-causal-reopen",
		intake.Policy{MaxQuestionRounds: 6},
	)
	if err != nil {
		t.Fatal(err)
	}
	state = applyProjectedQuestions(t, state, initial, questionRef(DimensionU1))
	state = applyIntakeChoices(t, state, intake.Choice{
		QuestionRef: intake.QuestionRef(questionRef(DimensionU1)),
		OptionRef:   intake.OptionRef(optionRef(DimensionU1, "team")),
	})

	afterAudience, err := Evaluate(Input{
		Facts:      Facts{TargetUsers: DeclarationDeclared},
		Selections: []Selection{preset(DimensionU1, "team")},
	})
	if err != nil {
		t.Fatal(err)
	}
	access, found := findDimensionQuestion(afterAudience.Questions(), DimensionT1)
	if !found {
		t.Fatal("team audience did not activate T1")
	}
	if !reflect.DeepEqual(
		access.DependsOn(),
		[]QuestionRef{questionRef(DimensionU1)},
	) {
		t.Fatalf("T1 dependencies = %#v", access.DependsOn())
	}
	state = applyProjectedQuestions(t, state, afterAudience, access.Ref())
	recommended, found := access.RecommendedOption()
	if !found {
		t.Fatal("T1 recommendation missing")
	}
	state = applyIntakeChoices(t, state, intake.Choice{
		QuestionRef: intake.QuestionRef(access.Ref()),
		OptionRef:   intake.OptionRef(recommended.Ref()),
	})
	state = applyIntakeChoices(t, state, intake.Choice{
		QuestionRef: intake.QuestionRef(questionRef(DimensionU1)),
		OptionRef:   intake.OptionRef(optionRef(DimensionU1, "personal")),
	})
	reopened := state.ReopenedDecisions()
	if len(reopened) != 1 ||
		reopened[0].QuestionRef != intake.QuestionRef(access.Ref()) {
		t.Fatalf("reopened = %#v, want T1", reopened)
	}
}

func TestResolvedParentsRemainOnRulesAndPackQuestions(t *testing.T) {
	t.Parallel()

	calendar := mustPackRef(t, "pack:calendar")
	result, err := Evaluate(Input{
		Facts: Facts{
			Surface:                SurfaceNativeMobile,
			IntegrationAuth:        DeclarationMissing,
			IntegrationCriticality: DeclarationMissing,
			TargetUsers:            DeclarationMissing,
		},
		Selections: []Selection{
			preset(DimensionU1, "team"),
			preset(DimensionU2, "native_mobile"),
			preset(DimensionU4, "persisted_user_data"),
			preset(DimensionU7, "external_services"),
			preset(DimensionU10, "cloud_container"),
			preset(DimensionU11, "realtime"),
		},
		PackRefs: []catalog.PackRef{calendar},
	})
	if err != nil {
		t.Fatal(err)
	}
	var packQuestion, governance, targetUsers Question
	for _, question := range result.Questions() {
		switch {
		case question.PackRef().String() == "pack:calendar":
			packQuestion = question
		case question.Ref() == QuestionRef("intake-question:wizard.r5"):
			governance = question
		case question.Ref() == QuestionRef("intake-question:wizard.r8"):
			targetUsers = question
		}
	}
	assertExactDependencies(
		t,
		"pack",
		packQuestion,
		questionRef(DimensionU7),
	)
	assertExactDependencies(
		t,
		"R5",
		governance,
		questionRef(DimensionU7),
	)
	assertExactDependencies(
		t,
		"R8",
		targetUsers,
		questionRef(DimensionU1),
	)

	deployment, found := findDimensionQuestion(result.Questions(), DimensionU10)
	if !found {
		t.Fatal("R7 did not re-question U10")
	}
	assertExactDependencies(
		t,
		"R7",
		deployment,
		questionRef(DimensionU2),
	)

	persistence, found := findDimensionQuestion(result.Questions(), DimensionT4)
	if !found {
		t.Fatal("R4 did not activate T4")
	}
	assertExactDependencies(
		t,
		"T4 dedupe",
		persistence,
		questionRef(DimensionU4),
	)

	resilience, found := findDimensionQuestion(result.Questions(), DimensionT7)
	if !found {
		t.Fatal("typed integrations/scale did not activate T7")
	}
	assertExactDependencies(
		t,
		"T7 order",
		resilience,
		questionRef(DimensionU7),
		questionRef(DimensionU11),
	)
}

func TestDynamicDependencyAccumulatorExcludesSelfReferences(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{
		Facts: Facts{
			Surface:       SurfaceNativeMobile,
			SharingIntent: SharingShared,
		},
		Selections: []Selection{
			preset(DimensionU1, "personal"),
			preset(DimensionU2, "native_desktop"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []DimensionRef{DimensionU1, DimensionU2} {
		question, found := findDimensionQuestion(result.Questions(), ref)
		if !found {
			t.Fatalf("missing contradicted %s", ref)
		}
		if len(question.DependsOn()) != 0 {
			t.Fatalf("%s retained self dependency: %#v", ref, question.DependsOn())
		}
	}
	for _, issue := range result.Issues() {
		if issue.Dimension() == "" {
			continue
		}
		for _, dependency := range issue.DependsOn() {
			if dependency == issue.Dimension() {
				t.Fatalf("%s retained self dependency", issue.Ref())
			}
		}
	}
}

func TestEvaluateConcurrentCallsDoNotShareMutableState(t *testing.T) {
	t.Parallel()

	const workers = 32
	var wait sync.WaitGroup
	errors := make(chan error, workers)
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := Evaluate(Input{Facts: Facts{Surface: SurfaceKernelModule}})
			if err == nil && len(result.Questions()) == 0 {
				err = domainError(ErrorInvalidArgument, "empty_concurrent_result")
			}
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestEvaluateRejectsOnlyStructurallyInvalidInput(t *testing.T) {
	t.Parallel()

	unknownPack, err := catalog.NewPackRef("pack:missing")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input Input
		code  ErrorCode
	}{
		{
			name: "unknown dimension",
			input: Input{Selections: []Selection{{
				Dimension: "U99", Option: "intake-option:wizard.u99.nope",
			}}},
			code: ErrorUnknownDimension,
		},
		{
			name: "unknown option",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1, Option: "intake-option:wizard.u1.nope",
			}}},
			code: ErrorUnknownOption,
		},
		{
			name: "unknown supplemental question",
			input: Input{QuestionSelections: []QuestionSelection{{
				Question: "intake-question:wizard.unknown",
				Option:   "intake-option:wizard.unknown.value",
			}}},
			code: ErrorUnknownQuestion,
		},
		{
			name: "free option without text",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1, Option: optionRef(DimensionU1, "custom"),
			}}},
			code: ErrorInvalidFreeText,
		},
		{
			name: "preset with free text",
			input: Input{Selections: []Selection{{
				Dimension: DimensionU1,
				Option:    optionRef(DimensionU1, "personal"),
				FreeText:  "hidden untyped override",
			}}},
			code: ErrorInvalidFreeText,
		},
		{
			name:  "invalid typed fact",
			input: Input{Facts: Facts{Surface: "web_by_keyword"}},
			code:  ErrorInvalidArgument,
		},
		{
			name:  "unknown explicit pack",
			input: Input{PackRefs: []catalog.PackRef{unknownPack}},
			code:  ErrorUnknownPack,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Evaluate(test.input)
			if ErrorCodeOf(err) != test.code {
				t.Fatalf("error = %v, code = %s, want %s", err, ErrorCodeOf(err), test.code)
			}
		})
	}
}

func TestAnsweredDimensionClosesWithoutSilentReplacement(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{
		Facts:      Facts{SharingIntent: SharingPersonal},
		Selections: []Selection{preset(DimensionU1, "personal")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findDimensionQuestion(result.Questions(), DimensionU1); found {
		t.Fatal("resolved U1 was reintroduced")
	}
	for _, item := range result.DefaultProposals() {
		if item.Dimension() == DimensionU1 {
			t.Fatal("product decision was converted to technical default")
		}
	}
}

func preset(dimension DimensionRef, value string) Selection {
	return Selection{Dimension: dimension, Option: optionRef(dimension, value)}
}

func mustPackRef(t *testing.T, value string) catalog.PackRef {
	t.Helper()
	ref, err := catalog.NewPackRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func findDimensionIssue(
	values []Issue,
	dimension DimensionRef,
	kind IssueKind,
) (Issue, bool) {
	for _, value := range values {
		if value.Dimension() == dimension && value.Kind() == kind {
			return value, true
		}
	}
	return Issue{}, false
}

func findDimensionQuestion(
	values []Question,
	dimension DimensionRef,
) (Question, bool) {
	for _, value := range values {
		if value.Dimension() == dimension {
			return value, true
		}
	}
	return Question{}, false
}

func containsIssueRef(values []IssueRef, want IssueRef) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func optionByKind(values []Option, want OptionKind) (Option, bool) {
	for _, value := range values {
		if value.Kind() == want {
			return value, true
		}
	}
	return Option{}, false
}

func hasDefaultProposal(values []DefaultProposal, want DimensionRef) bool {
	for _, value := range values {
		if value.Dimension() == want {
			return true
		}
	}
	return false
}

func questionRefs(values []Question) []QuestionRef {
	result := make([]QuestionRef, len(values))
	for index, value := range values {
		result[index] = value.Ref()
	}
	return result
}

func applyProjectedQuestions(
	t *testing.T,
	state intake.State,
	result Result,
	refs ...QuestionRef,
) intake.State {
	t.Helper()
	selected := make(map[QuestionRef]struct{}, len(refs))
	for _, ref := range refs {
		selected[ref] = struct{}{}
	}
	issueIndex := issueIndex(result.Issues())
	issues := make(map[IssueRef]Issue)
	questions := make([]intake.Question, 0, len(refs))
	for _, question := range result.Questions() {
		if _, found := selected[question.Ref()]; !found {
			continue
		}
		questions = append(questions, question.IntakeQuestion())
		for _, issueRef := range question.DerivedFrom() {
			issues[issueRef] = issueIndex[issueRef]
		}
	}
	projectedIssues := make([]intake.Issue, 0, len(issues))
	for _, issue := range result.Issues() {
		if _, found := issues[issue.Ref()]; found {
			projectedIssues = append(projectedIssues, issue.IntakeIssue())
		}
	}
	next, err := intake.Apply(state, intake.Change{
		StateRef:         state.Ref(),
		ExpectedRevision: state.Revision(),
		Origin:           intake.OriginForm,
		Issues:           projectedIssues,
		Questions:        questions,
	})
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func applyIntakeChoices(
	t *testing.T,
	state intake.State,
	choices ...intake.Choice,
) intake.State {
	t.Helper()
	next, err := intake.Apply(state, intake.Change{
		StateRef:         state.Ref(),
		ExpectedRevision: state.Revision(),
		Origin:           intake.OriginForm,
		Choices:          choices,
	})
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func assertExactDependencies(
	t *testing.T,
	name string,
	question Question,
	want ...QuestionRef,
) {
	t.Helper()
	if question.Ref() == "" {
		t.Fatalf("%s question missing", name)
	}
	if !reflect.DeepEqual(question.DependsOn(), want) {
		t.Fatalf("%s dependencies = %#v, want %#v", name, question.DependsOn(), want)
	}
}
