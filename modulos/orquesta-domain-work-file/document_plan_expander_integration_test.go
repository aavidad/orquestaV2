package orquestadomainworkfile_test

import (
	"context"
	"testing"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestFileDomainWorkJobCreatorV0IntegraDocumentPlanExpanderReplayTrasReinstanciar(t *testing.T) {
	dir := t.TempDir()
	creator := mustNewFileDomainWorkJobCreatorV0(t, dir)
	request := orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{
		Plan:          validDomainWorkFileDocumentPlanV0(),
		CorrelationID: "corr-document-plan-file-001",
		RequestedBy:   "integration-test",
		InterfaceRefs: []string{"domain-work.v0"},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  "run-document-plan-file-001",
		}},
		EvidenceRefs: []string{"evidence-document-plan-file-001"},
	}

	first, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{
			JobCreator: creator,
		},
	)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Status != orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 ||
		len(first.Issues) != 0 ||
		len(first.CreatedJobs) != 5 {
		t.Fatalf("first=%+v", first)
	}
	refsByIdempotency := map[string]string{}
	for _, job := range first.CreatedJobs {
		if job.JobRef == "" ||
			job.DomainRef != "dominio-demo" ||
			job.CorrelationID != "corr-document-plan-file-001" ||
			len(job.ExternalRefs) == 0 ||
			len(job.EvidenceRefs) == 0 {
			t.Fatalf("job=%+v", job)
		}
		refsByIdempotency[job.IdempotencyKey] = job.JobRef
	}

	reopened := mustNewFileDomainWorkJobCreatorV0(t, dir)
	second, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{
			JobCreator: reopened,
		},
	)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Status != orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 ||
		len(second.CreatedJobs) != len(first.CreatedJobs) {
		t.Fatalf("second=%+v", second)
	}
	for _, job := range second.CreatedJobs {
		if got, want := job.JobRef, refsByIdempotency[job.IdempotencyKey]; got != want {
			t.Fatalf("idempotency=%s got=%s want=%s", job.IdempotencyKey, got, want)
		}
	}
	if jobs := listDomainWorkFileJobsForTestV0(t, reopened); len(jobs) != len(first.CreatedJobs) {
		t.Fatalf("jobs=%+v first=%+v", jobs, first.CreatedJobs)
	}
}

func validDomainWorkFileDocumentPlanV0() orquestadomainwork.DomainDocumentPlanV0 {
	return orquestadomainwork.DomainDocumentPlanV0{
		PlanRef:        "plan-doc-file-001",
		DomainRef:      "dominio-demo",
		WorkKind:       orquestadomainwork.DomainWorkKindPlanDocumentV0,
		DocumentKind:   "manual",
		ScopeRef:       "scope-001",
		LanguageCode:   "es",
		Title:          "Manual de prueba",
		Objective:      "Planificar un documento ejecutable por trabajos de dominio.",
		TargetAudience: "Equipo editorial",
		Sections: []orquestadomainwork.DomainDocumentPlanSectionV0{{
			SectionRef: "section-001",
			Order:      1,
			Title:      "Seccion inicial",
			Objective:  "Redactar el bloque base.",
			WorkKind:   "draft_content_block",
			AcceptanceCriteria: []string{
				"Bloque revisable.",
			},
			SourceRefs: []string{"source-001"},
		}},
		Visuals: []orquestadomainwork.DomainDocumentPlanVisualV0{{
			VisualRef:    "visual-001",
			VisualType:   "diagram",
			PlacementRef: "section-001",
			Objective:    "Crear una ayuda visual.",
			WorkKind:     "generate_visual_asset",
			SourceRefs:   []string{"source-001"},
		}},
		ReviewSteps: []orquestadomainwork.DomainDocumentPlanReviewV0{
			{
				ReviewRef: "review-quality",
				Order:     1,
				WorkKind:  "review_quality",
				Objective: "Revisar calidad.",
			},
			{
				ReviewRef: "validate-document",
				Order:     2,
				WorkKind:  "validate_topic",
				Objective: "Validar el documento.",
			},
			{
				ReviewRef: "assemble-document",
				Order:     3,
				WorkKind:  "assemble_topic",
				Objective: "Ensamblar el entregable final.",
			},
		},
		Deliverables: []orquestadomainwork.DomainDocumentPlanDeliverableV0{{
			DeliverableRef: "deliverable-manual",
			ArtifactType:   "assembled_topic",
			Title:          "Manual ensamblado",
			Required:       true,
		}},
		QualityCriteria: []string{"trazabilidad"},
		Constraints:     []string{"sin_placeholder"},
		SourceRefs:      []string{"source-001"},
		EvidenceRefs:    []string{"evidence-plan-001"},
	}
}
