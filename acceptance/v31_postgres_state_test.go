package acceptance_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	statePostgres "orquesta/internal/adapters/state/postgres"
	"orquesta/internal/goal"
)

const v31PostgresStateFixturePath = "acceptance/fixtures/v31_postgres_state.json"

type v31PostgresStateFixture struct {
	SchemaVersion        int      `json:"schema_version"`
	FixtureID            string   `json:"fixture_id"`
	ContractID           string   `json:"contract_id"`
	CapabilityIDs        []string `json:"capability_ids"`
	Status               string   `json:"status"`
	AdapterPath          string   `json:"adapter_path"`
	ImplementedBehaviors []string `json:"implemented_behaviors"`
	DeferredGates        []string `json:"deferred_gates"`
}

func TestAcceptanceV31PostgresStateFoundation(t *testing.T) {
	fixture := readV31PostgresStateFixture(t)
	assertV31PostgresStateFixture(t, fixture)

	serverTime := time.Date(2026, 8, 11, 11, 15, 0, 0, time.FixedZone("postgres", 90*60))
	state := &v31PostgresState{serverTime: serverTime, revision: 5}
	foundation := newV31PostgresFoundation(t, state)
	request := v31PostgresRequest(t)
	receipt, err := foundation.ApplyAtomicMutation(context.Background(), request)
	v31PostgresNoError(t, err)
	if receipt.Revision != 5 || !receipt.TransactionAt.Equal(serverTime) || receipt.TransactionAt.Location() != time.UTC ||
		!reflect.DeepEqual(state.orderSnapshot(), []string{"begin", "clock", "cas", "event", "outbox", "commit"}) {
		t.Fatalf("atomic PostgreSQL foundation receipt=%+v order=%v", receipt, state.orderSnapshot())
	}

	staleState := &v31PostgresState{serverTime: serverTime, revision: 5, stale: true}
	staleFoundation := newV31PostgresFoundation(t, staleState)
	if _, err := staleFoundation.ApplyAtomicMutation(context.Background(), request); statePostgres.ErrorCode(err) != statePostgres.ErrorStaleFence ||
		!reflect.DeepEqual(staleState.orderSnapshot(), []string{"begin", "clock", "cas", "rollback"}) {
		t.Fatalf("stale fence error=%v order=%v", err, staleState.orderSnapshot())
	}

	canceledState := &v31PostgresState{serverTime: serverTime, revision: 5}
	canceledFoundation := newV31PostgresFoundation(t, canceledState)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := canceledFoundation.ApplyAtomicMutation(ctx, request); statePostgres.ErrorCode(err) != statePostgres.ErrorContext || len(canceledState.orderSnapshot()) != 0 {
		t.Fatalf("canceled transaction error=%v order=%v", err, canceledState.orderSnapshot())
	}
}

func TestV31PostgresStateFoundationDoesNotClaimOPS11Accreditation(t *testing.T) {
	fixture := readV31PostgresStateFixture(t)
	if fixture.Status != "implemented_foundation_not_accredited" ||
		!reflect.DeepEqual(fixture.DeferredGates, []string{
			"concrete_postgresql_driver_and_complete_schema",
			"shared_sqlite_postgresql_state_repository_suite",
			"real_postgresql_cluster_concurrency_and_multihost",
			"migration_and_cutover_without_dual_write",
			"sealed_candidate_receipt",
		}) {
		t.Fatalf("V31 PostgreSQL foundation overclaims closure: %+v", fixture)
	}
}

func readV31PostgresStateFixture(t *testing.T) v31PostgresStateFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(evidenceRepositoryRoot(t), v31PostgresStateFixturePath))
	v31PostgresNoError(t, err)
	var fixture v31PostgresStateFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	v31PostgresNoError(t, decoder.Decode(&fixture))
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("trailing fixture data: %v", err)
	}
	canonical, err := json.MarshalIndent(fixture, "", "  ")
	v31PostgresNoError(t, err)
	if !bytes.Equal(data, append(canonical, '\n')) {
		t.Fatal("V31 PostgreSQL fixture is not canonical strict JSON")
	}
	return fixture
}

func assertV31PostgresStateFixture(t *testing.T, fixture v31PostgresStateFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.FixtureID != "v31_postgres_state_foundation" ||
		fixture.ContractID != "AC-V31-POSTGRES-S3-MULTIHOST" ||
		!reflect.DeepEqual(fixture.CapabilityIDs, []string{"OPS-11"}) ||
		fixture.Status != "implemented_foundation_not_accredited" ||
		fixture.AdapterPath != "internal/adapters/state/postgres" ||
		!reflect.DeepEqual(fixture.ImplementedBehaviors, []string{
			"serializable_atomic_snapshot_event_outbox_transaction",
			"postgresql_transaction_time_is_authoritative",
			"expected_revision_generation_token_fence_and_lease_cas",
			"stale_fence_rolls_back_before_event_and_outbox",
			"database_and_context_failures_never_commit",
		}) {
		t.Fatalf("invalid V31 PostgreSQL fixture: %+v", fixture)
	}
}

