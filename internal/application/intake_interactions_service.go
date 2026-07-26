package application

import (
	"context"
	"encoding/json"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

// AcceptIntakeRecommendationsRequest identifies one exact question round.
// The service compiles it to the ordinary intake Apply operation and persists
// it through the same IntakeStore CAS and receipt path.
type AcceptIntakeRecommendationsRequest struct {
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	StateRef             intake.Ref
	ExpectedRevision     intake.Revision
	Origin               intake.Origin
	QuestionRound        uint32
	AuthorizationReceipt identity.AuthorizationReceipt
}

type GetIntakeContextRequest struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	Context    intake.ContextRequest
}

func (service *IntakeService) AcceptIntakeRecommendations(
	ctx context.Context,
	request AcceptIntakeRecommendationsRequest,
) (IntakeResult, error) {
	if err := validateIntakeRequestScope(
		request.RequestRef,
		request.ActorRef,
		request.ProjectRef,
	); err != nil {
		return IntakeResult{}, err
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationApply,
		request.RequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	if err := validateIntakeAuthorization(
		request.AuthorizationReceipt,
		request.ActorRef,
		request.ProjectRef,
		authorizationRequestRef,
	); err != nil {
		return IntakeResult{}, err
	}
	current, err := service.GetIntake(ctx, GetIntakeRequest{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef,
	})
	if err != nil {
		return IntakeResult{}, err
	}
	source, err := intakeStateAtRevision(
		current.State,
		request.ExpectedRevision,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	change, err := intake.BuildAcceptRecommendationsChange(
		source,
		intake.AcceptRecommendationsRequest{
			StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
			Origin: request.Origin, QuestionRound: request.QuestionRound,
		},
	)
	if err != nil {
		return IntakeResult{}, err
	}
	encoded, err := json.Marshal(change)
	if err != nil {
		return IntakeResult{}, errors.New("application.intake_change_invalid")
	}
	fingerprint := fingerprintFields(
		"orquesta.intake.apply.v1",
		request.ActorRef.String(),
		request.ProjectRef.String(),
		string(encoded),
		request.AuthorizationReceipt.Ref(),
	)
	replay := IntakeReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		Operation: IntakeOperationApply, ActorRef: request.ActorRef,
		ProjectRef: request.ProjectRef, StateRef: request.StateRef,
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}
	if record, found, replayErr := service.replay(
		ctx,
		replay,
		nil,
		request.ExpectedRevision,
	); replayErr != nil {
		return IntakeResult{}, replayErr
	} else if found {
		return IntakeResult{Record: record}, nil
	}
	next, err := intake.Apply(current.State, change)
	if err != nil {
		return IntakeResult{}, err
	}
	receipt, err := buildIntakeReceipt(
		IntakeOperationApply,
		request.RequestRef,
		fingerprint,
		request.ActorRef,
		request.ProjectRef,
		next,
		request.ExpectedRevision,
		request.AuthorizationReceipt.Ref(),
	)
	if err != nil {
		return IntakeResult{}, err
	}
	persisted, changed, err := service.store.ApplyIntake(ctx, IntakeApplyState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		AuthorizationReceipt: request.AuthorizationReceipt,
		ExpectedRevision:     request.ExpectedRevision,
		State:                next, Receipt: receipt,
	})
	if err != nil {
		return service.recoverIntakeConflict(
			ctx,
			replay,
			&next,
			request.ExpectedRevision,
			err,
		)
	}
	if err := validateIntakeMutationRecord(
		replay,
		next,
		request.ExpectedRevision,
		persisted,
	); err != nil {
		return IntakeResult{}, err
	}
	return IntakeResult{Record: persisted, Changed: changed}, nil
}

