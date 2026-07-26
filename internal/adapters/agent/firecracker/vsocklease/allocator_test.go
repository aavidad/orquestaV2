package vsocklease

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *testClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *testClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

func (clock *testClock) Set(now time.Time) {
	clock.mu.Lock()
	clock.now = now
	clock.mu.Unlock()
}

func openTestDatabase(t *testing.T, path string, installSchema bool) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=foreign_keys(1)",
		path,
	)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(32)
	if err := database.PingContext(context.Background()); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if installSchema {
		for _, statement := range SchemaStatements() {
			if _, err := database.ExecContext(context.Background(), statement); err != nil {
				_ = database.Close()
				t.Fatal(err)
			}
		}
	}
	return database
}

func testConfig(
	database *sql.DB,
	clock *testClock,
	minimumCID uint32,
	maximumCID uint32,
) Config {
	return Config{
		DB: database, AdapterRef: "adapter:firecracker-vsock-cid",
		PoolRef: "vsock-pool:host-128g-16", MinimumGuestCID: minimumCID,
		MaximumGuestCID: maximumCID, MinimumLeaseDuration: time.Second,
		MaximumLeaseDuration: time.Hour,
		Now:                  clock.Now,
	}
}

func testReservationRequest(
	t *testing.T,
	index int,
) ports.AgentMicroVMVsockCIDReservationRequest {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:firecracker-vsock")
	goalRef, _ := goal.NewGoalRef("goal:firecracker-vsock")
	workItemRef, _ := goal.NewWorkItemRef(fmt.Sprintf("work:firecracker-vsock:%d", index))
	executionRef, _ := goal.NewExecutionRef(fmt.Sprintf("execution:firecracker-vsock:%d", index))
	return ports.AgentMicroVMVsockCIDReservationRequest{
		PoolRef: "vsock-pool:host-128g-16",
		Scope: ports.AgentMicroVMNetworkScope{
			ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef,
			ExecutionRef: executionRef, PlanGeneration: 4, AppSpecGeneration: 3,
			ExecutionAttempt: 1, SpecHash: strings.Repeat("a", 64),
			AgentRef: fmt.Sprintf("agent:codex:%d", index),
		},
		OwnerRef:       fmt.Sprintf("launch-identity:firecracker-vsock:%d", index),
		LeaseDuration:  10 * time.Minute,
		IdempotencyKey: fmt.Sprintf("reserve-vsock-cid:%d", index),
	}
}

func openTestAllocator(
	t *testing.T,
	database *sql.DB,
	clock *testClock,
	minimumCID uint32,
	maximumCID uint32,
) *Allocator {
	t.Helper()
	allocator, err := Open(
		context.Background(),
		testConfig(database, clock, minimumCID, maximumCID),
	)
	if err != nil {
		t.Fatal(err)
	}
	return allocator
}

func TestAllocatorReservesSixteenUniqueCIDsAcrossConcurrentAdapters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-cids.db")
	firstDB := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = firstDB.Close() })
	secondDB := openTestDatabase(t, path, false)
	t.Cleanup(func() { _ = secondDB.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)}
	allocators := []*Allocator{
		openTestAllocator(t, firstDB, clock, 32, 47),
		openTestAllocator(t, secondDB, clock, 32, 47),
	}

	type result struct {
		lease ports.AgentMicroVMVsockCIDLease
		err   error
	}
	results := make(chan result, 16)
	requests := make([]ports.AgentMicroVMVsockCIDReservationRequest, 16)
	for index := range requests {
		requests[index] = testReservationRequest(t, index)
	}
	var group sync.WaitGroup
	for index := range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			lease, err := allocators[index%len(allocators)].Reserve(
				context.Background(), requests[index],
			)
			results <- result{lease: lease, err: err}
		}()
	}
	group.Wait()
	close(results)

	seen := make(map[uint32]string, 16)
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.lease.GuestCID < 3 || result.lease.GuestCID > 47 {
			t.Fatalf("CID escaped pool: %+v", result.lease)
		}
		if previous, duplicate := seen[result.lease.GuestCID]; duplicate {
			t.Fatalf("duplicate CID %d for %q and %q", result.lease.GuestCID, previous, result.lease.LeaseRef)
		}
		seen[result.lease.GuestCID] = result.lease.LeaseRef
	}
	if len(seen) != 16 {
		t.Fatalf("unique CIDs = %d, want 16", len(seen))
	}
	if _, err := allocators[0].Reserve(
		context.Background(), testReservationRequest(t, 16),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.capacity_reached" {
		t.Fatalf("seventeenth reservation = %v", err)
	}
}

func TestAllocatorReserveIsConcurrentAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-cids.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)}
	allocator := openTestAllocator(t, database, clock, 73, 73)
	request := testReservationRequest(t, 1)

	leases := make(chan ports.AgentMicroVMVsockCIDLease, 12)
	errorsFound := make(chan error, 12)
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			lease, err := allocator.Reserve(context.Background(), request)
			leases <- lease
			errorsFound <- err
		}()
	}
	group.Wait()
	close(leases)
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
	var expected ports.AgentMicroVMVsockCIDLease
	for lease := range leases {
		if expected.LeaseRef == "" {
			expected = lease
			continue
		}
		if lease != expected {
			t.Fatalf("idempotent receipt changed:\nfirst=%+v\nretry=%+v", expected, lease)
		}
	}
	var reservations int
	if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if reservations != 1 {
		t.Fatalf("reservations = %d, want 1", reservations)
	}

	conflict := request
	conflict.OwnerRef = "launch-identity:other"
	if _, err := allocator.Reserve(
		context.Background(), conflict,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.idempotency_conflict" {
		t.Fatalf("semantic idempotency conflict = %v", err)
	}
}

