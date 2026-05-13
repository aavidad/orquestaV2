package orquestaappchange

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestDomainWorkJobRequestFromAppChangeV0ConvierteExternalWork(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.ActorRef = "actor-ref-001"
	request.CurrentStateRefs = []string{" delivery-ref-web-001 ", "delivery-ref-web-001"}
	request.Constraints = []string{"sin cambiar contratos publicos"}
	request.MetadataRefs = []string{" evidence-ref-001 "}
	request.ExternalWork.InterfaceRefs = []string{
		" mcp-contract-ref-agenda-v0 ",
		"mcp-contract-ref-agenda-v0",
	}
	request.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{
		{Name: " topic_ref ", Value: " topic-ref-week-view "},
		{
			Name:   "source_refs",
			Values: []string{" source-ref-calendar ", "source-ref-calendar"},
		},
		{
			Name:   "source_refs",
			Values: []string{"source-ref-calendar"},
		},
	}

	job, ok := DomainWorkJobRequestFromAppChangeV0(request)

	if !ok {
		t.Fatalf("se esperaba job de domain work")
	}
	if job.SchemaVersion != orquestadomainwork.DomainWorkJobRequestSchemaV0 ||
		job.RequestID != "req-change-001" ||
		job.CorrelationID != "corr-change-001" ||
		job.IdempotencyKey != "change-ref-001" ||
		job.RequestedBy != AppChangeDefaultRequestedByV0 ||
		job.DomainRef != "project-ref-agenda" ||
		job.WorkKind != "programming" ||
		job.Objective != "Quiero mejorar la web con vista semanal." {
		t.Fatalf("job=%+v", job)
	}
	if len(job.InterfaceRefs) != 1 ||
		job.InterfaceRefs[0] != "mcp-contract-ref-agenda-v0" ||
		len(job.WorkRefs) != 2 ||
		job.WorkRefs[0] != "job-ref-agenda-week-view" ||
		job.WorkRefs[1] != "domain-work-ref-agenda-week-view" ||
		len(job.InputRefs) != 1 ||
		job.InputRefs[0] != "delivery-ref-web-001" ||
		len(job.InputFields) != 2 ||
		job.InputFields[0].Name != "topic_ref" ||
		job.InputFields[0].Value != "topic-ref-week-view" ||
		job.InputFields[1].Name != "source_refs" ||
		len(job.InputFields[1].Values) != 1 ||
		job.InputFields[1].Values[0] != "source-ref-calendar" ||
		len(job.Constraints) != 1 ||
		job.Constraints[0] != "sin cambiar contratos publicos" ||
		len(job.AcceptanceCriteria) != 1 ||
		job.AcceptanceCriteria[0] != "vista semanal visible" ||
		len(job.EvidenceRefs) != 1 ||
		job.EvidenceRefs[0] != "evidence-ref-001" {
		t.Fatalf("refs inesperadas: %+v", job)
	}
	if !hasDomainWorkExternalRefV0(job.ExternalRefs, "run_ref", "run-ref-agenda-001") ||
		!hasDomainWorkExternalRefV0(job.ExternalRefs, "app_ref", "app-ref-agenda") ||
		!hasDomainWorkExternalRefV0(job.ExternalRefs, "change_ref", "change-ref-001") ||
		!hasDomainWorkExternalRefV0(job.ExternalRefs, "actor_ref", "actor-ref-001") {
		t.Fatalf("external_refs=%+v", job.ExternalRefs)
	}
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(job); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainWorkJobRequestFromAppChangeV0SinExternalWorkNoEmiteJob(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	request.ExternalWork = nil

	job, ok := DomainWorkJobRequestFromAppChangeV0(request)

	if ok || job.SchemaVersion != "" {
		t.Fatalf("job=%+v ok=%v", job, ok)
	}
}

func TestDomainWorkJobRequestFromAppChangeV0NoMutaSolicitud(t *testing.T) {
	request := validAppChangeRequestForTestV0()
	originalWorkRef := request.ExternalWork.WorkRefs[0]
	originalJobRef := request.ExternalWork.JobRef

	job, ok := DomainWorkJobRequestFromAppChangeV0(request)
	if !ok {
		t.Fatalf("se esperaba job de domain work")
	}
	job.WorkRefs[0] = "job-ref-mutado"
	job.WorkRefs[1] = "domain-work-ref-mutado"
	job.InterfaceRefs[0] = "mcp-contract-ref-mutado"
	job.AcceptanceCriteria[0] = "criterio mutado"
	job.InputFields[0].Value = "topic-ref-mutado"
	job.InputFields[1].Values[0] = "source-ref-mutado"

	if request.ExternalWork.JobRef != originalJobRef ||
		request.ExternalWork.WorkRefs[0] != originalWorkRef ||
		request.ExternalWork.InterfaceRefs[0] != "mcp-contract-ref-agenda-v0" ||
		request.ExternalWork.InputFields[0].Value != "topic-ref-week-view" ||
		request.ExternalWork.InputFields[1].Values[0] != "source-ref-calendar" ||
		request.AcceptanceCriteria[0] != "vista semanal visible" {
		t.Fatalf("request mutada: %+v", request)
	}
}

func hasDomainWorkExternalRefV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
	kind string,
	ref string,
) bool {
	for _, item := range refs {
		if item.Kind == kind && item.Ref == ref {
			return true
		}
	}
	return false
}
