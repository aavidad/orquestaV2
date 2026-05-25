package orquestadomainworkfile_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
)

func TestFileDomainWorkJobCreatorV0ListRecordsFiltraConANDYLimite(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	reqA := validDomainWorkFileJobRequestV0()
	reqA.DomainRef = "dominio-a"
	reqA.WorkKind = "draft_content_block"
	reqA.IdempotencyKey = "idem-a"
	reqA.CorrelationID = "corr-a"
	reqA.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: "run-a"},
		{Kind: "app_ref", Ref: "app-a"},
	}
	jobA := createDomainWorkFileJobForTestV0(t, creator, reqA)

	reqB := validDomainWorkFileJobRequestV0()
	reqB.DomainRef = "dominio-a"
	reqB.WorkKind = "generate_visual_asset"
	reqB.IdempotencyKey = "idem-b"
	reqB.CorrelationID = "corr-b"
	reqB.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: "run-a"},
		{Kind: "app_ref", Ref: "app-b"},
	}
	jobB := createDomainWorkFileJobForTestV0(t, creator, reqB)

	reqC := validDomainWorkFileJobRequestV0()
	reqC.DomainRef = "dominio-b"
	reqC.WorkKind = "draft_content_block"
	reqC.IdempotencyKey = "idem-c"
	reqC.CorrelationID = "corr-c"
	reqC.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: "run-b"},
		{Kind: "app_ref", Ref: "app-a"},
	}
	jobC := createDomainWorkFileJobForTestV0(t, creator, reqC)

	records := listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{
			DomainRef: "dominio-a",
			WorkKind:  "draft_content_block",
			Status:    orquestadomainwork.DomainWorkStatusAcceptedV0,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
				Kind: "run_ref",
				Ref:  "run-a",
			}},
			Limit: 1,
		},
	)
	if len(records) != 1 ||
		records[0].Job.JobRef != jobA.JobRef ||
		records[0].Request.DomainRef != "dominio-a" {
		t.Fatalf("records=%+v jobA=%+v", records, jobA)
	}

	assertDomainWorkFileRecordJobRefV0(t, creator, orquestadomainwork.DomainWorkJobRecordFilterV0{
		JobRef: jobB.JobRef,
	}, jobB.JobRef)
	assertDomainWorkFileRecordJobRefV0(t, creator, orquestadomainwork.DomainWorkJobRecordFilterV0{
		CorrelationID: "corr-c",
	}, jobC.JobRef)

	records = listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
				Kind: "run_ref",
				Ref:  "app-a",
			}},
		},
	)
	if len(records) != 0 {
		t.Fatalf("external_ref no debe matchear solo por ref: %+v", records)
	}
}

func TestFileDomainWorkJobCreatorV0ListRecordsFiltroVacioEquivaleAListOrdenada(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	var refs []string
	for _, id := range []string{"c", "a", "b"} {
		request := validDomainWorkFileJobRequestV0()
		request.DomainRef = "dominio-" + id
		request.IdempotencyKey = "idem-" + id
		refs = append(refs, createDomainWorkFileJobForTestV0(t, creator, request).JobRef)
	}
	sort.Strings(refs)

	jobs := listDomainWorkFileJobsForTestV0(t, creator)
	records := listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{},
	)
	if len(records) != len(jobs) || len(records) != len(refs) {
		t.Fatalf("records=%+v jobs=%+v refs=%+v", records, jobs, refs)
	}
	for index := range records {
		if records[index].Job.JobRef != refs[index] ||
			jobs[index].JobRef != refs[index] ||
			records[index].Request.DomainRef == "" {
			t.Fatalf("index=%d records=%+v jobs=%+v refs=%+v", index, records, jobs, refs)
		}
	}

	limited := listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{Limit: 2},
	)
	if len(limited) != 2 || limited[0].Job.JobRef != refs[0] || limited[1].Job.JobRef != refs[1] {
		t.Fatalf("limited=%+v refs=%+v", limited, refs)
	}
}

func TestFileDomainWorkJobCreatorV0ListRecordsPersisteTrasReinstanciar(t *testing.T) {
	dir := t.TempDir()
	creator := mustNewFileDomainWorkJobCreatorV0(t, dir)
	request := validDomainWorkFileJobRequestV0()
	request.DomainRef = "dominio-persistente"
	request.WorkKind = "draft_content_block"
	request.IdempotencyKey = "idem-persistente"
	request.CorrelationID = "corr-persistente"
	job := createDomainWorkFileJobForTestV0(t, creator, request)

	reopened := mustNewFileDomainWorkJobCreatorV0(t, dir)
	records := listDomainWorkFileRecordsForTestV0(
		t,
		reopened,
		orquestadomainwork.DomainWorkJobRecordFilterV0{
			JobRef:         job.JobRef,
			DomainRef:      "dominio-persistente",
			IdempotencyKey: "idem-persistente",
		},
	)
	if len(records) != 1 ||
		records[0].Job.JobRef != job.JobRef ||
		records[0].Request.CorrelationID != "corr-persistente" ||
		records[0].Request.SchemaVersion != orquestadomainwork.DomainWorkJobRequestSchemaV0 {
		t.Fatalf("records=%+v job=%+v", records, job)
	}
}

func TestFileDomainWorkJobCreatorV0ListRecordsRespetaContextoCancelado(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	createDomainWorkFileJobForTestV0(t, creator, validDomainWorkFileJobRequestV0())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records, err := creator.ListDomainWorkJobRecordsV0(
		ctx,
		orquestadomainwork.DomainWorkJobRecordFilterV0{},
	)
	if !errors.Is(err, context.Canceled) || records != nil {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestFileDomainWorkJobCreatorV0ListRecordsDevuelveCopiasDefensivas(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	request := validDomainWorkFileJobRequestV0()
	request.InputFields = append(request.InputFields, orquestadomainwork.DomainWorkFieldV0{
		Name:      "payload",
		Values:    []string{"original"},
		ValueJSON: []byte(`{"k":"v"}`),
	})
	createDomainWorkFileJobForTestV0(t, creator, request)

	records := listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: "dominio-demo"},
	)
	if len(records) != 1 {
		t.Fatalf("records=%+v", records)
	}
	records[0].Request.InputFields[1].Values[0] = "mutado"
	records[0].Request.InputFields[1].ValueJSON[0] = '['
	records[0].Request.ExternalRefs[0].Ref = "mutado"
	records[0].Job.EvidenceRefs[0] = "mutado"

	again := listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: "dominio-demo"},
	)
	if again[0].Request.InputFields[1].Values[0] != "original" ||
		string(again[0].Request.InputFields[1].ValueJSON) != `{"k":"v"}` ||
		again[0].Request.ExternalRefs[0].Ref != "run-001" ||
		again[0].Job.EvidenceRefs[0] != "evidence-001" {
		t.Fatalf("again=%+v", again)
	}
}

func assertDomainWorkFileRecordJobRefV0(
	t *testing.T,
	creator *orquestadomainworkfile.FileDomainWorkJobCreatorV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
	want string,
) {
	t.Helper()
	records := listDomainWorkFileRecordsForTestV0(t, creator, filter)
	if len(records) != 1 || records[0].Job.JobRef != want {
		t.Fatalf("records=%+v want=%q filter=%+v", records, want, filter)
	}
}
