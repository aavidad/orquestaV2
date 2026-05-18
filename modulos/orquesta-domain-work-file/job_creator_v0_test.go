package orquestadomainworkfile_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestadomainworkcontracttest "orquesta/modulos/orquesta-domain-work/contracttest"
)

func TestFileDomainWorkJobCreatorV0CumpleContratoRecordStore(t *testing.T) {
	orquestadomainworkcontracttest.RunDomainWorkJobRecordStoreContractV0(
		t,
		func(t testing.TB) orquestadomainwork.DomainWorkJobRecordStorePortV0 {
			t.Helper()
			creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(t.TempDir())
			if err != nil {
				t.Fatalf("NewFileDomainWorkJobCreatorV0: %v", err)
			}
			return creator
		},
	)
}

func TestFileDomainWorkJobCreatorV0ReplayTrasReinstanciarDevuelveMismoJob(t *testing.T) {
	dir := t.TempDir()
	creator := mustNewFileDomainWorkJobCreatorV0(t, dir)
	request := validDomainWorkFileJobRequestV0()

	first, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
		first.JobRef == "" ||
		first.DomainRef != "dominio-demo" {
		t.Fatalf("first=%+v", first)
	}

	reopened := mustNewFileDomainWorkJobCreatorV0(t, dir)
	retry := request
	retry.RequestID = "req-domain-work-file-retry"
	second, err := reopened.CreateDomainWorkJobV0(context.Background(), retry)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.JobRef != first.JobRef ||
		second.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if jobs := listDomainWorkFileJobsForTestV0(t, reopened); len(jobs) != 1 || jobs[0].JobRef != first.JobRef {
		t.Fatalf("jobs=%+v first=%+v", jobs, first)
	}
}

func TestFileDomainWorkJobCreatorV0ConflictoIdempotenciaNoSobrescribeArchivo(t *testing.T) {
	dir := t.TempDir()
	creator := mustNewFileDomainWorkJobCreatorV0(t, dir)
	request := validDomainWorkFileJobRequestV0()

	first, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	conflict := request
	conflict.Objective = "Crear otro paquete con la misma clave."
	second, err := creator.CreateDomainWorkJobV0(context.Background(), conflict)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		len(second.Issues) != 1 ||
		second.Issues[0].Code != orquestadomainworkfile.ErrDomainWorkFileIdempotencyConflictV0 {
		t.Fatalf("second=%+v", second)
	}

	reopened := mustNewFileDomainWorkJobCreatorV0(t, dir)
	jobs := listDomainWorkFileJobsForTestV0(t, reopened)
	if len(jobs) != 1 ||
		jobs[0].JobRef != first.JobRef ||
		jobs[0].WorkKind != first.WorkKind {
		t.Fatalf("jobs=%+v first=%+v", jobs, first)
	}
}

func TestFileDomainWorkJobCreatorV0RequestInvalidoNoEscribeEstado(t *testing.T) {
	dir := t.TempDir()
	creator := mustNewFileDomainWorkJobCreatorV0(t, dir)
	request := validDomainWorkFileJobRequestV0()
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
	if jobs := listDomainWorkFileJobsForTestV0(t, creator); len(jobs) != 0 {
		t.Fatalf("jobs=%+v", jobs)
	}
	if _, err := os.Stat(creator.SnapshotPathV0()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot exists or unexpected err: %v", err)
	}
}

