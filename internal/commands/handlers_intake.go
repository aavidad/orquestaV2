package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

type createIntakeInput struct {
	IntakeRef         string  `json:"intake_ref"`
	MaxQuestionRounds *uint32 `json:"max_question_rounds"`
}

type applyIntakeInput struct {
	IntakeRef        string            `json:"intake_ref"`
	ExpectedRevision intake.Revision   `json:"expected_revision"`
	Origin           intake.Origin     `json:"origin"`
	Issues           []intake.Issue    `json:"issues"`
	Questions        []intake.Question `json:"questions"`
	Choices          []intake.Choice   `json:"choices"`
}

type intakeMutationView struct {
	IntakeRef  string `json:"intake_ref"`
	ProjectRef string `json:"project_ref"`
	Revision   uint64 `json:"revision"`
	ReceiptRef string `json:"receipt_ref"`
}

type intakeStateView struct {
	StateSchema       string            `json:"state_schema"`
	IntakeRef         string            `json:"intake_ref"`
	ActorRef          string            `json:"actor_ref"`
	ProjectRef        string            `json:"project_ref"`
	Revision          uint64            `json:"revision"`
	MaxQuestionRounds uint32            `json:"max_question_rounds"`
	QuestionRounds    uint32            `json:"question_rounds"`
	Issues            []intake.Issue    `json:"issues"`
	Questions         []intake.Question `json:"questions"`
	Decisions         []intake.Decision `json:"decisions"`
	History           []intake.Mutation `json:"history"`
}

func handleCreateIntake(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
	defaultPolicy intake.Policy,
) (json.RawMessage, error) {
	var input createIntakeInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	policy := defaultPolicy
	if input.MaxQuestionRounds != nil {
		policy.MaxQuestionRounds = *input.MaxQuestionRounds
	}
	result, err := api.CreateIntake(ctx, bound.access, application.CreateIntakeRequest{
		RequestRef: bound.requestRef,
		ActorRef:   bound.principal.ActorRef,
		ProjectRef: bound.projectRef,
		StateRef:   intake.Ref(input.IntakeRef),
		Policy:     policy,
	})
	return marshalApplication(struct {
		Intake intakeMutationView `json:"intake"`
	}{Intake: projectIntakeMutation(result.Record)}, normalizeIntakeError(err))
}

func handleGetIntake(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		IntakeRef string `json:"intake_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	record, err := api.GetIntake(ctx, bound.access, application.GetIntakeRequest{
		ActorRef: bound.principal.ActorRef, ProjectRef: bound.projectRef,
		StateRef: intake.Ref(input.IntakeRef),
	})
	return marshalApplication(struct {
		Intake intakeStateView `json:"intake"`
	}{Intake: projectIntakeState(record)}, normalizeIntakeError(err))
}

func handleApplyIntake(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input applyIntakeInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.ApplyIntake(ctx, bound.access, application.ApplyIntakeRequest{
		RequestRef: bound.requestRef,
		ActorRef:   bound.principal.ActorRef,
		ProjectRef: bound.projectRef,
		Change: intake.Change{
			StateRef: intake.Ref(input.IntakeRef), ExpectedRevision: input.ExpectedRevision,
			Origin: input.Origin, Issues: input.Issues, Questions: input.Questions, Choices: input.Choices,
		},
	})
	return marshalApplication(struct {
		Intake intakeMutationView `json:"intake"`
	}{Intake: projectIntakeMutation(result.Record)}, normalizeIntakeError(err))
}

func projectIntakeMutation(record application.IntakeRecord) intakeMutationView {
	return intakeMutationView{
		IntakeRef:  string(record.State.Ref()),
		ProjectRef: record.ProjectRef.String(),
		Revision:   uint64(record.State.Revision()),
		ReceiptRef: record.Receipt.Ref,
	}
}

func projectIntakeState(record application.IntakeRecord) intakeStateView {
	state := record.State
	return intakeStateView{
		StateSchema: state.Schema(), IntakeRef: string(state.Ref()),
		ActorRef: record.ActorRef.String(), ProjectRef: record.ProjectRef.String(),
		Revision: uint64(state.Revision()), MaxQuestionRounds: state.Policy().MaxQuestionRounds,
		QuestionRounds: state.QuestionRounds(), Issues: nonNil(state.Issues()),
		Questions: nonNil(state.Questions()), Decisions: nonNil(state.Decisions()),
		History: nonNil(state.History()),
	}
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func normalizeIntakeError(err error) error {
	switch intake.ErrorCodeOf(err) {
	case intake.ErrorRevisionConflict, intake.ErrorStateMismatch:
		return commandError{code: CodeConflict, cause: err}
	case intake.ErrorInvalidArgument, intake.ErrorInvalidRef, intake.ErrorInvalidOrigin,
		intake.ErrorChannelStateCreation, intake.ErrorDuplicateRef, intake.ErrorIssueNotFound,
		intake.ErrorQuestionNotFound, intake.ErrorOptionNotFound, intake.ErrorRecommendationCount,
		intake.ErrorMessageKeyInvalid, intake.ErrorRoundLimit, intake.ErrorChoiceConflict,
		intake.ErrorQuestionRound, intake.ErrorRecommendationsDone,
		intake.ErrorDependencyCycle, intake.ErrorDependencyPending:
		return commandError{code: CodeInvalidRequest, cause: err}
	default:
		return err
	}
}
