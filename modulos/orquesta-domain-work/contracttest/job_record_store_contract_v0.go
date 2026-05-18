package orquestadomainworkcontracttest

import (
	"context"
	"errors"
	"sort"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type DomainWorkJobRecordStoreFactoryV0 func(testing.TB) orquestadomainwork.DomainWorkJobRecordStorePortV0

func RunDomainWorkJobRecordStoreContractV0(
	t *testing.T,
	newStore DomainWorkJobRecordStoreFactoryV0,
) {
	t.Helper()
	t.Run("create_replay_conflict_invalid", func(t *testing.T) {
		store := newDomainWorkRecordStoreForContractV0(t, newStore)
		request := validDomainWorkContractJobRequestV0("base")

		first := createDomainWorkContractJobForTestV0(t, store, request)
		retry := request
		retry.RequestID = "request-ref-contract-retry"
		second := createDomainWorkContractJobForTestV0(t, store, retry)
		if second.JobRef != first.JobRef {
			t.Fatalf("replay job_ref=%q want %q", second.JobRef, first.JobRef)
		}
		records := listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: request.DomainRef},
		)
		if len(records) != 1 || records[0].Job.JobRef != first.JobRef {
			t.Fatalf("records=%+v first=%+v", records, first)
		}

		conflict := request
		conflict.Objective = "objetivo distinto con la misma clave idempotente"
		conflictJob, err := store.CreateDomainWorkJobV0(context.Background(), conflict)
		if err != nil {
			t.Fatalf("conflict CreateDomainWorkJobV0: %v", err)
		}
		if conflictJob.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
			len(conflictJob.Issues) == 0 {
			t.Fatalf("conflictJob=%+v", conflictJob)
		}
		records = listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: request.DomainRef},
		)
		if len(records) != 1 || records[0].Job.JobRef != first.JobRef {
			t.Fatalf("records after conflict=%+v first=%+v", records, first)
		}

		invalid := validDomainWorkContractJobRequestV0("invalid")
		invalid.DomainRef = ""
		invalidJob, err := store.CreateDomainWorkJobV0(context.Background(), invalid)
		if err != nil {
			t.Fatalf("invalid CreateDomainWorkJobV0: %v", err)
		}
		if invalidJob.Status != orquestadomainwork.DomainWorkStatusInvalidV0 {
			t.Fatalf("invalidJob=%+v", invalidJob)
		}
		records = listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{IdempotencyKey: invalid.IdempotencyKey},
		)
		if len(records) != 0 {
			t.Fatalf("invalid request persisted records=%+v", records)
		}
	})

	t.Run("list_records_filters_limit_and_order", func(t *testing.T) {
		store := newDomainWorkRecordStoreForContractV0(t, newStore)
		reqA := validDomainWorkContractJobRequestV0("a")
		reqA.DomainRef = "domain-a"
		reqA.WorkKind = "draft_content_block"
		reqA.IdempotencyKey = "idem-shared"
		reqA.CorrelationID = "corr-a"
		reqA.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-a"},
			{Kind: "app_ref", Ref: "app-a"},
		}
		jobA := createDomainWorkContractJobForTestV0(t, store, reqA)

		reqB := validDomainWorkContractJobRequestV0("b")
		reqB.DomainRef = "domain-a"
		reqB.WorkKind = "generate_visual_asset"
		reqB.IdempotencyKey = "idem-b"
		reqB.CorrelationID = "corr-b"
		reqB.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-a"},
			{Kind: "app_ref", Ref: "app-b"},
		}
		jobB := createDomainWorkContractJobForTestV0(t, store, reqB)

		reqC := validDomainWorkContractJobRequestV0("c")
		reqC.DomainRef = "domain-b"
		reqC.WorkKind = "draft_content_block"
		reqC.IdempotencyKey = "idem-shared"
		reqC.CorrelationID = "corr-c"
		reqC.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-b"},
			{Kind: "app_ref", Ref: "app-a"},
		}
		jobC := createDomainWorkContractJobForTestV0(t, store, reqC)

		all := listDomainWorkContractRecordsForTestV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{})
		allRefs := domainWorkContractRecordJobRefsV0(all)
		if len(allRefs) != 3 || !sort.StringsAreSorted(allRefs) {
			t.Fatalf("allRefs=%+v all=%+v", allRefs, all)
		}

		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			DomainRef: "domain-a",
		}, 2)
		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			WorkKind: "draft_content_block",
		}, 2)
		assertDomainWorkContractRecordJobRefV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			JobRef: jobB.JobRef,
		}, jobB.JobRef)
		assertDomainWorkContractRecordJobRefV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			CorrelationID: "corr-c",
		}, jobC.JobRef)
		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			IdempotencyKey: "idem-shared",
		}, 2)
		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			Status: orquestadomainwork.DomainWorkStatusAcceptedV0,
		}, 3)
		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{Kind: "run_ref", Ref: "run-a"}},
		}, 2)
		assertDomainWorkContractRecordCountV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{Kind: "run_ref", Ref: "app-a"}},
		}, 0)
		assertDomainWorkContractRecordJobRefV0(t, store, orquestadomainwork.DomainWorkJobRecordFilterV0{
			DomainRef: "domain-a",
			WorkKind:  "draft_content_block",
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
				Kind: "run_ref",
				Ref:  "run-a",
			}},
		}, jobA.JobRef)

		limited := listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{Limit: 2},
		)
		limitedRefs := domainWorkContractRecordJobRefsV0(limited)
		if len(limitedRefs) != 2 || limitedRefs[0] != allRefs[0] || limitedRefs[1] != allRefs[1] {
			t.Fatalf("limitedRefs=%+v allRefs=%+v", limitedRefs, allRefs)
		}
	})

	t.Run("list_records_returns_defensive_copies", func(t *testing.T) {
		store := newDomainWorkRecordStoreForContractV0(t, newStore)
		request := validDomainWorkContractJobRequestV0("copy")
		createDomainWorkContractJobForTestV0(t, store, request)

		records := listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: request.DomainRef},
		)
		if len(records) != 1 ||
			len(records[0].Request.InputFields) == 0 ||
			len(records[0].Request.InputFields[0].Values) == 0 ||
			len(records[0].Request.InputFields[0].ValueJSON) == 0 ||
			len(records[0].Request.ExternalRefs) == 0 ||
			len(records[0].Job.EvidenceRefs) == 0 {
			t.Fatalf("record no preparado para copia defensiva: %+v", records)
		}
		records[0].Request.InputFields[0].Values[0] = "mutado"
		records[0].Request.InputFields[0].ValueJSON[0] = '['
		records[0].Request.ExternalRefs[0].Ref = "mutado"
		records[0].Job.EvidenceRefs[0] = "mutado"

		again := listDomainWorkContractRecordsForTestV0(
			t,
			store,
			orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: request.DomainRef},
		)
		if again[0].Request.InputFields[0].Values[0] != "es" ||
			string(again[0].Request.InputFields[0].ValueJSON) != `{"kind":"contract"}` ||
			again[0].Request.ExternalRefs[0].Ref != "run-copy" ||
			again[0].Job.EvidenceRefs[0] != "evidence-copy" {
			t.Fatalf("again=%+v", again)
		}
	})

	t.Run("context_cancelled", func(t *testing.T) {
		store := newDomainWorkRecordStoreForContractV0(t, newStore)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := store.CreateDomainWorkJobV0(ctx, validDomainWorkContractJobRequestV0("cancel"))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("create err=%v", err)
		}
		records, err := store.ListDomainWorkJobRecordsV0(ctx, orquestadomainwork.DomainWorkJobRecordFilterV0{})
		if !errors.Is(err, context.Canceled) || records != nil {
			t.Fatalf("records=%+v err=%v", records, err)
		}
	})
}

