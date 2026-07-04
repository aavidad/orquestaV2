package orquestamcp

import (
	"context"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	MCPCodebaseQueryToolNameV0    = "orquesta.codebase.query.v0"
	MCPCodebaseQueryToolVersionV0 = "v0"
	MCPCodebaseQueryResourceURIV0 = "orquesta://contracts/codebase-query/v0"
	MCPCodebaseQueryHTTPPathV0    = "/api/v0/codebase/query"

	MCPCodebaseQueryHTTPErrorCodeV0             = "codebase_query_http_error"
	MCPCodebaseQueryHTTPNotConfiguredCodeV0     = "codebase_query_no_configurado"
	MCPCodebaseQueryHTTPExecutorErrorCodeV0     = "codebase_query_error"
	MCPCodebaseQueryHTTPUnsupportedPathCodeV0   = MCPPublicErrPathUnsupportedV0
	MCPCodebaseQueryHTTPUnsupportedMethodCodeV0 = MCPPublicErrMethodNotAllowedV0
)

type MCPCodebaseQueryToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPCodebaseQueryToolInputV0 = orquestacontext.CodeContextQueryV0
type MCPCodebaseQueryToolResultV0 = orquestacontext.CodeContextResultV0

type MCPCodebaseQueryToolExecutorV0 struct {
	Broker orquestacontext.CodeContextQueryPortV0
}

func MCPCodebaseQueryDescriptorV0() MCPCodebaseQueryToolDescriptorV0 {
	return MCPCodebaseQueryToolDescriptorV0{
		Name:        MCPCodebaseQueryToolNameV0,
		Version:     MCPCodebaseQueryToolVersionV0,
		InputSchema: "code_context_query:{schema_version,repository_ref,query,query_kind?,scope?,commit_ref?,max_results?,max_bytes?,cache_only?,allow_external_indexer?}",
		Output:      "code_context_result:{estado,provider_kind,cache_status,results,issues,diagnostics}",
		ResourceURI: MCPCodebaseQueryResourceURIV0,
		Invariantes: []string{
			"los agentes consultan Orquesta y no arrancan codebase-memory-mcp propio",
			"la indexacion externa solo puede activarla el broker central con opt-in explicito",
			"las consultas de lectura son compactas, cacheables y acotadas por resultados y bytes",
			"repo_map devuelve rutas, tipos/funciones y snippets minimos por el mismo broker central",
			"callers/imports/module_exports/relevant_snippets tienen fallback estructurado sin indexador externo",
			"rg/documentos siguen siendo fallback para strings exactos y docs no indexados",
			"sin HOME, OAuth, tokens, rutas privadas ni procesos externos dentro de orquesta-mcp",
		},
	}
}

func (executor MCPCodebaseQueryToolExecutorV0) Execute(
	ctx context.Context,
	input MCPCodebaseQueryToolInputV0,
) (MCPCodebaseQueryToolResultV0, error) {
	input.RequestedBy = firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-codebase-query")
	if executor.Broker == nil {
		return orquestacontext.CodeContextResultV0{
			SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
			Estado:        orquestacontext.CodeContextEstadoErrorV0,
			RequestRef:    input.RequestRef,
			CorrelationID: input.CorrelationID,
			RepositoryRef: input.RepositoryRef,
			QueryKind:     input.QueryKind,
			IndexerPolicy: orquestacontext.CodeContextProviderPolicyCentralOnlyV0,
			Issues: []orquestacontext.CodeContextIssueV0{{
				Code:    orquestacontext.ErrCodeContextProveedorNoConfiguradoV0,
				Field:   "broker",
				Message: "broker central de contexto no configurado",
			}},
		}, nil
	}
	return executor.Broker.QueryCodeContextV0(ctx, input)
}
