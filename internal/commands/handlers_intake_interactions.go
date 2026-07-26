package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

type acceptIntakeRecommendationsInput struct {
	IntakeRef        string          `json:"intake_ref"`
	ExpectedRevision intake.Revision `json:"expected_revision"`
	Origin           intake.Origin   `json:"origin"`
	QuestionRound    uint32          `json:"question_round"`
}

type getIntakeContextInput struct {
	IntakeRef        string               `json:"intake_ref"`
	ExpectedRevision intake.Revision      `json:"expected_revision"`
	Origin           intake.Origin        `json:"origin"`
	Kind             intake.ContextKind   `json:"kind"`
	QuestionRefs     []intake.QuestionRef `json:"question_refs"`
}

type intakeContextView struct {
	StateRef  intake.Ref                  `json:"state_ref"`
	Revision  intake.Revision             `json:"revision"`
	Origin    intake.Origin               `json:"origin"`
	Kind      intake.ContextKind          `json:"kind"`
	Issues    []intake.Issue              `json:"issues"`
	Questions []intakeQuestionContextView `json:"questions"`
}

type intakeQuestionContextView struct {
	Question         intakeContextQuestionView `json:"question"`
	CurrentDecision  *intake.Decision          `json:"current_decision,omitempty"`
	ReopenedDecision *intake.ReopenedDecision  `json:"reopened_decision,omitempty"`
}

type intakeContextQuestionView struct {
	Ref         intake.QuestionRef   `json:"ref"`
	DerivedFrom []intake.IssueRef    `json:"derived_from"`
	DependsOn   []intake.QuestionRef `json:"depends_on"`
	PromptKey   intake.MessageKey    `json:"prompt_key"`
	WhyKey      intake.MessageKey    `json:"why_key"`
	Options     []intake.Option      `json:"options"`
}

func handleAcceptIntakeRecommendations(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input acceptIntakeRecommendationsInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.AcceptIntakeRecommendations(
		ctx,
		bound.access,
		application.AcceptIntakeRecommendationsRequest{
			RequestRef:       bound.requestRef,
			ActorRef:         bound.principal.ActorRef,
			ProjectRef:       bound.projectRef,
			StateRef:         intake.Ref(input.IntakeRef),
			ExpectedRevision: input.ExpectedRevision,
			Origin:           input.Origin,
			QuestionRound:    input.QuestionRound,
		},
	)
	return marshalApplication(struct {
		Intake intakeMutationView `json:"intake"`
	}{Intake: projectIntakeMutation(result.Record)}, normalizeIntakeError(err))
}

func handleGetIntakeContext(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input getIntakeContextInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	contextView, err := api.GetIntakeContext(
		ctx,
		bound.access,
		application.GetIntakeContextRequest{
			ActorRef:   bound.principal.ActorRef,
			ProjectRef: bound.projectRef,
			Context: intake.ContextRequest{
				StateRef:         intake.Ref(input.IntakeRef),
				ExpectedRevision: input.ExpectedRevision,
				Origin:           input.Origin,
				Kind:             input.Kind,
				QuestionRefs:     nonNil(append([]intake.QuestionRef(nil), input.QuestionRefs...)),
			},
		},
	)
	return marshalApplication(struct {
		Context intakeContextView `json:"context"`
	}{Context: projectIntakeContext(contextView)}, normalizeIntakeError(err))
}

func projectIntakeContext(source intake.Context) intakeContextView {
	questions := make([]intakeQuestionContextView, 0, len(source.Questions))
	for _, item := range source.Questions {
		question := item.Question
		questions = append(questions, intakeQuestionContextView{
			Question: intakeContextQuestionView{
				Ref:         question.Ref,
				DerivedFrom: nonNil(append([]intake.IssueRef(nil), question.DerivedFrom...)),
				DependsOn:   nonNil(append([]intake.QuestionRef(nil), question.DependsOn...)),
				PromptKey:   question.PromptKey,
				WhyKey:      question.WhyKey,
				Options:     nonNil(append([]intake.Option(nil), question.Options...)),
			},
			CurrentDecision:  cloneIntakeDecision(item.CurrentDecision),
			ReopenedDecision: cloneReopenedDecision(item.ReopenedDecision),
		})
	}
	return intakeContextView{
		StateRef:  source.StateRef,
		Revision:  source.Revision,
		Origin:    source.Origin,
		Kind:      source.Kind,
		Issues:    nonNil(append([]intake.Issue(nil), source.Issues...)),
		Questions: nonNil(questions),
	}
}

func cloneIntakeDecision(source *intake.Decision) *intake.Decision {
	if source == nil {
		return nil
	}
	result := *source
	return &result
}

func cloneReopenedDecision(source *intake.ReopenedDecision) *intake.ReopenedDecision {
	if source == nil {
		return nil
	}
	result := *source
	result.InvalidatedBy = nonNil(append([]intake.DecisionChange(nil), source.InvalidatedBy...))
	return &result
}
