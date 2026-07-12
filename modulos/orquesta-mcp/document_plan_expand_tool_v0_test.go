package orquestamcp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestMCPDocumentPlanExpandDescriptorV0DeclaraContrato(t *testing.T) {
	descriptor := MCPDocumentPlanExpandDescriptorV0()
	if descriptor.Name != MCPDocumentPlanExpandToolNameV0 ||
		descriptor.Version != MCPDocumentPlanExpandToolVersionV0 ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
}

func TestMCPDocumentPlanExpandExecutorV0PreviewDelegaEnCore(t *testing.T) {
	executor := MCPDocumentPlanExpandToolExecutorV0{}
	input := documentPlanExpandInputForTestV0(MCPDocumentPlanExpandActionPreviewV0)

	result, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoOKV0 ||
		result.Expansion == nil ||
		len(result.Expansion.Jobs) != 1 ||
		result.Expansion.Jobs[0].WorkKind != "draft_content_block" ||
		result.RequestID != "req-document-plan-expand-001" ||
		result.CorrelationID != "corr-document-plan-expand-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDocumentPlanExpandExecutorV0CreateJobsUsaPuerto(t *testing.T) {
	creator := &fakeMCPDocumentPlanExpandCreatorV0{}
	executor := NewMCPDocumentPlanExpandToolExecutorV0(creator)

	result, err := executor.Execute(context.Background(), documentPlanExpandInputForTestV0(MCPDocumentPlanExpandActionCreateJobsV0))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoOKV0 ||
		result.Creation == nil ||
		len(result.Creation.CreatedJobs) != 1 ||
		creator.calls != 1 ||
		creator.requests[0].CorrelationID != "corr-document-plan-expand-001" {
		t.Fatalf("result=%+v creator=%+v", result, creator)
	}
}

