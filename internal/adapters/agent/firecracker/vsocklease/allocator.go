package vsocklease

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"orquesta/internal/ports"
)

const (
	reservationStateActive   = "active"
	reservationStateExpired  = "expired"
	reservationStateReleased = "released"
)

// Config is supplied by composition. DB must be the deployment's canonical
// transactional state database with schema installed by its migration owner;
// this adapter neither opens another database nor applies migrations.
type Config struct {
	DB                   *sql.DB
	AdapterRef           string
	PoolRef              string
	MinimumGuestCID      uint32
	MaximumGuestCID      uint32
	MaximumLeaseDuration time.Duration
	Now                  func() time.Time
}

type Allocator struct {
	config Config
}

var _ ports.AgentMicroVMVsockCIDAllocator = (*Allocator)(nil)

type Error struct {
	Code string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var allocatorErr *Error
	if errors.As(err, &allocatorErr) {
		return allocatorErr.Code
	}
	return ""
}

// SchemaStatements describes the tables required in the canonical state
// database. It is deliberately data-only: the canonical migration owner must
// review and install them before wiring this planned adapter.
func SchemaStatements() []string {
	return append([]string(nil), schemaStatements...)
}

func Open(ctx context.Context, config Config) (*Allocator, error) {
	if config.DB == nil ||
		!validRef(config.AdapterRef) ||
		!validRef(config.PoolRef) ||
		config.MinimumGuestCID < ports.AgentMicroVMMinimumGuestCID ||
		config.MaximumGuestCID < config.MinimumGuestCID ||
		config.MaximumGuestCID == ports.AgentMicroVMReservedAnyCID ||
		config.MaximumLeaseDuration <= 0 ||
		config.Now == nil {
		return nil, allocatorError("config_invalid")
	}
	if err := config.DB.PingContext(ctx); err != nil {
		return nil, allocatorError("store_unavailable")
	}
	if err := verifySchema(ctx, config.DB); err != nil {
		return nil, err
	}
	return &Allocator{config: config}, nil
}

func (allocator *Allocator) Reserve(
	ctx context.Context,
	request ports.AgentMicroVMVsockCIDReservationRequest,
) (ports.AgentMicroVMVsockCIDLease, error) {
	if allocator == nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("unavailable")
	}
	if err := ports.ValidateAgentMicroVMVsockCIDReservationRequest(request); err != nil ||
		request.PoolRef != allocator.config.PoolRef ||
		request.LeaseDuration > allocator.config.MaximumLeaseDuration {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("reservation_request_invalid")
	}
	requestDigest, err := ports.AgentMicroVMVsockCIDReservationRequestDigest(request)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("reservation_request_invalid")
	}
	scopeDigest, err := ports.AgentMicroVMNetworkScopeDigest(request.Scope)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("reservation_request_invalid")
	}
	now := allocator.now()
	connection, err := allocator.beginImmediate(ctx)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	defer rollback(connection)
	if err := expireLeases(ctx, connection, request.PoolRef, now); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}

	operation, found, err := readOperation(ctx, connection, request.PoolRef, request.IdempotencyKey)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if found {
		if operation.Kind != "reserve" || operation.RequestDigest != requestDigest {
			return ports.AgentMicroVMVsockCIDLease{}, allocatorError("idempotency_conflict")
		}
		record, err := readReservationByLease(ctx, connection, operation.LeaseRef)
		if err != nil {
			return ports.AgentMicroVMVsockCIDLease{}, mapReadError(err)
		}
		if record.State != reservationStateActive || !record.Lease.ExpiresAt.After(now) {
			return ports.AgentMicroVMVsockCIDLease{}, allocatorError("reservation_replay_denied")
		}
		if err := commit(ctx, connection); err != nil {
			return ports.AgentMicroVMVsockCIDLease{}, err
		}
		return record.Lease, nil
	}

	used, err := readActiveCIDs(ctx, connection, request.PoolRef)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	guestCID, ok := allocator.selectCID(requestDigest, used)
	if !ok {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("capacity_reached")
	}
	fencingToken, err := nextFencingToken(ctx, connection, request.PoolRef, guestCID)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	lease := ports.AgentMicroVMVsockCIDLease{
		PoolRef: request.PoolRef, ScopeDigest: scopeDigest,
		ExecutionRef: request.Scope.ExecutionRef.String(), AgentRef: request.Scope.AgentRef,
		OwnerRef: request.OwnerRef, RequestDigest: requestDigest,
		IdempotencyKey: request.IdempotencyKey, GuestCID: guestCID,
		FencingToken: fencingToken, Revision: 1, AcquiredAt: now,
		ExpiresAt: now.Add(request.LeaseDuration), AdapterRef: allocator.config.AdapterRef,
	}
	lease.LeaseRef = ports.AgentMicroVMVsockCIDLeaseRef(lease)
	lease.VsockBackendRef = ports.AgentMicroVMVsockBackendRef(lease.LeaseRef)
	lease.ReceiptRef = ports.AgentMicroVMVsockCIDLeaseReceiptRef(lease)
	if err := ports.ValidateAgentMicroVMVsockCIDLease(lease); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("lease_invalid")
	}
	if err := insertReservation(ctx, connection, lease); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if err := insertOperation(ctx, connection, operationRecord{
		PoolRef: request.PoolRef, IdempotencyKey: request.IdempotencyKey,
		Kind: "reserve", RequestDigest: requestDigest, LeaseRef: lease.LeaseRef,
		Result: mustJSON(lease), CompletedAt: now,
	}); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if err := commit(ctx, connection); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	return lease, nil
}

