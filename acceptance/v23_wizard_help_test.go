package acceptance_test

import (
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/i18n"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
)

func TestV23WizardHelpSurfaceIsCompleteAndReadOnly(t *testing.T) {
	const stateRef intake.Ref = "intake:v23-help"
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
			{Ref: "intake-issue:delivery-gap", Kind: intake.IssueGap,
				Field: "delivery", DetailKey: "intake.issue.delivery.missing"},
		},
		Questions: []intake.Question{audience, delivery},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := state

	contextual, err := intake.ReemitContext(state, intake.ContextRequest{
		StateRef: stateRef, ExpectedRevision: state.Revision(),
		Origin: intake.OriginForm, Kind: intake.ContextHelp,
		QuestionRefs: []intake.QuestionRef{delivery.Ref},
	})
	if err != nil {
		t.Fatal(err)
	}
	explainAllRequest := intake.ContextRequest{
		StateRef: stateRef, ExpectedRevision: state.Revision(),
		Origin: intake.OriginChat, Kind: intake.ContextHelp,
	}
	explainAll, err := intake.ReemitContext(state, explainAllRequest)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := intake.ReemitContext(state, explainAllRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(contextual.Questions) != 1 || contextual.Questions[0].Question.Ref != delivery.Ref ||
		len(contextual.Issues) != 1 || contextual.Issues[0].Ref != "intake-issue:delivery-gap" {
		t.Fatalf("contextual help leaked unrelated facts: %+v", contextual)
	}
	if len(explainAll.Questions) != 2 || explainAll.Questions[0].Question.Ref != audience.Ref ||
		explainAll.Questions[1].Question.Ref != delivery.Ref ||
		!reflect.DeepEqual(explainAll, replayed) {
		t.Fatalf("explain-all is incomplete or unstable: %+v / %+v", explainAll, replayed)
	}
	contextual.Questions[0].Question.Options[0].Ref = "intake-option:mutated"
	again, err := intake.ReemitContext(state, explainAllRequest)
	if err != nil || again.Questions[1].Question.Options[0].Ref != delivery.Options[0].Ref ||
		!reflect.DeepEqual(state, before) || state.Revision() != 2 ||
		state.QuestionRounds() != 1 || len(state.History()) != 1 {
		t.Fatalf("help mutated causal state or retained caller memory: context=%+v err=%v", again, err)
	}

	translations, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	keys, explainAllOption := v23WizardHelpExampleKeys()
	if !explainAllOption || len(keys) == 0 {
		t.Fatal("built-in explain-all help/example contract is absent")
	}
	for key := range keys {
		for _, locale := range []string{"es", "en"} {
			text, textErr := translations.Text(locale, key)
			if textErr != nil || strings.TrimSpace(text) == "" {
				t.Fatalf("missing localized help/example locale=%s key=%s err=%v", locale, key, textErr)
			}
		}
	}
}

func v23WizardHelpExampleKeys() (map[string]struct{}, bool) {
	keys := make(map[string]struct{})
	add := func(help, example catalog.MessageKey) {
		keys[help.String()] = struct{}{}
		keys[example.String()] = struct{}{}
	}
	explainAll := false
	for _, question := range catalog.BuiltIn().ComposeAll().Questions() {
		add(question.HelpKey(), question.ExampleKey())
		for _, option := range question.Options() {
			add(option.HelpKey(), option.ExampleKey())
			explainAll = explainAll || option.Ref().String() == "option:assistance_level.explain_all"
		}
	}
	return keys, explainAll
}
