package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeApplyPublicContractCarriesTypedFreeTextAndEnforcesRuneLimit(t *testing.T) {
	api := &captureIntakeApplication{fakeApplication: newFakeApplication()}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes:         1 << 20,
			MaxListLimit:            100,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := publicFreeTextApplyPayload(strings.Repeat("á", intake.MaxAnswerTextRunes))
	result := invoke(
		t, dispatcher, "orquesta.intakes.apply",
		"request:intake-free-text", payload, false,
	)
	if result.Failure != nil {
		t.Fatalf("free-text apply=%+v", result)
	}
	if len(api.applies) != 1 ||
		len(api.applies[0].Change.Questions) != 1 ||
		len(api.applies[0].Change.Questions[0].Options) != 2 ||
		!api.applies[0].Change.Questions[0].Options[1].AcceptsText ||
		len(api.applies[0].Change.Choices) != 1 ||
		api.applies[0].Change.Choices[0].AnswerText !=
			strings.Repeat("á", intake.MaxAnswerTextRunes) {
		t.Fatalf("application request=%+v", api.applies)
	}

	tooLong := publicFreeTextApplyPayload(
		strings.Repeat("á", intake.MaxAnswerTextRunes) + "界",
	)
	rejected := invoke(
		t, dispatcher, "orquesta.intakes.apply",
		"request:intake-free-text-too-long", tooLong, false,
	)
	if rejected.Failure == nil || rejected.Failure.Code != CodeInvalidRequest ||
		len(api.applies) != 1 {
		t.Fatalf("too-long=%+v application calls=%d", rejected, len(api.applies))
	}
}

func TestIntakePublicReadSchemasPreserveFreeTextCapabilityAndAnswer(t *testing.T) {
	state, err := intake.NewState(
		"intake:public-free-text",
		intake.Policy{MaxQuestionRounds: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	state, err = intake.Apply(state, intake.Change{
		StateRef: "intake:public-free-text", ExpectedRevision: 1,
		Origin: intake.OriginChat,
		Issues: []intake.Issue{{
			Ref: "intake-issue:scope", Kind: intake.IssueGap,
			Field: "scope", DetailKey: "intake.issue.scope.missing",
		}},
		Questions: []intake.Question{publicFreeTextQuestion()},
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:scope",
			OptionRef:   "intake-option:scope-other",
			AnswerText:  "Colectivo ágil",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	getOutput, err := json.Marshal(struct {
		Intake intakeStateView `json:"intake"`
	}{Intake: projectIntakeState(application.IntakeRecord{State: state})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = validatePayload(
		intakeDefinition(t, "orquesta.intakes.get").OutputSchema,
		getOutput,
	); err != nil {
		t.Fatalf("get output rejected: %v\n%s", err, getOutput)
	}
	if !strings.Contains(string(getOutput), `"accepts_text":true`) ||
		!strings.Contains(string(getOutput), `"answer_text":"Colectivo ágil"`) {
		t.Fatalf("get output lost free text: %s", getOutput)
	}

	context, err := intake.ReemitContext(state, intake.ContextRequest{
		StateRef: "intake:public-free-text", ExpectedRevision: 2,
		Origin: intake.OriginForm, Kind: intake.ContextHelp,
	})
	if err != nil {
		t.Fatal(err)
	}
	contextOutput, err := json.Marshal(struct {
		Context intakeContextView `json:"context"`
	}{Context: projectIntakeContext(context)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = validatePayload(
		intakeDefinition(t, "orquesta.intakes.context.get").OutputSchema,
		contextOutput,
	); err != nil {
		t.Fatalf("context output rejected: %v\n%s", err, contextOutput)
	}
	if !strings.Contains(string(contextOutput), `"accepts_text":true`) ||
		!strings.Contains(string(contextOutput), `"answer_text":"Colectivo ágil"`) {
		t.Fatalf("context output lost free text: %s", contextOutput)
	}
}

func publicFreeTextApplyPayload(answer string) map[string]any {
	return map[string]any{
		"intake_ref": "intake:test", "expected_revision": 1, "origin": "chat",
		"issues": []any{map[string]any{
			"ref": "intake-issue:scope", "kind": "gap",
			"field": "scope", "detail_key": "intake.issue.scope.missing",
		}},
		"questions": []any{map[string]any{
			"ref":          "intake-question:scope",
			"derived_from": []any{"intake-issue:scope"},
			"depends_on":   []any{},
			"prompt_key":   "intake.question.scope.prompt",
			"why_key":      "intake.question.scope.why",
			"options": []any{
				map[string]any{
					"ref":           "intake-option:scope-web",
					"label_key":     "intake.option.scope.web.label",
					"rationale_key": "intake.option.scope.web.rationale",
					"recommended":   true,
				},
				map[string]any{
					"ref":           "intake-option:scope-other",
					"label_key":     "intake.option.scope.other.label",
					"rationale_key": "intake.option.scope.other.rationale",
					"recommended":   false,
					"accepts_text":  true,
				},
			},
		}},
		"choices": []any{map[string]any{
			"question_ref": "intake-question:scope",
			"option_ref":   "intake-option:scope-other",
			"answer_text":  answer,
		}},
	}
}

func publicFreeTextQuestion() intake.Question {
	return intake.Question{
		Ref:         "intake-question:scope",
		DerivedFrom: []intake.IssueRef{"intake-issue:scope"},
		PromptKey:   "intake.question.scope.prompt",
		WhyKey:      "intake.question.scope.why",
		Options: []intake.Option{
			{
				Ref:          "intake-option:scope-web",
				LabelKey:     "intake.option.scope.web.label",
				RationaleKey: "intake.option.scope.web.rationale",
				Recommended:  true,
			},
			{
				Ref:          "intake-option:scope-other",
				LabelKey:     "intake.option.scope.other.label",
				RationaleKey: "intake.option.scope.other.rationale",
				AcceptsText:  true,
			},
		},
	}
}

func intakeDefinition(t *testing.T, id string) Definition {
	t.Helper()
	for _, definition := range CanonicalDefinitions() {
		if definition.ID == id {
			return definition
		}
	}
	t.Fatalf("definition %s missing", id)
	return Definition{}
}
