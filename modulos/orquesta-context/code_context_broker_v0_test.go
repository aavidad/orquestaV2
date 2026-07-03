package orquestacontext

import (
	"context"
	"sync"
	"testing"
	"time"
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

func TestCodeContextBrokerV0RepoMapHeredaCacheDedupeYFingerprint(t *testing.T) {
	provider := &fakeCodeContextProviderV0{hits: []CodeContextHitV0{{
		Kind:    "function",
		Path:    "modulos/orquesta-context/code_context_broker_v0.go",
		Line:    172,
		Symbol:  "QueryCodeContextV0",
		Summary: "funcion QueryCodeContextV0 en mapa compacto",
		Snippet: "func (broker *CodeContextBrokerV0) QueryCodeContextV0(...)",
		Score:   1,
	}}}
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider: provider,
		Cache:    NewInMemoryCodeContextCacheV0(),
	})
	query := validCodeContextQueryTestV0()
	query.QueryKind = CodeContextQueryKindRepoMapV0
	query.Query = "CodeContextBroker"
	query.WorktreeFingerprint = "fingerprint-ref-repo-map-clean"

	first, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	second, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if first.CacheStatus != CodeContextCacheStoreV0 || second.CacheStatus != CodeContextCacheHitV0 {
		t.Fatalf("cache statuses first=%q second=%q", first.CacheStatus, second.CacheStatus)
	}
	if provider.callCountV0() != 1 {
		t.Fatalf("provider calls=%d, want 1", provider.callCountV0())
	}
	dirty := query
	dirty.WorktreeFingerprint = "fingerprint-ref-repo-map-dirty"
	dirty.DirtyWorktree = true
	if _, err := broker.QueryCodeContextV0(context.Background(), dirty); err != nil {
		t.Fatalf("dirty query: %v", err)
	}
	if provider.callCountV0() != 2 {
		t.Fatalf("provider calls=%d, want 2 por fingerprint distinto", provider.callCountV0())
	}

	concurrentProvider := &fakeCodeContextProviderV0{delay: 50 * time.Millisecond}
	concurrentBroker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:     concurrentProvider,
		ProviderKind: CodeContextProviderKindFallbackRGV0,
		Timeout:      time.Second,
	})
	var wg sync.WaitGroup
	results := make(chan CodeContextResultV0, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := concurrentBroker.QueryCodeContextV0(context.Background(), query)
			if err != nil {
				t.Errorf("concurrent repo_map query: %v", err)
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	if concurrentProvider.callCountV0() != 1 {
		t.Fatalf("concurrent provider calls=%d, want 1", concurrentProvider.callCountV0())
	}
	var joined bool
	for result := range results {
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code == "code_context_inflight_joined" {
				joined = true
			}
		}
	}
	if !joined {
		t.Fatalf("repo_map no reutilizo inflight")
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

func TestCodeContextBrokerV0ExigeLeaseCentralParaCodebaseMCP(t *testing.T) {
	query := validCodeContextQueryTestV0()
	query.AllowExternalIndexer = true
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:               &fakeCodeContextProviderV0{},
		ProviderKind:           CodeContextProviderKindCodebaseMCPV0,
		ExternalIndexerEnabled: true,
	})

	result, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	requireCodeContextIssueTestV0(t, result, ErrCodeContextLeaseRequeridoV0)
}

func TestCodeContextBrokerV0PermiteCodebaseMCPConOptInCentralYConsulta(t *testing.T) {
	query := validCodeContextQueryTestV0()
	query.AllowExternalIndexer = true
	leases := NewInMemoryCodeContextToolLeaseStoreV0()
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:               &fakeCodeContextProviderV0{},
		ProviderRef:            "provider-ref-codebase-central",
		ProviderKind:           CodeContextProviderKindCodebaseMCPV0,
		ExternalIndexerEnabled: true,
		ToolLeasePort:          leases,
		Clock:                  fixedCodeContextClockTestV0(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)),
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
	completed, err := leases.ListCodeContextToolLeasesV0(context.Background(), CodeContextToolLeaseListFilterV0{
		Status: CodeContextToolLeaseStatusCompletedV0,
	})
	if err != nil {
		t.Fatalf("list leases: %v", err)
	}
	if len(completed) != 1 || completed[0].ProviderKind != CodeContextProviderKindCodebaseMCPV0 {
		t.Fatalf("leases=%+v", completed)
	}
	if completed[0].OwnerRef != CodeContextToolOwnerRefV0(query) || completed[0].OwnerRef == query.RequestedBy {
		t.Fatalf("owner_ref=%q requested_by=%q", completed[0].OwnerRef, query.RequestedBy)
	}
}