func TestAllocatorRestartRecoveryExpiryAndABAProtection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-cids.db")
	clock := &testClock{now: time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)}
	firstDB := openTestDatabase(t, path, true)
	first := openTestAllocator(t, firstDB, clock, 91, 91)
	firstRequest := testReservationRequest(t, 1)
	firstRequest.LeaseDuration = time.Minute
	oldLease, err := first.Reserve(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatal(err)
	}

	secondDB := openTestDatabase(t, path, false)
	t.Cleanup(func() { _ = secondDB.Close() })
	second := openTestAllocator(t, secondDB, clock, 91, 91)
	recovery := ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: oldLease.PoolRef, ScopeDigest: oldLease.ScopeDigest,
		LeaseRef: oldLease.LeaseRef, OwnerRef: oldLease.OwnerRef,
		FencingToken: oldLease.FencingToken,
	}
	recovered, err := second.Recover(context.Background(), recovery)
	if err != nil || recovered != oldLease {
		t.Fatalf("restart recovery = %+v, %v", recovered, err)
	}

	clock.Advance(time.Minute)
	if _, err := second.Recover(
		context.Background(), recovery,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_inactive" {
		t.Fatalf("inclusive expiry recovery = %v", err)
	}
	if _, err := second.Reserve(
		context.Background(), firstRequest,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.reservation_replay_denied" {
		t.Fatalf("expired acquire replay = %v", err)
	}

	newRequest := testReservationRequest(t, 2)
	newLease, err := second.Reserve(context.Background(), newRequest)
	if err != nil {
		t.Fatal(err)
	}
	if newLease.GuestCID != oldLease.GuestCID ||
		newLease.FencingToken <= oldLease.FencingToken ||
		newLease.LeaseRef == oldLease.LeaseRef {
		t.Fatalf("ABA fence not advanced:\nold=%+v\nnew=%+v", oldLease, newLease)
	}
	oldRelease := ports.AgentMicroVMVsockCIDReleaseRequest{
		PoolRef: oldLease.PoolRef, ScopeDigest: oldLease.ScopeDigest,
		LeaseRef: oldLease.LeaseRef, OwnerRef: oldLease.OwnerRef,
		FencingToken: oldLease.FencingToken, ExpectedRevision: oldLease.Revision,
		IdempotencyKey: "release-old-expired-lease",
	}
	if _, err := second.Release(
		context.Background(), oldRelease,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_inactive" {
		t.Fatalf("old lease release = %v", err)
	}
	if _, err := second.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: newLease.PoolRef, ScopeDigest: newLease.ScopeDigest,
		LeaseRef: newLease.LeaseRef, OwnerRef: newLease.OwnerRef,
		FencingToken: newLease.FencingToken,
	}); err != nil {
		t.Fatalf("new owner lost lease after ABA attempt: %v", err)
	}
}

func TestAllocatorFailsClosedOnDurableClockRegressionAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-clock.db")
	start := time.Date(2026, 7, 26, 11, 30, 0, 0, time.UTC)
	clock := &testClock{now: start}
	firstDB := openTestDatabase(t, path, true)
	first := openTestAllocator(t, firstDB, clock, 92, 92)
	request := testReservationRequest(t, 1)
	request.LeaseDuration = time.Minute
	lease, err := first.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Minute)
	recovery := ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken,
	}
	if _, err := first.Recover(
		context.Background(), recovery,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_inactive" {
		t.Fatalf("expiry observation = %v", err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatal(err)
	}

	clock.Set(start.Add(30 * time.Second))
	secondDB := openTestDatabase(t, path, false)
	t.Cleanup(func() { _ = secondDB.Close() })
	second := openTestAllocator(t, secondDB, clock, 92, 92)
	regressedOperations := map[string]func() error{
		"reserve": func() error {
			_, err := second.Reserve(context.Background(), testReservationRequest(t, 2))
			return err
		},
		"renew": func() error {
			_, err := second.Renew(context.Background(), ports.AgentMicroVMVsockCIDRenewalRequest{
				PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
				LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
				FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
				LeaseDuration: 2 * time.Minute, IdempotencyKey: "renew-regressed-clock",
			})
			return err
		},
		"recover": func() error {
			_, err := second.Recover(context.Background(), recovery)
			return err
		},
		"release": func() error {
			_, err := second.Release(context.Background(), ports.AgentMicroVMVsockCIDReleaseRequest{
				PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
				LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
				FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
				IdempotencyKey: "release-regressed-clock",
			})
			return err
		},
	}
	for name, operation := range regressedOperations {
		t.Run(name, func(t *testing.T) {
			if err := operation(); ErrorCode(err) !=
				"agent_firecracker_vsock_cid.clock_regressed" {
				t.Fatalf("regressed clock = %v", err)
			}
		})
	}

	var highWater int64
	if err := secondDB.QueryRow(`
SELECT high_water_unix_nano
FROM agent_microvm_vsock_cid_clock
WHERE pool_ref = ?`, lease.PoolRef).Scan(&highWater); err != nil {
		t.Fatal(err)
	}
	if want := start.Add(2 * time.Minute).UnixNano(); highWater != want {
		t.Fatalf("high-water = %d, want %d", highWater, want)
	}
	var state string
	if err := secondDB.QueryRow(`
SELECT state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != reservationStateExpired {
		t.Fatalf("expired tombstone rolled back: state=%q", state)
	}

	clock.Set(start.Add(2 * time.Minute))
	replacement, err := second.Reserve(
		context.Background(), testReservationRequest(t, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.GuestCID != lease.GuestCID ||
		replacement.FencingToken <= lease.FencingToken {
		t.Fatalf("replacement lost ABA fence: old=%+v new=%+v", lease, replacement)
	}
}

func TestAllocatorEnforcesMinimumLeaseDurationAtReserveAndRenew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-minimum-lease.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 11, 45, 0, 0, time.UTC)}
	config := testConfig(database, clock, 93, 93)
	config.MinimumLeaseDuration = 5 * time.Second
	allocator, err := Open(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	tooShort := testReservationRequest(t, 1)
	tooShort.LeaseDuration = config.MinimumLeaseDuration - time.Nanosecond
	if _, err := allocator.Reserve(
		context.Background(), tooShort,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.reservation_request_invalid" {
		t.Fatalf("short reservation = %v", err)
	}

	valid := testReservationRequest(t, 2)
	valid.LeaseDuration = time.Minute
	lease, err := allocator.Reserve(context.Background(), valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := allocator.Renew(context.Background(), ports.AgentMicroVMVsockCIDRenewalRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
		LeaseDuration:  config.MinimumLeaseDuration - time.Nanosecond,
		IdempotencyKey: "renew-too-short",
	}); ErrorCode(err) != "agent_firecracker_vsock_cid.renewal_request_invalid" {
		t.Fatalf("short renewal = %v", err)
	}

	invalidBounds := config
	invalidBounds.MaximumLeaseDuration = invalidBounds.MinimumLeaseDuration - time.Nanosecond
	if _, err := Open(
		context.Background(), invalidBounds,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.config_invalid" {
		t.Fatalf("inverted lease bounds = %v", err)
	}
}

func TestAllocatorRejectsUnrepresentableLeaseTimes(t *testing.T) {
	t.Run("reserve", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-reserve-overflow.db")
		database := openTestDatabase(t, path, true)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Unix(0, math.MaxInt64-int64(500*time.Millisecond)).UTC()}
		allocator := openTestAllocator(t, database, clock, 94, 94)
		request := testReservationRequest(t, 1)
		request.LeaseDuration = time.Second
		if _, err := allocator.Reserve(
			context.Background(), request,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_time_unrepresentable" {
			t.Fatalf("overflowing reservation = %v", err)
		}
		var reservations int
		if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_reservations`).Scan(&reservations); err != nil {
			t.Fatal(err)
		}
		if reservations != 0 {
			t.Fatalf("overflowing reservation persisted %d rows", reservations)
		}
	})

	t.Run("renew and schema order", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-renew-overflow.db")
		database := openTestDatabase(t, path, true)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Unix(0, math.MaxInt64-int64(5*time.Second)).UTC()}
		allocator := openTestAllocator(t, database, clock, 95, 95)
		request := testReservationRequest(t, 1)
		request.LeaseDuration = 2 * time.Second
		lease, err := allocator.Reserve(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		clock.Set(time.Unix(0, math.MaxInt64-int64(4*time.Second)).UTC())
		renewal := ports.AgentMicroVMVsockCIDRenewalRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
			LeaseDuration: 3 * time.Second, IdempotencyKey: "renew-near-time-ceiling",
		}
		renewed, err := allocator.Renew(context.Background(), renewal)
		if err != nil {
			t.Fatal(err)
		}
		if renewed.ExpiresAt.UnixNano() != math.MaxInt64-int64(time.Second) {
			t.Fatalf("renewal expiry = %v", renewed.ExpiresAt)
		}

		clock.Set(time.Unix(0, math.MaxInt64-int64(2*time.Second)).UTC())
		renewal.ExpectedRevision = renewed.Revision
		renewal.LeaseDuration = 3 * time.Second
		renewal.IdempotencyKey = "renew-over-time-ceiling"
		if _, err := allocator.Renew(
			context.Background(), renewal,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_time_unrepresentable" {
			t.Fatalf("overflowing renewal = %v", err)
		}
		if _, err := database.Exec(`
UPDATE agent_microvm_vsock_cid_reservations
SET expires_at_unix_nano = acquired_at_unix_nano
WHERE lease_ref = ?`, lease.LeaseRef); err == nil {
			t.Fatal("schema accepted non-increasing lease timestamps")
		}
		if _, err := allocator.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
			PoolRef: renewed.PoolRef, ScopeDigest: renewed.ScopeDigest,
			LeaseRef: renewed.LeaseRef, OwnerRef: renewed.OwnerRef,
			FencingToken: renewed.FencingToken,
		}); err != nil {
			t.Fatalf("overflow rejection changed active lease: %v", err)
		}
	})
}

