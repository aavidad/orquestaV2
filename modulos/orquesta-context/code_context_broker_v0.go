package orquestacontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CodeContextQuerySchemaVersionV0  = "code_context_query.v0"
	CodeContextResultSchemaVersionV0 = "code_context_result.v0"

	CodeContextEstadoOKV0    = "ok"
	CodeContextEstadoErrorV0 = "error"

	CodeContextQueryKindSearchV0       = "search"
	CodeContextQueryKindSymbolV0       = "symbol"
	CodeContextQueryKindArchitectureV0 = "architecture"
	CodeContextQueryKindRepoMapV0      = "repo_map"

	CodeContextCacheHitV0   = "hit"
	CodeContextCacheMissV0  = "miss"
	CodeContextCacheStoreV0 = "stored"

	CodeContextProviderPolicyCentralOnlyV0 = "central_broker_only"
	CodeContextProviderKindFallbackRGV0    = "fallback_rg"
	CodeContextProviderKindCodebaseMCPV0   = "codebase_memory_mcp"

	ErrCodeContextSchemaNoSoportadoV0      = "code_context_schema_no_soportado"
	ErrCodeContextCampoRequeridoV0         = "code_context_campo_requerido"
	ErrCodeContextQueryKindNoSoportadoV0   = "code_context_query_kind_no_soportado"
	ErrCodeContextLimiteInvalidoV0         = "code_context_limite_invalido"
	ErrCodeContextProveedorNoConfiguradoV0 = "code_context_proveedor_no_configurado"
	ErrCodeContextCacheOnlyMissV0          = "code_context_cache_only_miss"
	ErrCodeContextProveedorTimeoutV0       = "code_context_proveedor_timeout"
	ErrCodeContextProveedorErrorV0         = "code_context_proveedor_error"
	ErrCodeContextConcurrenciaV0           = "code_context_concurrencia_agotada"
	ErrCodeContextLeaseRequeridoV0         = "code_context_lease_requerido"
	ErrCodeContextLeaseErrorV0             = "code_context_lease_error"
	ErrCodeContextLeaseActivoV0            = "code_context_lease_activo"
)

const (
	defaultCodeContextMaxResultsV0    = 8
	defaultCodeContextMaxBytesV0      = 12000
	defaultCodeContextSnippetBytesV0  = 700
	defaultCodeContextTimeoutV0       = 3 * time.Second
	defaultCodeContextMaxConcurrentV0 = 4
	defaultCodeContextToolLeaseTTLV0  = 2 * defaultCodeContextTimeoutV0
)

type CodeContextQueryPortV0 interface {
	QueryCodeContextV0(context.Context, CodeContextQueryV0) (CodeContextResultV0, error)
}

type CodeContextProviderPortV0 interface {
	QueryCodeContextV0(context.Context, CodeContextQueryV0) (CodeContextResultV0, error)
}

type CodeContextToolLeasePortV0 interface {
	BeginCodeContextToolLeaseV0(context.Context, CodeContextToolLeaseRequestV0) (CodeContextToolLeaseV0, error)
	FinishCodeContextToolLeaseV0(context.Context, CodeContextToolLeaseCompletionV0) error
}

type CodeContextProviderDescriptorV0 struct {
	ProviderRef string
	Kind        string
}

type CodeContextCachePortV0 interface {
	LoadCodeContextResultV0(string) (CodeContextResultV0, bool)
	SaveCodeContextResultV0(string, CodeContextResultV0)
}

type CodeContextBrokerConfigV0 struct {
	Provider               CodeContextProviderPortV0
	Cache                  CodeContextCachePortV0
	ProviderRef            string
	ProviderKind           string
	ExternalIndexerEnabled bool
	MaxConcurrent          int
	DefaultMaxResults      int
	DefaultMaxBytes        int
	Timeout                time.Duration
	ToolLeaseTTL           time.Duration
	ToolLeasePort          CodeContextToolLeasePortV0
	Clock                  func() time.Time
}

