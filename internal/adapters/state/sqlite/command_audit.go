package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/ports"
)

var _ ports.CommandAuditStore = (*Repository)(nil)

func (repository *Repository) Begin(
	ctx context.Context,
	record ports.CommandAuditRecord,
) (ports.CommandAuditSession, error) {
	if ctx == nil || validateNewCommandAdmission(record) != nil {
		return ports.CommandAuditSession{}, invalid(errors.New("sqlite.command_admission_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.CommandAuditSession{}, err
	}
	defer func() { _ = transaction.Rollback() }()

	existing, found, err := readCommandAuditByIdentity(ctx, transaction, record)
	if err != nil {
		return ports.CommandAuditSession{}, err
	}
	if found {
		if !sameCommandAdmission(record, existing.Record) {
			return ports.CommandAuditSession{}, commandAuditConflict("sqlite.command_admission_replay_conflict")
		}
		if err := commit(transaction); err != nil {
			return ports.CommandAuditSession{}, err
		}
		return existing, nil
	}

	admittedAt := repository.now().Round(0).UTC()
	if admittedAt.IsZero() {
		return ports.CommandAuditSession{}, invalid(errors.New("sqlite.command_admission_time_invalid"))
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO command_invocations(
 ref,command_id,command_version,registry_digest,schema_digest,request_ref,input_digest,
 principal_ref,project_ref,authenticated_execution_ref,replay_mode,status,admitted_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.Ref, record.CommandID, record.CommandVersion, record.RegistryDigest,
		record.SchemaDigest, record.RequestRef, record.InputDigest, record.PrincipalRef,
		record.ProjectRef, record.AuthenticatedExecutionRef, string(record.ReplayMode),
		record.Status, requiredTime(admittedAt),
	); err != nil {
		if isSQLiteConstraintError(err) {
			return ports.CommandAuditSession{}, commandAuditConflict("sqlite.command_admission_identity_conflict")
		}
		return ports.CommandAuditSession{}, mapDatabaseError(err)
	}
	record.AdmittedAt = admittedAt
	session := ports.CommandAuditSession{
		Record: record, OutcomeRef: record.Ref + ":outcome", AdmissionCreated: true,
	}
	if err := commit(transaction); err != nil {
		return ports.CommandAuditSession{}, err
	}
	return session, nil
}

func (repository *Repository) Complete(
	ctx context.Context,
	request ports.CommandAuditCompletionRequest,
) (ports.CommandAuditCompletion, error) {
	if ctx == nil || !validText(request.RecordRef) ||
		validateNewCommandTerminal(request.RecordRef, request.Terminal) != nil {
		return ports.CommandAuditCompletion{}, invalid(errors.New("sqlite.command_outcome_invalid"))
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.CommandAuditCompletion{}, err
	}
	defer func() { _ = transaction.Rollback() }()

	session, found, err := readCommandAuditByRef(ctx, transaction, request.RecordRef)
	if err != nil {
		return ports.CommandAuditCompletion{}, err
	}
	if !found {
		return ports.CommandAuditCompletion{}, mapDatabaseError(sql.ErrNoRows)
	}
	if session.Terminal != nil {
		if !sameCommandTerminal(request.Terminal, *session.Terminal) {
			return ports.CommandAuditCompletion{}, commandAuditConflict("sqlite.command_outcome_replay_conflict")
		}
		if err := commit(transaction); err != nil {
			return ports.CommandAuditCompletion{}, err
		}
		return ports.CommandAuditCompletion{Terminal: *session.Terminal, Created: false}, nil
	}

	completedAt := repository.now().Round(0).UTC()
	if completedAt.IsZero() || completedAt.Before(session.Record.AdmittedAt) {
		return ports.CommandAuditCompletion{}, invalid(errors.New("sqlite.command_outcome_time_invalid"))
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO command_outcomes(
 ref,command_invocation_ref,status,error_code,output_digest,completed_at
) VALUES (?,?,?,?,?,?)`,
		request.Terminal.OutcomeRef, request.RecordRef, request.Terminal.Status,
		request.Terminal.ErrorCode, request.Terminal.OutputDigest, requiredTime(completedAt),
	); err != nil {
		if isSQLiteConstraintError(err) {
			return ports.CommandAuditCompletion{}, commandAuditConflict("sqlite.command_outcome_identity_conflict")
		}
		return ports.CommandAuditCompletion{}, mapDatabaseError(err)
	}
	terminal := request.Terminal
	terminal.CompletedAt = completedAt
	if err := commit(transaction); err != nil {
		return ports.CommandAuditCompletion{}, err
	}
	return ports.CommandAuditCompletion{Terminal: terminal, Created: true}, nil
}

func readCommandAuditByIdentity(
	ctx context.Context,
	source queryer,
	record ports.CommandAuditRecord,
) (ports.CommandAuditSession, bool, error) {
	return scanCommandAuditSession(source.QueryRowContext(ctx, `
SELECT invocation.ref,invocation.command_id,invocation.command_version,
 invocation.registry_digest,invocation.schema_digest,invocation.request_ref,
 invocation.input_digest,invocation.principal_ref,invocation.project_ref,
 invocation.authenticated_execution_ref,invocation.replay_mode,invocation.status,
 invocation.admitted_at,outcome.ref,outcome.status,outcome.error_code,
 outcome.output_digest,outcome.completed_at
FROM command_invocations invocation
LEFT JOIN command_outcomes outcome ON outcome.command_invocation_ref=invocation.ref
WHERE invocation.principal_ref=? AND invocation.project_ref=?
 AND invocation.command_id=? AND invocation.command_version=? AND invocation.request_ref=?`,
		record.PrincipalRef, record.ProjectRef, record.CommandID, record.CommandVersion,
		record.RequestRef,
	))
}

func readCommandAuditByRef(
	ctx context.Context,
	source queryer,
	ref string,
) (ports.CommandAuditSession, bool, error) {
	return scanCommandAuditSession(source.QueryRowContext(ctx, `
SELECT invocation.ref,invocation.command_id,invocation.command_version,
 invocation.registry_digest,invocation.schema_digest,invocation.request_ref,
 invocation.input_digest,invocation.principal_ref,invocation.project_ref,
 invocation.authenticated_execution_ref,invocation.replay_mode,invocation.status,
 invocation.admitted_at,outcome.ref,outcome.status,outcome.error_code,
 outcome.output_digest,outcome.completed_at
FROM command_invocations invocation
LEFT JOIN command_outcomes outcome ON outcome.command_invocation_ref=invocation.ref
WHERE invocation.ref=?`, ref))
}

func scanCommandAuditSession(row *sql.Row) (ports.CommandAuditSession, bool, error) {
	var session ports.CommandAuditSession
	var replayMode string
	var admittedAt int64
	var outcomeRef, outcomeStatus, errorCode, outputDigest sql.NullString
	var completedAt sql.NullInt64
	err := row.Scan(
		&session.Record.Ref, &session.Record.CommandID, &session.Record.CommandVersion,
		&session.Record.RegistryDigest, &session.Record.SchemaDigest, &session.Record.RequestRef,
		&session.Record.InputDigest, &session.Record.PrincipalRef, &session.Record.ProjectRef,
		&session.Record.AuthenticatedExecutionRef, &replayMode, &session.Record.Status,
		&admittedAt, &outcomeRef, &outcomeStatus, &errorCode, &outputDigest, &completedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.CommandAuditSession{}, false, nil
	}
	if err != nil {
		return ports.CommandAuditSession{}, false, mapDatabaseError(err)
	}
	session.Record.ReplayMode = ports.CommandReplayMode(replayMode)
	session.Record.AdmittedAt = time.Unix(0, admittedAt).UTC()
	session.OutcomeRef = session.Record.Ref + ":outcome"
	if outcomeRef.Valid {
		session.Terminal = &ports.CommandAuditTerminal{
			OutcomeRef: outcomeRef.String, Status: outcomeStatus.String,
			ErrorCode: errorCode.String, OutputDigest: outputDigest.String,
			CompletedAt: restoredTime(completedAt),
		}
	}
	return session, true, nil
}

func validateNewCommandAdmission(record ports.CommandAuditRecord) error {
	if !validText(record.Ref) || !validText(record.CommandID) || !validText(record.CommandVersion) ||
		!validSHA256Ref(record.RegistryDigest) || !validLowerHex(record.SchemaDigest, 64) ||
		!validText(record.RequestRef) || !validLowerHex(record.InputDigest, 64) ||
		!validText(record.PrincipalRef) || !validText(record.ProjectRef) ||
		!optionalText(record.AuthenticatedExecutionRef) ||
		(record.ReplayMode != ports.CommandReplayApplicationReceipt &&
			record.ReplayMode != ports.CommandReplayReadReexecute) ||
		record.Status != "admitted" || record.ErrorCode != "" || record.OutputDigest != "" ||
		!record.AdmittedAt.IsZero() || !record.CompletedAt.IsZero() {
		return errors.New("sqlite.command_admission_invalid")
	}
	return nil
}

func validateNewCommandTerminal(recordRef string, terminal ports.CommandAuditTerminal) error {
	if terminal.OutcomeRef != recordRef+":outcome" ||
		!validLowerHex(terminal.OutputDigest, 64) || !terminal.CompletedAt.IsZero() {
		return errors.New("sqlite.command_outcome_invalid")
	}
	switch terminal.Status {
	case "completed":
		if terminal.ErrorCode != "" {
			return errors.New("sqlite.command_outcome_invalid")
		}
	case "rejected":
		if !commandRejectedCode(terminal.ErrorCode) {
			return errors.New("sqlite.command_outcome_invalid")
		}
	case "failed":
		if terminal.ErrorCode != "unavailable" && terminal.ErrorCode != "internal" {
			return errors.New("sqlite.command_outcome_invalid")
		}
	default:
		return errors.New("sqlite.command_outcome_invalid")
	}
	return nil
}

func sameCommandAdmission(want, got ports.CommandAuditRecord) bool {
	return want.Ref == got.Ref && want.CommandID == got.CommandID &&
		want.CommandVersion == got.CommandVersion && want.RegistryDigest == got.RegistryDigest &&
		want.SchemaDigest == got.SchemaDigest && want.RequestRef == got.RequestRef &&
		want.InputDigest == got.InputDigest && want.PrincipalRef == got.PrincipalRef &&
		want.ProjectRef == got.ProjectRef &&
		want.AuthenticatedExecutionRef == got.AuthenticatedExecutionRef &&
		want.ReplayMode == got.ReplayMode && want.Status == got.Status &&
		!got.AdmittedAt.IsZero() && got.ErrorCode == "" && got.OutputDigest == "" &&
		got.CompletedAt.IsZero()
}

func sameCommandTerminal(want, got ports.CommandAuditTerminal) bool {
	return want.OutcomeRef == got.OutcomeRef && want.Status == got.Status &&
		want.ErrorCode == got.ErrorCode && want.OutputDigest == got.OutputDigest &&
		!got.CompletedAt.IsZero()
}

func commandRejectedCode(code string) bool {
	switch code {
	case "invalid_request", "unauthenticated", "forbidden", "not_found", "conflict":
		return true
	default:
		return false
	}
}

func validSHA256Ref(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validLowerHex(strings.TrimPrefix(value, "sha256:"), 64)
}

func validLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func commandAuditConflict(code string) error {
	return fmt.Errorf("%w: %s", ports.ErrCommandAuditConflict, code)
}