func newDomainWorkRecordStoreForContractV0(
	t testing.TB,
	newStore DomainWorkJobRecordStoreFactoryV0,
) orquestadomainwork.DomainWorkJobRecordStorePortV0 {
	t.Helper()
	if newStore == nil {
		t.Fatalf("newStore nil")
	}
	store := newStore(t)
	if store == nil {
		t.Fatalf("store nil")
	}
	return store
}

func validDomainWorkContractJobRequestV0(suffix string) orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "request-ref-contract-" + suffix,
		CorrelationID:  "corr-contract-" + suffix,
		IdempotencyKey: "idem-contract-" + suffix,
		RequestedBy:    "contract-test",
		DomainRef:      "domain-contract-" + suffix,
		InterfaceRefs:  []string{"domain-work.v0"},
		WorkKind:       "generate_content_package",
		WorkRefs:       []string{"scope-contract-" + suffix},
		Objective:      "Crear trabajo de contrato verificable.",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{{
			Name:      "language_code",
			Values:    []string{"es"},
			ValueJSON: []byte(`{"kind":"contract"}`),
		}},
		InputRefs: []string{"source-contract-" + suffix},
		Constraints: []string{
			"sin_placeholder",
		},
		AcceptanceCriteria: []string{
			"Entrega trazable.",
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  "run-" + suffix,
		}},
		EvidenceRefs: []string{"evidence-" + suffix},
	}
}

func createDomainWorkContractJobForTestV0(
	t testing.TB,
	store orquestadomainwork.DomainWorkJobRecordStorePortV0,
	request orquestadomainwork.DomainWorkJobRequestV0,
) orquestadomainwork.DomainWorkJobV0 {
	t.Helper()
	job, err := store.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 || job.JobRef == "" {
		t.Fatalf("job=%+v", job)
	}
	return job
}

func listDomainWorkContractRecordsForTestV0(
	t testing.TB,
	store orquestadomainwork.DomainWorkJobRecordStorePortV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) []orquestadomainwork.DomainWorkJobRecordV0 {
	t.Helper()
	records, err := store.ListDomainWorkJobRecordsV0(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListDomainWorkJobRecordsV0: %v", err)
	}
	return records
}

func assertDomainWorkContractRecordCountV0(
	t testing.TB,
	store orquestadomainwork.DomainWorkJobRecordStorePortV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
	want int,
) {
	t.Helper()
	records := listDomainWorkContractRecordsForTestV0(t, store, filter)
	if len(records) != want {
		t.Fatalf("len(records)=%d want %d filter=%+v records=%+v", len(records), want, filter, records)
	}
}

func assertDomainWorkContractRecordJobRefV0(
	t testing.TB,
	store orquestadomainwork.DomainWorkJobRecordStorePortV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
	want string,
) {
	t.Helper()
	records := listDomainWorkContractRecordsForTestV0(t, store, filter)
	if len(records) != 1 || records[0].Job.JobRef != want {
		t.Fatalf("records=%+v want job_ref=%q filter=%+v", records, want, filter)
	}
}

func domainWorkContractRecordJobRefsV0(
	records []orquestadomainwork.DomainWorkJobRecordV0,
) []string {
	refs := make([]string, 0, len(records))
	for _, record := range records {
		refs = append(refs, record.Job.JobRef)
	}
	return refs
}