type CodeContextBrokerV0 struct {
	config   CodeContextBrokerConfigV0
	sem      chan struct{}
	mu       sync.Mutex
	inflight map[string]*codeContextBrokerInflightV0
}

type CodeContextQueryV0 struct {
	SchemaVersion        string   `json:"schema_version"`
	RequestRef           string   `json:"request_ref,omitempty"`
	CorrelationID        string   `json:"correlation_id,omitempty"`
	RepositoryRef        string   `json:"repository_ref"`
	WorktreeRef          string   `json:"worktree_ref,omitempty"`
	CommitRef            string   `json:"commit_ref,omitempty"`
	WorktreeFingerprint  string   `json:"worktree_fingerprint,omitempty"`
	DirtyWorktree        bool     `json:"dirty_worktree,omitempty"`
	QueryKind            string   `json:"query_kind,omitempty"`
	Query                string   `json:"query"`
	Scope                []string `json:"scope,omitempty"`
	MaxResults           int      `json:"max_results,omitempty"`
	MaxBytes             int      `json:"max_bytes,omitempty"`
	CacheOnly            bool     `json:"cache_only,omitempty"`
	AllowExternalIndexer bool     `json:"allow_external_indexer,omitempty"`
	RequestedBy          string   `json:"requested_by,omitempty"`
}

type CodeContextResultV0 struct {
	SchemaVersion       string                    `json:"schema_version"`
	Estado              string                    `json:"estado"`
	RequestRef          string                    `json:"request_ref,omitempty"`
	CorrelationID       string                    `json:"correlation_id,omitempty"`
	RepositoryRef       string                    `json:"repository_ref,omitempty"`
	WorktreeRef         string                    `json:"worktree_ref,omitempty"`
	CommitRef           string                    `json:"commit_ref,omitempty"`
	WorktreeFingerprint string                    `json:"worktree_fingerprint,omitempty"`
	DirtyWorktree       bool                      `json:"dirty_worktree,omitempty"`
	QueryKind           string                    `json:"query_kind,omitempty"`
	QueryHash           string                    `json:"query_hash,omitempty"`
	CacheStatus         string                    `json:"cache_status,omitempty"`
	ProviderRef         string                    `json:"provider_ref,omitempty"`
	ProviderKind        string                    `json:"provider_kind,omitempty"`
	IndexerPolicy       string                    `json:"indexer_policy,omitempty"`
	Results             []CodeContextHitV0        `json:"results,omitempty"`
	Diagnostics         []CodeContextDiagnosticV0 `json:"diagnostics,omitempty"`
	Issues              []CodeContextIssueV0      `json:"issues,omitempty"`
	EvidenceRefs        []string                  `json:"evidence_refs,omitempty"`
}

type CodeContextHitV0 struct {
	HitRef  string  `json:"hit_ref"`
	Kind    string  `json:"kind,omitempty"`
	Path    string  `json:"path,omitempty"`
	Line    int     `json:"line,omitempty"`
	Symbol  string  `json:"symbol,omitempty"`
	Summary string  `json:"summary,omitempty"`
	Snippet string  `json:"snippet,omitempty"`
	Score   float64 `json:"score,omitempty"`
}

type CodeContextDiagnosticV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type CodeContextIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewCodeContextBrokerV0(config CodeContextBrokerConfigV0) *CodeContextBrokerV0 {
	config = normalizeCodeContextBrokerConfigV0(config)
	return &CodeContextBrokerV0{
		config: config,
		sem:    make(chan struct{}, config.MaxConcurrent),
	}
}

