package application

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const (
	intakeDossierConfirmationAuthorizationPrefix = "authorization-request:intake-dossier-confirm:"
	intakeDossierConfirmationReceiptPrefix       = "intake-dossier-confirmation-receipt:"
	intakeDossierConfirmationReasonPrefix        = "operator.intake_dossier_confirmation:"
)

type ConfirmIntakeDossierRequest struct {
	RequestRef string
	DossierRef IntakeDossierRef
	Confirm    bool
}

type ConfirmIntakeDossierResult struct {
	Record       GoalRecord
	Confirmation IntakeDossierConfirmation
	Created      bool
}

// IntakeDossierConfirmationAuthorizationRequestRef binds authorization to one
// exact confirmation request.
func IntakeDossierConfirmationAuthorizationRequestRef(
	requestRef string,
) (string, error) {
	if !validIntakeRequestRef(requestRef) {
		return "", invalidIntakeRequestRefError()
	}
	return intakeDossierConfirmationAuthorizationPrefix + requestRef, nil
}

// IntakeDossierConfirmationAppSpecReason binds the immutable dossier ref into
// the AppSpec hash without adding another source field to the Goal lifecycle.
func IntakeDossierConfirmationAppSpecReason(
	dossierRef IntakeDossierRef,
) string {
	return intakeDossierConfirmationReasonPrefix + string(dossierRef)
}

func (orchestrator *Orchestrator) ConfirmIntakeDossier(
	ctx context.Context,
	access Access,
	request ConfirmIntakeDossierRequest,
) (ConfirmIntakeDossierResult, error) {
	if orchestrator == nil || orchestrator.intakeDossier == nil {
		return ConfirmIntakeDossierResult{}, errors.New("application.unavailable")
	}
	if !request.Confirm {
		return ConfirmIntakeDossierResult{}, errors.New("application.confirmation_required")
	}
	if err := validateConfirmIntakeDossierRequest(request); err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	authorizationRequestRef, err :=
		IntakeDossierConfirmationAuthorizationRequestRef(request.RequestRef)
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	now := orchestrator.clock.Now()
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsCreate, projectRef.String(), now,
		authorizationRequestRef,
	)
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	now = authorizationCausalFloor(now, authorization)

	dossierRecord, err := orchestrator.intakeDossier.GetIntakeDossier(
		ctx,
		GetIntakeDossierRequest{
			ActorRef: principal.ActorRef, ProjectRef: projectRef,
			DossierRef: request.DossierRef,
		},
	)
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	dossier := dossierRecord.Dossier
	plan := dossier.Plan()
	submission := SubmitRequest{
		RequestRef: request.RequestRef, Statement: dossier.Statement(),
		NormalizedObjective: dossier.Objective(), Confirm: true, Plan: &plan,
	}
	fingerprint := intakeDossierConfirmationFingerprint(
		request, principal, projectRef, dossier, authorization.Ref(),
	)
	createState, err := orchestrator.prepareGoalCreation(
		ctx, submission, principal, projectRef, authorization, now,
		IntakeDossierConfirmationAppSpecReason(dossier.Ref()), fingerprint,
	)
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	confirmation := buildIntakeDossierConfirmation(
		request.RequestRef, fingerprint, principal, dossier,
		createState.Goal, authorization.Ref(),
	)
	state := ConfirmIntakeDossierState{
		CreateGoal: createState, Dossier: dossier, Confirmation: confirmation,
	}
	if err := ValidateIntakeDossierConfirmationState(state); err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	persisted, created, err :=
		orchestrator.state.ConfirmIntakeDossierAndCreateGoal(ctx, state)
	if err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	if created {
		if err := validatePersistedCandidate(
			createState.Goal, createState.Executions, persisted.Goal,
		); err != nil {
			return ConfirmIntakeDossierResult{}, err
		}
	}
	if err := ValidatePersistedIntakeDossierConfirmationRecord(
		dossier, authorization, persisted,
	); err != nil {
		return ConfirmIntakeDossierResult{}, err
	}
	return ConfirmIntakeDossierResult{
		Record: persisted.Goal, Confirmation: persisted.Confirmation,
		Created: created,
	}, nil
}

func validateConfirmIntakeDossierRequest(
	request ConfirmIntakeDossierRequest,
) error {
	if !validIntakeRequestRef(request.RequestRef) {
		return invalidIntakeRequestRefError()
	}
	if !validIntakeDossierRef(string(request.DossierRef), "intake-dossier:") {
		return invalidIntakeDossier("dossier_ref", nil)
	}
	return nil
}