func TestAllocatorRejectsClockOutsideUnixNanoRange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-clock-outside-range.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })

	var outsideRange time.Time
	for year := 2263; year <= 9999; year++ {
		candidate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		unixNano := candidate.UnixNano()
		if unixNano > 0 && !time.Unix(0, unixNano).UTC().Equal(candidate) {
			outsideRange = candidate
			break
		}
	}
	if outsideRange.IsZero() {
		t.Fatal("test fixture did not find a positive wrapped UnixNano value")
	}
	clock := &testClock{now: outsideRange}
	allocator := openTestAllocator(t, database, clock, 96, 96)

	if _, err := allocator.Reserve(
		context.Background(), testReservationRequest(t, 1),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.clock_invalid" {
		t.Fatalf("out-of-range clock = %v", err)
	}
	for _, table := range []string{
		"agent_microvm_vsock_cid_clock",
		"agent_microvm_vsock_cid_reservations",
		"agent_microvm_vsock_cid_operations",
	} {
		var rows int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if rows != 0 {
			t.Fatalf("out-of-range clock persisted %d rows in %s", rows, table)
		}
	}
}

func TestAllocatorCommitsExpiryAndClockWhenFencingIsExhausted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-fencing-exhausted.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	start := time.Date(2026, 7, 26, 15, 30, 0, 0, time.UTC)
	clock := &testClock{now: start}
	allocator := openTestAllocator(t, database, clock, 136, 136)
	request := testReservationRequest(t, 1)
	request.LeaseDuration = time.Minute
	lease, err := allocator.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
UPDATE agent_microvm_vsock_cid_generations
SET fencing_token = ?
WHERE pool_ref = ? AND guest_cid = ?`,
		int64(math.MaxInt64), lease.PoolRef, lease.GuestCID,
	); err != nil {
		t.Fatal(err)
	}

	clock.Advance(2 * time.Minute)
	if _, err := allocator.Reserve(
		context.Background(), testReservationRequest(t, 2),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.fencing_exhausted" {
		t.Fatalf("exhausted fencing token = %v", err)
	}

	var state string
	if err := database.QueryRow(`
SELECT state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != reservationStateExpired {
		t.Fatalf("expiry rolled back after fencing exhaustion: state=%q", state)
	}
	var highWater int64
	if err := database.QueryRow(`
SELECT high_water_unix_nano
FROM agent_microvm_vsock_cid_clock
WHERE pool_ref = ?`, lease.PoolRef).Scan(&highWater); err != nil {
		t.Fatal(err)
	}
	if want := start.Add(2 * time.Minute).UnixNano(); highWater != want {
		t.Fatalf("high-water = %d, want %d", highWater, want)
	}

	clock.Set(start.Add(30 * time.Second))
	if _, err := allocator.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken,
	}); ErrorCode(err) != "agent_firecracker_vsock_cid.clock_regressed" {
		t.Fatalf("regressed clock after fencing exhaustion = %v", err)
	}
}

func TestAllocatorPreservesTemporalFrontierAfterDownstreamTechnicalFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-downstream-failure.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	start := time.Date(2026, 7, 26, 15, 45, 0, 0, time.UTC)
	clock := &testClock{now: start}
	allocator := openTestAllocator(t, database, clock, 138, 138)
	request := testReservationRequest(t, 1)
	request.LeaseDuration = time.Minute
	lease, err := allocator.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
CREATE TRIGGER agent_microvm_vsock_cid_fail_generation_update
BEFORE UPDATE OF fencing_token ON agent_microvm_vsock_cid_generations
BEGIN
    SELECT RAISE(ABORT, 'injected downstream generation failure');
END`); err != nil {
		t.Fatal(err)
	}

	clock.Set(start.Add(2 * time.Minute))
	if _, err := allocator.Reserve(
		context.Background(), testReservationRequest(t, 2),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
		t.Fatalf("downstream technical failure = %v", err)
	}

	var state string
	if err := database.QueryRow(`
