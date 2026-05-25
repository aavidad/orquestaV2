package orquestadomainworkfile_test

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
)

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