func (allocator *Allocator) Renew(
	ctx context.Context,
	request ports.AgentMicroVMVsockCIDRenewalRequest,
) (ports.AgentMicroVMVsockCIDLease, error) {
	if allocator == nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("unavailable")
	}
	if err := ports.ValidateAgentMicroVMVsockCIDRenewalRequest(request); err != nil ||
		request.PoolRef != allocator.config.PoolRef ||
		request.LeaseDuration > allocator.config.MaximumLeaseDuration {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("renewal_request_invalid")
	}
	requestDigest, _ := ports.AgentMicroVMVsockCIDRenewalRequestDigest(request)
	now := allocator.now()
	connection, err := allocator.beginImmediate(ctx)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	defer rollback(connection)
	if err := expireLeases(ctx, connection, request.PoolRef, now); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if operation, found, err := readOperation(
		ctx, connection, request.PoolRef, request.IdempotencyKey,
	); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	} else if found {
		if operation.Kind != "renew" || operation.RequestDigest != requestDigest {
			return ports.AgentMicroVMVsockCIDLease{}, allocatorError("idempotency_conflict")
		}
		var lease ports.AgentMicroVMVsockCIDLease
		if json.Unmarshal(operation.Result, &lease) != nil ||
			ports.ValidateAgentMicroVMVsockCIDLease(lease) != nil {
			return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_corrupt")
		}
		record, err := readReservationByLease(ctx, connection, lease.LeaseRef)
		if err != nil {
			return ports.AgentMicroVMVsockCIDLease{}, mapReadError(err)
		}
		if record.State != reservationStateActive || !record.Lease.ExpiresAt.After(now) {
			return ports.AgentMicroVMVsockCIDLease{}, allocatorError("lease_inactive")
		}
		if err := commit(ctx, connection); err != nil {
			return ports.AgentMicroVMVsockCIDLease{}, err
		}
		return lease, nil
	}

	record, err := readReservationByLease(ctx, connection, request.LeaseRef)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, mapReadError(err)
	}
	if err := validateCurrentOwner(record, request.PoolRef, request.ScopeDigest,
		request.OwnerRef, request.FencingToken); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	if record.State != reservationStateActive || !record.Lease.ExpiresAt.After(now) {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("lease_inactive")
	}
	if record.Lease.Revision != request.ExpectedRevision {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("stale_revision")
	}
	newExpiry := now.Add(request.LeaseDuration)
	if !newExpiry.After(record.Lease.ExpiresAt) {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("renewal_not_extending")
	}
	record.Lease.Revision++
	record.Lease.ExpiresAt = newExpiry
	record.Lease.ReceiptRef = ports.AgentMicroVMVsockCIDLeaseReceiptRef(record.Lease)
	if _, err := connection.ExecContext(ctx, `
UPDATE agent_microvm_vsock_cid_reservations
SET revision = ?, expires_at_unix_nano = ?, receipt_ref = ?
WHERE lease_ref = ? AND state = ? AND revision = ?`,
		record.Lease.Revision, record.Lease.ExpiresAt.UnixNano(), record.Lease.ReceiptRef,
		record.Lease.LeaseRef, reservationStateActive, request.ExpectedRevision,
	); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if err := insertOperation(ctx, connection, operationRecord{
		PoolRef: request.PoolRef, IdempotencyKey: request.IdempotencyKey,
		Kind: "renew", RequestDigest: requestDigest, LeaseRef: request.LeaseRef,
		Result: mustJSON(record.Lease), CompletedAt: now,
	}); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	if err := commit(ctx, connection); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	return record.Lease, nil
}

