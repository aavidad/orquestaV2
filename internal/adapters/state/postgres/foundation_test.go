package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestFoundationCommitsAtomicMutationOnceUsingServerTime(t *testing.T) {
	serverTime := time.Date(2026, 8, 11, 10, 30, 0, 123, time.FixedZone("server", 2*60*60))
	state := newFakeState(serverTime, 8)
	database := openFakeDatabase(t, state)
	foundation := newTestFoundation(t, database)
	request := validAtomicMutation(t)

	receipt, err := foundation.ApplyAtomicMutation(context.Background(), request)
	postgresNoError(t, err)
	if receipt.MutationRef != request.MutationRef || receipt.Revision != 8 ||
		!receipt.TransactionAt.Equal(serverTime) || receipt.TransactionAt.Location() != time.UTC {
		t.Fatalf("receipt=%+v", receipt)
	}
	if got := state.orderSnapshot(); !reflect.DeepEqual(got, []string{"begin", "clock", "cas", "event", "outbox", "commit"}) {
		t.Fatalf("transaction order=%v", got)
	}
	if state.beginCount != 1 || state.commitCount != 1 || state.rollbackCount != 0 {
		t.Fatalf("begin=%d commit=%d rollback=%d", state.beginCount, state.commitCount, state.rollbackCount)
	}
	assertServerTimeArguments(t, state, serverTime.UTC())
}

func TestFoundationFencedCASGuardsEveryCausalDimension(t *testing.T) {
	for _, marker := range []string{
		"project_ref = $4", "goal_ref = $5", "work_item_ref = $6", "revision = $7",
		"plan_generation = $8", "lease_token = $9", "lease_fence = $10", "lease_expires_at > $2",
		"RETURNING revision",
	} {
		if !strings.Contains(fencedCASSQL, marker) {
			t.Fatalf("fenced CAS lacks %q", marker)
		}
	}
	state := newFakeState(time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC), 8)
	request := validAtomicMutation(t)
	foundation := newTestFoundation(t, openFakeDatabase(t, state))
	_, err := foundation.ApplyAtomicMutation(context.Background(), request)
	postgresNoError(t, err)

	state.mu.Lock()
	defer state.mu.Unlock()
	var arguments []driver.NamedValue
	for _, call := range state.calls {
		if call.stage == "cas" {
			arguments = call.arguments
			break
		}
	}
	want := []any{
		request.Snapshot, state.serverTime.UTC(), request.MutationRef,
		request.ProjectRef.String(), request.GoalRef.String(), request.WorkItemRef.String(),
		int64(request.ExpectedRevision), int64(request.PlanGeneration), request.LeaseToken, int64(request.LeaseFence),
	}
	if len(arguments) != len(want) {
		t.Fatalf("CAS arguments=%d want=%d", len(arguments), len(want))
	}
	for index := range want {
		if !reflect.DeepEqual(arguments[index].Value, want[index]) {
			t.Fatalf("CAS argument %d=%#v want=%#v", index+1, arguments[index].Value, want[index])
		}
	}
}

func TestFoundationRollsBackEveryPreCommitFailure(t *testing.T) {
	for _, stage := range []string{"clock", "cas", "event", "outbox"} {
		t.Run(stage, func(t *testing.T) {
			state := newFakeState(time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC), 8)
			state.failStage = stage
			database := openFakeDatabase(t, state)
			foundation := newTestFoundation(t, database)
			if receipt, err := foundation.ApplyAtomicMutation(context.Background(), validAtomicMutation(t)); receipt.MutationRef != "" || ErrorCode(err) != ErrorUnavailable {
				t.Fatalf("stage=%s receipt=%+v error=%v", stage, receipt, err)
			}
			if state.beginCount != 1 || state.commitCount != 0 || state.rollbackCount != 1 {
				t.Fatalf("stage=%s begin=%d commit=%d rollback=%d order=%v",
					stage, state.beginCount, state.commitCount, state.rollbackCount, state.orderSnapshot())
			}
		})
	}
}

func TestFoundationRejectsStaleFenceBeforeEventOrOutbox(t *testing.T) {
	state := newFakeState(time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC), 8)
	state.staleFence = true
	database := openFakeDatabase(t, state)
	foundation := newTestFoundation(t, database)

	if receipt, err := foundation.ApplyAtomicMutation(context.Background(), validAtomicMutation(t)); receipt.MutationRef != "" || ErrorCode(err) != ErrorStaleFence {
		t.Fatalf("receipt=%+v error=%v", receipt, err)
	}
	if got := state.orderSnapshot(); !reflect.DeepEqual(got, []string{"begin", "clock", "cas", "rollback"}) {
		t.Fatalf("stale fence order=%v", got)
	}
}

