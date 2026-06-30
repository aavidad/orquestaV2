package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestMCPCodebaseQueryTransportV0RegistradoYDelegado(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		CodebaseQuery: MCPCodebaseQueryToolExecutorV0{
			Broker: &fakeMCPCodeContextBrokerV0{},
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw, err := transport.CallToolV0(context.Background(), MCPCodebaseQueryToolNameV0, validMCPCodebaseQueryInputTestV0())
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var result MCPCodebaseQueryToolResultV0
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, raw)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.ProviderKind != orquestacontext.CodeContextProviderKindFallbackRGV0 {
		t.Fatalf("provider=%q", result.ProviderKind)
	}
}

func TestMCPCodebaseQueryInputSchemaV0ExponeCamposYEnum(t *testing.T) {
	fields, ok := MCPTransportToolInputFieldsV0(MCPCodebaseQueryToolNameV0)
	if !ok {
		t.Fatalf("schema no encontrado")
	}
	required := map[string]bool{}
	enums := map[string][]string{}
	for _, field := range fields {
		required[field.Name] = field.Required
		enums[field.Name] = field.Enum
	}
	if !required["repository_ref"] || !required["query"] {
		t.Fatalf("required=%+v", required)
	}
	if len(enums["query_kind"]) == 0 {
		t.Fatalf("query_kind sin enum: %+v", fields)
	}
}

func TestMCPCodebaseQueryToolExecutorV0RellenaRequestedByPorDefecto(t *testing.T) {
	broker := &fakeMCPCodeContextBrokerV0{}
	input := validMCPCodebaseQueryInputTestV0()
	input.RequestedBy = ""

	if _, err := (MCPCodebaseQueryToolExecutorV0{Broker: broker}).Execute(context.Background(), input); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if broker.last.RequestedBy != "orquesta-mcp-codebase-query" {
		t.Fatalf("requested_by=%q", broker.last.RequestedBy)
	}
}

type fakeMCPCodeContextBrokerV0 struct {
	last orquestacontext.CodeContextQueryV0
}

func (fake *fakeMCPCodeContextBrokerV0) QueryCodeContextV0(
	_ context.Context,
	query orquestacontext.CodeContextQueryV0,
) (orquestacontext.CodeContextResultV0, error) {
	fake.last = query
	return orquestacontext.CodeContextResultV0{
		SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
		Estado:        orquestacontext.CodeContextEstadoOKV0,
		RequestRef:    query.RequestRef,
		RepositoryRef: query.RepositoryRef,
		QueryKind:     query.QueryKind,
		ProviderKind:  orquestacontext.CodeContextProviderKindFallbackRGV0,
		IndexerPolicy: orquestacontext.CodeContextProviderPolicyCentralOnlyV0,
		Results: []orquestacontext.CodeContextHitV0{{
			HitRef:  "hit-ref-test",
			Path:    "modulos/orquesta-mcp/codebase_query_tool_v0.go",
			Summary: "tool codebase query",
		}},
	}, nil
}

func validMCPCodebaseQueryInputTestV0() MCPCodebaseQueryToolInputV0 {
	return MCPCodebaseQueryToolInputV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RequestRef:    "request-ref-mcp-codebase-test",
		RepositoryRef: "repo-ref-orquesta",
		QueryKind:     orquestacontext.CodeContextQueryKindSearchV0,
		Query:         "codebase query",
		MaxResults:    3,
		MaxBytes:      3000,
	}
}