SELECT state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != reservationStateExpired {
		t.Fatalf("technical failure revived expired lease: state=%q", state)
	}
	var highWater int64
	if err := database.QueryRow(`
SELECT high_water_unix_nano
FROM agent_microvm_vsock_cid_clock
WHERE pool_ref = ?`, lease.PoolRef).Scan(&highWater); err != nil {
		t.Fatal(err)
	}
	if want := start.Add(2 * time.Minute).UnixNano(); highWater != want {
		t.Fatalf("high-water = %d, want %d", highWater, want)
	}
	var reservations, operations int
	if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if reservations != 1 || operations != 1 {
		t.Fatalf("partial business survived: reservations=%d operations=%d", reservations, operations)
	}

	clock.Set(start.Add(time.Minute))
	if _, err := allocator.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken,
	}); ErrorCode(err) != "agent_firecracker_vsock_cid.clock_regressed" {
		t.Fatalf("T1 recovery after T2 technical failure = %v", err)
	}
}

func TestAllocatorRetriesLostRollbackToResponseWithoutCommittingBusiness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-lost-rollback-to-response.db")
	database := openTestDatabase(t, path, true)
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	start := time.Date(2026, 7, 26, 15, 45, 30, 0, time.UTC)
	clock := &testClock{now: start}
	allocator := openTestAllocator(t, database, clock, 145, 145)
	request := testReservationRequest(t, 1)
	request.LeaseDuration = time.Minute
	lease, err := allocator.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
CREATE TRIGGER agent_microvm_vsock_cid_fail_second_operation
BEFORE INSERT ON agent_microvm_vsock_cid_operations
WHEN NEW.idempotency_key = 'reserve-vsock-cid:2'
BEGIN
    SELECT RAISE(ABORT, 'injected operation failure after reservation insert');
END`); err != nil {
		t.Fatal(err)
	}
	originalExec := allocator.execTransaction
	var rollbackToAttempts int
	allocator.execTransaction = func(
		ctx context.Context,
		connection *sql.Conn,
		statement string,
	) (sql.Result, error) {
		if statement == businessSavepointRollback {
			rollbackToAttempts++
			result, err := originalExec(ctx, connection, statement)
			if err != nil {
				return result, err
			}
			if rollbackToAttempts == 1 {
				return result, errors.New("injected lost ROLLBACK TO response")
			}
			return result, nil
		}
		return originalExec(ctx, connection, statement)
	}

	clock.Set(start.Add(2 * time.Minute))
	if _, err := allocator.Reserve(
		context.Background(), testReservationRequest(t, 2),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
		t.Fatalf("downstream failure with lost ROLLBACK TO response = %v", err)
	}
	if rollbackToAttempts != businessRollbackToMaximumAttempts {
		t.Fatalf("ROLLBACK TO attempts = %d, want %d",
			rollbackToAttempts, businessRollbackToMaximumAttempts)
	}
	allocator.execTransaction = originalExec

	var reservations, operations int
	if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	var fencingToken int64
	if err := database.QueryRow(`
SELECT fencing_token
FROM agent_microvm_vsock_cid_generations
WHERE pool_ref = ? AND guest_cid = ?`,
		lease.PoolRef, lease.GuestCID,
	).Scan(&fencingToken); err != nil {
		t.Fatal(err)
	}
	if reservations != 1 || operations != 1 || fencingToken != int64(lease.FencingToken) {
		t.Fatalf("business survived lost rollback response: reservations=%d operations=%d fencing=%d",
			reservations, operations, fencingToken)
	}
	assertTemporalFrontierBlocksT1(t, database, allocator, clock, lease,
		start.Add(2*time.Minute), reservationStateExpired)
}

func TestAllocatorPreservesTemporalFrontierBeforeBusinessSavepoint(t *testing.T) {
	t.Run("expiry update fails after clock advance", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-expiry-failure.db")
		database := openTestDatabase(t, path, true)
		t.Cleanup(func() { _ = database.Close() })
		start := time.Date(2026, 7, 26, 15, 46, 0, 0, time.UTC)
		clock := &testClock{now: start}
		allocator := openTestAllocator(t, database, clock, 143, 143)
		request := testReservationRequest(t, 1)
		request.LeaseDuration = time.Minute
		lease, err := allocator.Reserve(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`
CREATE TRIGGER agent_microvm_vsock_cid_fail_expiry
BEFORE UPDATE OF state ON agent_microvm_vsock_cid_reservations
WHEN NEW.state = 'expired'
BEGIN
    SELECT RAISE(ABORT, 'injected expiry failure');
END`); err != nil {
			t.Fatal(err)
		}

		clock.Set(start.Add(2 * time.Minute))
		if _, err := allocator.Recover(
			context.Background(), recoveryRequest(lease),
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("expiry failure = %v", err)
		}
		assertTemporalFrontierBlocksT1(t, database, allocator, clock, lease,
			start.Add(2*time.Minute), reservationStateActive)
	})

	t.Run("savepoint response is lost after creation", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-savepoint-response-lost.db")
		database := openTestDatabase(t, path, true)
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		start := time.Date(2026, 7, 26, 15, 47, 0, 0, time.UTC)
		clock := &testClock{now: start}
		allocator := openTestAllocator(t, database, clock, 144, 144)
		request := testReservationRequest(t, 1)
		request.LeaseDuration = time.Minute
		lease, err := allocator.Reserve(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		originalExec := allocator.execTransaction
		var injected bool
		allocator.execTransaction = func(
			ctx context.Context,
			connection *sql.Conn,
			statement string,
		) (sql.Result, error) {
			if statement == businessSavepointCreate && !injected {
				injected = true
				result, err := originalExec(ctx, connection, statement)
				if err != nil {
					return result, err
				}
				return result, errors.New("injected lost savepoint response")
			}
			return originalExec(ctx, connection, statement)
		}

		clock.Set(start.Add(2 * time.Minute))
		if _, err := allocator.Recover(
			context.Background(), recoveryRequest(lease),
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("lost savepoint response = %v", err)
		}
		if !injected {
			t.Fatal("savepoint failpoint was not reached")
		}
		allocator.execTransaction = originalExec
		assertTemporalFrontierBlocksT1(t, database, allocator, clock, lease,
			start.Add(2*time.Minute), reservationStateExpired)
	})
}

func assertTemporalFrontierBlocksT1(
	t *testing.T,
	database *sql.DB,
	allocator *Allocator,
	clock *testClock,
	lease ports.AgentMicroVMVsockCIDLease,
	highWaterTime time.Time,
	wantState string,
) {
	t.Helper()
	var state string
	if err := database.QueryRow(`
SELECT state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != wantState {
		t.Fatalf("reservation state = %q, want %q", state, wantState)
	}
	var highWater int64
	if err := database.QueryRow(`
SELECT high_water_unix_nano
FROM agent_microvm_vsock_cid_clock
WHERE pool_ref = ?`, lease.PoolRef).Scan(&highWater); err != nil {
		t.Fatal(err)
	}
	if want := highWaterTime.UnixNano(); highWater != want {
		t.Fatalf("high-water = %d, want %d", highWater, want)
	}
	clock.Set(lease.AcquiredAt.Add(time.Minute))
	if _, err := allocator.Recover(
		context.Background(), recoveryRequest(lease),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.clock_regressed" {
		t.Fatalf("T1 recovery after T2 failure = %v", err)
	}
}

