package postgres

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"orquesta/internal/goal"
)

const (
	ErrorInvalid       = "postgres.foundation_invalid"
	ErrorContext       = "postgres.foundation_context"
	ErrorUnavailable   = "postgres.foundation_unavailable"
	ErrorCommitUnknown = "postgres.foundation_commit_unknown"
	ErrorStaleFence    = "postgres.foundation_stale_fence"
	ErrorInvariant     = "postgres.foundation_invariant"
)

const transactionTimeSQL = `SELECT transaction_timestamp()`

const fencedCASSQL = `UPDATE work_items
SET snapshot = $1,
    revision = revision + 1,
    updated_at = $2,
    last_mutation_ref = $3
WHERE project_ref = $4
  AND goal_ref = $5
  AND work_item_ref = $6
  AND revision = $7
  AND plan_generation = $8
  AND lease_token = $9
  AND lease_fence = $10
  AND lease_expires_at > $2
RETURNING revision`

const insertEventSQL = `INSERT INTO events
  (ref, mutation_ref, project_ref, goal_ref, work_item_ref, revision, occurred_at, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const insertOutboxSQL = `INSERT INTO outbox
  (ref, mutation_ref, project_ref, goal_ref, work_item_ref, revision, available_at, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const revalidateFencedMutationSQL = `SELECT TRUE
FROM work_items
WHERE project_ref = $1
  AND goal_ref = $2
  AND work_item_ref = $3
  AND revision = $4
  AND plan_generation = $5
  AND lease_token = $6
  AND lease_fence = $7
  AND last_mutation_ref = $8
  AND lease_expires_at > clock_timestamp()
FOR UPDATE`

// ContractError exposes stable machine codes without leaking SQL, DSNs or
// backend details.
type ContractError struct {
	Code  string
	cause error
}

func (err *ContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *ContractError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func ErrorCode(err error) string {
	var contractErr *ContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

// AtomicMutationRequest is one narrow, reusable PostgreSQL transaction
// primitive. It is not StateRepository and does not choose lifecycle state.
// Snapshot, event and outbox payloads are decisions already made by the
// application writer.
type AtomicMutationRequest struct {
	ProjectRef       goal.ProjectRef
	GoalRef          goal.GoalRef
	WorkItemRef      goal.WorkItemRef
	ExpectedRevision goal.Revision
	PlanGeneration   goal.PlanGeneration
	LeaseToken       string
	LeaseFence       uint64
	MutationRef      string
	EventRef         string
	OutboxRef        string
	Snapshot         []byte
	EventPayload     []byte
	OutboxPayload    []byte
}

// AtomicMutationReceipt proves only the local transaction outcome. It is not
// product evidence and cannot accredit OPS-11.
type AtomicMutationReceipt struct {
	MutationRef   string
	Revision      goal.Revision
	TransactionAt time.Time
}

// Foundation owns no pool configuration, DSN, driver, schema migration or
// clock. Composition injects an already configured database/sql handle.
type Foundation struct {
	database *sql.DB
}

func NewFoundation(database *sql.DB) (*Foundation, error) {
	if database == nil {
		return nil, contractError(ErrorInvalid, nil)
	}
	return &Foundation{database: database}, nil
}

func (foundation *Foundation) ApplyAtomicMutation(
	ctx context.Context,
	request AtomicMutationRequest,
) (AtomicMutationReceipt, error) {
	if foundation == nil || foundation.database == nil {
		return AtomicMutationReceipt{}, contractError(ErrorUnavailable, nil)
	}
	if err := ctx.Err(); err != nil {
		return AtomicMutationReceipt{}, contractError(ErrorContext, err)
	}
	if err := validateAtomicMutation(request); err != nil {
		return AtomicMutationReceipt{}, err
	}

	transaction, err := foundation.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	open := true
	defer func() {
		if open {
			_ = transaction.Rollback()
		}
	}()

	var transactionAt time.Time
	if err := transaction.QueryRowContext(ctx, transactionTimeSQL).Scan(&transactionAt); err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	if transactionAt.IsZero() {
		return AtomicMutationReceipt{}, contractError(ErrorInvariant, nil)
	}
	transactionAt = transactionAt.UTC()

	var revision int64
	err = transaction.QueryRowContext(ctx, fencedCASSQL,
		request.Snapshot, transactionAt, request.MutationRef,
		request.ProjectRef.String(), request.GoalRef.String(), request.WorkItemRef.String(),
		int64(request.ExpectedRevision), int64(request.PlanGeneration), request.LeaseToken, int64(request.LeaseFence),
	).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return AtomicMutationReceipt{}, contractError(ErrorStaleFence, nil)
	}
	if err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	if revision != int64(request.ExpectedRevision)+1 {
		return AtomicMutationReceipt{}, contractError(ErrorInvariant, nil)
	}

	common := []any{
		request.MutationRef, request.ProjectRef.String(), request.GoalRef.String(),
		request.WorkItemRef.String(), revision, transactionAt,
	}
	if _, err := transaction.ExecContext(ctx, insertEventSQL,
		request.EventRef, common[0], common[1], common[2], common[3], common[4], common[5], request.EventPayload,
	); err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	if _, err := transaction.ExecContext(ctx, insertOutboxSQL,
		request.OutboxRef, common[0], common[1], common[2], common[3], common[4], common[5], request.OutboxPayload,
	); err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	var authorityLive bool
	err = transaction.QueryRowContext(ctx, revalidateFencedMutationSQL,
		request.ProjectRef.String(), request.GoalRef.String(), request.WorkItemRef.String(), revision,
		int64(request.PlanGeneration), request.LeaseToken, int64(request.LeaseFence), request.MutationRef,
	).Scan(&authorityLive)
	if errors.Is(err, sql.ErrNoRows) {
		return AtomicMutationReceipt{}, contractError(ErrorStaleFence, nil)
	}
	if err != nil {
		return AtomicMutationReceipt{}, databaseError(err)
	}
	if !authorityLive {
		return AtomicMutationReceipt{}, contractError(ErrorInvariant, nil)
	}
	if err := transaction.Commit(); err != nil {
		open = false
		return AtomicMutationReceipt{}, contractError(ErrorCommitUnknown, err)
	}
	open = false
	return AtomicMutationReceipt{
		MutationRef: request.MutationRef,
		Revision:    goal.Revision(revision), TransactionAt: transactionAt,
	}, nil
}

func validateAtomicMutation(request AtomicMutationRequest) error {
	if request.ProjectRef.String() == "" || request.GoalRef.String() == "" || request.WorkItemRef.String() == "" ||
		request.ExpectedRevision == 0 || uint64(request.ExpectedRevision) >= math.MaxInt64 ||
		request.PlanGeneration == 0 || uint64(request.PlanGeneration) > math.MaxInt64 ||
		request.LeaseFence == 0 || request.LeaseFence > math.MaxInt64 ||
		!validOpaque(request.LeaseToken) || !validOpaque(request.MutationRef) ||
		!validOpaque(request.EventRef) || !validOpaque(request.OutboxRef) ||
		request.Snapshot == nil || request.EventPayload == nil || request.OutboxPayload == nil {
		return contractError(ErrorInvalid, nil)
	}
	return nil
}

func validOpaque(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

func databaseError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return contractError(ErrorContext, err)
	}
	return contractError(ErrorUnavailable, err)
}

func contractError(code string, cause error) error {
	return &ContractError{Code: code, cause: cause}
}