func TestMCPDocumentPlanExpandExecutorV0CreateJobsRequierePuertoAntesDeValidarPlan(t *testing.T) {
	input := MCPDocumentPlanExpandToolInputV0{Action: MCPDocumentPlanExpandActionCreateJobsV0}
	result, err := (MCPDocumentPlanExpandToolExecutorV0{}).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoErrorV0 || len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPDocumentPlanExpandCreatorUnavailableV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDocumentPlanExpandExecutorV0RechazaMasDeCuarentaJobs(t *testing.T) {
	plan := validDocumentPlanForMCPExpandTestV0()
	plan.Sections = make([]orquestadomainwork.DomainDocumentPlanSectionV0, 41)
	for index := range plan.Sections {
		plan.Sections[index] = orquestadomainwork.DomainDocumentPlanSectionV0{
			SectionRef:     fmt.Sprintf("section-%02d", index),
			Order:          index + 1,
			Title:          fmt.Sprintf("Section %d", index),
			Objective:      "Redactar contenido valido.",
			WorkKind:       "redaccion_tema",
			TargetWordsMin: 10,
			TargetWordsMax: 20,
			SourceRefs:     []string{"source-001"},
		}
	}
	input := documentPlanExpandInputForTestV0(MCPDocumentPlanExpandActionCreateJobsV0)
	input.ExpansionRequest.Plan = plan
	creator := &fakeMCPDocumentPlanExpandCreatorV0{}

	result, err := (MCPDocumentPlanExpandToolExecutorV0{JobCreator: creator}).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoErrorV0 || len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPDocumentPlanExpandJobLimitExceededV0 || creator.calls != 0 ||
		result.Creation != nil || result.Expansion != nil {
		t.Fatalf("result=%+v creator=%+v", result, creator)
	}
}

func TestMCPDocumentPlanExpandExecutorV0PreviewRechazaMasDeCuarentaJobs(t *testing.T) {
	plan := validDocumentPlanForMCPExpandTestV0()
	plan.Sections = make([]orquestadomainwork.DomainDocumentPlanSectionV0, 41)
	for index := range plan.Sections {
		plan.Sections[index] = orquestadomainwork.DomainDocumentPlanSectionV0{
			SectionRef: fmt.Sprintf("preview-section-%02d", index), Order: index + 1,
			Title: "Section", Objective: "Redactar contenido valido.", WorkKind: "redaccion_tema",
			TargetWordsMin: 10, TargetWordsMax: 20, SourceRefs: []string{"source-001"},
		}
	}
	input := documentPlanExpandInputForTestV0(MCPDocumentPlanExpandActionPreviewV0)
	input.ExpansionRequest.Plan = plan

	result, err := (MCPDocumentPlanExpandToolExecutorV0{}).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoErrorV0 || result.Expansion != nil ||
		len(result.Errores) != 1 || result.Errores[0].Code != MCPDocumentPlanExpandJobLimitExceededV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDocumentPlanExpandExecutorV0ConservaJobsCreadosAnteFalloParcial(t *testing.T) {
	creator := &fakeMCPDocumentPlanExpandCreatorV0{errAt: 1, err: errors.New("creator unavailable")}
	plan := validDocumentPlanForMCPExpandTestV0()
	plan.Sections = append(plan.Sections, orquestadomainwork.DomainDocumentPlanSectionV0{
		SectionRef: "section-002", Order: 2, Title: "Dos", Objective: "Redactar segundo contenido.",
		WorkKind: "redaccion_tema", TargetWordsMin: 10, TargetWordsMax: 20, SourceRefs: []string{"source-001"},
	})
	input := documentPlanExpandInputForTestV0(MCPDocumentPlanExpandActionCreateJobsV0)
	input.ExpansionRequest.Plan = plan

	result, err := (MCPDocumentPlanExpandToolExecutorV0{JobCreator: creator}).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDocumentPlanExpandEstadoErrorV0 || result.Creation == nil ||
		len(result.Creation.CreatedJobs) != 1 || creator.calls != 2 || len(result.Errores) == 0 ||
		result.Errores[len(result.Errores)-1].Code != MCPDocumentPlanExpandCreatorUnavailableV0 {
		t.Fatalf("result=%+v creator=%+v", result, creator)
	}
}

func documentPlanExpandInputForTestV0(action string) MCPDocumentPlanExpandToolInputV0 {
	return MCPDocumentPlanExpandToolInputV0{
		RequestID:     "req-document-plan-expand-001",
		CorrelationID: "corr-document-plan-expand-001",
		Action:        action,
		ExpansionRequest: orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{
			Plan: validDocumentPlanForMCPExpandTestV0(),
		},
	}
}

func validDocumentPlanForMCPExpandTestV0() orquestadomainwork.DomainDocumentPlanV0 {
	return orquestadomainwork.DomainDocumentPlanV0{
		PlanRef: "plan-001", DomainRef: "domain-demo", WorkKind: "plan_temario", DocumentKind: "documento_formativo",
		ScopeRef: "scope-001", LanguageCode: "es", Title: "Plan", Objective: "Planificar contenido.", TargetAudience: "Estudiantes",
		EstimatedPagesMin: 1, EstimatedPagesMax: 2,
		Sections: []orquestadomainwork.DomainDocumentPlanSectionV0{{
			SectionRef: "section-001", Order: 1, Title: "Uno", Objective: "Redactar contenido.", WorkKind: "redaccion_tema",
			TargetWordsMin: 10, TargetWordsMax: 20, SourceRefs: []string{"source-001"},
		}},
		Deliverables:    []orquestadomainwork.DomainDocumentPlanDeliverableV0{{DeliverableRef: "deliverable-001", ArtifactType: "content_block", Title: "Contenido", Required: true}},
		QualityCriteria: []string{"calidad"},
		SourceRefs:      []string{"source-001"},
	}
}

type fakeMCPDocumentPlanExpandCreatorV0 struct {
	requests []orquestadomainwork.DomainWorkJobRequestV0
	calls    int
	errAt    int
	err      error
}

func (fake *fakeMCPDocumentPlanExpandCreatorV0) CreateDomainWorkJobV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	call := fake.calls
	fake.calls++
	fake.requests = append(fake.requests, request)
	if fake.err != nil && call == fake.errAt {
		return orquestadomainwork.DomainWorkJobV0{}, fake.err
	}
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         "job-" + request.RequestID,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
	}, nil
}
