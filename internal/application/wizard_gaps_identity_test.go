package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsRequestScopedPacksCanChangeEvaluationWithoutChangingMutation(
	t *testing.T,
) {
	evaluator := wizardGapsV1Evaluator(t)
	packs := catalog.DomainPackRefs()[:2]
	state, err := intake.NewState(
		"intake:wizard-pack-context",
		intake.Policy{MaxQuestionRounds: 6},
	)
	if err != nil {
		t.Fatal(err)
	}
	all, err := evaluateWizardGaps(state, gaps.Facts{}, packs, evaluator)
	if err != nil {
		t.Fatal(err)
	}
	issueRefs := make(map[intake.IssueRef]struct{})
	questions := make([]intake.Question, 0, 3)
	for _, question := range all.Questions() {
		if question.PackRef().String() == "" &&
			question.Dimension() != gaps.DimensionU7 {
			continue
		}
		projected := question.IntakeQuestion()
		questions = append(questions, projected)
		for _, ref := range projected.DerivedFrom {
			issueRefs[ref] = struct{}{}
		}
	}
	issues := make([]intake.Issue, 0, len(issueRefs))
	for _, issue := range all.Issues() {
		projected := issue.IntakeIssue()
		if _, selected := issueRefs[projected.Ref]; selected {
			issues = append(issues, projected)
		}
	}
	state, err = intake.Apply(state, intake.Change{
		StateRef: state.Ref(), ExpectedRevision: state.Revision(),
		Origin: intake.OriginForm, Issues: issues, Questions: questions,
		Derivation: evaluator.Identity(),
	})
	if err != nil {
		t.Fatal(err)
	}
	left, err := evaluateWizardGaps(
		state,
		gaps.Facts{},
		[]catalog.PackRef{packs[0]},
		evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := evaluateWizardGaps(
		state,
		gaps.Facts{},
		[]catalog.PackRef{packs[1]},
		evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	leftChange, err := wizardGapsChange(
		state,
		intake.OriginForm,
		evaluator.Identity(),
		left,
	)
	if err != nil {
		t.Fatal(err)
	}
	rightChange, err := wizardGapsChange(
		state,
		intake.OriginForm,
		evaluator.Identity(),
		right,
	)
	if err != nil {
		t.Fatal(err)
	}
	leftJSON, _ := json.Marshal(leftChange)
	rightJSON, _ := json.Marshal(rightChange)
	if len(leftChange.Questions) == 0 ||
		!bytes.Equal(leftJSON, rightJSON) ||
		left.PackRefs()[0] == right.PackRefs()[0] {
		t.Fatalf(
			"left_packs=%v right_packs=%v left_change=%s right_change=%s",
			left.PackRefs(),
			right.PackRefs(),
			leftJSON,
			rightJSON,
		)
	}
	// Same mutation means the regular Intake receipt can replay Record only.
	// Advisory Evaluation remains request-scoped and explicitly non-exact.
	if (ApplyWizardGapsResult{}).EvaluationReplayExact {
		t.Fatal("request-scoped evaluation advertised exact replay")
	}
}

func TestWizardGapsHistoricalEvaluatorReplaySurvivesRegistryEvolution(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-versioned-replay",
		1,
	)
	first, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	parent := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU1)
	parentChoice, found := parent.RecommendedOption()
	if !found {
		t.Fatal("U1 recommendation missing")
	}
	if _, err = system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-versioned-later",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 2,
				Origin: intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(parent.Ref()),
					OptionRef:   intake.OptionRef(parentChoice.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:wizard-gaps-versioned-later",
			),
		},
	); err != nil {
		t.Fatal(err)
	}

	v1 := wizardGapsV1Evaluator(t)
	v1Identity := v1.Identity()
	v2Identity, err := intake.NewDerivationIdentity(
		v1Identity.Schema,
		"v2",
		strings.Repeat("e", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	v2 := testWizardGapsEvaluator{identity: v2Identity, delegate: v1}
	evolved, err := newWizardGapsServiceWithEvaluatorResolver(
		system.service,
		testWizardGapsEvaluatorResolver{evaluators: []wizardGapsEvaluator{v1, v2}},
	)
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := evolved.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed || !replayed.RequestRefReserved ||
		replayed.EvaluationReplayExact ||
		replayed.EvaluatorIdentity != v1Identity ||
		replayed.Record.Receipt != first.Record.Receipt {
		t.Fatalf(
			"first=%+v replayed=%+v",
			first.Record.Receipt,
			replayed,
		)
	}

	request.EvaluatorIdentity = v2Identity
	if _, err = evolved.ApplyWizardGaps(
		context.Background(),
		request,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("same request with V2 identity err=%v", err)
	}
}

type testWizardGapsEvaluator struct {
	identity intake.DerivationIdentity
	delegate wizardGapsEvaluator
}

func (value testWizardGapsEvaluator) Identity() intake.DerivationIdentity {
	return value.identity
}

func (value testWizardGapsEvaluator) Dimensions() []gaps.DimensionDescriptor {
	return value.delegate.Dimensions()
}

func (value testWizardGapsEvaluator) Rules() []gaps.RuleDescriptor {
	return value.delegate.Rules()
}

func (value testWizardGapsEvaluator) Evaluate(input gaps.Input) (gaps.Result, error) {
	return value.delegate.Evaluate(input)
}

type testWizardGapsEvaluatorResolver struct {
	evaluators []wizardGapsEvaluator
}

func (value testWizardGapsEvaluatorResolver) Resolve(
	identity intake.DerivationIdentity,
) (wizardGapsEvaluator, error) {
	for _, evaluator := range value.evaluators {
		if evaluator.Identity() == identity {
			return evaluator, nil
		}
	}
	return nil, &gaps.DomainError{
		Code: gaps.ErrorUnsupportedEvaluator, Field: "evaluator_identity",
	}
}

func TestWizardGapsRejectsExistingRefWhoseCompletePayloadDiffers(t *testing.T) {
	t.Run("new facts change existing question", func(t *testing.T) {
		system, service := newWizardGapsTestSystem(t)
		first, err := service.ApplyWizardGaps(
			context.Background(),
			wizardGapsRequest(
				t,
				system,
				"request:wizard-gaps-collision-base",
				1,
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		before := SnapshotIntake(first.Record.State)
		applyCalls := system.store.applyCalls
		request := wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-collision-facts",
			first.Record.State.Revision(),
		)
		request.Facts.Surface = gaps.SurfaceServerService
		if _, err = service.ApplyWizardGaps(
			context.Background(),
			request,
		); !IsStateError(err, StateConflict) ||
			!errors.Is(err, ErrWizardGapsProjectionConflict) {
			t.Fatalf("collision err=%v", err)
		}
		current, getErr := system.service.GetIntake(
			context.Background(),
			GetIntakeRequest{
				ActorRef: system.actor, ProjectRef: system.project,
				StateRef: "intake:shared",
			},
		)
		if getErr != nil ||
			system.store.applyCalls != applyCalls ||
			!reflectIntakeSnapshotEqual(SnapshotIntake(current.State), before) {
			t.Fatalf(
				"get_err=%v calls=%d/%d current=%+v before=%+v",
				getErr,
				applyCalls,
				system.store.applyCalls,
				SnapshotIntake(current.State),
				before,
			)
		}
	})

	for _, test := range []struct {
		name   string
		change intake.Change
	}{
		{
			name: "foreign issue owns canonical ref",
			change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 1,
				Origin: intake.OriginForm,
				Issues: []intake.Issue{{
					Ref:  "intake-issue:wizard.dimension.u1",
					Kind: intake.IssueContradiction, Field: "foreign.field",
					DetailKey: "foreign.issue.detail",
				}},
				Questions: []intake.Question{
					foreignWizardCollisionQuestion(
						"intake-question:foreign-issue",
						"intake-issue:wizard.dimension.u1",
					),
				},
			},
		},
		{
			name: "foreign question owns canonical ref",
			change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 1,
				Origin: intake.OriginForm,
				Issues: []intake.Issue{{
					Ref:  "intake-issue:foreign-question",
					Kind: intake.IssueGap, Field: "foreign.field",
					DetailKey: "foreign.issue.detail",
				}},
				Questions: []intake.Question{
					foreignWizardCollisionQuestion(
						"intake-question:wizard.u1",
						"intake-issue:foreign-question",
					),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			system, service := newWizardGapsTestSystem(t)
			foreign, err := system.service.ApplyIntake(
				context.Background(),
				ApplyIntakeRequest{
					RequestRef: "request:wizard-gaps-foreign",
					ActorRef:   system.actor, ProjectRef: system.project,
					Change: test.change,
					AuthorizationReceipt: system.authorizationFor(
						t,
						IntakeOperationApply,
						"request:wizard-gaps-foreign",
					),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			applyCalls := system.store.applyCalls
			_, err = service.ApplyWizardGaps(
				context.Background(),
				wizardGapsRequest(
					t,
					system,
					"request:wizard-gaps-foreign-evaluate",
					foreign.Record.State.Revision(),
				),
			)
			if !IsStateError(err, StateConflict) ||
				!errors.Is(err, ErrWizardGapsProjectionConflict) ||
				system.store.applyCalls != applyCalls {
				t.Fatalf(
					"err=%v calls=%d/%d",
					err,
					applyCalls,
					system.store.applyCalls,
				)
			}
		})
	}
}

func TestWizardGapsNoOpDoesNotReserveRequestRef(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-seed", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	request := wizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-unreserved-noop",
		first.Record.State.Revision(),
	)
	noOp, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if noOp.Changed || noOp.RequestRefReserved {
		t.Fatalf("no-op result=%+v", noOp)
	}

	integration := mustWizardDimensionQuestion(
		t,
		first.Evaluation,
		gaps.DimensionU7,
	)
	choice, found := integration.RecommendedOption()
	if !found {
		t.Fatal("U7 recommendation missing")
	}
	answered, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-unreserved-answer",
			ActorRef:   system.actor, ProjectRef: system.project,
			Change: intake.Change{
				StateRef:         "intake:shared",
				ExpectedRevision: first.Record.State.Revision(),
				Origin:           intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(integration.Ref()),
					OptionRef:   intake.OptionRef(choice.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:wizard-gaps-unreserved-answer",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	request.ExpectedRevision = answered.Record.State.Revision()
	applied, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Changed || !applied.RequestRefReserved ||
		applied.Record.Receipt.RequestRef != request.RequestRef {
		t.Fatalf("later reuse result=%+v", applied)
	}
}

func TestWizardGapsRejectsCanonicalOptionOwnedByForeignAnsweredQuestion(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	poisoned, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-foreign-option",
			ActorRef:   system.actor, ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 1,
				Origin: intake.OriginForm,
				Issues: []intake.Issue{{
					Ref: "intake-issue:foreign-option", Kind: intake.IssueGap,
					Field: "foreign.field", DetailKey: "foreign.issue.detail",
				}},
				Questions: []intake.Question{{
					Ref:         "intake-question:foreign-option",
					DerivedFrom: []intake.IssueRef{"intake-issue:foreign-option"},
					PromptKey:   "foreign.question.prompt", WhyKey: "foreign.question.why",
					Options: []intake.Option{
						{
							Ref:          "intake-option:wizard.u1.personal",
							LabelKey:     "foreign.option.personal.label",
							RationaleKey: "foreign.option.personal.rationale",
							Recommended:  true,
						},
						{
							Ref:          "intake-option:foreign-option.alternative",
							LabelKey:     "foreign.option.alternative.label",
							RationaleKey: "foreign.option.alternative.rationale",
						},
					},
				}},
				Choices: []intake.Choice{{
					QuestionRef: "intake-question:foreign-option",
					OptionRef:   "intake-option:wizard.u1.personal",
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:wizard-gaps-foreign-option",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	_, err = service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-foreign-option-evaluate",
			poisoned.Record.State.Revision(),
		),
	)
	if !IsStateError(err, StateConflict) ||
		!errors.Is(err, ErrWizardGapsProjectionConflict) ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("err=%v calls=%d/%d", err, applyCalls, system.store.applyCalls)
	}
}

func TestWizardGapsRejectsActiveSupplementalOptionOwnedByForeignAnsweredQuestion(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-supplemental-option-base",
			1,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := wizardGapsV1Evaluator(t)
	evaluation, err := evaluator.Evaluate(gaps.Input{
		Selections: []gaps.Selection{{
			Dimension: gaps.DimensionU7,
			Option:    "intake-option:wizard.u7.external_services",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var supplemental gaps.Question
	for _, question := range evaluation.Questions() {
		if question.Ref() == "intake-question:wizard.r5" {
			supplemental = question
			break
		}
	}
	option, found := supplemental.RecommendedOption()
	if !found {
		t.Fatal("R5 recommendation missing")
	}
	poisoned, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-supplemental-option-poison",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: first.Record.State.Revision(),
				Origin: intake.OriginForm,
				Issues: []intake.Issue{{
					Ref:  "intake-issue:foreign-supplemental-option",
					Kind: intake.IssueGap, Field: "foreign.field",
					DetailKey: "foreign.issue.detail",
				}},
				Questions: []intake.Question{{
					Ref: "intake-question:foreign-supplemental-option",
					DerivedFrom: []intake.IssueRef{
						"intake-issue:foreign-supplemental-option",
					},
					PromptKey: "foreign.question.prompt",
					WhyKey:    "foreign.question.why",
					Options: []intake.Option{
						{
							Ref:          intake.OptionRef(option.Ref()),
							LabelKey:     "foreign.option.canonical.label",
							RationaleKey: "foreign.option.canonical.rationale",
							Recommended:  true,
						},
						{
							Ref:          "intake-option:foreign-supplemental.alternative",
							LabelKey:     "foreign.option.alternative.label",
							RationaleKey: "foreign.option.alternative.rationale",
						},
					},
				}},
				Choices: []intake.Choice{
					{
						QuestionRef: "intake-question:wizard.u7",
						OptionRef:   "intake-option:wizard.u7.external_services",
					},
					{
						QuestionRef: "intake-question:foreign-supplemental-option",
						OptionRef:   intake.OptionRef(option.Ref()),
					},
				},
			},
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:wizard-gaps-supplemental-option-poison",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	_, err = service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-supplemental-option-evaluate",
			poisoned.Record.State.Revision(),
		),
	)
	if !IsStateError(err, StateConflict) ||
		!errors.Is(err, ErrWizardGapsProjectionConflict) ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("err=%v calls=%d/%d", err, applyCalls, system.store.applyCalls)
	}
}

func TestWizardGapsRejectsAlteredCanonicalAnsweredQuestionPayload(
	t *testing.T,
) {
	for _, test := range []struct {
		name   string
		mutate func(*intake.Question)
	}{
		{
			name: "prompt",
			mutate: func(question *intake.Question) {
				question.PromptKey = "foreign.question.prompt"
			},
		},
		{
			name: "recommendation",
			mutate: func(question *intake.Question) {
				recommendedIndex := -1
				for index := range question.Options {
					if question.Options[index].Recommended {
						recommendedIndex = index
					}
					question.Options[index].Recommended = false
				}
				alternative := (recommendedIndex + 1) % len(question.Options)
				if recommendedIndex < 0 || alternative == recommendedIndex {
					panic("test requires distinct recommendation")
				}
				question.Options[alternative].Recommended = true
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			system, service := newWizardGapsTestSystem(t)
			evaluator := wizardGapsV1Evaluator(t)
			evaluation, err := evaluator.Evaluate(gaps.Input{})
			if err != nil {
				t.Fatal(err)
			}
			source := mustWizardDimensionQuestion(t, evaluation, gaps.DimensionU1)
			question := source.IntakeQuestion()
			test.mutate(&question)
			recommended, found := source.RecommendedOption()
			if !found {
				t.Fatal("U1 recommendation missing")
			}
			var issue intake.Issue
			for _, candidate := range evaluation.Issues() {
				if candidate.Ref() == source.DerivedFrom()[0] {
					issue = candidate.IntakeIssue()
					break
				}
			}
			if issue.Ref == "" {
				t.Fatal("U1 issue missing")
			}
			requestRef := "request:wizard-gaps-forged-payload-" + test.name
			poisoned, err := system.service.ApplyIntake(
				context.Background(),
				ApplyIntakeRequest{
					RequestRef: requestRef,
					ActorRef:   system.actor, ProjectRef: system.project,
					Change: intake.Change{
						StateRef: "intake:shared", ExpectedRevision: 1,
						Origin: intake.OriginForm, Derivation: evaluator.Identity(),
						Issues:    []intake.Issue{issue},
						Questions: []intake.Question{question},
						Choices: []intake.Choice{{
							QuestionRef: question.Ref,
							OptionRef:   intake.OptionRef(recommended.Ref()),
						}},
					},
					AuthorizationReceipt: system.authorizationFor(
						t,
						IntakeOperationApply,
						requestRef,
					),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			applyCalls := system.store.applyCalls
			_, err = service.ApplyWizardGaps(
				context.Background(),
				wizardGapsRequest(
					t,
					system,
					requestRef+"-evaluate",
					poisoned.Record.State.Revision(),
				),
			)
			if !IsStateError(err, StateConflict) ||
				!errors.Is(err, ErrWizardGapsProjectionConflict) ||
				system.store.applyCalls != applyCalls {
				t.Fatalf("err=%v calls=%d/%d", err, applyCalls, system.store.applyCalls)
			}
		})
	}
}

func TestWizardGapsRejectsForeignSupplementalIssueProvenance(t *testing.T) {
	calendar, err := catalog.NewPackRef("pack:calendar")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		rule         gaps.RuleRef
		answer       *intake.Choice
		evaluate     gaps.Input
		requestPacks []catalog.PackRef
	}{
		{
			name: "R3 suffixed pack issue", rule: gaps.RuleR3,
			evaluate:     gaps.Input{PackRefs: []catalog.PackRef{calendar}},
			requestPacks: []catalog.PackRef{calendar},
		},
		{
			name: "R5 governance issue", rule: gaps.RuleR5,
			answer: &intake.Choice{
				QuestionRef: "intake-question:wizard.u7",
				OptionRef:   "intake-option:wizard.u7.external_services",
			},
			evaluate: gaps.Input{Selections: []gaps.Selection{{
				Dimension: gaps.DimensionU7,
				Option:    "intake-option:wizard.u7.external_services",
			}}},
		},
		{
			name: "R8 target users issue", rule: gaps.RuleR8,
			answer: &intake.Choice{
				QuestionRef: "intake-question:wizard.u1",
				OptionRef:   "intake-option:wizard.u1.team",
			},
			evaluate: gaps.Input{Selections: []gaps.Selection{{
				Dimension: gaps.DimensionU1,
				Option:    "intake-option:wizard.u1.team",
			}}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			evaluator := wizardGapsV1Evaluator(t)
			state, err := intake.NewState(
				"intake:supplemental-provenance",
				intake.Policy{MaxQuestionRounds: 6},
			)
			if err != nil {
				t.Fatal(err)
			}
			initial, err := evaluator.Evaluate(gaps.Input{})
			if err != nil {
				t.Fatal(err)
			}
			change, err := wizardGapsChange(
				state,
				intake.OriginForm,
				evaluator.Identity(),
				initial,
			)
			if err != nil {
				t.Fatal(err)
			}
			state, err = intake.Apply(state, change)
			if err != nil {
				t.Fatal(err)
			}
			if test.answer != nil {
				state, err = intake.Apply(state, intake.Change{
					StateRef: state.Ref(), ExpectedRevision: state.Revision(),
					Origin:  intake.OriginForm,
					Choices: []intake.Choice{*test.answer},
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			expected, err := evaluator.Evaluate(test.evaluate)
			if err != nil {
				t.Fatal(err)
			}
			var issue intake.Issue
			for _, candidate := range expected.Issues() {
				if candidate.RuleRef() == test.rule {
					issue = candidate.IntakeIssue()
					break
				}
			}
			if issue.Ref == "" {
				t.Fatalf("rule %s issue missing", test.rule)
			}
			state, err = intake.Apply(state, intake.Change{
				StateRef: state.Ref(), ExpectedRevision: state.Revision(),
				Origin: intake.OriginChat, Issues: []intake.Issue{issue},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = evaluateWizardGaps(
				state,
				gaps.Facts{},
				test.requestPacks,
				evaluator,
			); !IsStateError(err, StateConflict) ||
				!errors.Is(err, ErrWizardGapsProjectionConflict) {
				t.Fatalf("rule %s foreign issue err=%v", test.rule, err)
			}
		})
	}
}

func foreignWizardCollisionQuestion(
	ref intake.QuestionRef,
	issueRef intake.IssueRef,
) intake.Question {
	return intake.Question{
		Ref: ref, DerivedFrom: []intake.IssueRef{issueRef},
		PromptKey: "foreign.question.prompt",
		WhyKey:    "foreign.question.why",
		Options: []intake.Option{
			{
				Ref:          "intake-option:foreign-a",
				LabelKey:     "foreign.option.a.label",
				RationaleKey: "foreign.option.a.rationale",
				Recommended:  true,
			},
			{
				Ref:          "intake-option:foreign-b",
				LabelKey:     "foreign.option.b.label",
				RationaleKey: "foreign.option.b.rationale",
			},
		},
	}
}

func reflectIntakeSnapshotEqual(left, right IntakeSnapshot) bool {
	leftDigest, leftErr := intakeSnapshotDigest(left)
	rightDigest, rightErr := intakeSnapshotDigest(right)
	return leftErr == nil && rightErr == nil && leftDigest == rightDigest
}

func intakeSnapshotDigest(value IntakeSnapshot) (string, error) {
	state, err := RestoreIntake(value)
	if err != nil {
		return "", err
	}
	return IntakeStateDigest(state)
}
