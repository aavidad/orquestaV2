package sqlite

import (
	"errors"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

const intakeReceiptPrefix = "intake-receipt:"

func validateIntakeReplayRequest(request application.IntakeReplayRequest) error {
	if !validIntakeRequestRef(request.RequestRef) ||
		!validCanonicalHash(request.RequestFingerprint) ||
		!validText(request.AuthorizationReceiptRef) ||
		(request.Operation != application.IntakeOperationCreate &&
			request.Operation != application.IntakeOperationApply) {
		return errors.New("sqlite.intake_replay_invalid")
	}
	return validateIntakeScope(request.ActorRef, request.ProjectRef, request.StateRef)
}

func validateIntakeCreateState(state application.IntakeCreateState) ([]byte, error) {
	if err := validateIntakeMutationIdentity(
		state.RequestRef, state.RequestFingerprint,
		state.ActorRef, state.ProjectRef, state.State, state.Receipt,
	); err != nil {
		return nil, err
	}
	if err := validateIntakeAuthorization(
		state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
		state.Receipt.AuthorizationReceiptRef, state.Receipt.Operation, state.RequestRef,
	); err != nil {
		return nil, err
	}
	if state.Receipt.Operation != application.IntakeOperationCreate ||
		state.Receipt.PreviousRevision != 0 ||
		state.State.Revision() != 1 ||
		state.Receipt.Revision != state.State.Revision() {
		return nil, errors.New("sqlite.intake_create_invalid")
	}
	if err := application.ValidateIntakeChain([]application.IntakeRecord{{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}}); err != nil {
		return nil, errors.Join(errors.New("sqlite.intake_create_chain_invalid"), err)
	}
	return encodeAndValidateIntakeDigest(state.State, state.Receipt.StateDigest)
}

func validateIntakeApplyState(state application.IntakeApplyState) ([]byte, error) {
	if err := validateIntakeMutationIdentity(
		state.RequestRef, state.RequestFingerprint,
		state.ActorRef, state.ProjectRef, state.State, state.Receipt,
	); err != nil {
		return nil, err
	}
	if err := validateIntakeAuthorization(
		state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
		state.Receipt.AuthorizationReceiptRef, state.Receipt.Operation, state.RequestRef,
	); err != nil {
		return nil, err
	}
	if state.Receipt.Operation != application.IntakeOperationApply ||
		state.ExpectedRevision == 0 ||
		uint64(state.ExpectedRevision) >= maxSQLiteInteger ||
		state.Receipt.PreviousRevision != state.ExpectedRevision ||
		state.State.Revision() != state.ExpectedRevision+1 ||
		state.Receipt.Revision != state.State.Revision() {
		return nil, errors.New("sqlite.intake_apply_invalid")
	}
	return encodeAndValidateIntakeDigest(state.State, state.Receipt.StateDigest)
}

func validateIntakeMutationIdentity(
	requestRef, requestFingerprint string,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	state intake.State,
	receipt application.IntakeReceipt,
) error {
	if !validIntakeRequestRef(requestRef) || !validCanonicalHash(requestFingerprint) ||
		!validIntakeReceiptRef(receipt.Ref) ||
		receipt.RequestRef != requestRef ||
		receipt.RequestFingerprint != requestFingerprint ||
		receipt.ActorRef != actorRef ||
		receipt.ProjectRef != projectRef ||
		receipt.StateRef != state.Ref() ||
		!validCanonicalHash(receipt.StateDigest) ||
		!validText(receipt.AuthorizationReceiptRef) ||
		uint64(state.Revision()) > maxSQLiteInteger ||
		uint64(receipt.PreviousRevision) > maxSQLiteInteger ||
		uint64(receipt.Revision) > maxSQLiteInteger {
		return errors.New("sqlite.intake_mutation_invalid")
	}
	if err := validateIntakeScope(actorRef, projectRef, state.Ref()); err != nil {
		return err
	}
	return nil
}

func validateIntakeAuthorization(
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	receiptRef string,
	operation application.IntakeOperation,
	requestRef string,
) error {
	request := receipt.Decision().Request()
	principal := request.Principal()
	expectedRequestRef, err := application.IntakeAuthorizationRequestRef(operation, requestRef)
	if err != nil ||
		receipt.Ref() == "" || receipt.Ref() != receiptRef ||
		identity.ValidatePrincipal(principal) != nil ||
		request.RequestRef() != expectedRequestRef ||
		principal.ActorRef != actorRef ||
		request.ProjectRef() != projectRef ||
		request.Permission() != identity.PermissionGoalsCreate ||
		request.ResourceRef() != projectRef.String() ||
		receipt.Decision().Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(receipt.Decision().Role(), identity.PermissionGoalsCreate) {
		return errors.New("sqlite.intake_authorization_invalid")
	}
	return nil
}

func validateIntakeScope(actorRef goal.ActorRef, projectRef goal.ProjectRef, stateRef intake.Ref) error {
	if actorRef.String() == "" || projectRef.String() == "" || stateRef == "" {
		return errors.New("sqlite.intake_scope_invalid")
	}
	if restored, err := goal.NewActorRef(actorRef.String()); err != nil || restored != actorRef {
		return errors.New("sqlite.intake_actor_invalid")
	}
	if restored, err := goal.NewProjectRef(projectRef.String()); err != nil || restored != projectRef {
		return errors.New("sqlite.intake_project_invalid")
	}
	// NewState is the domain-owned syntax authority for an intake identity.
	if _, err := intake.NewState(stateRef, intake.Policy{MaxQuestionRounds: 1}); err != nil {
		return err
	}
	return nil
}

func encodeAndValidateIntakeDigest(state intake.State, expected string) ([]byte, error) {
	encoded, err := encodeIntakeState(state)
	if err != nil {
		return nil, err
	}
	if intakeStateDigest(encoded) != expected {
		return nil, errors.New("sqlite.intake_state_digest_invalid")
	}
	return encoded, nil
}

func validatePersistedIntakeRecord(record application.IntakeRecord, storedJSON []byte) error {
	receipt := record.Receipt
	if err := validateIntakeScope(record.ActorRef, record.ProjectRef, record.State.Ref()); err != nil {
		return err
	}
	if !validIntakeReceiptRef(receipt.Ref) ||
		!validIntakeRequestRef(receipt.RequestRef) ||
		!validCanonicalHash(receipt.RequestFingerprint) ||
		(receipt.Operation != application.IntakeOperationCreate &&
			receipt.Operation != application.IntakeOperationApply) ||
		receipt.ActorRef != record.ActorRef ||
		receipt.ProjectRef != record.ProjectRef ||
		receipt.StateRef != record.State.Ref() ||
		receipt.Revision != record.State.Revision() ||
		receipt.Revision != receipt.PreviousRevision+1 ||
		(receipt.Operation == application.IntakeOperationCreate && receipt.PreviousRevision != 0) ||
		(receipt.Operation == application.IntakeOperationApply && receipt.PreviousRevision == 0) ||
		!validCanonicalHash(receipt.StateDigest) ||
		!validText(receipt.AuthorizationReceiptRef) ||
		uint64(receipt.Revision) > maxSQLiteInteger ||
		intakeStateDigest(storedJSON) != receipt.StateDigest {
		return errors.New("sqlite.intake_record_invalid")
	}
	canonical, err := encodeIntakeState(record.State)
	if err != nil {
		return err
	}
	if string(canonical) != string(storedJSON) {
		return errors.New("sqlite.intake_snapshot_not_canonical")
	}
	return nil
}

func validIntakeRequestRef(value string) bool {
	return validText(value) && len(value) <= 512 &&
		!strings.ContainsAny(value, "\x00\n\r")
}

func validIntakeReceiptRef(value string) bool {
	return strings.HasPrefix(value, intakeReceiptPrefix) &&
		len(value) == len(intakeReceiptPrefix)+64 &&
		validCanonicalHash(strings.TrimPrefix(value, intakeReceiptPrefix))
}
