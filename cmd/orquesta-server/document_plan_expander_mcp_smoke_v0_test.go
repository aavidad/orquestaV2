package main

import (
	"context"
	"encoding/json"
	"testing"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestDocumentPlanExpandMCPRealHTTPSmokeV0(t *testing.T) {
	creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(t.TempDir())
	if err != nil {
		t.Fatalf("file creator: %v", err)
	}
	domainWork := orquestamcp.NewMCPDomainWorkToolExecutorV0(creator, creator)
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{
		DomainWork:         domainWork,
		DocumentPlanExpand: serverDocumentPlanExpandExecutorV0(domainWork),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	if mcpToolByNameTestV0(tools.Tools, orquestamcp.MCPDocumentPlanExpandToolNameV0) == nil {
		t.Fatalf("tools/list no publica document plan expand: %+v", tools.Tools)
	}

	preview := callDocumentPlanExpandMCPV0(t, server.URL, orquestamcp.MCPDocumentPlanExpandActionPreviewV0)
	if preview.Estado != orquestamcp.MCPDocumentPlanExpandEstadoOKV0 || preview.Expansion == nil || len(preview.Expansion.Jobs) != 4 {
		t.Fatalf("preview=%+v", preview)
	}
	first := callDocumentPlanExpandMCPV0(t, server.URL, orquestamcp.MCPDocumentPlanExpandActionCreateJobsV0)
	if first.Estado != orquestamcp.MCPDocumentPlanExpandEstadoOKV0 || first.Creation == nil || len(first.Creation.CreatedJobs) != 4 {
		t.Fatalf("first=%+v", first)
	}
	second := callDocumentPlanExpandMCPV0(t, server.URL, orquestamcp.MCPDocumentPlanExpandActionCreateJobsV0)
	if second.Estado != orquestamcp.MCPDocumentPlanExpandEstadoOKV0 || second.Creation == nil || len(second.Creation.CreatedJobs) != 4 {
		t.Fatalf("second=%+v", second)
	}
	for idx, job := range first.Creation.CreatedJobs {
		if second.Creation.CreatedJobs[idx].JobRef != job.JobRef {
			t.Fatalf("replay idx=%d got=%s want=%s", idx, second.Creation.CreatedJobs[idx].JobRef, job.JobRef)
		}
	}
	jobs, err := creator.ListDomainWorkJobsV0(context.Background())
	if err != nil || len(jobs) != 4 {
		t.Fatalf("persisted jobs=%+v err=%v", jobs, err)
	}
}

func callDocumentPlanExpandMCPV0(t *testing.T, baseURL string, action string) orquestamcp.MCPDocumentPlanExpandToolResultV0 {
	t.Helper()
	var call mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, baseURL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": orquestamcp.MCPDocumentPlanExpandToolNameV0,
		"arguments": orquestamcp.MCPDocumentPlanExpandToolInputV0{
			RequestID:        "request-ref-document-plan-expand-mcp-smoke",
			CorrelationID:    "corr-document-plan-expand-mcp-smoke",
			Action:           action,
			ExpansionRequest: documentPlanExpandMCPRequestV0(),
		},
	}, &call)
	if call.IsError || len(call.Content) != 1 {
		t.Fatalf("call=%+v", call)
	}
	var result orquestamcp.MCPDocumentPlanExpandToolResultV0
	if err := json.Unmarshal([]byte(call.Content[0].Text), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return result
}

func documentPlanExpandMCPRequestV0() orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0 {
	return orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0{
		Plan: orquestadomainwork.DomainDocumentPlanV0{
			PlanRef: "plan-document-plan-expand-mcp-smoke", DomainRef: "domain-mcp-smoke", WorkKind: orquestadomainwork.DomainWorkKindPlanDocumentV0,
			DocumentKind: "guide", ScopeRef: "scope-mcp-smoke", LanguageCode: "es", Title: "Plan MCP", Objective: "Ejercitar la expansión durable.", TargetAudience: "editores",
			Sections: []orquestadomainwork.DomainDocumentPlanSectionV0{
				{SectionRef: "section-one", Order: 1, Title: "Uno", Objective: "Redactar uno.", WorkKind: "draft_content_block", SourceRefs: []string{"source-mcp-smoke"}},
				{SectionRef: "section-two", Order: 2, Title: "Dos", Objective: "Redactar dos.", WorkKind: "draft_content_block", SourceRefs: []string{"source-mcp-smoke"}},
			},
			Visuals:      []orquestadomainwork.DomainDocumentPlanVisualV0{{VisualRef: "visual-one", VisualType: "diagram", PlacementRef: "section-one", Objective: "Crear visual.", WorkKind: "generate_visual_asset", SourceRefs: []string{"source-mcp-smoke"}}},
			ReviewSteps:  []orquestadomainwork.DomainDocumentPlanReviewV0{{ReviewRef: "review-one", Order: 1, WorkKind: "review_quality", Objective: "Revisar."}},
			Deliverables: []orquestadomainwork.DomainDocumentPlanDeliverableV0{{DeliverableRef: "deliverable-one", ArtifactType: "document", Title: "Plan MCP", Required: true}},
			SourceRefs:   []string{"source-mcp-smoke"}, EvidenceRefs: []string{"evidence-mcp-smoke"},
		},
		CorrelationID: "corr-document-plan-expand-mcp-smoke", RequestedBy: "mcp-smoke",
	}
}
