package application

import (
	"context"
	"errors"
	"hash"
	"strconv"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

const intakeDossierAuthorizationRequestPrefix = "authorization-request:intake-dossier-generate:"

// IntakeDossierGenerationReceipt is the deterministic receipt for one
// accepted dossier preparation. It binds the complete request fingerprint to
// the immutable dossier and the exact intake revision from which it was built.
type IntakeDossierGenerationReceipt struct {
	Ref                     string
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	StateRevision           intake.Revision
	StateDigest             string
	SourceIntakeReceiptRef  string
	DossierRef              IntakeDossierRef
	DossierDigest           string
	PlanDigest              string
	AuthorizationReceiptRef string
}

type IntakeDossierRecord struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	Dossier    IntakeDossier
	Receipt    IntakeDossierGenerationReceipt
}

type IntakeDossierReplayRequest struct {
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	DossierRef              IntakeDossierRef
	StateRef                intake.Ref
	ExpectedRevision        intake.Revision
	SourceIntakeReceiptRef  string
	PlanDigest              string
	AuthorizationReceiptRef string
}

type IntakeDossierCreateState struct {
	RequestRef             string
	RequestFingerprint     string
	ActorRef               goal.ActorRef
	ProjectRef             goal.ProjectRef
	ExpectedRevision       intake.Revision
	SourceIntakeReceiptRef string
	AuthorizationReceipt   identity.AuthorizationReceipt
	Dossier                IntakeDossier
	Receipt                IntakeDossierGenerationReceipt
}

// IntakeDossierStore is separate from IntakeStore because dossier history is
// immutable proposal state, not another intake or Goal lifecycle. Create must
// atomically enforce actor/project/request idempotency and verify that the
// current intake revision and source receipt still match CreateState.
//
// A dossier is content-addressed and may be prepared by multiple request refs.
// GetIntakeDossier must always return the canonical record from the first
// committed creation of that DossierRef; later generation receipts remain
// available only through their exact request replay and never replace it.
type IntakeDossierStore interface {
	ReplayIntakeDossier(context.Context, IntakeDossierReplayRequest) (IntakeDossierRecord, bool, error)
	CreateIntakeDossier(context.Context, IntakeDossierCreateState) (IntakeDossierRecord, bool, error)
	GetIntakeDossier(context.Context, goal.ActorRef, goal.ProjectRef, IntakeDossierRef) (IntakeDossierRecord, error)
}

type PrepareIntakeDossierRequest struct {
	RequestRef             string
	ActorRef               goal.ActorRef
	ProjectRef             goal.ProjectRef
	StateRef               intake.Ref
	ExpectedRevision       intake.Revision
	SourceIntakeReceiptRef string
	Plan                   PlanSpec
	Input                  IntakeDossierInput
	AuthorizationReceipt   identity.AuthorizationReceipt
}

type GetIntakeDossierRequest struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	DossierRef IntakeDossierRef
}

type IntakeDossierResult struct {
	Record  IntakeDossierRecord
	Created bool
}

type IntakeDossierService struct {
	intakes  IntakeStore
	dossiers IntakeDossierStore
}

func NewIntakeDossierService(
	intakes IntakeStore,
	dossiers IntakeDossierStore,
) (*IntakeDossierService, error) {
	if intakes == nil {
		return nil, errors.New("application.intake_store_required")
	}
	if dossiers == nil {
		return nil, errors.New("application.intake_dossier_store_required")
	}
	return &IntakeDossierService{intakes: intakes, dossiers: dossiers}, nil
}

// IntakeDossierAuthorizationRequestRef binds authorization to one dossier
// preparation request. The receipt must grant goals.create for the project.
func IntakeDossierAuthorizationRequestRef(requestRef string) (string, error) {
	if !validIntakeRequestRef(requestRef) {
		return "", invalidIntakeRequestRefError()
	}
	return intakeDossierAuthorizationRequestPrefix + requestRef, nil
}