func intakeDossierConfirmationFingerprint(
	request ConfirmIntakeDossierRequest,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	dossier IntakeDossier,
	authorizationReceiptRef string,
) string {
	return fingerprintFields(
		"orquesta.intake.dossier.confirm.v1",
		request.RequestRef,
		principal.Ref.String(),
		principal.ActorRef.String(),
		projectRef.String(),
		string(dossier.Ref()),
		dossier.Digest(),
		string(dossier.StateRef()),
		strconv.FormatUint(uint64(dossier.StateRevision()), 10),
		dossier.StateDigest(),
		dossier.SourceIntakeReceiptRef(),
		dossier.PlanDigest(),
		authorizationReceiptRef,
	)
}

func buildIntakeDossierConfirmation(
	requestRef string,
	fingerprint string,
	principal identity.Principal,
	dossier IntakeDossier,
	aggregate goal.Goal,
	authorizationReceiptRef string,
) IntakeDossierConfirmation {
	appSpec := aggregate.AppSpec()
	confirmedAt := appSpec.ConfirmedAt()
	refDigest := fingerprintFields(
		"orquesta.intake.dossier.confirmation-receipt.v1",
		requestRef,
		fingerprint,
		principal.Ref.String(),
		string(dossier.Ref()),
		aggregate.Ref().String(),
		appSpec.Ref().String(),
		appSpec.Hash(),
		authorizationReceiptRef,
		confirmedAt.Format(time.RFC3339Nano),
	)
	return IntakeDossierConfirmation{
		Ref:        intakeDossierConfirmationReceiptPrefix + refDigest,
		RequestRef: requestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ActorRef: principal.ActorRef,
		ProjectRef: dossier.ProjectRef(),
		StateRef:   dossier.StateRef(), StateRevision: dossier.StateRevision(),
		StateDigest:            dossier.StateDigest(),
		SourceIntakeReceiptRef: dossier.SourceIntakeReceiptRef(),
		DossierRef:             dossier.Ref(), DossierDigest: dossier.Digest(),
		PlanDigest: dossier.PlanDigest(),
		GoalRef:    aggregate.Ref(), AppSpecRef: appSpec.Ref(),
		SpecHash:                appSpec.Hash(),
		AuthorizationReceiptRef: authorizationReceiptRef,
		ConfirmedAt:             confirmedAt,
	}
}

// ValidateIntakeDossierConfirmationState is shared with durable adapters. It
// validates application-owned semantics; adapters still revalidate persisted
// authorization and source fences inside their transaction.
func ValidateIntakeDossierConfirmationState(
	state ConfirmIntakeDossierState,
) error {
	restored, err := RestoreIntakeDossier(SnapshotIntakeDossier(state.Dossier))
	if err != nil || !reflect.DeepEqual(
		SnapshotIntakeDossier(restored), SnapshotIntakeDossier(state.Dossier),
	) {
		return &StateError{Code: StateInvalid, Cause: err}
	}
	create := state.CreateGoal
	confirmation := state.Confirmation
	appSpec := create.Goal.AppSpec()
	intent := appSpec.Intent()
	if create.RequestRef != confirmation.RequestRef ||
		create.RequestFingerprint != confirmation.RequestFingerprint ||
		create.RequestedBy != confirmation.PrincipalRef ||
		create.AuthorizationReceipt.Ref() !=
			confirmation.AuthorizationReceiptRef ||
		create.Goal.Ref() != confirmation.GoalRef ||
		create.Goal.Actor() != confirmation.ActorRef ||
		create.Goal.Project() != confirmation.ProjectRef ||
		intent.Statement() != state.Dossier.Statement() ||
		appSpec.Objective() != state.Dossier.Objective() ||
		appSpec.Reason() !=
			IntakeDossierConfirmationAppSpecReason(state.Dossier.Ref()) ||
		appSpec.ConfirmedBy() != confirmation.ActorRef ||
		appSpec.Ref() != confirmation.AppSpecRef ||
		appSpec.Hash() != confirmation.SpecHash ||
		appSpec.ConfirmedAt() != confirmation.ConfirmedAt ||
		confirmation.StateRef != state.Dossier.StateRef() ||
		confirmation.StateRevision != state.Dossier.StateRevision() ||
		confirmation.StateDigest != state.Dossier.StateDigest() ||
		confirmation.SourceIntakeReceiptRef !=
			state.Dossier.SourceIntakeReceiptRef() ||
		confirmation.DossierRef != state.Dossier.Ref() ||
		confirmation.DossierDigest != state.Dossier.Digest() ||
		confirmation.PlanDigest != state.Dossier.PlanDigest() ||
		confirmation.ProjectRef != state.Dossier.ProjectRef() ||
		confirmation.ActorRef != state.Dossier.ActorRef() ||
		!validIntakeDossierConfirmationRef(confirmation.Ref) ||
		!intakeDossierPlanMatchesInitialGoal(
			state.Dossier.Plan(), create.Goal,
		) {
		return &StateError{Code: StateInvalid}
	}
	authorizationRequestRef, err :=
		IntakeDossierConfirmationAuthorizationRequestRef(create.RequestRef)
	if err != nil || validateIntakeAuthorization(
		create.AuthorizationReceipt,
		confirmation.ActorRef,
		confirmation.ProjectRef,
		authorizationRequestRef,
	) != nil {
		return &StateError{Code: StateInvalid, Cause: err}
	}
	expected := buildIntakeDossierConfirmation(
		create.RequestRef,
		create.RequestFingerprint,
		create.AuthorizationReceipt.Decision().Request().Principal(),
		state.Dossier,
		create.Goal,
		create.AuthorizationReceipt.Ref(),
	)
	if expected != confirmation {
		return &StateError{Code: StateInvalid}
	}
	return nil
}

