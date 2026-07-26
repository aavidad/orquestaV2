package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type IntakeService struct {
	store IntakeStore
}

func NewIntakeService(store IntakeStore) (*IntakeService, error) {
	if store == nil {
		return nil, errors.New("application.intake_store_required")
	}
	return &IntakeService{store: store}, nil
}

func (service *IntakeService) CreateIntake(
	ctx context.Context,
	request CreateIntakeRequest,
) (IntakeResult, error) {
	if err := validateIntakeRequestScope(request.RequestRef, request.ActorRef, request.ProjectRef); err != nil {
		return IntakeResult{}, err
	}
	if err := validateIntakeAuthorization(
		request.AuthorizationReceipt, request.ActorRef, request.ProjectRef,
	); err != nil {
		return IntakeResult{}, err
	}
	state, err := intake.NewState(request.StateRef, request.Policy)
	if err != nil {
		return IntakeResult{}, err
	}
	fingerprint := fingerprintFields(
		"orquesta.intake.create.v1",
		request.ActorRef.String(),
		request.ProjectRef.String(),
		string(request.StateRef),
		strconv.FormatUint(uint64(request.Policy.MaxQuestionRounds), 10),
		request.AuthorizationReceipt.Ref(),
	)
	receipt, err := buildIntakeReceipt(
		IntakeOperationCreate, request.RequestRef, fingerprint,
		request.ActorRef, request.ProjectRef, state, 0,
		request.AuthorizationReceipt.Ref(),
	)
	if err != nil {
		return IntakeResult{}, err
	}
	replay := IntakeReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		Operation: IntakeOperationCreate, ActorRef: request.ActorRef,
		ProjectRef: request.ProjectRef, StateRef: request.StateRef,
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}
	if record, found, replayErr := service.replay(ctx, replay, &state, 0); replayErr != nil {
		return IntakeResult{}, replayErr
	} else if found {
		return IntakeResult{Record: record}, nil
	}
	persisted, changed, err := service.store.CreateIntake(ctx, IntakeCreateState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		AuthorizationReceipt: request.AuthorizationReceipt,
		State:                state, Receipt: receipt,
	})
	if err != nil {
		return service.recoverIntakeConflict(ctx, replay, &state, 0, err)
	}
	if err := validateIntakeMutationRecord(replay, state, 0, persisted); err != nil {
		return IntakeResult{}, err
	}
	return IntakeResult{Record: persisted, Changed: changed}, nil
}

func (service *IntakeService) GetIntake(
	ctx context.Context,
	request GetIntakeRequest,
) (IntakeRecord, error) {
	if service == nil || service.store == nil {
		return IntakeRecord{}, errors.New("application.unavailable")
	}
	if err := validateIntakeScope(request.ActorRef, request.ProjectRef); err != nil {
		return IntakeRecord{}, err
	}
	if !validIntakeStateRef(request.StateRef) {
		return IntakeRecord{}, &intake.DomainError{
			Code: intake.ErrorInvalidRef, Field: "state_ref",
		}
	}
	record, err := service.store.GetIntake(
		ctx, request.ActorRef, request.ProjectRef, request.StateRef,
	)
	if err != nil {
		return IntakeRecord{}, err
	}
	if err := validateStoredIntakeRecord(
		request.ActorRef, request.ProjectRef, request.StateRef, record,
	); err != nil {
		return IntakeRecord{}, err
	}
	return record, nil
}