func recoveryRequest(
	lease ports.AgentMicroVMVsockCIDLease,
) ports.AgentMicroVMVsockCIDRecoveryRequest {
	return ports.AgentMicroVMVsockCIDRecoveryRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken,
	}
}

func TestAllocatorRequiresExactlyOneRenewOrReleaseWrite(t *testing.T) {
	t.Run("renew", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-renew-write-skipped.db")
		database := openTestDatabase(t, path, true)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Date(2026, 7, 26, 15, 50, 0, 0, time.UTC)}
		allocator := openTestAllocator(t, database, clock, 141, 141)
		lease, err := allocator.Reserve(
			context.Background(), testReservationRequest(t, 1),
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`
CREATE TRIGGER agent_microvm_vsock_cid_skip_renew
BEFORE UPDATE OF revision ON agent_microvm_vsock_cid_reservations
WHEN NEW.revision > OLD.revision
BEGIN
    SELECT RAISE(IGNORE);
END`); err != nil {
			t.Fatal(err)
		}
		renewal := ports.AgentMicroVMVsockCIDRenewalRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
			LeaseDuration: 20 * time.Minute, IdempotencyKey: "renew-write-skipped",
		}
		if _, err := allocator.Renew(
			context.Background(), renewal,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("skipped renewal write = %v", err)
		}
		var revision uint64
		var operationCount int
		if err := database.QueryRow(`
SELECT revision
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&revision); err != nil {
			t.Fatal(err)
		}
		if err := database.QueryRow(`
SELECT COUNT(*)
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ?`,
			renewal.PoolRef, renewal.IdempotencyKey,
		).Scan(&operationCount); err != nil {
			t.Fatal(err)
		}
		if revision != lease.Revision || operationCount != 0 {
			t.Fatalf("skipped renewal left partial effects: revision=%d operations=%d",
				revision, operationCount)
		}
	})

	t.Run("release", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-release-write-skipped.db")
		database := openTestDatabase(t, path, true)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Date(2026, 7, 26, 15, 55, 0, 0, time.UTC)}
		allocator := openTestAllocator(t, database, clock, 142, 142)
		lease, err := allocator.Reserve(
			context.Background(), testReservationRequest(t, 1),
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`
CREATE TRIGGER agent_microvm_vsock_cid_skip_release
BEFORE UPDATE OF state ON agent_microvm_vsock_cid_reservations
WHEN NEW.state = 'released'
BEGIN
    SELECT RAISE(IGNORE);
END`); err != nil {
			t.Fatal(err)
		}
		release := ports.AgentMicroVMVsockCIDReleaseRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
			IdempotencyKey: "release-write-skipped",
		}
		if _, err := allocator.Release(
			context.Background(), release,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("skipped release write = %v", err)
		}
		var state string
		var operationCount int
		if err := database.QueryRow(`
SELECT state
FROM agent_microvm_vsock_cid_reservations
WHERE lease_ref = ?`, lease.LeaseRef).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if err := database.QueryRow(`
SELECT COUNT(*)
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ?`,
			release.PoolRef, release.IdempotencyKey,
		).Scan(&operationCount); err != nil {
			t.Fatal(err)
		}
		if state != reservationStateActive || operationCount != 0 {
			t.Fatalf("skipped release left partial effects: state=%q operations=%d",
				state, operationCount)
		}
	})
}

func TestAllocatorDiscardsConnectionWhenBeginResponseIsLost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-lost-begin-response.db")
	database := openTestDatabase(t, path, true)
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 16, 10, 0, 0, time.UTC)}
	allocator := openTestAllocator(t, database, clock, 146, 146)
	originalExec := allocator.execTransaction
	var injected bool
	allocator.execTransaction = func(
		ctx context.Context,
		connection *sql.Conn,
		statement string,
	) (sql.Result, error) {
		if statement == "BEGIN IMMEDIATE" && !injected {
			injected = true
			result, err := originalExec(ctx, connection, statement)
			if err != nil {
				return result, err
			}
			return result, errors.New("injected lost BEGIN response")
		}
		return originalExec(ctx, connection, statement)
	}
	request := testReservationRequest(t, 1)
	if _, err := allocator.Reserve(
		context.Background(), request,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
		t.Fatalf("lost BEGIN response = %v", err)
	}
	if !injected {
		t.Fatal("BEGIN failpoint was not reached")
	}
	allocator.execTransaction = originalExec

	var reservations, operations, clocks int
	for table, destination := range map[string]*int{
		"agent_microvm_vsock_cid_reservations": &reservations,
		"agent_microvm_vsock_cid_operations":   &operations,
		"agent_microvm_vsock_cid_clock":        &clocks,
	} {
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	if reservations != 0 || operations != 0 || clocks != 0 {
		t.Fatalf("lost BEGIN response left effects: reservations=%d operations=%d clocks=%d",
			reservations, operations, clocks)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := allocator.Reserve(ctx, request); err != nil {
		t.Fatalf("discarded BEGIN connection left lock or transaction alive: %v", err)
	}
}

func TestAllocatorDiscardsConnectionsWhenTransactionCleanupFails(t *testing.T) {
	t.Run("commit failure rolls back", func(t *testing.T) {
		testCommitFailureLeavesNoAmbiguousEffects(t, false)
	})
	t.Run("commit and rollback failure discard physical connection", func(t *testing.T) {
		testCommitFailureLeavesNoAmbiguousEffects(t, true)
	})
	t.Run("lost commit response replays durable receipt", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-lost-commit-response.db")
		database := openTestDatabase(t, path, true)
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Date(2026, 7, 26, 16, 15, 0, 0, time.UTC)}
		allocator := openTestAllocator(t, database, clock, 137, 137)
		originalExec := allocator.execTransaction
		var commitInjected bool
		var rollbackAttempts int
		allocator.execTransaction = func(
			ctx context.Context,
			connection *sql.Conn,
			statement string,
		) (sql.Result, error) {
			if statement == "ROLLBACK" {
				rollbackAttempts++
			}
			if statement == "COMMIT" && !commitInjected {
				commitInjected = true
				result, err := originalExec(ctx, connection, statement)
				if err != nil {
					return result, err
				}
				return result, errors.New("injected lost commit response")
			}
			return originalExec(ctx, connection, statement)
		}
		request := testReservationRequest(t, 1)
		if _, err := allocator.Reserve(
			context.Background(), request,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("lost commit response = %v", err)
		}
		if rollbackAttempts != 1 {
			t.Fatalf("rollback attempts after completed commit = %d, want 1 compensating attempt", rollbackAttempts)
		}
		allocator.execTransaction = originalExec

		replayed, err := allocator.Reserve(context.Background(), request)
		if err != nil {
			t.Fatalf("idempotent replay after lost commit response = %v", err)
		}
		var reservations, operations int
		if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_reservations`).Scan(&reservations); err != nil {
			t.Fatal(err)
		}
		if err := database.QueryRow(`
SELECT COUNT(*) FROM agent_microvm_vsock_cid_operations`).Scan(&operations); err != nil {
			t.Fatal(err)
		}
		if reservations != 1 || operations != 1 {
			t.Fatalf("replay duplicated effects: reservations=%d operations=%d", reservations, operations)
		}
		var persistedLeaseRef string
		if err := database.QueryRow(`
SELECT lease_ref
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ?`,
			request.PoolRef, request.IdempotencyKey,
		).Scan(&persistedLeaseRef); err != nil {
			t.Fatal(err)
		}
		if replayed.LeaseRef != persistedLeaseRef {
			t.Fatalf("replayed lease %q != durable lease %q", replayed.LeaseRef, persistedLeaseRef)
		}
	})
	t.Run("lost renewal commit response replays exact durable receipt", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-lost-renew-commit-response.db")
		database := openTestDatabase(t, path, true)
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Date(2026, 7, 26, 16, 20, 0, 0, time.UTC)}
		allocator := openTestAllocator(t, database, clock, 139, 139)
		lease, err := allocator.Reserve(
			context.Background(), testReservationRequest(t, 1),
		)
		if err != nil {
			t.Fatal(err)
		}
		clock.Advance(time.Minute)
		renewal := ports.AgentMicroVMVsockCIDRenewalRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
			LeaseDuration: 20 * time.Minute, IdempotencyKey: "renew-lost-commit-response",
		}
		restore := injectLostCommitResponse(t, allocator)
		if _, err := allocator.Renew(
			context.Background(), renewal,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("lost renewal commit response = %v", err)
		}
		restore()

		replayed, err := allocator.Renew(context.Background(), renewal)
		if err != nil {
			t.Fatalf("renewal replay after lost commit response = %v", err)
		}
		var resultJSON []byte
		var operationCount int
		if err := database.QueryRow(`
SELECT result_json
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ? AND operation_kind = 'renew'`,
			renewal.PoolRef, renewal.IdempotencyKey,
		).Scan(&resultJSON); err != nil {
			t.Fatal(err)
		}
		var persisted ports.AgentMicroVMVsockCIDLease
		if err := json.Unmarshal(resultJSON, &persisted); err != nil {
			t.Fatal(err)
		}
		if replayed != persisted {
			t.Fatalf("renewal replay changed receipt:\nreplayed=%+v\npersisted=%+v", replayed, persisted)
		}
		if err := database.QueryRow(`
SELECT COUNT(*)
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ? AND operation_kind = 'renew'`,
			renewal.PoolRef, renewal.IdempotencyKey,
		).Scan(&operationCount); err != nil {
			t.Fatal(err)
		}
		if operationCount != 1 {
			t.Fatalf("renewal operations = %d, want 1", operationCount)
		}
	})
	t.Run("lost release commit response replays exact durable receipt", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-lost-release-commit-response.db")
		database := openTestDatabase(t, path, true)
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		clock := &testClock{now: time.Date(2026, 7, 26, 16, 25, 0, 0, time.UTC)}
		allocator := openTestAllocator(t, database, clock, 140, 140)
		lease, err := allocator.Reserve(
			context.Background(), testReservationRequest(t, 1),
		)
		if err != nil {
			t.Fatal(err)
		}
		clock.Advance(time.Minute)
		release := ports.AgentMicroVMVsockCIDReleaseRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
			IdempotencyKey: "release-lost-commit-response",
		}
		restore := injectLostCommitResponse(t, allocator)
		if _, err := allocator.Release(
			context.Background(), release,
		); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("lost release commit response = %v", err)
		}
		restore()

		replayed, err := allocator.Release(context.Background(), release)
		if err != nil {
			t.Fatalf("release replay after lost commit response = %v", err)
		}
		var resultJSON []byte
		var operationCount int
		if err := database.QueryRow(`
SELECT result_json
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ? AND operation_kind = 'release'`,
			release.PoolRef, release.IdempotencyKey,
		).Scan(&resultJSON); err != nil {
			t.Fatal(err)
		}
		var persisted ports.AgentMicroVMVsockCIDReleaseReceipt
		if err := json.Unmarshal(resultJSON, &persisted); err != nil {
			t.Fatal(err)
		}
		if replayed != persisted {
			t.Fatalf("release replay changed receipt:\nreplayed=%+v\npersisted=%+v", replayed, persisted)
		}
		if err := database.QueryRow(`
SELECT COUNT(*)
FROM agent_microvm_vsock_cid_operations
WHERE pool_ref = ? AND idempotency_key = ? AND operation_kind = 'release'`,
			release.PoolRef, release.IdempotencyKey,
		).Scan(&operationCount); err != nil {
			t.Fatal(err)
		}
		if operationCount != 1 {
			t.Fatalf("release operations = %d, want 1", operationCount)
		}
	})
	t.Run("temporal-only commit and rollback failure discard physical connection", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-rollback-failure.db")
		database := openTestDatabase(t, path, true)
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		start := time.Date(2026, 7, 26, 16, 0, 0, 0, time.UTC)
		clock := &testClock{now: start}
		allocator := openTestAllocator(t, database, clock, 132, 133)
		request := testReservationRequest(t, 1)
		request.LeaseDuration = time.Minute
		lease, err := allocator.Reserve(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		clock.Advance(2 * time.Minute)
		if _, err := allocator.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken,
		}); ErrorCode(err) != "agent_firecracker_vsock_cid.lease_inactive" {
			t.Fatalf("persist expiry = %v", err)
		}

		clock.Set(start.Add(30 * time.Second))
		originalExec := allocator.execTransaction
		allocator.execTransaction = func(
			ctx context.Context,
			connection *sql.Conn,
			statement string,
		) (sql.Result, error) {
			if statement == "COMMIT" || statement == "ROLLBACK" {
				return nil, errors.New("injected temporal cleanup failure")
			}
			return originalExec(ctx, connection, statement)
		}
		if _, err := allocator.Recover(context.Background(), ports.AgentMicroVMVsockCIDRecoveryRequest{
			PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
			LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
			FencingToken: lease.FencingToken,
		}); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
			t.Fatalf("clock regression with temporal cleanup failure = %v", err)
		}
		allocator.execTransaction = originalExec
		clock.Set(start.Add(2 * time.Minute))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := allocator.Reserve(ctx, testReservationRequest(t, 2)); err != nil {
			t.Fatalf("discarded transaction left database blocked: %v", err)
		}
	})
}

