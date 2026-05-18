package orquestadomainworkmemory_test

import (
	"context"
	"testing"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
)

func TestInMemoryDomainWorkJobCreatorV0IntegraDocumentPlanExpanderOffline(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	request := orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{
		Plan:          validDomainWorkMemoryDocumentPlanV0(),
		CorrelationID: "corr-document-plan-memory-001",
		RequestedBy:   "integration-test",
		InterfaceRefs: []string{"domain-work.v0"},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  "run-document-plan-001",
		}},
		EvidenceRefs: []string{"evidence-document-plan-001"},
	}

	first, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{
			JobCreator: creator,
		},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0 first: %v", err)
	}
	if first.Status != orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 ||
		len(first.Issues) != 0 ||
		len(first.RequestedJobs) != 5 ||
		len(first.CreatedJobs) != 5 {
		t.Fatalf("first=%+v", first)
	}
	refsByIdempotency := map[string]string{}
	for i, job := range first.CreatedJobs {
		requested := first.RequestedJobs[i]
		if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
			job.JobRef == "" ||
			job.DomainRef != requested.DomainRef ||
			job.WorkKind != requested.WorkKind ||
			job.CorrelationID != requested.CorrelationID ||
			job.IdempotencyKey != requested.IdempotencyKey ||
			len(job.ExternalRefs) == 0 ||
			len(job.EvidenceRefs) == 0 {
			t.Fatalf("job[%d]=%+v requested=%+v", i, job, requested)
		}
		refsByIdempotency[job.IdempotencyKey] = job.JobRef
	}

	second, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{
			JobCreator: creator,
		},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0 second: %v", err)
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
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != len(first.CreatedJobs) {
		t.Fatalf("jobs=%+v first=%+v", jobs, first.CreatedJobs)
	}
}

func TestInMemoryDomainWorkJobCreatorV0NoGuardaJobsSiPlanInvalido(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	plan := validDomainWorkMemoryDocumentPlanV0()
	plan.Sections = nil

	result, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{Plan: plan},
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{
			JobCreator: creator,
		},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if result.Status != orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationStatusInvalidV0 ||
		len(result.Issues) == 0 ||
		len(result.CreatedJobs) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if jobs := listDomainWorkMemoryJobsForTestV0(t, creator); len(jobs) != 0 {
		t.Fatalf("jobs=%+v", jobs)
	}
}

func validDomainWorkMemoryDocumentPlanV0() orquestadomainwork.DomainDocumentPlanV0 {
	return orquestadomainwork.DomainDocumentPlanV0{
		PlanRef:        "plan-doc-memory-001",
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
