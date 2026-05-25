package orquestadomainworkcontracttest

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
