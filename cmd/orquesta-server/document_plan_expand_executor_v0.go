package main

import (
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// serverDocumentPlanExpandExecutorV0 deliberately reaches the same DomainWork
// executor used by orquesta.domain_work.v0. Therefore create_jobs shares its
// configured durable backend and replay semantics rather than creating a
// second store for document-plan-derived jobs.
func serverDocumentPlanExpandExecutorV0(
	executor orquestamcp.MCPDomainWorkExecutorPortV0,
) orquestamcp.MCPDocumentPlanExpandToolExecutorV0 {
	return orquestamcp.NewMCPDocumentPlanExpandToolExecutorV0(
		serverDomainWorkMCPJobCreatorV0{executor: executor},
	)
}