func testCommitFailureLeavesNoAmbiguousEffects(t *testing.T, failRollback bool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vsock-commit-failure.db")
	database := openTestDatabase(t, path, true)
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 16, 30, 0, 0, time.UTC)}
	allocator := openTestAllocator(t, database, clock, 134, 134)
	originalExec := allocator.execTransaction
	allocator.execTransaction = func(
		ctx context.Context,
		connection *sql.Conn,
		statement string,
	) (sql.Result, error) {
		if statement == "COMMIT" || failRollback && statement == "ROLLBACK" {
			return nil, errors.New("injected transaction failure")
		}
		return originalExec(ctx, connection, statement)
	}
	request := testReservationRequest(t, 1)
	if _, err := allocator.Reserve(
		context.Background(), request,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.store_unavailable" {
		t.Fatalf("injected commit failure = %v", err)
	}
	allocator.execTransaction = originalExec

	var reservations, operations, clocks int
	for table, destination := range map[string]*int{
		"agent_microvm_vsock_cid_reservations": &reservations,
		"agent_microvm_vsock_cid_operations":   &operations,
		"agent_microvm_vsock_cid_clock":        &clocks,
	} {
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	if reservations != 0 || operations != 0 || clocks != 0 {
		t.Fatalf("ambiguous effects after commit failure: reservations=%d operations=%d clocks=%d",
			reservations, operations, clocks)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := allocator.Reserve(ctx, request); err != nil {
		t.Fatalf("retry after transaction cleanup = %v", err)
	}
}

func injectLostCommitResponse(t *testing.T, allocator *Allocator) func() {
	t.Helper()
	originalExec := allocator.execTransaction
	var injected bool
	allocator.execTransaction = func(
		ctx context.Context,
		connection *sql.Conn,
		statement string,
	) (sql.Result, error) {
		if statement == "COMMIT" && !injected {
			injected = true
			result, err := originalExec(ctx, connection, statement)
			if err != nil {
				return result, err
			}
			return result, errors.New("injected lost commit response")
		}
		return originalExec(ctx, connection, statement)
	}
	return func() {
		allocator.execTransaction = originalExec
		if !injected {
			t.Fatal("COMMIT failpoint was not reached")
		}
	}
}

func TestAllocatorRejectsForeignReleaseAndKeepsReleaseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-cids.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)}
	allocator := openTestAllocator(t, database, clock, 101, 101)
	lease, err := allocator.Reserve(context.Background(), testReservationRequest(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	release := ports.AgentMicroVMVsockCIDReleaseRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: "launch-identity:foreign",
		FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
		IdempotencyKey: "release-vsock-cid:foreign",
	}
	if _, err := allocator.Release(
		context.Background(), release,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.foreign_owner" {
		t.Fatalf("foreign release = %v", err)
	}
	release.OwnerRef = lease.OwnerRef
	release.IdempotencyKey = "release-vsock-cid:owner"
	first, err := allocator.Release(context.Background(), release)
	if err != nil {
		t.Fatal(err)
	}
	second, err := allocator.Release(context.Background(), release)
	if err != nil || second != first {
		t.Fatalf("idempotent release = %+v, %v; want %+v", second, err, first)
	}
}