func intakeStateAtRevision(
	current intake.State,
	revision intake.Revision,
) (intake.State, error) {
	if revision == current.Revision() {
		return current, nil
	}
	if revision == 0 || revision > current.Revision() ||
		uint64(revision-1) > uint64(len(current.History())) {
		return intake.State{}, &intake.DomainError{
			Code:  intake.ErrorRevisionConflict,
			Field: "request.expected_revision",
		}
	}
	snapshot := SnapshotIntake(current)
	mutationCount := int(revision - 1)
	snapshot.History = append(
		[]intake.Mutation(nil),
		snapshot.History[:mutationCount]...,
	)
	issues, questions, questionVersions, questionTransitions, decisions := 0, 0, 0, 0, 0
	var rounds uint32
	for _, mutation := range snapshot.History {
		issues += mutation.IssuesAdded
		questions += mutation.QuestionsAdded
		questionVersions += mutation.QuestionsAdded +
			mutation.QuestionsRevised +
			mutation.QuestionsRetired
		questionTransitions += mutation.QuestionsRevised +
			mutation.QuestionsRetired
		decisions += mutation.ChoicesRecorded
		rounds = mutation.QuestionRound
	}
	if issues > len(snapshot.Issues) ||
		(len(snapshot.QuestionVersions) == 0 && questions > len(snapshot.Questions)) ||
		(len(snapshot.QuestionVersions) > 0 &&
			questionVersions > len(snapshot.QuestionVersions)) ||
		decisions > len(snapshot.Decisions) {
		return intake.State{}, &StateError{Code: StateConflict}
	}
	snapshot.Revision = revision
	snapshot.QuestionRounds = rounds
	snapshot.Issues = append([]intake.Issue(nil), snapshot.Issues[:issues]...)
	if len(snapshot.QuestionVersions) == 0 {
		snapshot.Questions = cloneSnapshotQuestions(snapshot.Questions[:questions])
	} else {
		snapshot.QuestionVersions = append(
			[]intake.QuestionVersion(nil),
			snapshot.QuestionVersions[:questionVersions]...,
		)
		snapshot.Questions = activeQuestionsFromVersions(snapshot.QuestionVersions)
		if questionTransitions == 0 {
			snapshot.QuestionVersions = nil
		}
	}
	snapshot.Decisions = append(
		[]intake.Decision(nil),
		snapshot.Decisions[:decisions]...,
	)
	return RestoreIntake(snapshot)
}

func activeQuestionsFromVersions(
	versions []intake.QuestionVersion,
) []intake.Question {
	order := make([]intake.QuestionRef, 0)
	active := make(map[intake.QuestionRef]intake.Question)
	for _, version := range versions {
		if version.Retired {
			delete(active, version.Question.Ref)
			for index, ref := range order {
				if ref == version.Question.Ref {
					order = append(order[:index], order[index+1:]...)
					break
				}
			}
			continue
		}
		if _, found := active[version.Question.Ref]; !found {
			order = append(order, version.Question.Ref)
		}
		active[version.Question.Ref] = cloneSnapshotQuestion(version.Question)
	}
	result := make([]intake.Question, 0, len(order))
	for _, ref := range order {
		result = append(result, active[ref])
	}
	return result
}

func (service *IntakeService) GetIntakeContext(
	ctx context.Context,
	request GetIntakeContextRequest,
) (intake.Context, error) {
	record, err := service.GetIntake(ctx, GetIntakeRequest{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.Context.StateRef,
	})
	if err != nil {
		return intake.Context{}, err
	}
	return intake.ReemitContext(record.State, request.Context)
}

func (orchestrator *Orchestrator) AcceptIntakeRecommendations(
	ctx context.Context,
	access Access,
	request AcceptIntakeRecommendationsRequest,
) (IntakeResult, error) {
	if orchestrator == nil || orchestrator.intake == nil {
		return IntakeResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationApply,
		request.RequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.intake.AcceptIntakeRecommendations(ctx, request)
}

func (orchestrator *Orchestrator) GetIntakeContext(
	ctx context.Context,
	access Access,
	request GetIntakeContextRequest,
) (intake.Context, error) {
	if orchestrator == nil || orchestrator.intake == nil {
		return intake.Context{}, errors.New("application.unavailable")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return intake.Context{}, err
	}
	if _, err = orchestrator.authorizeRead(
		ctx,
		access,
		identity.PermissionGoalsGet,
		string(request.Context.StateRef),
	); err != nil {
		return intake.Context{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	return orchestrator.intake.GetIntakeContext(ctx, request)
}