func (service *IntakeDossierService) PrepareIntakeDossier(
	ctx context.Context,
	request PrepareIntakeDossierRequest,
) (IntakeDossierResult, error) {
	if service == nil || service.intakes == nil || service.dossiers == nil {
		return IntakeDossierResult{}, errors.New("application.unavailable")
	}
	if err := validatePrepareIntakeDossierRequest(request); err != nil {
		return IntakeDossierResult{}, err
	}
	authorizationRequestRef, err := IntakeDossierAuthorizationRequestRef(request.RequestRef)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	if err := validateIntakeAuthorization(
		request.AuthorizationReceipt,
		request.ActorRef,
		request.ProjectRef,
		authorizationRequestRef,
	); err != nil {
		return IntakeDossierResult{}, err
	}

	planDigest := IntakeDossierPlanDigest(request.Plan)
	fingerprint := intakeDossierRequestFingerprint(request, planDigest)
	replay := IntakeDossierReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		SourceIntakeReceiptRef:  request.SourceIntakeReceiptRef,
		PlanDigest:              planDigest,
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}
	if record, found, replayErr := service.replay(ctx, replay); replayErr != nil {
		return IntakeDossierResult{}, replayErr
	} else if found {
		return IntakeDossierResult{Record: record}, nil
	}

	current, err := service.intakes.GetIntake(
		ctx, request.ActorRef, request.ProjectRef, request.StateRef,
	)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	if err := validateStoredIntakeRecord(
		request.ActorRef, request.ProjectRef, request.StateRef, current,
	); err != nil {
		return IntakeDossierResult{}, err
	}
	if current.State.Revision() != request.ExpectedRevision ||
		current.Receipt.Ref != request.SourceIntakeReceiptRef {
		return IntakeDossierResult{}, &StateError{Code: StateConflict}
	}

	dossier, err := BuildIntakeDossier(current, request.Plan, request.Input)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	replay.DossierRef = dossier.Ref()
	receipt, err := BuildIntakeDossierGenerationReceipt(
		request.RequestRef, fingerprint, dossier, request.AuthorizationReceipt.Ref(),
	)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	persisted, created, err := service.dossiers.CreateIntakeDossier(
		ctx,
		IntakeDossierCreateState{
			RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
			ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
			ExpectedRevision:       request.ExpectedRevision,
			SourceIntakeReceiptRef: request.SourceIntakeReceiptRef,
			AuthorizationReceipt:   request.AuthorizationReceipt,
			Dossier:                dossier, Receipt: receipt,
		},
	)
	if err != nil {
		if IsStateError(err, StateConflict) {
			if recovered, found, replayErr := service.replay(ctx, replay); replayErr == nil && found {
				return IntakeDossierResult{Record: recovered}, nil
			}
		}
		return IntakeDossierResult{}, err
	}
	if err := validateIntakeDossierMutationRecord(replay, dossier, persisted); err != nil {
		return IntakeDossierResult{}, err
	}
	cloned, err := cloneIntakeDossierRecord(persisted)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	return IntakeDossierResult{Record: cloned, Created: created}, nil
}

func (service *IntakeDossierService) GetIntakeDossier(
	ctx context.Context,
	request GetIntakeDossierRequest,
) (IntakeDossierRecord, error) {
	if service == nil || service.dossiers == nil {
		return IntakeDossierRecord{}, errors.New("application.unavailable")
	}
	if err := validateIntakeScope(request.ActorRef, request.ProjectRef); err != nil {
		return IntakeDossierRecord{}, err
	}
	if !validIntakeDossierRef(string(request.DossierRef), "intake-dossier:") {
		return IntakeDossierRecord{}, invalidIntakeDossier("dossier_ref", nil)
	}
	record, err := service.dossiers.GetIntakeDossier(
		ctx, request.ActorRef, request.ProjectRef, request.DossierRef,
	)
	if err != nil {
		return IntakeDossierRecord{}, err
	}
	if err := validateStoredIntakeDossierRecord(
		request.ActorRef, request.ProjectRef, request.DossierRef, record,
	); err != nil {
		return IntakeDossierRecord{}, err
	}
	return cloneIntakeDossierRecord(record)
}

func (service *IntakeDossierService) replay(
	ctx context.Context,
	request IntakeDossierReplayRequest,
) (IntakeDossierRecord, bool, error) {
	record, found, err := service.dossiers.ReplayIntakeDossier(ctx, request)
	if err != nil || !found {
		return IntakeDossierRecord{}, false, err
	}
	if err := validateIntakeDossierReplayRecord(request, record); err != nil {
		return IntakeDossierRecord{}, false, err
	}
	cloned, err := cloneIntakeDossierRecord(record)
	if err != nil {
		return IntakeDossierRecord{}, false, err
	}
	return cloned, true, nil
}