func TestFoundationRejectsInvalidReturnedRevisionAndCommitError(t *testing.T) {
	t.Run("revision invariant", func(t *testing.T) {
		state := newFakeState(time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC), 99)
		foundation := newTestFoundation(t, openFakeDatabase(t, state))
		if _, err := foundation.ApplyAtomicMutation(context.Background(), validAtomicMutation(t)); ErrorCode(err) != ErrorInvariant {
			t.Fatalf("error=%v", err)
		}
		if state.commitCount != 0 || state.rollbackCount != 1 {
			t.Fatalf("commit=%d rollback=%d", state.commitCount, state.rollbackCount)
		}
	})
	t.Run("commit error is not followed by second transaction decision", func(t *testing.T) {
		state := newFakeState(time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC), 8)
		state.failStage = "commit"
		foundation := newTestFoundation(t, openFakeDatabase(t, state))
		if _, err := foundation.ApplyAtomicMutation(context.Background(), validAtomicMutation(t)); ErrorCode(err) != ErrorUnavailable {
			t.Fatalf("error=%v", err)
		}
		if state.commitCount != 1 || state.rollbackCount != 0 {
			t.Fatalf("commit=%d rollback=%d order=%v", state.commitCount, state.rollbackCount, state.orderSnapshot())
		}
	})
}

func TestFoundationPreservesContextFailureAndNeverStartsCanceledWork(t *testing.T) {
	t.Run("canceled before begin", func(t *testing.T) {
		state := newFakeState(time.Now(), 8)
		foundation := newTestFoundation(t, openFakeDatabase(t, state))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := foundation.ApplyAtomicMutation(ctx, validAtomicMutation(t))
		if ErrorCode(err) != ErrorContext || !errors.Is(err, context.Canceled) || state.beginCount != 0 {
			t.Fatalf("error=%v begins=%d", err, state.beginCount)
		}
	})
	t.Run("canceled inside transaction", func(t *testing.T) {
		state := newFakeState(time.Now(), 8)
		state.failStage = "clock_context"
		foundation := newTestFoundation(t, openFakeDatabase(t, state))
		_, err := foundation.ApplyAtomicMutation(context.Background(), validAtomicMutation(t))
		if ErrorCode(err) != ErrorContext || !errors.Is(err, context.Canceled) || state.rollbackCount != 1 {
			t.Fatalf("error=%v rollback=%d", err, state.rollbackCount)
		}
	})
}

func TestFoundationValidatesTypedBoundaryBeforeDatabase(t *testing.T) {
	if foundation, err := NewFoundation(nil); foundation != nil || ErrorCode(err) != ErrorInvalid {
		t.Fatalf("NewFoundation(nil) foundation=%v error=%v", foundation, err)
	}
	state := newFakeState(time.Now(), 8)
	foundation := newTestFoundation(t, openFakeDatabase(t, state))
	request := validAtomicMutation(t)
	request.LeaseFence = 0
	if _, err := foundation.ApplyAtomicMutation(context.Background(), request); ErrorCode(err) != ErrorInvalid {
		t.Fatalf("error=%v", err)
	}
	if state.beginCount != 0 {
		t.Fatalf("invalid request began %d transactions", state.beginCount)
	}
}

func assertServerTimeArguments(t *testing.T, state *fakeSQLState, want time.Time) {
	t.Helper()
	state.mu.Lock()
	defer state.mu.Unlock()
	wantPositions := map[string]int{"cas": 1, "event": 6, "outbox": 6}
	for _, call := range state.calls {
		position, found := wantPositions[call.stage]
		if !found {
			continue
		}
		got, ok := call.arguments[position].Value.(time.Time)
		if !ok || !got.Equal(want) || got.Location() != time.UTC {
			t.Fatalf("%s transaction time=%#v want=%v", call.stage, call.arguments[position].Value, want)
		}
	}
}

func validAtomicMutation(t *testing.T) AtomicMutationRequest {
	t.Helper()
	project, err := goal.NewProjectRef("project:v31-postgres")
	postgresNoError(t, err)
	goalRef, err := goal.NewGoalRef("goal:v31-postgres")
	postgresNoError(t, err)
	item, err := goal.NewWorkItemRef("work-item:v31-postgres")
	postgresNoError(t, err)
	return AtomicMutationRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: item,
		ExpectedRevision: 7, PlanGeneration: 3,
		LeaseToken: "lease-token:v31", LeaseFence: 11,
		MutationRef: "mutation:v31", EventRef: "event:v31", OutboxRef: "action:v31",
		Snapshot: []byte(`{"revision":8}`), EventPayload: []byte(`{"kind":"updated"}`),
		OutboxPayload: []byte(`{"kind":"continue"}`),
	}
}

func newTestFoundation(t *testing.T, database *sql.DB) *Foundation {
	t.Helper()
	foundation, err := NewFoundation(database)
	postgresNoError(t, err)
	return foundation
}

