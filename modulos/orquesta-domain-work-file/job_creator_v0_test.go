package orquestadomainworkfile_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
