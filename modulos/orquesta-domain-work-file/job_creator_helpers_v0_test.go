package orquestadomainworkfile_test

import (
	"context"
	"os"
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

func validDomainWorkFileArtifactSubmissionV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(
		orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "req-domain-work-file-artifact-001",
			CorrelationID:  "corr-domain-work-file-001",
			IdempotencyKey: "domain-work-file-artifact-idem-001",
			RequestedBy:    "test",
			DomainRef:      "dominio-demo",
			JobRef:         "job-ref-domain-work-file-001",
			ArtifactRef:    "artifact-ref-domain-work-file-001",
			ArtifactType:   "content_package",
			Summary:        "Artefacto local de prueba.",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "body",
				Value: "contenido",
			}},
			PayloadRefs: []string{"payload-ref-001"},
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
				Kind: "run_ref",
				Ref:  "run-001",
			}},
			EvidenceRefs: []string{"evidence-artifact-001"},
			CompleteJob:  true,
		},
	)
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

func pathExistsForDomainWorkFileTestV0(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