// ValidatePersistedIntakeDossierConfirmationRecord validates an immutable
// confirmation against the current, monotonically evolved Goal projection.
// Durable adapters use it during recovery; creation uses the stricter
// ValidateIntakeDossierConfirmationState contract above.
func ValidatePersistedIntakeDossierConfirmationRecord(
	dossier IntakeDossier,
	authorization identity.AuthorizationReceipt,
	record IntakeDossierConfirmationRecord,
) error {
	confirmation := record.Confirmation
	authorizationRequestRef, err :=
		IntakeDossierConfirmationAuthorizationRequestRef(confirmation.RequestRef)
	if err != nil ||
		authorization.Ref() != confirmation.AuthorizationReceiptRef ||
		validateIntakeAuthorization(
			authorization, confirmation.ActorRef, confirmation.ProjectRef,
			authorizationRequestRef,
		) != nil {
		return &StateError{Code: StateConflict, Cause: err}
	}
	principal := authorization.Decision().Request().Principal()
	request := ConfirmIntakeDossierRequest{
		RequestRef: confirmation.RequestRef,
		DossierRef: confirmation.DossierRef,
		Confirm:    true,
	}
	fingerprint := intakeDossierConfirmationFingerprint(
		request, principal, confirmation.ProjectRef, dossier,
		authorization.Ref(),
	)
	return validatePersistedIntakeDossierConfirmation(
		request,
		fingerprint,
		principal,
		dossier,
		authorization.Ref(),
		record,
	)
}

