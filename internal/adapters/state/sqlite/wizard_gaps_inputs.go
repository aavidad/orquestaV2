package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

var _ application.WizardGapsStore = (*Repository)(nil)

func (repository *Repository) ReplayWizardGapsInput(
	ctx context.Context,
	request application.WizardGapsInputReplayRequest,
) (application.WizardGapsInputRecord, bool, error) {
	if err := validateWizardGapsInputReplayRequest(request); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, found, err := readWizardGapsInput(ctx, transaction, request)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	return record, found, nil
}

func (repository *Repository) ApplyWizardGapsMutation(
	ctx context.Context,
	reservation application.WizardGapsMutationReservation,
) (application.WizardGapsInputRecord, bool, error) {
	input := reservation.Input
	state := reservation.Intake
	snapshotJSON, err := validateIntakeApplyState(state)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	if err := validateWizardGapsMutationReservation(reservation); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	packRefsJSON, selectionsJSON, err := encodeWizardGapsInput(input)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	replayRequest := wizardGapsInputReplayRequest(input)
	replayed, found, err := readWizardGapsInput(ctx, transaction, replayRequest)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.WizardGapsInputRecord{}, false, err
		}
		return replayed, false, nil
	}
	intakeReplay := application.IntakeReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Operation: state.Receipt.Operation, ActorRef: state.ActorRef,
		ProjectRef: state.ProjectRef, StateRef: state.State.Ref(),
		AuthorizationReceiptRef: state.Receipt.AuthorizationReceiptRef,
	}
	if _, found, err := readIntakeReplay(
		ctx,
		transaction,
		intakeReplay,
	); err != nil {
		return application.WizardGapsInputRecord{}, false, err
	} else if found {
		return application.WizardGapsInputRecord{}, false, conflict(
			errors.New("sqlite.wizard_gaps_input_late_attachment_forbidden"),
		)
	}
	source, err := readCurrentIntakeRecord(
		ctx, transaction, input.ActorRef, input.ProjectRef, input.StateRef,
	)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	if source.Receipt.Ref != input.SourceIntakeReceiptRef ||
		source.State.Revision() != input.ExpectedRevision {
		return application.WizardGapsInputRecord{}, false, conflict(
			errors.New("sqlite.wizard_gaps_input_source_conflict"),
		)
	}
	outcome, err := applyIntakeMutationInTransaction(
		ctx,
		transaction,
		state,
		snapshotJSON,
	)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	record := application.WizardGapsInputRecord{
		Receipt: input, SourceRecord: source, OutcomeRecord: outcome,
	}
	if err := application.ValidateWizardGapsInputRecord(record); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	if err := insertWizardGapsInput(
		ctx, transaction, input, packRefsJSON, selectionsJSON,
	); err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	persisted, found, err := readWizardGapsInput(ctx, transaction, replayRequest)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	if !found {
		return application.WizardGapsInputRecord{}, false, invalid(
			errors.New("sqlite.wizard_gaps_input_missing_after_insert"),
		)
	}
	if err := commit(transaction); err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	return persisted, true, nil
}