func newV31PostgresFoundation(t *testing.T, state *v31PostgresState) *statePostgres.Foundation {
	t.Helper()
	database := sql.OpenDB(v31PostgresConnector{state: state})
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	foundation, err := statePostgres.NewFoundation(database)
	v31PostgresNoError(t, err)
	return foundation
}

func v31PostgresRequest(t *testing.T) statePostgres.AtomicMutationRequest {
	t.Helper()
	project, err := goal.NewProjectRef("project:v31-acceptance")
	v31PostgresNoError(t, err)
	goalRef, err := goal.NewGoalRef("goal:v31-acceptance")
	v31PostgresNoError(t, err)
	item, err := goal.NewWorkItemRef("work-item:v31-acceptance")
	v31PostgresNoError(t, err)
	return statePostgres.AtomicMutationRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: item,
		ExpectedRevision: 4, PlanGeneration: 2, LeaseToken: "lease:v31-acceptance", LeaseFence: 7,
		MutationRef: "mutation:v31-acceptance", EventRef: "event:v31-acceptance", OutboxRef: "action:v31-acceptance",
		Snapshot: []byte(`{"revision":5}`), EventPayload: []byte(`{"event":"updated"}`),
		OutboxPayload: []byte(`{"action":"continue"}`),
	}
}

type v31PostgresState struct {
	mu         sync.Mutex
	serverTime time.Time
	revision   int64
	stale      bool
	active     bool
	order      []string
}

func (state *v31PostgresState) record(value string) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.order = append(state.order, value)
}

func (state *v31PostgresState) orderSnapshot() []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]string(nil), state.order...)
}

type v31PostgresConnector struct{ state *v31PostgresState }

func (connector v31PostgresConnector) Connect(context.Context) (driver.Conn, error) {
	return &v31PostgresConnection{state: connector.state}, nil
}
func (connector v31PostgresConnector) Driver() driver.Driver {
	return v31PostgresDriver{state: connector.state}
}

type v31PostgresDriver struct{ state *v31PostgresState }

func (driverValue v31PostgresDriver) Open(string) (driver.Conn, error) {
	return &v31PostgresConnection{state: driverValue.state}, nil
}

type v31PostgresConnection struct{ state *v31PostgresState }

func (*v31PostgresConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*v31PostgresConnection) Close() error { return nil }
func (connection *v31PostgresConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}
func (connection *v31PostgresConnection) BeginTx(
	ctx context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.Isolation != driver.IsolationLevel(sql.LevelSerializable) || options.ReadOnly {
		return nil, errors.New("invalid transaction options")
	}
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	if connection.state.active {
		return nil, errors.New("transaction already active")
	}
	connection.state.active = true
	connection.state.order = append(connection.state.order, "begin")
	return &v31PostgresTransaction{state: connection.state}, nil
}

func (connection *v31PostgresConnection) QueryContext(
	ctx context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.HasPrefix(query, "SELECT transaction_timestamp()") {
		connection.state.record("clock")
		return &v31PostgresRows{columns: []string{"transaction_timestamp"}, values: [][]driver.Value{{connection.state.serverTime}}}, nil
	}
	if strings.HasPrefix(query, "UPDATE work_items") {
		connection.state.record("cas")
		if connection.state.stale {
			return &v31PostgresRows{columns: []string{"revision"}}, nil
		}
		return &v31PostgresRows{columns: []string{"revision"}, values: [][]driver.Value{{connection.state.revision}}}, nil
	}
	return nil, errors.New("unexpected query")
}

func (connection *v31PostgresConnection) ExecContext(
	ctx context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch {
	case strings.HasPrefix(query, "INSERT INTO events"):
		connection.state.record("event")
	case strings.HasPrefix(query, "INSERT INTO outbox"):
		connection.state.record("outbox")
	default:
		return nil, errors.New("unexpected exec")
	}
	return driver.RowsAffected(1), nil
}

type v31PostgresTransaction struct{ state *v31PostgresState }

func (transaction *v31PostgresTransaction) Commit() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	transaction.state.active = false
	transaction.state.order = append(transaction.state.order, "commit")
	return nil
}
func (transaction *v31PostgresTransaction) Rollback() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	transaction.state.active = false
	transaction.state.order = append(transaction.state.order, "rollback")
	return nil
}

type v31PostgresRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *v31PostgresRows) Columns() []string { return rows.columns }
func (*v31PostgresRows) Close() error           { return nil }
func (rows *v31PostgresRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

func v31PostgresNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