func (broker *CodeContextBrokerV0) QueryCodeContextV0(
	ctx context.Context,
	query CodeContextQueryV0,
) (CodeContextResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query = normalizeCodeContextQueryV0(query, broker.config)
	if issues := ValidateCodeContextQueryV0(query); len(issues) > 0 {
		return newCodeContextErrorResultV0(query, issues), nil
	}
	key := CodeContextCacheKeyV0(query)
	if broker.config.Cache != nil {
		if cached, ok := broker.config.Cache.LoadCodeContextResultV0(key); ok {
			cached.CacheStatus = CodeContextCacheHitV0
			cached.Diagnostics = append(cached.Diagnostics, codeContextDiagnosticV0(
				"code_context_cache_hit",
				"cache",
				"resultado reutilizado por Orquesta; el agente no arranca indexador propio",
			))
			return cached, nil
		}
	}
	if query.CacheOnly {
		return newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(ErrCodeContextCacheOnlyMissV0, "cache_only", "consulta no existe en cache central"),
		}), nil
	}
	if broker.config.Provider == nil {
		return newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(ErrCodeContextProveedorNoConfiguradoV0, "provider", "broker sin proveedor central configurado"),
		}), nil
	}
	if broker.config.ProviderKind == CodeContextProviderKindCodebaseMCPV0 &&
		(!broker.config.ExternalIndexerEnabled || !query.AllowExternalIndexer) {
		return newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(
				ErrCodeContextProveedorNoConfiguradoV0,
				"allow_external_indexer",
				"codebase-memory-mcp solo puede usarse por broker central con opt-in explicito",
			),
		}), nil
	}
	if broker.config.ProviderKind == CodeContextProviderKindCodebaseMCPV0 &&
		broker.config.ToolLeasePort == nil {
		return newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(
				ErrCodeContextLeaseRequeridoV0,
				"tool_lease",
				"codebase-memory-mcp requiere lease central de Orquesta antes de ejecutar el proveedor",
			),
		}), nil
	}
	if call, leader := broker.beginInflightV0(key); !leader {
		select {
		case <-call.done:
			result := call.result
			result.Diagnostics = append(result.Diagnostics, codeContextDiagnosticV0(
				"code_context_inflight_joined",
				"query_hash",
				"consulta concurrente reutilizada por el broker central",
			))
			return result, call.err
		case <-ctx.Done():
			return newCodeContextErrorResultV0(query, []CodeContextIssueV0{
				codeContextIssueV0(ErrCodeContextProveedorTimeoutV0, "inflight", "consulta concurrente no termino antes del contexto"),
			}), nil
		}
	} else {
		defer func() {
			broker.finishInflightV0(key, call)
		}()
	}
	if ok := broker.acquireV0(ctx); !ok {
		result := newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(ErrCodeContextConcurrenciaV0, "concurrency", "broker de contexto ocupado"),
		})
		broker.storeInflightResultV0(key, result, nil)
		return result, nil
	}
	defer broker.releaseV0()

	lease, leaseBegun, leaseErr := broker.beginToolLeaseV0(ctx, query)
	if leaseErr != nil {
		code := ErrCodeContextLeaseErrorV0
		message := "no se pudo registrar lease central para proveedor de contexto"
		if leaseErr == errCodeContextLeaseActivoV0 {
			code = ErrCodeContextLeaseActivoV0
			message = "codebase-memory-mcp ya tiene un lease activo para este repositorio; no se arranca otra instancia"
		}
		result := newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(code, "tool_lease", message),
		})
		broker.storeInflightResultV0(key, result, nil)
		return result, nil
	}

	runCtx := ctx
	cancel := func() {}
	if broker.config.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, broker.config.Timeout)
	}
	defer cancel()
	result, err := broker.config.Provider.QueryCodeContextV0(runCtx, query)
	if err != nil {
		broker.finishToolLeaseV0(ctx, lease, leaseBegun, CodeContextToolLeaseCompletionFailedV0)
		code := ErrCodeContextProveedorErrorV0
		if runCtx.Err() == context.DeadlineExceeded {
			code = ErrCodeContextProveedorTimeoutV0
		}
		result := newCodeContextErrorResultV0(query, []CodeContextIssueV0{
			codeContextIssueV0(code, "provider", "proveedor central no devolvio contexto util"),
		})
		broker.storeInflightResultV0(key, result, nil)
		return result, nil
	}
	broker.finishToolLeaseV0(ctx, lease, leaseBegun, CodeContextToolLeaseCompletionCompletedV0)
	result = normalizeCodeContextResultV0(result, query, broker.config)
	result.CacheStatus = CodeContextCacheStoreV0
	if broker.config.Cache != nil && result.Estado == CodeContextEstadoOKV0 {
		broker.config.Cache.SaveCodeContextResultV0(key, result)
	}
	broker.storeInflightResultV0(key, result, nil)
	return result, nil
}