func TestFileDomainWorkJobCreatorV0EscribeSnapshotEstructurado(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	job, err := creator.CreateDomainWorkJobV0(context.Background(), validDomainWorkFileJobRequestV0())
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	data, err := os.ReadFile(creator.SnapshotPathV0())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var snapshot struct {
		SchemaVersion string `json:"schema_version"`
		Records       []struct {
			DomainRef      string `json:"domain_ref"`
			IdempotencyKey string `json:"idempotency_key"`
			Fingerprint    string `json:"fingerprint"`
			Job            struct {
				JobRef string `json:"job_ref"`
			} `json:"job"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if snapshot.SchemaVersion != orquestadomainworkfile.DomainWorkFileJobCreatorSnapshotSchemaV0 ||
		len(snapshot.Records) != 1 ||
		snapshot.Records[0].DomainRef != "dominio-demo" ||
		snapshot.Records[0].IdempotencyKey != "domain-work-file-idem-001" ||
		snapshot.Records[0].Fingerprint == "" ||
		snapshot.Records[0].Job.JobRef != job.JobRef {
		t.Fatalf("snapshot=%+v job=%+v", snapshot, job)
	}
}

func TestFileDomainWorkJobCreatorV0CorrupcionLecturaFallaSinMutar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domain_work_jobs_v0.json")
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(dir); err == nil {
		t.Fatalf("expected corruption error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{` {
		t.Fatalf("corrupt snapshot mutated: %q", data)
	}
}

func TestFileDomainWorkJobCreatorV0RespetaContextoCancelado(t *testing.T) {
	creator := mustNewFileDomainWorkJobCreatorV0(t, t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := creator.CreateDomainWorkJobV0(ctx, validDomainWorkFileJobRequestV0())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if jobs := listDomainWorkFileJobsForTestV0(t, creator); len(jobs) != 0 {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func TestFileDomainWorkJobCreatorV0RequiereDirectorioAbsoluto(t *testing.T) {
	if _, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0("relative"); err == nil {
		t.Fatalf("expected relative dir error")
	}
}

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

	records = listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{JobRef: jobB.JobRef},
	)
	if len(records) != 1 || records[0].Job.JobRef != jobB.JobRef {
		t.Fatalf("records=%+v jobB=%+v", records, jobB)
	}

	records = listDomainWorkFileRecordsForTestV0(
		t,
		creator,
		orquestadomainwork.DomainWorkJobRecordFilterV0{CorrelationID: "corr-c"},
	)
	if len(records) != 1 || records[0].Job.JobRef != jobC.JobRef {
		t.Fatalf("records=%+v jobC=%+v", records, jobC)
	}

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

func validDomainWorkFileJobRequestV0() orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:      "req-domain-work-file-001",
		CorrelationID:  "corr-domain-work-file-001",
		IdempotencyKey: "domain-work-file-idem-001",
		RequestedBy:    "test",
		DomainRef:      "dominio-demo",
		InterfaceRefs:  []string{"domain-work.v0"},
		WorkKind:       "generate_content_package",
		WorkRefs:       []string{"scope-001"},
		Objective:      "Crear un paquete de contenido verificable.",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{{
			Name:  "language_code",
			Value: "es",
		}},
		InputRefs: []string{"source-001"},
		Constraints: []string{
			"sin_placeholder",
		},
		AcceptanceCriteria: []string{
			"Entrega trazable.",
		},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  "run-001",
		}},
		EvidenceRefs: []string{"evidence-001"},
	}
}

func mustNewFileDomainWorkJobCreatorV0(
	t *testing.T,
	dir string,
) *orquestadomainworkfile.FileDomainWorkJobCreatorV0 {
	t.Helper()
	creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(dir)
	if err != nil {
		t.Fatalf("NewFileDomainWorkJobCreatorV0: %v", err)
	}
	return creator
}

func listDomainWorkFileJobsForTestV0(
	t *testing.T,
	creator *orquestadomainworkfile.FileDomainWorkJobCreatorV0,
) []orquestadomainwork.DomainWorkJobV0 {
	t.Helper()
	jobs, err := creator.ListDomainWorkJobsV0(context.Background())
	if err != nil {
		t.Fatalf("ListDomainWorkJobsV0: %v", err)
	}
	return jobs
}

func createDomainWorkFileJobForTestV0(
	t *testing.T,
	creator *orquestadomainworkfile.FileDomainWorkJobCreatorV0,
	request orquestadomainwork.DomainWorkJobRequestV0,
) orquestadomainwork.DomainWorkJobV0 {
	t.Helper()
	job, err := creator.CreateDomainWorkJobV0(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 || job.JobRef == "" {
		t.Fatalf("job=%+v", job)
	}
	return job
}

func listDomainWorkFileRecordsForTestV0(
	t *testing.T,
	creator *orquestadomainworkfile.FileDomainWorkJobCreatorV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) []orquestadomainwork.DomainWorkJobRecordV0 {
	t.Helper()
	records, err := creator.ListDomainWorkJobRecordsV0(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListDomainWorkJobRecordsV0: %v", err)
	}
	return records
}
