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

		requiredTestConflict := request
		requiredTestConflict.RequiredTests = []orquestadomainwork.DomainWorkRequiredTestV0{{
			TestRef: "domain-test-ref-contract-conflict",
		}}
		requiredTestJob, err := store.CreateDomainWorkJobV0(
			context.Background(),
			requiredTestConflict,
		)
		if err != nil {
			t.Fatalf("required tests conflict CreateDomainWorkJobV0: %v", err)
		}
		if requiredTestJob.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
			len(requiredTestJob.Issues) == 0 {
			t.Fatalf("requiredTestJob=%+v", requiredTestJob)
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
