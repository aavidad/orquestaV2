package orquestamcp

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
)

func TestMCPDomainWorkToolExecutorV0ExponeJobRecordSourceSiBackendLoSoporta(t *testing.T) {
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	executor := NewMCPDomainWorkToolExecutorV0(creator, nil)
	if _, err := executor.Execute(context.Background(), MCPDomainWorkToolInputV0{
		Action: MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-mcp-job-source-001",
			CorrelationID:  "corr-mcp-job-source-001",
			IdempotencyKey: "idem-mcp-job-source-001",
			RequestedBy:    orquestadomainwork.DomainWorkDefaultRequestedByV0,
			DomainRef:      "opes",
			WorkKind:       "generate_html_site",
			Objective:      "crear job visible para reconciliacion",
		},
	}); err != nil {
		t.Fatalf("Execute create_job: %v", err)
	}

	records, err := executor.ListDomainWorkJobRecordsV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRecordFilterV0{DomainRef: "opes"},
	)
	if err != nil {
		t.Fatalf("ListDomainWorkJobRecordsV0: %v", err)
	}
	if len(records) != 1 || records[0].Job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("records=%+v", records)
	}
}