func ValidateCodeContextQueryV0(query CodeContextQueryV0) []CodeContextIssueV0 {
	var issues []CodeContextIssueV0
	if query.SchemaVersion != CodeContextQuerySchemaVersionV0 {
		issues = append(issues, codeContextIssueV0(ErrCodeContextSchemaNoSoportadoV0, "schema_version", "schema_version no soportada"))
	}
	for _, required := range []struct {
		field string
		value string
	}{
		{"repository_ref", query.RepositoryRef},
		{"query", query.Query},
	} {
		if strings.TrimSpace(required.value) == "" {
			issues = append(issues, codeContextIssueV0(ErrCodeContextCampoRequeridoV0, required.field, "campo requerido"))
		}
	}
	switch query.QueryKind {
	case CodeContextQueryKindSearchV0, CodeContextQueryKindSymbolV0, CodeContextQueryKindArchitectureV0, CodeContextQueryKindRepoMapV0:
	default:
		issues = append(issues, codeContextIssueV0(ErrCodeContextQueryKindNoSoportadoV0, "query_kind", "tipo de consulta no soportado"))
	}
	if query.MaxResults < 1 || query.MaxResults > 50 {
		issues = append(issues, codeContextIssueV0(ErrCodeContextLimiteInvalidoV0, "max_results", "limite de resultados invalido"))
	}
	if query.MaxBytes < 1000 || query.MaxBytes > 64000 {
		issues = append(issues, codeContextIssueV0(ErrCodeContextLimiteInvalidoV0, "max_bytes", "limite de bytes invalido"))
	}
	return issues
}

func CodeContextCacheKeyV0(query CodeContextQueryV0) string {
	query = normalizeCodeContextQueryV0(query, CodeContextBrokerConfigV0{})
	raw := strings.Join([]string{
		query.RepositoryRef,
		query.WorktreeRef,
		query.CommitRef,
		query.WorktreeFingerprint,
		strconv.FormatBool(query.DirtyWorktree),
		query.QueryKind,
		query.Query,
		strings.Join(query.Scope, "\x00"),
		strconv.Itoa(query.MaxResults),
		strconv.Itoa(query.MaxBytes),
	}, "\x1f")
	sum := sha256.Sum256([]byte(raw))
	return "code-context-sha256-" + hex.EncodeToString(sum[:])[:24]
}

