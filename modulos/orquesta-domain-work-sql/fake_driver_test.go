package orquestadomainworksql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const domainWorkSQLFakeDriverNameV0 = "orquesta_domain_work_sql_fake_v0"

var (
	domainWorkSQLFakeDriverOnceV0 sync.Once
	domainWorkSQLFakeDriverSeqV0  atomic.Uint64
	domainWorkSQLFakeStoresV0     sync.Map
)

type domainWorkSQLFakeStateV0 struct {
	mu             sync.Mutex
	records        map[string]domainWorkSQLFakeRecordV0
	queries        []string
	insertRaceMode domainWorkSQLFakeInsertRaceModeV0
}

type domainWorkSQLFakeInsertRaceModeV0 string

const (
	domainWorkSQLFakeInsertRaceSameV0     domainWorkSQLFakeInsertRaceModeV0 = "same"
	domainWorkSQLFakeInsertRaceConflictV0 domainWorkSQLFakeInsertRaceModeV0 = "conflict"
)

type domainWorkSQLFakeRecordV0 struct {
	DomainRef      string
	IdempotencyKey string
	JobRef         string
	CorrelationID  string
	WorkKind       string
	Status         string
	RequestJSON    []byte
	JobJSON        []byte
	Fingerprint    string
}

type domainWorkSQLFakeDriverV0 struct{}

type domainWorkSQLFakeConnV0 struct {
	state *domainWorkSQLFakeStateV0
}

type domainWorkSQLFakeRowsV0 struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func openDomainWorkSQLFakeDriverDBV0(t testing.TB) *sql.DB {
	t.Helper()
	db, _ := openDomainWorkSQLFakeDriverDBAndStateV0(t)
	return db
}

func openDomainWorkSQLFakeDriverDBAndStateV0(t testing.TB) (*sql.DB, *domainWorkSQLFakeStateV0) {
	t.Helper()
	domainWorkSQLFakeDriverOnceV0.Do(func() {
		sql.Register(domainWorkSQLFakeDriverNameV0, domainWorkSQLFakeDriverV0{})
	})
	dsn := fmt.Sprintf("domain-work-sql-fake-%d", domainWorkSQLFakeDriverSeqV0.Add(1))
	state := &domainWorkSQLFakeStateV0{
		records: map[string]domainWorkSQLFakeRecordV0{},
	}
	domainWorkSQLFakeStoresV0.Store(dsn, state)
	db, err := sql.Open(domainWorkSQLFakeDriverNameV0, dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() {
		domainWorkSQLFakeStoresV0.Delete(dsn)
	})
	return db, state
}

func (domainWorkSQLFakeDriverV0) Open(name string) (driver.Conn, error) {
	value, ok := domainWorkSQLFakeStoresV0.Load(name)
	if !ok {
		return nil, fmt.Errorf("fake dsn missing")
	}
	return &domainWorkSQLFakeConnV0{
		state: value.(*domainWorkSQLFakeStateV0),
	}, nil
}

func (conn *domainWorkSQLFakeConnV0) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare no soportado")
}

func (conn *domainWorkSQLFakeConnV0) Close() error {
	return nil
}

func (conn *domainWorkSQLFakeConnV0) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("tx no soportado")
}

func (conn *domainWorkSQLFakeConnV0) ExecContext(
	_ context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Result, error) {
	normalized := strings.TrimSpace(query)
	if !strings.HasPrefix(normalized, "INSERT INTO ") {
		return nil, fmt.Errorf("exec query no soportada: %s", normalized)
	}
	if len(args) != 9 {
		return nil, fmt.Errorf("args=%d want 9", len(args))
	}
	record := domainWorkSQLFakeRecordV0{
		DomainRef:      stringArgV0(args[0].Value),
		IdempotencyKey: stringArgV0(args[1].Value),
		JobRef:         stringArgV0(args[2].Value),
		CorrelationID:  stringArgV0(args[3].Value),
		WorkKind:       stringArgV0(args[4].Value),
		Status:         stringArgV0(args[5].Value),
		RequestJSON:    bytesArgV0(args[6].Value),
		JobJSON:        bytesArgV0(args[7].Value),
		Fingerprint:    stringArgV0(args[8].Value),
	}
	key := record.DomainRef + "\x00" + record.IdempotencyKey
	conn.state.mu.Lock()
	defer conn.state.mu.Unlock()
	conn.state.queries = append(conn.state.queries, normalized)
	if conn.state.insertRaceMode != "" {
		raceRecord := record
		if conn.state.insertRaceMode == domainWorkSQLFakeInsertRaceConflictV0 {
			raceRecord.Fingerprint = "fingerprint-distinta"
		}
		conn.state.insertRaceMode = ""
		if _, exists := conn.state.records[key]; !exists {
			conn.state.records[key] = raceRecord
		}
		return nil, fmt.Errorf("duplicate_key")
	}
	if _, exists := conn.state.records[key]; exists {
		return nil, fmt.Errorf("duplicate_key")
	}
	for _, existing := range conn.state.records {
		if existing.JobRef == record.JobRef {
			return nil, fmt.Errorf("duplicate_job_ref")
		}
	}
	conn.state.records[key] = record
	return driver.RowsAffected(1), nil
}