func validatePrepareIntakeDossierRequest(request PrepareIntakeDossierRequest) error {
	if err := validateIntakeRequestScope(
		request.RequestRef, request.ActorRef, request.ProjectRef,
	); err != nil {
		return err
	}
	if !validIntakeStateRef(request.StateRef) {
		return &intake.DomainError{Code: intake.ErrorInvalidRef, Field: "state_ref"}
	}
	if request.ExpectedRevision == 0 {
		return invalidIntakeDossier("expected_revision", nil)
	}
	if !validIntakeDossierRef(request.SourceIntakeReceiptRef, "intake-receipt:") {
		return invalidIntakeDossier("source_intake_receipt_ref", nil)
	}
	if err := validateIntakeDossierInput(request.Input); err != nil {
		return err
	}
	return validateIntakeDossierPlan(request.Plan)
}

func intakeDossierRequestFingerprint(
	request PrepareIntakeDossierRequest,
	planDigest string,
) string {
	digest := fingerprintDigest(
		"orquesta.intake.dossier.generate.v1",
		request.ActorRef.String(), request.ProjectRef.String(),
		string(request.StateRef),
		strconv.FormatUint(uint64(request.ExpectedRevision), 10),
		request.SourceIntakeReceiptRef, planDigest,
		request.AuthorizationReceipt.Ref(),
	)
	writeIntakeDossierInputFingerprint(digest, request.Input)
	return fingerprintHex(digest)
}

// IntakeDossierGenerationFingerprint lets recovery adapters recompute the
// application-owned request fingerprint from a restored immutable dossier.
// Invalid dossier/auth inputs return an empty fingerprint.
func IntakeDossierGenerationFingerprint(
	dossier IntakeDossier,
	authorizationReceiptRef string,
) string {
	if !validApplicationRef(authorizationReceiptRef) {
		return ""
	}
	if _, err := RestoreIntakeDossier(SnapshotIntakeDossier(dossier)); err != nil {
		return ""
	}
	// RequestRef is the idempotency key and intentionally does not enter the
	// payload fingerprint, matching other application mutations.
	digest := fingerprintDigest(
		"orquesta.intake.dossier.generate.v1",
		dossier.ActorRef().String(), dossier.ProjectRef().String(),
		string(dossier.StateRef()),
		strconv.FormatUint(uint64(dossier.StateRevision()), 10),
		dossier.SourceIntakeReceiptRef(), dossier.PlanDigest(),
		authorizationReceiptRef,
	)
	writeIntakeDossierInputFingerprint(digest, IntakeDossierInput{
		Statement: dossier.Statement(), Objective: dossier.Objective(),
		Sections: dossier.Sections(), Diagrams: dossier.Diagrams(),
		RiskRefs: dossier.RiskRefs(),
	})
	return fingerprintHex(digest)
}

func writeIntakeDossierInputFingerprint(digest hash.Hash, input IntakeDossierInput) {
	writeFingerprintField(digest, input.Statement)
	writeFingerprintField(digest, input.Objective)
	writeFingerprintField(digest, strconv.Itoa(len(input.Sections)))
	for _, section := range input.Sections {
		writeFingerprintField(digest, string(section.Ref))
		writeFingerprintField(digest, string(section.Kind))
		writeFingerprintField(digest, string(section.TitleKey))
		writeFingerprintField(digest, section.Markdown)
	}
	writeFingerprintField(digest, strconv.Itoa(len(input.Diagrams)))
	for _, diagram := range input.Diagrams {
		writeFingerprintField(digest, string(diagram.Ref))
		writeFingerprintField(digest, string(diagram.Purpose))
		writeFingerprintField(digest, string(diagram.Kind))
		writeFingerprintField(digest, diagram.Source)
		writeFingerprintField(digest, string(diagram.AltTextKey))
	}
	writeFingerprintField(digest, strconv.Itoa(len(input.RiskRefs)))
	for _, riskRef := range input.RiskRefs {
		writeFingerprintField(digest, string(riskRef))
	}
}