func normalizeCodeContextBrokerConfigV0(config CodeContextBrokerConfigV0) CodeContextBrokerConfigV0 {
	config.ProviderRef = trimContextV0(config.ProviderRef)
	if config.ProviderRef == "" {
		config.ProviderRef = "provider-ref-orquesta-code-context"
	}
	config.ProviderKind = trimContextV0(config.ProviderKind)
	if config.ProviderKind == "" {
		config.ProviderKind = CodeContextProviderKindFallbackRGV0
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = defaultCodeContextMaxConcurrentV0
	}
	if config.DefaultMaxResults <= 0 {
		config.DefaultMaxResults = defaultCodeContextMaxResultsV0
	}
	if config.DefaultMaxBytes <= 0 {
		config.DefaultMaxBytes = defaultCodeContextMaxBytesV0
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultCodeContextTimeoutV0
	}
	if config.ToolLeaseTTL <= 0 {
		config.ToolLeaseTTL = defaultCodeContextToolLeaseTTLV0
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	return config
}

func normalizeCodeContextQueryV0(
	query CodeContextQueryV0,
	config CodeContextBrokerConfigV0,
) CodeContextQueryV0 {
	query.SchemaVersion = trimContextV0(query.SchemaVersion)
	if query.SchemaVersion == "" {
		query.SchemaVersion = CodeContextQuerySchemaVersionV0
	}
	query.RequestRef = trimContextV0(query.RequestRef)
	query.CorrelationID = trimContextV0(query.CorrelationID)
	query.RepositoryRef = trimContextV0(query.RepositoryRef)
	query.WorktreeRef = trimContextV0(query.WorktreeRef)
	query.CommitRef = trimContextV0(query.CommitRef)
	query.WorktreeFingerprint = trimContextV0(query.WorktreeFingerprint)
	query.QueryKind = trimContextV0(query.QueryKind)
	if query.QueryKind == "" {
		query.QueryKind = CodeContextQueryKindSearchV0
	}
	query.Query = trimContextV0(query.Query)
	query.Scope = compactContextStringsV0(query.Scope)
	query.RequestedBy = trimContextV0(query.RequestedBy)
	if query.MaxResults <= 0 {
		query.MaxResults = config.DefaultMaxResults
	}
	if query.MaxResults <= 0 {
		query.MaxResults = defaultCodeContextMaxResultsV0
	}
	if query.MaxBytes <= 0 {
		query.MaxBytes = config.DefaultMaxBytes
	}
	if query.MaxBytes <= 0 {
		query.MaxBytes = defaultCodeContextMaxBytesV0
	}
	return query
}

func normalizeCodeContextResultV0(
	result CodeContextResultV0,
	query CodeContextQueryV0,
	config CodeContextBrokerConfigV0,
) CodeContextResultV0 {
	result.SchemaVersion = CodeContextResultSchemaVersionV0
	result.Estado = trimContextV0(result.Estado)
	if result.Estado == "" {
		result.Estado = CodeContextEstadoOKV0
	}
	result.RequestRef = firstNonEmptyContextV0(result.RequestRef, query.RequestRef)
	result.CorrelationID = firstNonEmptyContextV0(result.CorrelationID, query.CorrelationID)
	result.RepositoryRef = firstNonEmptyContextV0(result.RepositoryRef, query.RepositoryRef)
	result.WorktreeRef = firstNonEmptyContextV0(result.WorktreeRef, query.WorktreeRef)
	result.CommitRef = firstNonEmptyContextV0(result.CommitRef, query.CommitRef)
	result.WorktreeFingerprint = firstNonEmptyContextV0(result.WorktreeFingerprint, query.WorktreeFingerprint)
	result.DirtyWorktree = result.DirtyWorktree || query.DirtyWorktree
	result.QueryKind = firstNonEmptyContextV0(result.QueryKind, query.QueryKind)
	result.QueryHash = firstNonEmptyContextV0(result.QueryHash, CodeContextCacheKeyV0(query))
	result.ProviderRef = firstNonEmptyContextV0(result.ProviderRef, config.ProviderRef)
	result.ProviderKind = firstNonEmptyContextV0(result.ProviderKind, config.ProviderKind)
	result.IndexerPolicy = CodeContextProviderPolicyCentralOnlyV0
	result.Results = limitCodeContextHitsV0(result.Results, query.MaxResults, query.MaxBytes)
	result.EvidenceRefs = compactContextStringsV0(result.EvidenceRefs)
	result.Diagnostics = append(result.Diagnostics, codeContextDiagnosticV0(
		"code_context_central_broker",
		"provider",
		"consulta servida por Orquesta; los agentes no deben arrancar codebase-memory-mcp propio",
	))
	return result
}

func limitCodeContextHitsV0(hits []CodeContextHitV0, maxResults int, maxBytes int) []CodeContextHitV0 {
	if maxResults <= 0 {
		maxResults = defaultCodeContextMaxResultsV0
	}
	if maxBytes <= 0 {
		maxBytes = defaultCodeContextMaxBytesV0
	}
	out := make([]CodeContextHitV0, 0, len(hits))
	total := 0
	for _, hit := range hits {
		if len(out) >= maxResults {
			break
		}
		hit.HitRef = trimContextV0(hit.HitRef)
		if hit.HitRef == "" {
			hit.HitRef = "code-context-hit-" + contextIntV0(len(out)+1)
		}
		hit.Kind = trimContextV0(hit.Kind)
		hit.Path = trimContextV0(hit.Path)
		hit.Symbol = trimContextV0(hit.Symbol)
		hit.Summary = trimContextV0(hit.Summary)
		hit.Snippet = trimCodeContextSnippetV0(hit.Snippet, defaultCodeContextSnippetBytesV0)
		total += len(hit.Path) + len(hit.Symbol) + len(hit.Summary) + len(hit.Snippet)
		if total > maxBytes {
			break
		}
		out = append(out, hit)
	}
	return out
}

func trimCodeContextSnippetV0(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	if maxBytes < 16 {
		return value[:maxBytes]
	}
	return strings.TrimSpace(value[:maxBytes-12]) + " [truncated]"
}

func newCodeContextErrorResultV0(query CodeContextQueryV0, issues []CodeContextIssueV0) CodeContextResultV0 {
	query = normalizeCodeContextQueryV0(query, CodeContextBrokerConfigV0{})
	return CodeContextResultV0{
		SchemaVersion:       CodeContextResultSchemaVersionV0,
		Estado:              CodeContextEstadoErrorV0,
		RequestRef:          query.RequestRef,
		CorrelationID:       query.CorrelationID,
		RepositoryRef:       query.RepositoryRef,
		WorktreeRef:         query.WorktreeRef,
		CommitRef:           query.CommitRef,
		WorktreeFingerprint: query.WorktreeFingerprint,
		DirtyWorktree:       query.DirtyWorktree,
		QueryKind:           query.QueryKind,
		QueryHash:           CodeContextCacheKeyV0(query),
		IndexerPolicy:       CodeContextProviderPolicyCentralOnlyV0,
		Issues:              issues,
	}
}

func (broker *CodeContextBrokerV0) acquireV0(ctx context.Context) bool {
	select {
	case broker.sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (broker *CodeContextBrokerV0) releaseV0() {
	select {
	case <-broker.sem:
	default:
	}
}

type codeContextBrokerInflightV0 struct {
	done   chan struct{}
	result CodeContextResultV0
	err    error
}

func (broker *CodeContextBrokerV0) beginInflightV0(key string) (*codeContextBrokerInflightV0, bool) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.inflight == nil {
		broker.inflight = map[string]*codeContextBrokerInflightV0{}
	}
	if call, ok := broker.inflight[key]; ok {
		return call, false
	}
	call := &codeContextBrokerInflightV0{done: make(chan struct{})}
	broker.inflight[key] = call
	return call, true
}

func (broker *CodeContextBrokerV0) storeInflightResultV0(key string, result CodeContextResultV0, err error) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if call, ok := broker.inflight[key]; ok {
		call.result = result
		call.err = err
	}
}

func (broker *CodeContextBrokerV0) finishInflightV0(key string, call *codeContextBrokerInflightV0) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if current, ok := broker.inflight[key]; ok && current == call {
		close(call.done)
		delete(broker.inflight, key)
	}
}