func validatePersistedIntakeDossierConfirmation(
	request ConfirmIntakeDossierRequest,
	fingerprint string,
	principal identity.Principal,
	dossier IntakeDossier,
	authorizationReceiptRef string,
	record IntakeDossierConfirmationRecord,
) error {
	confirmation := record.Confirmation
	aggregate := record.Goal.Goal
	if record.Goal.RequestRef != request.RequestRef ||
		record.Goal.RequestFingerprint != fingerprint ||
		record.Goal.RequestedBy != principal.Ref ||
		confirmation.RequestRef != request.RequestRef ||
		confirmation.RequestFingerprint != fingerprint ||
		confirmation.PrincipalRef != principal.Ref ||
		confirmation.ActorRef != principal.ActorRef ||
		confirmation.ProjectRef != dossier.ProjectRef() ||
		confirmation.StateRef != dossier.StateRef() ||
		confirmation.StateRevision != dossier.StateRevision() ||
		confirmation.StateDigest != dossier.StateDigest() ||
		confirmation.SourceIntakeReceiptRef != dossier.SourceIntakeReceiptRef() ||
		confirmation.DossierRef != dossier.Ref() ||
		confirmation.DossierDigest != dossier.Digest() ||
		confirmation.PlanDigest != dossier.PlanDigest() ||
		confirmation.AuthorizationReceiptRef != authorizationReceiptRef ||
		confirmation.GoalRef != aggregate.Ref() ||
		confirmation.AppSpecRef != aggregate.AppSpec().Ref() ||
		confirmation.SpecHash != aggregate.SpecHash() ||
		confirmation.ConfirmedAt != aggregate.AppSpec().ConfirmedAt() ||
		aggregate.Actor() != principal.ActorRef ||
		aggregate.Project() != dossier.ProjectRef() ||
		aggregate.AppSpec().Intent().Statement() != dossier.Statement() ||
		aggregate.AppSpec().Objective() != dossier.Objective() ||
		aggregate.AppSpec().Reason() !=
			IntakeDossierConfirmationAppSpecReason(dossier.Ref()) ||
		aggregate.AppSpec().ConfirmedBy() != principal.ActorRef ||
		!validIntakeDossierConfirmationRef(confirmation.Ref) ||
		!intakeDossierPlanMatchesLiveGoal(dossier.Plan(), aggregate) {
		return &StateError{Code: StateConflict}
	}
	expected := buildIntakeDossierConfirmation(
		request.RequestRef, fingerprint, principal, dossier, aggregate,
		authorizationReceiptRef,
	)
	if confirmation != expected ||
		validateLiveIntakeDossierGoalRecord(
			record.Goal, dossier, principal, authorizationReceiptRef,
		) != nil {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateLiveIntakeDossierGoalRecord(
	record GoalRecord,
	dossier IntakeDossier,
	principal identity.Principal,
	authorizationReceiptRef string,
) error {
	aggregate := record.Goal
	snapshot := aggregate.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil ||
		!intakeDossierPlanMatchesLiveGoal(dossier.Plan(), aggregate) ||
		len(record.Executions) == 0 ||
		len(record.BudgetEnvelopes) != 3 ||
		len(record.WorkItemAuthorities) < len(dossier.Plan().WorkItems) {
		return &StateError{Code: StateConflict, Cause: err}
	}
	initialItems := make(map[goal.WorkItemRef]struct{}, len(dossier.Plan().WorkItems))
	for index := range dossier.Plan().WorkItems {
		itemRef, err := goal.NewWorkItemRef(snapshot.WorkItems[index].Ref)
		if err != nil {
			return &StateError{Code: StateConflict, Cause: err}
		}
		initialItems[itemRef] = struct{}{}
	}
	seenExecutions := make(map[goal.ExecutionRef]struct{}, len(record.Executions))
	initialExecutionFound := false
	for _, execution := range record.Executions {
		_, itemFound := aggregate.WorkItem(execution.WorkItemRef)
		_, duplicate := seenExecutions[execution.Ref]
		if execution.Ref.String() == "" || duplicate || !itemFound ||
			execution.GoalRef != aggregate.Ref() ||
			execution.AttemptNo == 0 ||
			execution.MaxExecutionAttempts < execution.AttemptNo ||
			execution.PlanGeneration == 0 ||
			execution.PlanGeneration > aggregate.PlanGeneration() ||
			execution.AppSpecGeneration != aggregate.AppSpec().Generation() ||
			execution.SpecHash != aggregate.SpecHash() ||
			execution.CreatedAt.IsZero() {
			return &StateError{Code: StateConflict}
		}
		seenExecutions[execution.Ref] = struct{}{}
		if _, initial := initialItems[execution.WorkItemRef]; initial {
			initialExecutionFound = true
		}
	}
	if !initialExecutionFound {
		return &StateError{Code: StateConflict}
	}
	authorityCounts := make(map[goal.WorkItemRef]int, len(initialItems))
	for _, authority := range record.WorkItemAuthorities {
		if _, initial := initialItems[authority.WorkItemRef]; !initial {
			continue
		}
		authorityCounts[authority.WorkItemRef]++
		if authorityCounts[authority.WorkItemRef] != 1 ||
			authority.PrincipalRef != principal.Ref ||
			authority.Permission != identity.PermissionGoalsCreate ||
			authority.Source != EffectApprovalSourceGoalConfirmation ||
			authority.AuthorizationReceipt.Ref() != authorizationReceiptRef ||
			validateWorkItemAuthority(aggregate, authority) != nil {
			return &StateError{Code: StateConflict}
		}
	}
	for itemRef := range initialItems {
		if authorityCounts[itemRef] != 1 {
			return &StateError{Code: StateConflict}
		}
	}
	if _, err := historicalEffectPolicy(record); err != nil {
		return &StateError{Code: StateConflict, Cause: err}
	}
	return nil
}

func validIntakeDossierConfirmationRef(value string) bool {
	return len(value) == len(intakeDossierConfirmationReceiptPrefix)+64 &&
		strings.HasPrefix(value, intakeDossierConfirmationReceiptPrefix) &&
		validIntakeDossierDigest(
			strings.TrimPrefix(value, intakeDossierConfirmationReceiptPrefix),
		)
}