func (conn *domainWorkSQLFakeConnV0) QueryContext(
	_ context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	normalized := strings.TrimSpace(query)
	conn.state.mu.Lock()
	defer conn.state.mu.Unlock()
	conn.state.queries = append(conn.state.queries, normalized)
	switch {
	case domainWorkSQLFakeMatchesDomainKeyQueryV0(normalized):
		if len(args) != 2 {
			return nil, fmt.Errorf("args=%d want 2", len(args))
		}
		key := stringArgV0(args[0].Value) + "\x00" + stringArgV0(args[1].Value)
		record, ok := conn.state.records[key]
		rows := &domainWorkSQLFakeRowsV0{
			columns: []string{"fingerprint", "request_json", "job_json"},
		}
		if ok {
			rows.values = append(rows.values, []driver.Value{
				record.Fingerprint,
				append([]byte(nil), record.RequestJSON...),
				append([]byte(nil), record.JobJSON...),
			})
		}
		return rows, nil
	case domainWorkSQLFakeMatchesJobRefQueryV0(normalized):
		if len(args) != 1 {
			return nil, fmt.Errorf("args=%d want 1", len(args))
		}
		jobRef := stringArgV0(args[0].Value)
		rows := &domainWorkSQLFakeRowsV0{
			columns: []string{"job_ref"},
		}
		for _, record := range conn.state.records {
			if record.JobRef == jobRef {
				rows.values = append(rows.values, []driver.Value{record.JobRef})
				break
			}
		}
		return rows, nil
	case strings.Contains(normalized, "ORDER BY job_ref ASC"):
		records := make([]domainWorkSQLFakeRecordV0, 0, len(conn.state.records))
		for _, record := range conn.state.records {
			records = append(records, record)
		}
		sort.Slice(records, func(i int, j int) bool {
			return records[i].JobRef < records[j].JobRef
		})
		rows := &domainWorkSQLFakeRowsV0{
			columns: []string{"request_json", "job_json"},
		}
		for _, record := range records {
			rows.values = append(rows.values, []driver.Value{
				append([]byte(nil), record.RequestJSON...),
				append([]byte(nil), record.JobJSON...),
			})
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("query no soportada: %s", normalized)
	}
}

func domainWorkSQLFakeMatchesDomainKeyQueryV0(query string) bool {
	return strings.Contains(query, "WHERE domain_ref = ? AND idempotency_key = ?") ||
		strings.Contains(query, "WHERE domain_ref = $1 AND idempotency_key = $2")
}

func domainWorkSQLFakeMatchesJobRefQueryV0(query string) bool {
	return strings.Contains(query, "WHERE job_ref = ?") ||
		strings.Contains(query, "WHERE job_ref = $1")
}

func (rows *domainWorkSQLFakeRowsV0) Columns() []string {
	return append([]string(nil), rows.columns...)
}

func (rows *domainWorkSQLFakeRowsV0) Close() error {
	return nil
}

func (rows *domainWorkSQLFakeRowsV0) Next(dest []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	for index, value := range rows.values[rows.index] {
		dest[index] = value
	}
	rows.index++
	return nil
}

func stringArgV0(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func bytesArgV0(value any) []byte {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...)
	case string:
		return []byte(typed)
	default:
		return []byte(fmt.Sprint(typed))
	}
}