func openFakeDatabase(t *testing.T, state *fakeSQLState) *sql.DB {
	t.Helper()
	database := sql.OpenDB(fakeConnector{state: state})
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

type fakeCall struct {
	stage     string
	arguments []driver.NamedValue
}

type fakeSQLState struct {
	mu            sync.Mutex
	serverTime    time.Time
	nextRevision  int64
	failStage     string
	staleFence    bool
	active        bool
	beginCount    int
	commitCount   int
	rollbackCount int
	order         []string
	calls         []fakeCall
}

func newFakeState(serverTime time.Time, nextRevision int64) *fakeSQLState {
	return &fakeSQLState{serverTime: serverTime, nextRevision: nextRevision}
}

func (state *fakeSQLState) record(stage string, arguments []driver.NamedValue) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.order = append(state.order, stage)
	if arguments != nil {
		state.calls = append(state.calls, fakeCall{stage: stage, arguments: append([]driver.NamedValue(nil), arguments...)})
	}
}

func (state *fakeSQLState) orderSnapshot() []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]string(nil), state.order...)
}

type fakeConnector struct{ state *fakeSQLState }

func (connector fakeConnector) Connect(context.Context) (driver.Conn, error) {
	return &fakeConnection{state: connector.state}, nil
}

func (connector fakeConnector) Driver() driver.Driver { return fakeDriver{state: connector.state} }

type fakeDriver struct{ state *fakeSQLState }

func (driverValue fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConnection{state: driverValue.state}, nil
}

type fakeConnection struct{ state *fakeSQLState }

func (*fakeConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*fakeConnection) Close() error { return nil }
func (connection *fakeConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}

func (connection *fakeConnection) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	if connection.state.failStage == "begin" {
		return nil, errors.New("injected begin error")
	}
	if options.Isolation != driver.IsolationLevel(sql.LevelSerializable) || options.ReadOnly || connection.state.active {
		return nil, errors.New("invalid transaction options")
	}
	connection.state.active = true
	connection.state.beginCount++
	connection.state.order = append(connection.state.order, "begin")
	return &fakeTransaction{state: connection.state}, nil
}

func (connection *fakeConnection) QueryContext(
	ctx context.Context,
	query string,
	arguments []driver.NamedValue,
) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stage := ""
	switch query {
	case transactionTimeSQL:
		stage = "clock"
	case fencedCASSQL:
		stage = "cas"
	default:
		return nil, errors.New("unexpected query")
	}
	connection.state.record(stage, arguments)
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	if !connection.state.active {
		return nil, errors.New("query outside transaction")
	}
	if connection.state.failStage == "clock_context" && stage == "clock" {
		return nil, context.Canceled
	}
	if connection.state.failStage == stage {
		return nil, errors.New("injected " + stage + " error")
	}
	if stage == "clock" {
		return &fakeRows{columns: []string{"transaction_timestamp"}, values: [][]driver.Value{{connection.state.serverTime}}}, nil
	}
	if connection.state.staleFence {
		return &fakeRows{columns: []string{"revision"}}, nil
	}
	return &fakeRows{columns: []string{"revision"}, values: [][]driver.Value{{connection.state.nextRevision}}}, nil
}

func (connection *fakeConnection) ExecContext(
	ctx context.Context,
	query string,
	arguments []driver.NamedValue,
) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stage := ""
	switch query {
	case insertEventSQL:
		stage = "event"
	case insertOutboxSQL:
		stage = "outbox"
	default:
		return nil, errors.New("unexpected exec")
	}
	connection.state.record(stage, arguments)
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	if !connection.state.active {
		return nil, errors.New("exec outside transaction")
	}
	if connection.state.failStage == stage {
		return nil, errors.New("injected " + stage + " error")
	}
	return driver.RowsAffected(1), nil
}

type fakeTransaction struct{ state *fakeSQLState }

func (transaction *fakeTransaction) Commit() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	if !transaction.state.active {
		return errors.New("commit inactive transaction")
	}
	transaction.state.active = false
	transaction.state.commitCount++
	transaction.state.order = append(transaction.state.order, "commit")
	if transaction.state.failStage == "commit" {
		return errors.New("injected commit error")
	}
	return nil
}

func (transaction *fakeTransaction) Rollback() error {
	transaction.state.mu.Lock()
	defer transaction.state.mu.Unlock()
	if !transaction.state.active {
		return errors.New("rollback inactive transaction")
	}
	transaction.state.active = false
	transaction.state.rollbackCount++
	transaction.state.order = append(transaction.state.order, "rollback")
	return nil
}

type fakeRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *fakeRows) Columns() []string { return rows.columns }
func (*fakeRows) Close() error           { return nil }
func (rows *fakeRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

func postgresNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