func (broker *CodeContextBrokerV0) beginToolLeaseV0(
	ctx context.Context,
	query CodeContextQueryV0,
) (CodeContextToolLeaseV0, bool, error) {
	if broker.config.ToolLeasePort == nil {
		return CodeContextToolLeaseV0{}, false, nil
	}
	if broker.config.ProviderKind == CodeContextProviderKindCodebaseMCPV0 {
		if listPort, ok := broker.config.ToolLeasePort.(CodeContextToolLeaseListPortV0); ok {
			active, err := listPort.ListCodeContextToolLeasesV0(ctx, CodeContextToolLeaseListFilterV0{
				RepositoryRef: query.RepositoryRef,
				ToolRef:       broker.config.ProviderRef,
				Status:        CodeContextToolLeaseStatusActiveV0,
			})
			if err != nil {
				return CodeContextToolLeaseV0{}, false, err
			}
			if len(active) > 0 {
				return CodeContextToolLeaseV0{}, false, errCodeContextLeaseActivoV0
			}
		}
	}
	now := broker.config.Clock().UTC()
	request := CodeContextToolLeaseRequestV0{
		RequestRef:      query.RequestRef,
		CorrelationID:   query.CorrelationID,
		RepositoryRef:   query.RepositoryRef,
		WorktreeRef:     query.WorktreeRef,
		CommitRef:       query.CommitRef,
		QueryHash:       CodeContextCacheKeyV0(query),
		ToolRef:         broker.config.ProviderRef,
		ProviderKind:    broker.config.ProviderKind,
		OwnerRef:        CodeContextToolOwnerRefV0(query),
		StartedAt:       now.Format(time.RFC3339),
		LeaseTTLSeconds: int(broker.config.ToolLeaseTTL.Seconds()),
		EvidenceRefs:    []string{query.RequestRef},
	}
	lease, err := broker.config.ToolLeasePort.BeginCodeContextToolLeaseV0(ctx, request)
	if err != nil {
		return CodeContextToolLeaseV0{}, false, err
	}
	return lease, true, nil
}