func (service *IntakeService) ApplyIntake(
	ctx context.Context,
	request ApplyIntakeRequest,
) (IntakeResult, error) {
	if err := validateIntakeRequestScope(request.RequestRef, request.ActorRef, request.ProjectRef); err != nil {
		return IntakeResult{}, err
	}
	if err := validateIntakeAuthorization(
		request.AuthorizationReceipt, request.ActorRef, request.ProjectRef,
	); err != nil {
		return IntakeResult{}, err
	}
	encoded, err := json.Marshal(request.Change)
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
		ProjectRef: request.ProjectRef, StateRef: request.Change.StateRef,
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}
	if record, found, replayErr := service.replay(
		ctx, replay, nil, request.Change.ExpectedRevision,
	); replayErr != nil {
		return IntakeResult{}, replayErr
	} else if found {
		return IntakeResult{Record: record}, nil
	}

	current, err := service.store.GetIntake(
		ctx, request.ActorRef, request.ProjectRef, request.Change.StateRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	if err := validateStoredIntakeRecord(
		request.ActorRef, request.ProjectRef, request.Change.StateRef, current,
	); err != nil {
		return IntakeResult{}, err
	}
	next, err := intake.Apply(current.State, request.Change)
	if err != nil {
		return IntakeResult{}, err
	}
	receipt, err := buildIntakeReceipt(
		IntakeOperationApply, request.RequestRef, fingerprint,
		request.ActorRef, request.ProjectRef, next, request.Change.ExpectedRevision,
		request.AuthorizationReceipt.Ref(),
	)
	if err != nil {
		return IntakeResult{}, err
	}
	persisted, changed, err := service.store.ApplyIntake(ctx, IntakeApplyState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		AuthorizationReceipt: request.AuthorizationReceipt,
		ExpectedRevision:     request.Change.ExpectedRevision,
		State:                next, Receipt: receipt,
	})
	if err != nil {
		return service.recoverIntakeConflict(
			ctx, replay, &next, request.Change.ExpectedRevision, err,
		)
	}
	if err := validateIntakeMutationRecord(
		replay, next, request.Change.ExpectedRevision, persisted,
	); err != nil {
		return IntakeResult{}, err
	}
	return IntakeResult{Record: persisted, Changed: changed}, nil
}

func (service *IntakeService) replay(
	ctx context.Context,
	request IntakeReplayRequest,
	expected *intake.State,
	previousRevision intake.Revision,
) (IntakeRecord, bool, error) {
	if service == nil || service.store == nil {
		return IntakeRecord{}, false, errors.New("application.unavailable")
	}
	record, found, err := service.store.ReplayIntake(ctx, request)
	if err != nil || !found {
		return IntakeRecord{}, false, err
	}
	if expected != nil {
		err = validateIntakeMutationRecord(request, *expected, previousRevision, record)
	} else {
		err = validateIntakeReplayRecord(request, previousRevision, record)
	}
	return record, err == nil, err
}

func (service *IntakeService) recoverIntakeConflict(
	ctx context.Context,
	replay IntakeReplayRequest,
	expected *intake.State,
	previousRevision intake.Revision,
	persistErr error,
) (IntakeResult, error) {
	if !IsStateError(persistErr, StateConflict) {
		return IntakeResult{}, persistErr
	}
	record, found, replayErr := service.replay(ctx, replay, expected, previousRevision)
	if replayErr == nil && found {
		return IntakeResult{Record: record}, nil
	}
	return IntakeResult{}, persistErr
}

func buildIntakeReceipt(
	operation IntakeOperation,
	requestRef string,
	fingerprint string,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	state intake.State,
	previousRevision intake.Revision,
	authorizationReceiptRef string,
) (IntakeReceipt, error) {
	digest, err := IntakeStateDigest(state)
	if err != nil {
		return IntakeReceipt{}, err
	}
	revision := state.Revision()
	refDigest := fingerprintFields(
		"orquesta.intake.receipt.v1",
		string(operation), actorRef.String(), projectRef.String(), requestRef,
		fingerprint, string(state.Ref()),
		strconv.FormatUint(uint64(previousRevision), 10),
		strconv.FormatUint(uint64(revision), 10), digest,
		authorizationReceiptRef,
	)
	return IntakeReceipt{
		Ref:        "intake-receipt:" + refDigest,
		RequestRef: requestRef, RequestFingerprint: fingerprint,
		Operation: operation, ActorRef: actorRef, ProjectRef: projectRef,
		StateRef: state.Ref(), PreviousRevision: previousRevision,
		Revision: revision, StateDigest: digest,
		AuthorizationReceiptRef: authorizationReceiptRef,
	}, nil
}