func TestCodeContextBrokerV0NoArrancaCodebaseMCPConLeaseActivoDelRepo(t *testing.T) {
	query := validCodeContextQueryTestV0()
	query.AllowExternalIndexer = true
	leases := NewInMemoryCodeContextToolLeaseStoreV0()
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	if _, err := leases.BeginCodeContextToolLeaseV0(context.Background(), CodeContextToolLeaseRequestV0{
		RequestRef:      "request-ref-code-context-existing-lease",
		RepositoryRef:   query.RepositoryRef,
		QueryHash:       "query-hash-existing",
		ToolRef:         "provider-ref-codebase-central",
		ProviderKind:    CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-existing",
		StartedAt:       now.Format(time.RFC3339),
		LeaseTTLSeconds: 60,
	}); err != nil {
		t.Fatalf("begin existing lease: %v", err)
	}
	provider := &fakeCodeContextProviderV0{}
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:               provider,
		ProviderRef:            "provider-ref-codebase-central",
		ProviderKind:           CodeContextProviderKindCodebaseMCPV0,
		ExternalIndexerEnabled: true,
		ToolLeasePort:          leases,
		Clock:                  fixedCodeContextClockTestV0(now),
	})

	result, err := broker.QueryCodeContextV0(context.Background(), query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	requireCodeContextIssueTestV0(t, result, ErrCodeContextLeaseActivoV0)
	if provider.callCountV0() != 0 {
		t.Fatalf("provider calls=%d, want 0", provider.callCountV0())
	}
	active, err := leases.ListCodeContextToolLeasesV0(context.Background(), CodeContextToolLeaseListFilterV0{
		RepositoryRef: query.RepositoryRef,
		ToolRef:       "provider-ref-codebase-central",
		Status:        CodeContextToolLeaseStatusActiveV0,
	})
	if err != nil {
		t.Fatalf("list leases: %v", err)
	}
	if len(active) != 1 || active[0].OwnerRef != "owner-ref-existing" {
		t.Fatalf("active leases=%+v", active)
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

func TestCodeContextBrokerV0DeduplicaConsultasConcurrentesIguales(t *testing.T) {
	provider := &fakeCodeContextProviderV0{delay: 50 * time.Millisecond}
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider:      provider,
		ProviderKind:  CodeContextProviderKindFallbackRGV0,
		MaxConcurrent: 4,
		Timeout:       time.Second,
	})
	query := validCodeContextQueryTestV0()

	var wg sync.WaitGroup
	results := make(chan CodeContextResultV0, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := broker.QueryCodeContextV0(context.Background(), query)
			if err != nil {
				t.Errorf("query: %v", err)
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	if provider.callCountV0() != 1 {
		t.Fatalf("provider calls=%d, want 1", provider.callCountV0())
	}
	var joined bool
	for result := range results {
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code == "code_context_inflight_joined" {
				joined = true
			}
		}
	}
	if !joined {
		t.Fatalf("ninguna consulta concurrente reutilizo inflight")
	}
}

func TestCodeContextBrokerV0CacheDistingueFingerprintYWorktreeSucio(t *testing.T) {
	provider := &fakeCodeContextProviderV0{}
	broker := NewCodeContextBrokerV0(CodeContextBrokerConfigV0{
		Provider: provider,
		Cache:    NewInMemoryCodeContextCacheV0(),
	})
	first := validCodeContextQueryTestV0()
	first.WorktreeFingerprint = "fingerprint-ref-clean"
	second := first
	second.WorktreeFingerprint = "fingerprint-ref-dirty"
	second.DirtyWorktree = true

	if _, err := broker.QueryCodeContextV0(context.Background(), first); err != nil {
		t.Fatalf("first query: %v", err)
	}
	if _, err := broker.QueryCodeContextV0(context.Background(), second); err != nil {
		t.Fatalf("second query: %v", err)
	}
	if provider.callCountV0() != 2 {
		t.Fatalf("provider calls=%d, want 2 por fingerprint distinto", provider.callCountV0())
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
	mu    sync.Mutex
	calls int
	hits  []CodeContextHitV0
	delay time.Duration
}

func (provider *fakeCodeContextProviderV0) QueryCodeContextV0(
	ctx context.Context,
	query CodeContextQueryV0,
) (CodeContextResultV0, error) {
	if provider.delay > 0 {
		select {
		case <-time.After(provider.delay):
		case <-ctx.Done():
			return CodeContextResultV0{}, ctx.Err()
		}
	}
	provider.mu.Lock()
	provider.calls++
	provider.mu.Unlock()
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

func (provider *fakeCodeContextProviderV0) callCountV0() int {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.calls
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

func fixedCodeContextClockTestV0(now time.Time) func() time.Time {
	return func() time.Time { return now }
}