type codeContextLeaseActiveErrorV0 struct{}

func (codeContextLeaseActiveErrorV0) Error() string {
	return ErrCodeContextLeaseActivoV0
}

var errCodeContextLeaseActivoV0 error = codeContextLeaseActiveErrorV0{}

func (broker *CodeContextBrokerV0) finishToolLeaseV0(
	ctx context.Context,
	lease CodeContextToolLeaseV0,
	leaseBegun bool,
	status string,
) {
	if broker.config.ToolLeasePort == nil || !leaseBegun {
		return
	}
	_ = broker.config.ToolLeasePort.FinishCodeContextToolLeaseV0(ctx, CodeContextToolLeaseCompletionV0{
		LeaseRef:    lease.LeaseRef,
		ToolRef:     lease.ToolRef,
		CompletedAt: broker.config.Clock().UTC().Format(time.RFC3339),
		Status:      status,
		EvidenceRefs: []string{
			lease.LeaseRef,
		},
	})
}

func codeContextIssueV0(code string, field string, message string) CodeContextIssueV0 {
	return CodeContextIssueV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field), Message: strings.TrimSpace(message)}
}

func CodeContextToolOwnerRefV0(query CodeContextQueryV0) string {
	hash := CodeContextCacheKeyV0(query)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	if hash == "" {
		hash = "unknown"
	}
	return "owner-ref-code-context-" + hash
}

func codeContextDiagnosticV0(code string, field string, message string) CodeContextDiagnosticV0 {
	return CodeContextDiagnosticV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field), Message: strings.TrimSpace(message)}
}

func firstNonEmptyContextV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type InMemoryCodeContextCacheV0 struct {
	mu      sync.Mutex
	results map[string]CodeContextResultV0
}

func NewInMemoryCodeContextCacheV0() *InMemoryCodeContextCacheV0 {
	return &InMemoryCodeContextCacheV0{results: map[string]CodeContextResultV0{}}
}

func (cache *InMemoryCodeContextCacheV0) LoadCodeContextResultV0(key string) (CodeContextResultV0, bool) {
	if cache == nil {
		return CodeContextResultV0{}, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	result, ok := cache.results[strings.TrimSpace(key)]
	return result, ok
}

func (cache *InMemoryCodeContextCacheV0) SaveCodeContextResultV0(key string, result CodeContextResultV0) {
	if cache == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.results[key] = result
}
