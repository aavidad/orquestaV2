package orquestacontext

import (
	"context"
	"testing"
)

func TestCodeContextBrokerV0UsaCacheCentralSinReinvocarProveedor(t *testing.T) {
	provider := &fakeCodeContextProviderV0{}
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:     provider,
		Cache:        NewInMemoryCodeContextCacheV0(),
		ProviderRef:  "provider-ref-rg",
		ProviderKind: CodeContextProviderKindFallbackRGV0,
	})
	query := validCodeContextQueryTestV0()

	first, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	if first.Estado != CodeContextEstadoOKV0 || first.CacheStatus != CodeContextCacheStoreV0 {
		t.Fatalf("first result inesperado: %+v", first)
	}
	second, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if second.CacheStatus != CodeContextCacheHitV0 {
		t.Fatalf("cache status=%q result=%+v", second.CacheStatus, second)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls=%d, want 1", provider.calls)
	}
}

func TestCodeContextBrokerV0BloqueaCodebaseMCPHastaOptInCentral(t *testing.T) {
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:     &fakeCodeContextProviderV0{},
		ProviderKind: CodeContextProviderKindCodebaseMCPV0,
	})
	result, err := broker.QueryCodeContextV0(context.Background(), validCodeContextQueryTestV0())
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	requireCodeContextIssueTestV0(t, result, ErrCodeContextProveedorNoConfiguradoV0)
}

func TestCodeContextBrokerV0PermiteCodebaseMCPConOptInCentralYConsulta(t *testing.T) {
	query := validCodeContextQueryTestV0()
	query.AllowExternalIndexer = true
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:               &fakeCodeContextProviderV0{},
		ProviderRef:            "provider-ref-codebase-central",
		ProviderKind:           CodeContextProviderKindCodebaseMCPV0,
		ExternalIndexerEnabled: true,
	})

	result, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.Estado != CodeContextEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.ProviderKind != CodeContextProviderKindCodebaseMCPV0 {
		t.Fatalf("provider kind=%q", result.ProviderKind)
	}
}

func TestCodeContextBrokerV0LimitaResultadosYSnippets(t *testing.T) {
	query := validCodeContextQueryTestV0()
	query.MaxResults = 1
	query.MaxBytes = 1200
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider: &fakeCodeContextProviderV0{
			hits: []CodeContextHitV0{
				{Path: "modulos/a.go", Snippet: longTextContextTestV0(1200)},
				{Path: "modulos/b.go", Snippet: "no debe salir"},
			},
		},
	})

	result, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("results=%d result=%+v", len(result.Results), result)
	}
	if len(result.Results[0].Snippet) > defaultCodeContextSnippetBytesV0+20 {
		t.Fatalf("snippet no acotado: %d", len(result.Results[0].Snippet))
	}
}

func validCodeContextQueryTestV0() CodeContextQueryV0 {
	return CodeContextQueryV0{
		SchemaVersion: CodeContextQuerySchemaVersionV0,
		RequestRef:    "request-ref-code-context-test",
		RepositoryRef: "repo-ref-orquesta",
		CommitRef:     "commit-ref-test",
		QueryKind:     CodeContextQueryKindSearchV0,
		Query:         "external work run",
		Scope:         []string{"modulos/orquesta-mcp"},
		MaxResults:    4,
		MaxBytes:      4000,
		RequestedBy:   "test",
	}
}

type fakeCodeContextProviderV0 struct {
	calls int
	hits  []CodeContextHitV0
}

func (provider *fakeCodeContextProviderV0) QueryCodeContextV0(
	_ context.Context,
	query CodeContextQueryV0,
) (CodeContextResultV0, error) {
	provider.calls++
	hits := provider.hits
	if len(hits) == 0 {
		hits = []CodeContextHitV0{{
			Kind:    "file",
			Path:    "modulos/orquesta-mcp/external_work_run_http_v0.go",
			Line:    12,
			Summary: "ruta HTTP external-work run",
			Snippet: "const MCPExternalWorkRunHTTPPathV0 = ...",
			Score:   0.9,
		}}
	}
	return CodeContextResultV0{
		Estado:      CodeContextEstadoOKV0,
		RequestRef:  query.RequestRef,
		ProviderRef: "provider-ref-fake",
		Results:     hits,
	}, nil
}

func requireCodeContextIssueTestV0(t *testing.T, result CodeContextResultV0, code string) {
	t.Helper()
	for _, issue := range result.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrada en %+v", code, result)
}

func longTextContextTestV0(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'x'
	}
	return string(out)
}