func BuildIntakeDossierGenerationReceipt(
	requestRef string,
	requestFingerprint string,
	dossier IntakeDossier,
	authorizationReceiptRef string,
) (IntakeDossierGenerationReceipt, error) {
	if !validIntakeRequestRef(requestRef) ||
		!validIntakeDossierDigest(requestFingerprint) ||
		!validApplicationRef(authorizationReceiptRef) {
		return IntakeDossierGenerationReceipt{}, invalidIntakeDossier("generation_receipt", nil)
	}
	if _, err := RestoreIntakeDossier(SnapshotIntakeDossier(dossier)); err != nil {
		return IntakeDossierGenerationReceipt{}, err
	}
	refDigest := fingerprintFields(
		"orquesta.intake.dossier.generation-receipt.v1",
		requestRef, requestFingerprint,
		dossier.ActorRef().String(), dossier.ProjectRef().String(),
		string(dossier.StateRef()),
		strconv.FormatUint(uint64(dossier.StateRevision()), 10),
		dossier.StateDigest(), dossier.SourceIntakeReceiptRef(),
		string(dossier.Ref()), dossier.Digest(), dossier.PlanDigest(),
		authorizationReceiptRef,
	)
	return IntakeDossierGenerationReceipt{
		Ref:        "intake-dossier-generation-receipt:" + refDigest,
		RequestRef: requestRef, RequestFingerprint: requestFingerprint,
		ActorRef: dossier.ActorRef(), ProjectRef: dossier.ProjectRef(),
		StateRef: dossier.StateRef(), StateRevision: dossier.StateRevision(),
		StateDigest:            dossier.StateDigest(),
		SourceIntakeReceiptRef: dossier.SourceIntakeReceiptRef(),
		DossierRef:             dossier.Ref(), DossierDigest: dossier.Digest(),
		PlanDigest:              dossier.PlanDigest(),
		AuthorizationReceiptRef: authorizationReceiptRef,
	}, nil
}

func validateStoredIntakeDossierRecord(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef IntakeDossierRef,
	record IntakeDossierRecord,
) error {
	dossier := record.Dossier
	if record.ActorRef != actorRef || record.ProjectRef != projectRef ||
		dossier.ActorRef() != actorRef || dossier.ProjectRef() != projectRef ||
		dossier.Ref() != dossierRef {
		return &StateError{Code: StateConflict}
	}
	if _, err := RestoreIntakeDossier(SnapshotIntakeDossier(dossier)); err != nil {
		return &StateError{Code: StateConflict, Cause: err}
	}
	receipt := record.Receipt
	if fingerprint := IntakeDossierGenerationFingerprint(
		dossier, receipt.AuthorizationReceiptRef,
	); fingerprint == "" || receipt.RequestFingerprint != fingerprint {
		return &StateError{Code: StateConflict}
	}
	expected, err := BuildIntakeDossierGenerationReceipt(
		receipt.RequestRef, receipt.RequestFingerprint,
		dossier, receipt.AuthorizationReceiptRef,
	)
	if err != nil || expected != receipt {
		return &StateError{Code: StateConflict, Cause: err}
	}
	return nil
}

func validateIntakeDossierReplayRecord(
	replay IntakeDossierReplayRequest,
	record IntakeDossierRecord,
) error {
	if err := validateStoredIntakeDossierRecord(
		replay.ActorRef, replay.ProjectRef, record.Dossier.Ref(), record,
	); err != nil {
		return err
	}
	receipt := record.Receipt
	if receipt.RequestRef != replay.RequestRef ||
		receipt.RequestFingerprint != replay.RequestFingerprint ||
		(replay.DossierRef != "" && receipt.DossierRef != replay.DossierRef) ||
		receipt.StateRef != replay.StateRef ||
		receipt.StateRevision != replay.ExpectedRevision ||
		receipt.SourceIntakeReceiptRef != replay.SourceIntakeReceiptRef ||
		receipt.PlanDigest != replay.PlanDigest ||
		receipt.AuthorizationReceiptRef != replay.AuthorizationReceiptRef {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateIntakeDossierMutationRecord(
	replay IntakeDossierReplayRequest,
	expected IntakeDossier,
	record IntakeDossierRecord,
) error {
	if record.Dossier.Ref() != expected.Ref() ||
		record.Dossier.Digest() != expected.Digest() {
		return &StateError{Code: StateConflict}
	}
	return validateIntakeDossierReplayRecord(replay, record)
}

func cloneIntakeDossierRecord(record IntakeDossierRecord) (IntakeDossierRecord, error) {
	dossier, err := RestoreIntakeDossier(SnapshotIntakeDossier(record.Dossier))
	if err != nil {
		return IntakeDossierRecord{}, err
	}
	record.Dossier = dossier
	return record, nil
}
