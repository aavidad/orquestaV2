package orquestadomainworksql_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworksql "orquesta/modulos/orquesta-domain-work-sql"
	orquestadomainworkcontracttest "orquesta/modulos/orquesta-domain-work/contracttest"
)

func TestSQLDomainWorkJobRecordStoreV0CumpleContratoRecordStore(t *testing.T) {
	orquestadomainworkcontracttest.RunDomainWorkJobRecordStoreContractV0(
		t,
		func(t testing.TB) orquestadomainwork.DomainWorkJobRecordStorePortV0 {
			t.Helper()
			db := openDomainWorkSQLFakeDBV0(t)
			store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
				db,
				orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{},
			)
			if err != nil {
				t.Fatalf("NewSQLDomainWorkJobRecordStoreV0: %v", err)
			}
			return store
		},
	)
}

func TestSQLDomainWorkJobRecordStoreV0CumpleContratoRecordStoreConPlaceholdersDollar(t *testing.T) {
	orquestadomainworkcontracttest.RunDomainWorkJobRecordStoreContractV0(
		t,
		func(t testing.TB) orquestadomainwork.DomainWorkJobRecordStorePortV0 {
			t.Helper()
			db := openDomainWorkSQLFakeDBV0(t)
			store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
				db,
				orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
					PlaceholderStyle: orquestadomainworksql.SQLDomainWorkPlaceholderDollarV0,
				},
			)
			if err != nil {
				t.Fatalf("NewSQLDomainWorkJobRecordStoreV0: %v", err)
			}
			return store
		},
	)
}

func TestSQLDomainWorkJobRecordStoreV0RechazaDBNil(t *testing.T) {
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		nil,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{},
	)
	if err == nil || store != nil {
		t.Fatalf("store=%v err=%v", store, err)
	}
}

func TestSQLDomainWorkJobRecordStoreV0RechazaTablaInvalida(t *testing.T) {
	db := openDomainWorkSQLFakeDBV0(t)
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		db,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
			TableName: "domain_work_jobs;drop",
		},
	)
	if err == nil || store != nil {
		t.Fatalf("store=%v err=%v", store, err)
	}
}

func TestSQLDomainWorkJobRecordStoreV0RechazaPlaceholderInvalido(t *testing.T) {
	db := openDomainWorkSQLFakeDBV0(t)
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		db,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
			PlaceholderStyle: "named",
		},
	)
	if err == nil || store != nil {
		t.Fatalf("store=%v err=%v", store, err)
	}
}

func TestSQLDomainWorkJobRecordStoreV0UsaPlaceholderDollar(t *testing.T) {
	db, state := openDomainWorkSQLFakeDBAndStateV0(t)
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		db,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
			PlaceholderStyle: orquestadomainworksql.SQLDomainWorkPlaceholderDollarV0,
		},
	)
	if err != nil {
		t.Fatalf("NewSQLDomainWorkJobRecordStoreV0: %v", err)
	}
	job, err := store.CreateDomainWorkJobV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "req-dollar-001",
			CorrelationID:  "corr-dollar-001",
			IdempotencyKey: "idem-dollar-001",
			RequestedBy:    "test",
			DomainRef:      "domain-dollar",
			WorkKind:       "work-dollar",
			Objective:      "Comprobar dialecto dollar.",
		},
	)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("status=%q", job.Status)
	}
	state.mu.Lock()
	queries := append([]string(nil), state.queries...)
	state.mu.Unlock()
	joined := strings.Join(queries, "\n")
	if !strings.Contains(joined, "$1") || strings.Contains(joined, " = ?") {
		t.Fatalf("queries no usan placeholders dollar:\n%s", joined)
	}
}

func TestSQLDomainWorkJobRecordStoreV0UniqueViolationReleeYReproduceReplay(t *testing.T) {
	db, state := openDomainWorkSQLFakeDBAndStateV0(t)
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		db,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
			IsUniqueViolation: domainWorkSQLFakeIsUniqueViolationV0,
		},
	)
	if err != nil {
		t.Fatalf("NewSQLDomainWorkJobRecordStoreV0: %v", err)
	}
	state.mu.Lock()
	state.insertRaceMode = domainWorkSQLFakeInsertRaceSameV0
	state.mu.Unlock()
	job, err := store.CreateDomainWorkJobV0(
		context.Background(),
		validDomainWorkSQLRequestForTestV0("race-replay"),
	)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 || job.JobRef == "" {
		t.Fatalf("job=%+v", job)
	}
	records, err := store.ListDomainWorkJobRecordsV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRecordFilterV0{IdempotencyKey: "idem-race-replay"},
	)
	if err != nil {
		t.Fatalf("ListDomainWorkJobRecordsV0: %v", err)
	}
	if len(records) != 1 || records[0].Job.JobRef != job.JobRef {
		t.Fatalf("records=%+v job=%+v", records, job)
	}
}

func TestSQLDomainWorkJobRecordStoreV0UniqueViolationReleeYDevuelveConflicto(t *testing.T) {
	db, state := openDomainWorkSQLFakeDBAndStateV0(t)
	store, err := orquestadomainworksql.NewSQLDomainWorkJobRecordStoreV0(
		db,
		orquestadomainworksql.SQLDomainWorkJobRecordStoreConfigV0{
			IsUniqueViolation: domainWorkSQLFakeIsUniqueViolationV0,
		},
	)
	if err != nil {
		t.Fatalf("NewSQLDomainWorkJobRecordStoreV0: %v", err)
	}
	state.mu.Lock()
	state.insertRaceMode = domainWorkSQLFakeInsertRaceConflictV0
	state.mu.Unlock()
	job, err := store.CreateDomainWorkJobV0(
		context.Background(),
		validDomainWorkSQLRequestForTestV0("race-conflict"),
	)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusInvalidV0 {
		t.Fatalf("status=%q job=%+v", job.Status, job)
	}
	if len(job.Issues) != 1 ||
		job.Issues[0].Code != orquestadomainworksql.ErrDomainWorkSQLIdempotencyConflictV0 {
		t.Fatalf("issues=%+v", job.Issues)
	}
	records, err := store.ListDomainWorkJobRecordsV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRecordFilterV0{IdempotencyKey: "idem-race-conflict"},
	)
	if err != nil {
		t.Fatalf("ListDomainWorkJobRecordsV0: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records=%+v", records)
	}
}

func openDomainWorkSQLFakeDBV0(t testing.TB) *sql.DB {
	t.Helper()
	db := openDomainWorkSQLFakeDriverDBV0(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func openDomainWorkSQLFakeDBAndStateV0(t testing.TB) (*sql.DB, *domainWorkSQLFakeStateV0) {
	t.Helper()
	db, state := openDomainWorkSQLFakeDriverDBAndStateV0(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, state
}

func domainWorkSQLFakeIsUniqueViolationV0(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate_key")
}

func validDomainWorkSQLRequestForTestV0(suffix string) orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "req-" + suffix,
		CorrelationID:  "corr-" + suffix,
		IdempotencyKey: "idem-" + suffix,
		RequestedBy:    "test",
		DomainRef:      "domain-" + suffix,
		WorkKind:       "work-" + suffix,
		Objective:      "Comprobar persistencia SQL.",
	}
}
