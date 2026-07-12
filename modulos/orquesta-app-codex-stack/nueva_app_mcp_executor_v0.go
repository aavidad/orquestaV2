package orquestaappcodexstack

import (
	"context"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// codexStackNuevaAppExecutorV0 mantiene solicitar_nueva dentro del proceso:
// reutiliza el caso de uso Factory y evita un loop HTTP contra el propio server.
type codexStackNuevaAppExecutorV0 struct {
	Clock orquestafactoryhttp.AppSpecHTTPClockV0
}

func (executor codexStackNuevaAppExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPNuevaAppToolInputV0,
) (orquestamcp.MCPNuevaAppToolResultV0, error) {
	request, correlationID := orquestamcp.ToAppSpecRequestV0(input)
	spec, issues := orquestafactory.SolicitarNuevaAppV0(request, stackNowV0(executor.Clock))
	if len(issues) > 0 {
		return orquestamcp.NewMCPNuevaAppErrorResultV0(request.RequestID, correlationID, issues), nil
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		return orquestamcp.NewMCPNuevaAppErrorResultV0(request.RequestID, correlationID, issues), nil
	}
	return orquestamcp.NewMCPNuevaAppOKResultV0(spec, backlog, correlationID), nil
}