func (allocator *Allocator) Recover(
	ctx context.Context,
	request ports.AgentMicroVMVsockCIDRecoveryRequest,
) (ports.AgentMicroVMVsockCIDLease, error) {
	if allocator == nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("unavailable")
	}
	if err := ports.ValidateAgentMicroVMVsockCIDRecoveryRequest(request); err != nil ||
		request.PoolRef != allocator.config.PoolRef {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("recovery_request_invalid")
	}
	now := allocator.now()
	connection, err := allocator.beginImmediate(ctx)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	defer rollback(connection)
	if err := expireLeases(ctx, connection, request.PoolRef, now); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("store_unavailable")
	}
	record, err := readReservationByLease(ctx, connection, request.LeaseRef)
	if err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, mapReadError(err)
	}
	if err := validateCurrentOwner(record, request.PoolRef, request.ScopeDigest,
		request.OwnerRef, request.FencingToken); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	if record.State != reservationStateActive || !record.Lease.ExpiresAt.After(now) {
		return ports.AgentMicroVMVsockCIDLease{}, allocatorError("lease_inactive")
	}
	if err := commit(ctx, connection); err != nil {
		return ports.AgentMicroVMVsockCIDLease{}, err
	}
	return record.Lease, nil
}

func (allocator *Allocator) Release(
	ctx context.Context,
	request ports.AgentMicroVMVsockCIDReleaseRequest,
) (ports.AgentMicroVMVsockCIDReleaseReceipt, error) {
	if allocator == nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("unavailable")
	}
	if err := ports.ValidateAgentMicroVMVsockCIDReleaseRequest(request); err != nil ||
		request.PoolRef != allocator.config.PoolRef {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("release_request_invalid")
	}
	requestDigest, _ := ports.AgentMicroVMVsockCIDReleaseRequestDigest(request)
	now := allocator.now()
	connection, err := allocator.beginImmediate(ctx)
	if err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, err
	}
	defer rollback(connection)
	if err := expireLeases(ctx, connection, request.PoolRef, now); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("store_unavailable")
	}
	if operation, found, err := readOperation(
		ctx, connection, request.PoolRef, request.IdempotencyKey,
	); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("store_unavailable")
	} else if found {
		if operation.Kind != "release" || operation.RequestDigest != requestDigest {
			return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("idempotency_conflict")
		}
		var receipt ports.AgentMicroVMVsockCIDReleaseReceipt
		if json.Unmarshal(operation.Result, &receipt) != nil ||
			ports.ValidateAgentMicroVMVsockCIDReleaseReceipt(request, receipt) != nil {
			return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("store_corrupt")
		}
		if err := commit(ctx, connection); err != nil {
			return ports.AgentMicroVMVsockCIDReleaseReceipt{}, err
		}
		return receipt, nil
	}

	record, err := readReservationByLease(ctx, connection, request.LeaseRef)
	if err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, mapReadError(err)
	}
	if err := validateCurrentOwner(record, request.PoolRef, request.ScopeDigest,
		request.OwnerRef, request.FencingToken); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, err
	}
	if record.State != reservationStateActive || !record.Lease.ExpiresAt.After(now) {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("lease_inactive")
	}
	if record.Lease.Revision != request.ExpectedRevision {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("stale_revision")
	}
	if _, err := connection.ExecContext(ctx, `
UPDATE agent_microvm_vsock_cid_reservations
SET state = ?, released_at_unix_nano = ?
WHERE lease_ref = ? AND state = ? AND revision = ?`,
		reservationStateReleased, now.UnixNano(), request.LeaseRef,
		reservationStateActive, request.ExpectedRevision,
	); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("store_unavailable")
	}
	receipt := ports.AgentMicroVMVsockCIDReleaseReceipt{
		PoolRef: request.PoolRef, ScopeDigest: request.ScopeDigest,
		LeaseRef: request.LeaseRef, OwnerRef: request.OwnerRef,
		GuestCID: record.Lease.GuestCID, FencingToken: request.FencingToken,
		FinalRevision: request.ExpectedRevision, ReleasedAt: now,
		IdempotencyKey: request.IdempotencyKey, AdapterRef: allocator.config.AdapterRef,
	}
	receipt.ReceiptRef = ports.AgentMicroVMVsockCIDReleaseReceiptRef(receipt)
	if err := ports.ValidateAgentMicroVMVsockCIDReleaseReceipt(request, receipt); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("release_receipt_invalid")
	}
	if err := insertOperation(ctx, connection, operationRecord{
		PoolRef: request.PoolRef, IdempotencyKey: request.IdempotencyKey,
		Kind: "release", RequestDigest: requestDigest, LeaseRef: request.LeaseRef,
		Result: mustJSON(receipt), CompletedAt: now,
	}); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, allocatorError("store_unavailable")
	}
	if err := commit(ctx, connection); err != nil {
		return ports.AgentMicroVMVsockCIDReleaseReceipt{}, err
	}
	return receipt, nil
}