func TestAllocatorRenewsWithRevisionFenceAndIdempotency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vsock-cids.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 13, 0, 0, 0, time.UTC)}
	allocator := openTestAllocator(t, database, clock, 110, 110)
	request := testReservationRequest(t, 1)
	request.LeaseDuration = time.Minute
	lease, err := allocator.Reserve(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(30 * time.Second)
	renewal := ports.AgentMicroVMVsockCIDRenewalRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
		LeaseDuration: 2 * time.Minute, IdempotencyKey: "renew-vsock-cid:1",
	}
	renewed, err := allocator.Renew(context.Background(), renewal)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Revision != lease.Revision+1 || !renewed.ExpiresAt.After(lease.ExpiresAt) {
		t.Fatalf("lease not extended: before=%+v after=%+v", lease, renewed)
	}
	replayed, err := allocator.Renew(context.Background(), renewal)
	if err != nil || replayed != renewed {
		t.Fatalf("renew replay = %+v, %v; want %+v", replayed, err, renewed)
	}
	stale := renewal
	stale.IdempotencyKey = "renew-vsock-cid:stale"
	if _, err := allocator.Renew(
		context.Background(), stale,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.stale_revision" {
		t.Fatalf("stale renewal = %v", err)
	}
}

func TestAllocatorRequiresCanonicalSchemaAndSafeCIDRange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-schema.db")
	database := openTestDatabase(t, path, false)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 14, 0, 0, 0, time.UTC)}
	if _, err := Open(
		context.Background(), testConfig(database, clock, 32, 47),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.schema_unavailable" {
		t.Fatalf("missing schema = %v", err)
	}

	config := testConfig(database, clock, ports.AgentMicroVMVsockHostCID, 47)
	if _, err := Open(
		context.Background(), config,
	); ErrorCode(err) != "agent_firecracker_vsock_cid.config_invalid" {
		t.Fatalf("reserved CID range = %v", err)
	}

	driftPath := filepath.Join(t.TempDir(), "schema-drift.db")
	driftDatabase := openTestDatabase(t, driftPath, true)
	if _, err := driftDatabase.Exec(`
DROP INDEX agent_microvm_vsock_cid_one_active;
CREATE INDEX agent_microvm_vsock_cid_one_active
ON agent_microvm_vsock_cid_reservations (pool_ref, guest_cid)`); err != nil {
		t.Fatal(err)
	}
	if err := driftDatabase.Close(); err != nil {
		t.Fatal(err)
	}
	driftDatabase = openTestDatabase(t, driftPath, false)
	t.Cleanup(func() { _ = driftDatabase.Close() })
	if _, err := Open(
		context.Background(), testConfig(driftDatabase, clock, 32, 47),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.schema_mismatch" {
		t.Fatalf("non-unique schema = %v", err)
	}

	memoryDatabase, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = memoryDatabase.Close() })
	for _, statement := range SchemaStatements() {
		if _, err := memoryDatabase.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Open(
		context.Background(), testConfig(memoryDatabase, clock, 32, 47),
	); ErrorCode(err) != "agent_firecracker_vsock_cid.durability_unavailable" {
		t.Fatalf("in-memory store = %v", err)
	}
}