func IntakeStateDigest(state intake.State) (string, error) {
	encoded, err := json.Marshal(SnapshotIntake(state))
	if err != nil {
		return "", errors.New("application.intake_state_invalid")
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateIntakeRequestScope(
	requestRef string,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
) error {
	if !validApplicationRef(requestRef) {
		return errors.New("application.request_ref_invalid")
	}
	return validateIntakeScope(actorRef, projectRef)
}

func validateIntakeScope(actorRef goal.ActorRef, projectRef goal.ProjectRef) error {
	switch {
	case actorRef.String() == "":
		return errors.New("application.actor_ref_required")
	case projectRef.String() == "":
		return errors.New("application.project_ref_required")
	default:
		return nil
	}
}

func validateIntakeAuthorization(
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
) error {
	decision := receipt.Decision()
	request := decision.Request()
	principal := request.Principal()
	if receipt.Ref() == "" || receipt.RecordedAt().IsZero() ||
		decision.Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(decision.Role(), identity.PermissionGoalsCreate) ||
		request.Permission() != identity.PermissionGoalsCreate ||
		request.ProjectRef() != projectRef ||
		request.ResourceRef() != projectRef.String() ||
		principal.ActorRef != actorRef ||
		receipt.RecordedAt().Before(decision.DecidedAt()) {
		return ErrForbidden
	}
	return nil
}

func validIntakeStateRef(ref intake.Ref) bool {
	value := string(ref)
	if len(value) <= len("intake:") || len(value) > 512 ||
		!strings.HasPrefix(value, "intake:") || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func validateStoredIntakeRecord(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
	record IntakeRecord,
) error {
	if record.ActorRef != actorRef || record.ProjectRef != projectRef ||
		record.State.Ref() != stateRef {
		return &StateError{Code: StateConflict}
	}
	receipt := record.Receipt
	if receipt.ActorRef != actorRef || receipt.ProjectRef != projectRef ||
		receipt.StateRef != stateRef || receipt.Revision != record.State.Revision() ||
		!validApplicationRef(receipt.AuthorizationReceiptRef) ||
		(receipt.Operation != IntakeOperationCreate && receipt.Operation != IntakeOperationApply) {
		return &StateError{Code: StateConflict}
	}
	expected, err := buildIntakeReceipt(
		receipt.Operation, receipt.RequestRef, receipt.RequestFingerprint,
		actorRef, projectRef, record.State, receipt.PreviousRevision,
		receipt.AuthorizationReceiptRef,
	)
	if err != nil || expected != receipt {
		return &StateError{Code: StateConflict, Cause: err}
	}
	return nil
}

func validateIntakeMutationRecord(
	replay IntakeReplayRequest,
	expectedState intake.State,
	previousRevision intake.Revision,
	record IntakeRecord,
) error {
	if !reflectIntakeStateEqual(record.State, expectedState) {
		return &StateError{Code: StateConflict}
	}
	return validateIntakeReplayRecord(replay, previousRevision, record)
}

func validateIntakeReplayRecord(
	replay IntakeReplayRequest,
	previousRevision intake.Revision,
	record IntakeRecord,
) error {
	if err := validateStoredIntakeRecord(
		replay.ActorRef, replay.ProjectRef, replay.StateRef, record,
	); err != nil {
		return err
	}
	receipt := record.Receipt
	if receipt.RequestRef != replay.RequestRef ||
		receipt.RequestFingerprint != replay.RequestFingerprint ||
		receipt.Operation != replay.Operation ||
		receipt.AuthorizationReceiptRef != replay.AuthorizationReceiptRef ||
		receipt.PreviousRevision != previousRevision ||
		receipt.Revision == 0 || receipt.Revision-1 != previousRevision {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func reflectIntakeStateEqual(left, right intake.State) bool {
	leftDigest, leftErr := IntakeStateDigest(left)
	rightDigest, rightErr := IntakeStateDigest(right)
	return leftErr == nil && rightErr == nil && leftDigest == rightDigest
}