func (allocator *Allocator) now() time.Time {
	return allocator.config.Now().UTC()
}

func (allocator *Allocator) beginImmediate(ctx context.Context) (*sql.Conn, error) {
	connection, err := allocator.config.DB.Conn(ctx)
	if err != nil {
		return nil, allocatorError("store_unavailable")
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		_ = connection.Close()
		if ctx.Err() != nil {
			return nil, allocatorError("canceled")
		}
		return nil, allocatorError("store_unavailable")
	}
	return connection, nil
}

func (allocator *Allocator) selectCID(requestDigest string, used map[uint32]struct{}) (uint32, bool) {
	size := uint64(allocator.config.MaximumGuestCID) -
		uint64(allocator.config.MinimumGuestCID) + 1
	digest := sha256.Sum256([]byte(requestDigest))
	offset := binary.BigEndian.Uint64(digest[:8]) % size
	for checked := uint64(0); checked < size; checked++ {
		candidate := uint32(uint64(allocator.config.MinimumGuestCID) + (offset+checked)%size)
		if _, reserved := used[candidate]; !reserved {
			return candidate, true
		}
	}
	return 0, false
}

type reservationRecord struct {
	Lease ports.AgentMicroVMVsockCIDLease
	State string
}

func readReservationByLease(
	ctx context.Context,
	connection *sql.Conn,
	leaseRef string,
) (reservationRecord, error) {
	var lease ports.AgentMicroVMVsockCIDLease
	var state string
	var acquiredAt, expiresAt int64
	err := connection.QueryRowContext(ctx, `
SELECT pool_ref, scope_digest, execution_ref, agent_ref, owner_ref, request_digest,
       acquire_idempotency_key, guest_cid, fencing_token, revision,
       acquired_at_unix_nano, expires_at_unix_nano, lease_ref, backend_ref,
       adapter_ref, receipt_ref, state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, leaseRef).Scan(
		&lease.PoolRef, &lease.ScopeDigest, &lease.ExecutionRef, &lease.AgentRef,
		&lease.OwnerRef, &lease.RequestDigest, &lease.IdempotencyKey,
		&lease.GuestCID, &lease.FencingToken, &lease.Revision,
		&acquiredAt, &expiresAt, &lease.LeaseRef, &lease.VsockBackendRef,
		&lease.AdapterRef, &lease.ReceiptRef, &state,
	)
	if err != nil {
		return reservationRecord{}, err
	}
	lease.AcquiredAt = time.Unix(0, acquiredAt).UTC()
	lease.ExpiresAt = time.Unix(0, expiresAt).UTC()
	if ports.ValidateAgentMicroVMVsockCIDLease(lease) != nil {
		return reservationRecord{}, allocatorError("store_corrupt")
	}
	return reservationRecord{Lease: lease, State: state}, nil
}

func validateCurrentOwner(
	record reservationRecord,
	poolRef string,
	scopeDigest string,
	ownerRef string,
	fencingToken uint64,
) error {
	switch {
	case record.Lease.PoolRef != poolRef || record.Lease.ScopeDigest != scopeDigest:
		return allocatorError("lease_scope_mismatch")
	case record.Lease.OwnerRef != ownerRef:
		return allocatorError("foreign_owner")
	case record.Lease.FencingToken != fencingToken:
		return allocatorError("stale_fencing_token")
	default:
		return nil
	}
}

func readActiveCIDs(
	ctx context.Context,
	connection *sql.Conn,
	poolRef string,
) (map[uint32]struct{}, error) {
	rows, err := connection.QueryContext(ctx, `
SELECT guest_cid
FROM agent_microvm_vsock_cid_reservations
WHERE pool_ref = ? AND state = ?`, poolRef, reservationStateActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	used := make(map[uint32]struct{})
	for rows.Next() {
		var cid uint32
		if err := rows.Scan(&cid); err != nil {
			return nil, err
		}
		used[cid] = struct{}{}
	}
	return used, rows.Err()
}

func nextFencingToken(
	ctx context.Context,
	connection *sql.Conn,
	poolRef string,
	guestCID uint32,
) (uint64, error) {
	var current int64
	err := connection.QueryRowContext(ctx, `
SELECT fencing_token
FROM agent_microvm_vsock_cid_generations
WHERE pool_ref = ? AND guest_cid = ?`, poolRef, guestCID).Scan(&current)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := connection.ExecContext(ctx, `
INSERT INTO agent_microvm_vsock_cid_generations (pool_ref, guest_cid, fencing_token)
VALUES (?, ?, 1)`, poolRef, guestCID); err != nil {
			return 0, allocatorError("store_unavailable")
		}
		return 1, nil
	case err != nil:
		return 0, allocatorError("store_unavailable")
	case current <= 0 || current == math.MaxInt64:
		return 0, allocatorError("fencing_exhausted")
	default:
		next := current + 1
		if _, err := connection.ExecContext(ctx, `
UPDATE agent_microvm_vsock_cid_generations
SET fencing_token = ?
WHERE pool_ref = ? AND guest_cid = ? AND fencing_token = ?`,
			next, poolRef, guestCID, current,
		); err != nil {
			return 0, allocatorError("store_unavailable")
		}
		return uint64(next), nil
	}
}

func insertReservation(
	ctx context.Context,
	connection *sql.Conn,
	lease ports.AgentMicroVMVsockCIDLease,
) error {
	_, err := connection.ExecContext(ctx, `
INSERT INTO agent_microvm_vsock_cid_reservations (
    lease_ref, pool_ref, scope_digest, execution_ref, agent_ref, owner_ref,
    request_digest, acquire_idempotency_key, guest_cid, fencing_token, revision,
    acquired_at_unix_nano, expires_at_unix_nano, backend_ref, adapter_ref,
    receipt_ref, state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lease.LeaseRef, lease.PoolRef, lease.ScopeDigest, lease.ExecutionRef,
		lease.AgentRef, lease.OwnerRef, lease.RequestDigest, lease.IdempotencyKey,
		lease.GuestCID, lease.FencingToken, lease.Revision,
		lease.AcquiredAt.UnixNano(), lease.ExpiresAt.UnixNano(),
		lease.VsockBackendRef, lease.AdapterRef, lease.ReceiptRef,
		reservationStateActive,
	)
	return err
}

func expireLeases(
	ctx context.Context,
	connection *sql.Conn,
	poolRef string,
	now time.Time,
) error {
	_, err := connection.ExecContext(ctx, `
UPDATE agent_microvm_vsock_cid_reservations
SET state = ?
WHERE pool_ref = ? AND state = ? AND expires_at_unix_nano <= ?`,
		reservationStateExpired, poolRef, reservationStateActive, now.UnixNano(),
	)
	return err
}

type operationRecord struct {
	PoolRef        string
	IdempotencyKey string
	Kind           string
	RequestDigest  string
	LeaseRef       string
	Result         []byte
	CompletedAt    time.Time
}

func readOperation(
	ctx context.Context,
	connection *sql.Conn,
	poolRef string,
	idempotencyKey string,
) (operationRecord, bool, error) {
	var operation operationRecord
	var completedAt int64
	err := connection.QueryRowContext(ctx, `
SELECT pool_ref, idempotency_key, operation_kind, request_digest, lease_ref,
       result_json, completed_at_unix_nano
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ?`,
		poolRef, idempotencyKey,
	).Scan(
		&operation.PoolRef, &operation.IdempotencyKey, &operation.Kind,
		&operation.RequestDigest, &operation.LeaseRef, &operation.Result, &completedAt,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return operationRecord{}, false, nil
	case err != nil:
		return operationRecord{}, false, err
	default:
		operation.CompletedAt = time.Unix(0, completedAt).UTC()
		return operation, true, nil
	}
}

func insertOperation(
	ctx context.Context,
	connection *sql.Conn,
	operation operationRecord,
) error {
	_, err := connection.ExecContext(ctx, `
INSERT INTO agent_microvm_vsock_cid_operations (
    pool_ref, idempotency_key, operation_kind, request_digest, lease_ref,
    result_json, completed_at_unix_nano
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		operation.PoolRef, operation.IdempotencyKey, operation.Kind,
		operation.RequestDigest, operation.LeaseRef, operation.Result,
		operation.CompletedAt.UnixNano(),
	)
	return err
}

func commit(ctx context.Context, connection *sql.Conn) error {
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		_ = connection.Close()
		if ctx.Err() != nil {
			return allocatorError("canceled")
		}
		return allocatorError("store_unavailable")
	}
	return connection.Close()
}

func rollback(connection *sql.Conn) {
	if connection == nil {
		return
	}
	_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
	_ = connection.Close()
}

func mapReadError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return allocatorError("lease_not_found")
	}
	var allocatorErr *Error
	if errors.As(err, &allocatorErr) {
		return err
	}
	return allocatorError("store_unavailable")
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		panic("vsock lease receipt cannot fail JSON encoding: " + err.Error())
	}
	return payload
}

func validRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

func allocatorError(suffix string) error {
	return &Error{Code: "agent_firecracker_vsock_cid." + suffix}
}

func verifySchema(ctx context.Context, database *sql.DB) error {
	for index, object := range schemaObjects {
		var definition string
		err := database.QueryRowContext(ctx, `
SELECT sql
FROM sqlite_master
WHERE type = ? AND name = ?`, object.Kind, object.Name).Scan(&definition)
		if err != nil {
			return allocatorError("schema_unavailable")
		}
		if normalizeSQL(definition) != normalizeSQL(schemaStatements[index]) {
			return allocatorError("schema_mismatch")
		}
	}
	return nil
}

func normalizeSQL(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}

var schemaStatements = []string{
	`CREATE TABLE agent_microvm_vsock_cid_generations (
    pool_ref TEXT NOT NULL,
    guest_cid INTEGER NOT NULL CHECK (guest_cid >= 3 AND guest_cid < 4294967295),
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    PRIMARY KEY (pool_ref, guest_cid)
) STRICT`,
	`CREATE TABLE agent_microvm_vsock_cid_reservations (
    lease_ref TEXT PRIMARY KEY,
    pool_ref TEXT NOT NULL,
    scope_digest TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    agent_ref TEXT NOT NULL,
    owner_ref TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    acquire_idempotency_key TEXT NOT NULL,
    guest_cid INTEGER NOT NULL CHECK (guest_cid >= 3 AND guest_cid < 4294967295),
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    revision INTEGER NOT NULL CHECK (revision > 0),
    acquired_at_unix_nano INTEGER NOT NULL,
    expires_at_unix_nano INTEGER NOT NULL,
    backend_ref TEXT NOT NULL UNIQUE,
    adapter_ref TEXT NOT NULL,
    receipt_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('active', 'expired', 'released')),
    released_at_unix_nano INTEGER,
    UNIQUE (pool_ref, acquire_idempotency_key),
    UNIQUE (pool_ref, guest_cid, fencing_token)
) STRICT`,
	`CREATE UNIQUE INDEX agent_microvm_vsock_cid_one_active
ON agent_microvm_vsock_cid_reservations (pool_ref, guest_cid)
WHERE state = 'active'`,
	`CREATE TABLE agent_microvm_vsock_cid_operations (
    pool_ref TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    operation_kind TEXT NOT NULL CHECK (operation_kind IN ('reserve', 'renew', 'release')),
    request_digest TEXT NOT NULL,
    lease_ref TEXT NOT NULL,
    result_json BLOB NOT NULL,
    completed_at_unix_nano INTEGER NOT NULL,
    PRIMARY KEY (pool_ref, idempotency_key),
    FOREIGN KEY (lease_ref) REFERENCES agent_microvm_vsock_cid_reservations (lease_ref)
) STRICT`,
}

var schemaObjects = []struct {
	Kind string
	Name string
}{
	{Kind: "table", Name: "agent_microvm_vsock_cid_generations"},
	{Kind: "table", Name: "agent_microvm_vsock_cid_reservations"},
	{Kind: "index", Name: "agent_microvm_vsock_cid_one_active"},
	{Kind: "table", Name: "agent_microvm_vsock_cid_operations"},
}