func TestAllocatorRejectsFileBackedDurabilityPragmaManipulation(t *testing.T) {
	tests := map[string]string{
		"synchronous disabled":  "PRAGMA synchronous = OFF",
		"foreign keys disabled": "PRAGMA foreign_keys = OFF",
		"busy timeout disabled": "PRAGMA busy_timeout = 0",
	}
	for name, statement := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vsock-pragma.db")
			database := openTestDatabase(t, path, true)
			t.Cleanup(func() { _ = database.Close() })
			database.SetMaxOpenConns(1)
			clock := &testClock{now: time.Date(2026, 7, 26, 14, 30, 0, 0, time.UTC)}
			allocator := openTestAllocator(t, database, clock, 32, 47)
			if _, err := database.ExecContext(context.Background(), statement); err != nil {
				t.Fatal(err)
			}
			if _, err := allocator.Reserve(
				context.Background(), testReservationRequest(t, 1),
			); ErrorCode(err) != "agent_firecracker_vsock_cid.durability_unavailable" {
				t.Fatalf("manipulated %q = %v", statement, err)
			}
		})
	}

	t.Run("journal mode persists across restart", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "vsock-journal.db")
		database := openTestDatabase(t, path, true)
		if _, err := database.ExecContext(
			context.Background(), "PRAGMA journal_mode = DELETE",
		); err != nil {
			_ = database.Close()
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		dsn := fmt.Sprintf(
			"file:%s?_pragma=busy_timeout(10000)&_pragma=synchronous(FULL)&_pragma=foreign_keys(1)",
			path,
		)
		database, err := sql.Open("sqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = database.Close() })
		var journalMode string
		if err := database.QueryRowContext(
			context.Background(), "PRAGMA journal_mode",
		).Scan(&journalMode); err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(journalMode, "delete") {
			t.Fatalf("journal manipulation did not persist: %q", journalMode)
		}
		clock := &testClock{now: time.Date(2026, 7, 26, 14, 45, 0, 0, time.UTC)}
		if _, err := Open(
			context.Background(), testConfig(database, clock, 32, 47),
		); ErrorCode(err) != "agent_firecracker_vsock_cid.durability_unavailable" {
			t.Fatalf("persistent journal manipulation = %v", err)
		}
	})
}

func TestAllocatorEvaluatesTimeAfterContendedBegin(t *testing.T) {
	t.Run("reserve starts lease after lock acquisition", func(t *testing.T) {
		database, allocator, clock := newContentionAllocator(t, 120)
		releaseWriter := holdWriteTransaction(t, database)
		entered := observeTransactionBegin(allocator)
		type result struct {
			lease ports.AgentMicroVMVsockCIDLease
			err   error
		}
		resultChannel := make(chan result, 1)
		request := testReservationRequest(t, 1)
		go func() {
			lease, err := allocator.Reserve(context.Background(), request)
			resultChannel <- result{lease: lease, err: err}
		}()
		waitTransactionBegin(t, entered)
		clock.Advance(2 * time.Minute)
		releaseWriter()
		got := <-resultChannel
		if got.err != nil {
			t.Fatal(got.err)
		}
		wantAcquiredAt := time.Date(2026, 7, 26, 15, 2, 0, 0, time.UTC)
		if got.lease.AcquiredAt != wantAcquiredAt ||
			got.lease.ExpiresAt != wantAcquiredAt.Add(10*time.Minute) {
			t.Fatalf("lease used pre-lock clock: %+v", got.lease)
		}
	})

	tests := map[string]func(
		context.Context,
		*Allocator,
		ports.AgentMicroVMVsockCIDLease,
	) error{
		"renew observes expiry while waiting": func(
			ctx context.Context,
			allocator *Allocator,
			lease ports.AgentMicroVMVsockCIDLease,
		) error {
			_, err := allocator.Renew(ctx, ports.AgentMicroVMVsockCIDRenewalRequest{
				PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
				LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
				FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
				LeaseDuration: 2 * time.Minute, IdempotencyKey: "renew-after-contention",
			})
			return err
		},
		"recover observes expiry while waiting": func(
			ctx context.Context,
			allocator *Allocator,
			lease ports.AgentMicroVMVsockCIDLease,
		) error {
			_, err := allocator.Recover(ctx, ports.AgentMicroVMVsockCIDRecoveryRequest{
				PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
				LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
				FencingToken: lease.FencingToken,
			})
			return err
		},
		"release observes expiry while waiting": func(
			ctx context.Context,
			allocator *Allocator,
			lease ports.AgentMicroVMVsockCIDLease,
		) error {
			_, err := allocator.Release(ctx, ports.AgentMicroVMVsockCIDReleaseRequest{
				PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
				LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
				FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
				IdempotencyKey: "release-after-contention",
			})
			return err
		},
	}
	for name, operation := range tests {
		t.Run(name, func(t *testing.T) {
			database, allocator, clock := newContentionAllocator(t, 121)
			request := testReservationRequest(t, 1)
			request.LeaseDuration = time.Minute
			lease, err := allocator.Reserve(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			releaseWriter := holdWriteTransaction(t, database)
			entered := observeTransactionBegin(allocator)
			resultChannel := make(chan error, 1)
			go func() {
				resultChannel <- operation(context.Background(), allocator, lease)
			}()
			waitTransactionBegin(t, entered)
			clock.Advance(time.Minute)
			releaseWriter()
			if err := <-resultChannel; ErrorCode(err) !=
				"agent_firecracker_vsock_cid.lease_inactive" {
				t.Fatalf("operation used pre-lock clock: %v", err)
			}
		})
	}
}

func newContentionAllocator(
	t *testing.T,
	guestCID uint32,
) (*sql.DB, *Allocator, *testClock) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vsock-contention.db")
	database := openTestDatabase(t, path, true)
	t.Cleanup(func() { _ = database.Close() })
	clock := &testClock{now: time.Date(2026, 7, 26, 15, 0, 0, 0, time.UTC)}
	return database, openTestAllocator(t, database, clock, guestCID, guestCID), clock
}

func holdWriteTransaction(t *testing.T, database *sql.DB) func() {
	t.Helper()
	connection, err := database.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		_ = connection.Close()
		t.Fatal(err)
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
			_ = connection.Close()
		})
	}
	t.Cleanup(release)
	return release
}

func observeTransactionBegin(allocator *Allocator) <-chan struct{} {
	entered := make(chan struct{})
	original := allocator.beginTransaction
	var once sync.Once
	allocator.beginTransaction = func(ctx context.Context) (*sql.Conn, error) {
		once.Do(func() { close(entered) })
		return original(ctx)
	}
	return entered
}

func waitTransactionBegin(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("operation did not attempt BEGIN IMMEDIATE")
	}
}