func readWizardGapsInput(
	ctx context.Context,
	source queryer,
	request application.WizardGapsInputReplayRequest,
) (application.WizardGapsInputRecord, bool, error) {
	var record application.WizardGapsInputRecord
	receipt := &record.Receipt
	var (
		actorRef, projectRef, stateRef string
		origin, outcomeKind            string
		factSurface, factSharing       string
		factCorporate, factUsers       string
		factAuth, factCriticality      string
		packRefsJSON, selectionsJSON   string
		expectedRevision               int64
	)
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, actor_ref, project_ref, state_ref,
       expected_revision, origin, source_intake_receipt_ref,
       outcome_kind, outcome_receipt_ref,
       fact_surface, fact_sharing_intent, fact_corporate_identity,
       fact_target_users, fact_integration_auth, fact_integration_criticality,
       facts_digest, pack_refs_json, pack_refs_digest,
       selections_json, selections_digest,
       evaluator_schema, evaluator_version, evaluator_semantic_digest,
       authorization_receipt_ref
FROM wizard_gaps_input_receipts
WHERE actor_ref = ? AND project_ref = ? AND request_ref = ?`,
		request.ActorRef.String(), request.ProjectRef.String(), request.RequestRef,
	).Scan(
		&receipt.Ref, &receipt.RequestRef, &receipt.RequestFingerprint,
		&actorRef, &projectRef, &stateRef, &expectedRevision, &origin,
		&receipt.SourceIntakeReceiptRef, &outcomeKind, &receipt.OutcomeReceiptRef,
		&factSurface, &factSharing, &factCorporate, &factUsers,
		&factAuth, &factCriticality, &receipt.FactsDigest,
		&packRefsJSON, &receipt.PackRefsDigest,
		&selectionsJSON, &receipt.SelectionsDigest,
		&receipt.EvaluatorIdentity.Schema, &receipt.EvaluatorIdentity.Version,
		&receipt.EvaluatorIdentity.SemanticDigest,
		&receipt.AuthorizationReceiptRef,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return application.WizardGapsInputRecord{}, false, nil
	case err != nil:
		return application.WizardGapsInputRecord{}, false, mapDatabaseError(err)
	case expectedRevision <= 0 || uint64(expectedRevision) > maxSQLiteInteger:
		return application.WizardGapsInputRecord{}, false, invalid(
			errors.New("sqlite.wizard_gaps_input_revision_invalid"),
		)
	}
	actor, err := goal.NewActorRef(actorRef)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	project, err := goal.NewProjectRef(projectRef)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	receipt.ActorRef = actor
	receipt.ProjectRef = project
	receipt.StateRef = intake.Ref(stateRef)
	receipt.ExpectedRevision = intake.Revision(expectedRevision)
	receipt.Origin = intake.Origin(origin)
	receipt.OutcomeKind = application.WizardGapsRequestOutcomeKind(outcomeKind)
	receipt.Facts = gaps.Facts{
		Surface:                gaps.Surface(factSurface),
		SharingIntent:          gaps.SharingIntent(factSharing),
		CorporateIdentity:      gaps.Declaration(factCorporate),
		TargetUsers:            gaps.Declaration(factUsers),
		IntegrationAuth:        gaps.Declaration(factAuth),
		IntegrationCriticality: gaps.Declaration(factCriticality),
	}
	if err := decodeWizardGapsPackRefs(packRefsJSON, &receipt.PackRefs); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	if err := json.Unmarshal([]byte(selectionsJSON), &receipt.Selections); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(
			errors.Join(errors.New("sqlite.wizard_gaps_input_selections_invalid"), err),
		)
	}
	if receipt.RequestFingerprint != request.RequestFingerprint ||
		receipt.ActorRef != request.ActorRef ||
		receipt.ProjectRef != request.ProjectRef ||
		receipt.StateRef != request.StateRef ||
		receipt.ExpectedRevision != request.ExpectedRevision ||
		receipt.EvaluatorIdentity != request.EvaluatorIdentity ||
		receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef {
		return application.WizardGapsInputRecord{}, false, conflict(
			errors.New("sqlite.wizard_gaps_input_replay_conflict"),
		)
	}
	record.SourceRecord, err = readIntakeRecordByReceipt(
		ctx, source, receipt.SourceIntakeReceiptRef,
	)
	if err != nil {
		return application.WizardGapsInputRecord{}, false, err
	}
	switch receipt.OutcomeKind {
	case application.WizardGapsRequestOutcomeIntakeMutation:
		record.OutcomeRecord, err = readIntakeRecordByReceipt(
			ctx, source, receipt.OutcomeReceiptRef,
		)
	case application.WizardGapsRequestOutcomeNoOp:
		record.OutcomeRecord = record.SourceRecord
		outcome, found, outcomeErr := readWizardGapsNoOp(
			ctx,
			source,
			wizardGapsNoOpReplayRequestFromInput(request),
		)
		if outcomeErr != nil {
			err = outcomeErr
		} else if !found {
			err = errors.New("sqlite.wizard_gaps_input_outcome_missing")
		} else if outcome.Ref != receipt.OutcomeReceiptRef ||
			outcome.SourceIntakeReceiptRef != receipt.SourceIntakeReceiptRef ||
			outcome.RequestFingerprint != receipt.RequestFingerprint {
			err = errors.New("sqlite.wizard_gaps_input_outcome_binding_invalid")
		}
	default:
		err = errors.New("sqlite.wizard_gaps_input_outcome_kind_invalid")
	}
	if err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(err)
	}
	if err := application.ValidateWizardGapsInputRecord(record); err != nil {
		return application.WizardGapsInputRecord{}, false, invalid(
			errors.Join(errors.New("sqlite.wizard_gaps_input_invalid"), err),
		)
	}
	return record, true, nil
}

func insertWizardGapsInput(
	ctx context.Context,
	transaction *sql.Tx,
	receipt application.WizardGapsInputReceipt,
	packRefsJSON,
	selectionsJSON []byte,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO wizard_gaps_input_receipts(
    ref, request_ref, request_fingerprint, actor_ref, project_ref, state_ref,
    expected_revision, origin, source_intake_receipt_ref,
    outcome_kind, outcome_receipt_ref,
    fact_surface, fact_sharing_intent, fact_corporate_identity,
    fact_target_users, fact_integration_auth, fact_integration_criticality,
    facts_digest, pack_refs_json, pack_refs_digest,
    selections_json, selections_digest,
    evaluator_schema, evaluator_version, evaluator_semantic_digest,
    authorization_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.Ref, receipt.RequestRef, receipt.RequestFingerprint,
		receipt.ActorRef.String(), receipt.ProjectRef.String(), receipt.StateRef,
		int64(receipt.ExpectedRevision), string(receipt.Origin),
		receipt.SourceIntakeReceiptRef, string(receipt.OutcomeKind),
		receipt.OutcomeReceiptRef, string(receipt.Facts.Surface),
		string(receipt.Facts.SharingIntent),
		string(receipt.Facts.CorporateIdentity),
		string(receipt.Facts.TargetUsers),
		string(receipt.Facts.IntegrationAuth),
		string(receipt.Facts.IntegrationCriticality),
		receipt.FactsDigest, string(packRefsJSON), receipt.PackRefsDigest,
		string(selectionsJSON), receipt.SelectionsDigest,
		receipt.EvaluatorIdentity.Schema, receipt.EvaluatorIdentity.Version,
		receipt.EvaluatorIdentity.SemanticDigest,
		receipt.AuthorizationReceiptRef,
	)
	return mapDatabaseError(err)
}

func encodeWizardGapsInput(
	receipt application.WizardGapsInputReceipt,
) ([]byte, []byte, error) {
	packRefs := make([]string, len(receipt.PackRefs))
	for index, ref := range receipt.PackRefs {
		packRefs[index] = ref.String()
	}
	packRefsJSON, err := json.Marshal(packRefs)
	if err != nil {
		return nil, nil, err
	}
	selections := receipt.Selections
	if selections == nil {
		selections = []application.WizardGapsSelectionInput{}
	}
	selectionsJSON, err := json.Marshal(selections)
	if err != nil {
		return nil, nil, err
	}
	return packRefsJSON, selectionsJSON, nil
}

func decodeWizardGapsPackRefs(
	encoded string,
	destination *[]catalog.PackRef,
) error {
	var refs []string
	if err := json.Unmarshal([]byte(encoded), &refs); err != nil || refs == nil {
		return errors.Join(errors.New("sqlite.wizard_gaps_input_pack_refs_invalid"), err)
	}
	result := make([]catalog.PackRef, len(refs))
	for index, ref := range refs {
		value, err := catalog.NewPackRef(ref)
		if err != nil {
			return err
		}
		result[index] = value
	}
	*destination = result
	return nil
}

func validateWizardGapsInputReplayRequest(
	request application.WizardGapsInputReplayRequest,
) error {
	if !validIntakeRequestRef(request.RequestRef) ||
		!validCanonicalHash(request.RequestFingerprint) ||
		!validText(request.AuthorizationReceiptRef) ||
		request.ExpectedRevision == 0 ||
		uint64(request.ExpectedRevision) > maxSQLiteInteger {
		return errors.New("sqlite.wizard_gaps_input_replay_invalid")
	}
	identity, err := intake.NewDerivationIdentity(
		request.EvaluatorIdentity.Schema,
		request.EvaluatorIdentity.Version,
		request.EvaluatorIdentity.SemanticDigest,
	)
	if err != nil || identity != request.EvaluatorIdentity {
		return errors.New("sqlite.wizard_gaps_input_evaluator_invalid")
	}
	return validateIntakeScope(
		request.ActorRef, request.ProjectRef, request.StateRef,
	)
}

func validateWizardGapsMutationReservation(
	reservation application.WizardGapsMutationReservation,
) error {
	input := reservation.Input
	state := reservation.Intake
	if input.OutcomeKind != application.WizardGapsRequestOutcomeIntakeMutation ||
		input.OutcomeReceiptRef != state.Receipt.Ref ||
		input.RequestRef != state.RequestRef ||
		input.ActorRef != state.ActorRef ||
		input.ProjectRef != state.ProjectRef ||
		input.StateRef != state.State.Ref() ||
		input.ExpectedRevision != state.ExpectedRevision ||
		input.AuthorizationReceiptRef != state.AuthorizationReceipt.Ref() {
		return errors.New("sqlite.wizard_gaps_mutation_binding_invalid")
	}
	return nil
}

func wizardGapsInputReplayRequest(
	input application.WizardGapsInputReceipt,
) application.WizardGapsInputReplayRequest {
	return application.WizardGapsInputReplayRequest{
		RequestRef: input.RequestRef, RequestFingerprint: input.RequestFingerprint,
		ActorRef: input.ActorRef, ProjectRef: input.ProjectRef,
		StateRef: input.StateRef, ExpectedRevision: input.ExpectedRevision,
		EvaluatorIdentity:       input.EvaluatorIdentity,
		AuthorizationReceiptRef: input.AuthorizationReceiptRef,
	}
}

func wizardGapsNoOpReplayRequestFromInput(
	request application.WizardGapsInputReplayRequest,
) application.WizardGapsNoOpReplayRequest {
	return application.WizardGapsNoOpReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: request.RequestFingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		EvaluatorIdentity:       request.EvaluatorIdentity,
		AuthorizationReceiptRef: request.AuthorizationReceiptRef,
	}
}
