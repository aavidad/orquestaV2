package orquestadomainworkmemory_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
	orquestadomainworkcontracttest "orquesta/modulos/orquesta-domain-work/contracttest"
)

func TestInMemoryDomainWorkJobCreatorV0CumpleContratoRecordStore(t *testing.T) {
	orquestadomainworkcontracttest.RunDomainWorkJobRecordStoreContractV0(
		t,
		func(t testing.TB) orquestadomainwork.DomainWorkJobRecordStorePortV0 {
			t.Helper()
			return orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
		},
	)
}

func TestInMemoryDomainWorkJobCreatorV0CreaJobAceptado(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := validDomainWorkMemoryJobRequestV0()
	request.DomainRef = " dominio-demo "
	request.InterfaceRefs = []string{"domain-work.v0", "domain-work.v0"}

	job, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.SchemaVersion != orquestadomainwork.DomainWorkJobSchemaV0 ||
		job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		job.JobRef == "" ||
		job.DomainRef != "dominio-demo" ||
		job.WorkKind != "generate_content_package" ||
		job.CorrelationID != "corr-domain-work-memory-001" ||
		job.IdempotencyKey != "domain-work-memory-idem-001" ||
		len(job.ExternalRefs) != 1 ||
		len(job.EvidenceRefs) != 1 {
		t.Fatalf("job=%+v", job)
	}

	jobs := listDomainWorkMemoryJobsForTestV0(t, creator)
	if len(jobs) != 1 || jobs[0].JobRef != job.JobRef {
		t.Fatalf("jobs=%+v job=%+v", jobs, job)
	}
	job.ExternalRefs[0].Ref = "mutated"
	jobs = listDomainWorkMemoryJobsForTestV0(t, creator)
	if jobs[0].ExternalRefs[0].Ref == "mutated" {
		t.Fatalf("job interno mutado=%+v", jobs[0])
	}
}

func TestInMemoryDomainWorkJobCreatorV0ReplayIdempotenteDevuelveMismoJob(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := validDomainWorkMemoryJobRequestV0()

	first, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	retry := request
	retry.RequestID = "req-domain-work-memory-retry"
	second, err := creator.CreateDomainWorkJobV0(context.Background(), retry)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.JobRef != first.JobRef ||
		second.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 1 {
		t.Fatalf("jobs=%+v", jobs)
	}

	conflict := request
	conflict.Objective = "Crear otro paquete con la misma clave idempotente."
	third, err := creator.CreateDomainWorkJobV0(context.Background(), conflict)
	if err != nil {
		t.Fatalf("third: %v", err)
	}
	if third.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		len(third.Issues) != 1 ||
		third.Issues[0].Code != orquestadomainworkmemory.ErrDomainWorkMemoryIdempotencyConflictV0 {
		t.Fatalf("third=%+v", third)
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 1 || jobs[0].JobRef != first.JobRef {
		t.Fatalf("jobs=%+v first=%+v", jobs, first)
	}
}

func TestInMemoryDomainWorkJobCreatorV0RechazaRequestInvalidoSinGuardar(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := validDomainWorkMemoryJobRequestV0()
	request.DomainRef = ""

	job, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		len(job.Issues) == 0 ||
		job.Issues[0].Code != orquestadomainwork.ErrDomainWorkDomainRefRequiredV0 {
		t.Fatalf("job=%+v", job)
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 0 {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func TestInMemoryDomainWorkJobCreatorV0RespetaContextoCancelado(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := creator.CreateDomainWorkJobV0(ctx, validDomainWorkMemoryJobRequestV0())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 0 {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func TestInMemoryDomainWorkJobCreatorV0ReplayConcurrenteNoDuplica(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := validDomainWorkMemoryJobRequestV0()
	const workers = 20
	var wg sync.WaitGroup
	refs := make(chan string, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, err := creator.CreateDomainWorkJobV0(context.Background(), request)
			if err != nil {
				errs <- err
				return
			}
			refs <- job.JobRef
		}()
	}
	wg.Wait()
	close(refs)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("err=%v", err)
		}
	}
	var first string
	for ref := range refs {
		if first == "" {
			first = ref
		}
		if ref == "" || ref != first {
			t.Fatalf("ref=%q first=%q", ref, first)
		}
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 1 {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func TestInMemoryDomainWorkJobCreatorV0ListRecordsFiltraConAND(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	reqA := validDomainWorkMemoryJobRequestV0()
	reqA.DomainRef = "dominio-a"
	reqA.WorkKind = "draft_content_block"
	reqA.IdempotencyKey = "idem-a"
	reqA.CorrelationID = "corr-a"
	reqA.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{{
		Kind: "run_ref",
		Ref:  "run-a",
	}}
	jobA := createDomainWorkMemoryJobForTestV0(t, creator, reqA)

	reqB := validDomainWorkMemoryJobRequestV0()
	reqB.DomainRef = "dominio-a"
	reqB.WorkKind = "generate_visual_asset"
	reqB.IdempotencyKey = "idem-b"
	reqB.CorrelationID = "corr-b"
	reqB.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{{
		Kind: "run_ref",
		Ref:  "run-a",
	}}
	createDomainWorkMemoryJobForTestV0(t, creator, reqB)

	records := listDomainWorkMemoryRecordsForTestV0(
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
		},
	)
	if len(records) != 1 ||
		records[0].Job.JobRef != jobA.JobRef ||
		records[0].Request.CorrelationID != "corr-a" {
		t.Fatalf("records=%+v jobA=%+v", records, jobA)
	}
}

func TestInMemoryDomainWorkJobCreatorV0ListRecordsDevuelveCopiasDefensivas(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := validDomainWorkMemoryJobRequestV0()
	request.InputFields = append(request.InputFields, orquestadomainwork.DomainWorkFieldV0{
		Name:      "payload",
		Values:    []string{"original"},
		ValueJSON: []byte(`{"k":"v"}`),
	})
	createDomainWorkMemoryJobForTestV0(t, creator, request)

	records := listDomainWorkMemoryRecordsForTestV0(
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

	again := listDomainWorkMemoryRecordsForTestV0(
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

func TestInMemoryDomainWorkJobCreatorV0ListRecordsRespetaContextoCancelado(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	createDomainWorkMemoryJobForTestV0(t, creator, validDomainWorkMemoryJobRequestV0())
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
