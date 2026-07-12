package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// CouncilGateDecisionPortV0 lo implementa el servidor sobre los recibos durables
// del consejo. El gate NO decide: solo comprueba si hay una decision ACEPTADA.
type CouncilGateDecisionPortV0 interface {
	CouncilDecisionAcceptedV0(ctx context.Context, councilRef string) (bool, error)
}

// CouncilGateConfigV0 gobierna el gate de creacion. El operador pidio el consejo
// "en tiempo de creacion": esta es la pieza que hace que el ciclo lo convoque de
// verdad en vez de dejarlo como una tool que nadie llama.
//
// Required=false deja pasar y NO es un descuido: sin fuente real de miembros
// (brecha 2), un gate obligatorio bloquearia toda creacion. Se activa cuando esa
// fuente exista. Mientras tanto el gate esta cableado y probado, no muerto.
type CouncilGateConfigV0 struct {
	Required bool
	Decision CouncilGateDecisionPortV0
}

// CouncilGateDecisionRequiredCodeV0 es el codigo publico con el que el gate
// rechaza un arranque sin decision aceptada del consejo.
const CouncilGateDecisionRequiredCodeV0 = "council_decision_required"

// CouncilGateMisconfiguredCodeV0 delata un gate exigido sin puerto cableado. Es
// un fallo de despliegue, y se resuelve cerrando la puerta, no abriendola.
const CouncilGateMisconfiguredCodeV0 = "council_gate_misconfigured"

// CouncilRefForAppRequestV0 deriva el consejo de la PETICION DE CREACION de forma
// determinista: la misma peticion convoca siempre al mismo consejo, que es lo que
// permite la idempotencia del recibo.
func CouncilRefForAppRequestV0(requestID string) string {
	ref := strings.TrimSpace(requestID)
	if ref == "" {
		return ""
	}
	return "council-app-" + ref
}

type councilGatedArrancarDirectorExecutorV0 struct {
	inner  orquestamcp.MCPTransportArrancarDirectorAppExecutorV0
	config CouncilGateConfigV0
}

var _ orquestamcp.MCPTransportArrancarDirectorAppExecutorV0 = councilGatedArrancarDirectorExecutorV0{}

// newCouncilGatedArrancarDirectorExecutorV0 envuelve el arranque real.
//
// Si el gate no esta exigido, devuelve el ejecutor tal cual: cero coste y cero
// cambio de comportamiento.
//
// PERO si el gate SI esta exigido y falta el puerto de decision, NO se devuelve
// el ejecutor desnudo: eso seria fail-open por mala configuracion, y una puerta
// que desaparece cuando la configuras mal no es una puerta. Se devuelve un gate
// que rechaza siempre: fail-closed.
func newCouncilGatedArrancarDirectorExecutorV0(
	inner orquestamcp.MCPTransportArrancarDirectorAppExecutorV0,
	config CouncilGateConfigV0,
) orquestamcp.MCPTransportArrancarDirectorAppExecutorV0 {
	if !config.Required {
		return inner
	}
	return councilGatedArrancarDirectorExecutorV0{inner: inner, config: config}
}

func (executor councilGatedArrancarDirectorExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPArrancarDirectorAppToolResultV0, error) {
	requestID := strings.TrimSpace(input.AppSpecRequest.RequestID)
	if requestID == "" {
		requestID = strings.TrimSpace(input.RequestID)
	}
	councilRef := CouncilRefForAppRequestV0(requestID)

	// Gate exigido sin puerto de decision: mala configuracion. Se cierra, no se
	// abre. Un error de despliegue no puede desactivar un control.
	if executor.config.Decision == nil || executor.inner == nil {
		return orquestamcp.NewMCPArrancarDirectorAppIssuesResultV0(input, []orquestamcp.MCPValidationIssueV0{{
			Code:    CouncilGateMisconfiguredCodeV0,
			Field:   "council.gate_required",
			Message: "el gate del consejo esta exigido pero no hay puerto de decision cableado",
		}}), nil
	}

	accepted, err := executor.config.Decision.CouncilDecisionAcceptedV0(ctx, councilRef)
	if err != nil || !accepted {
		// Sin decision aceptada NO se programa. El consejo no es un tramite
		// posterior: es la puerta.
		return orquestamcp.NewMCPArrancarDirectorAppIssuesResultV0(input, []orquestamcp.MCPValidationIssueV0{{
			Code:    CouncilGateDecisionRequiredCodeV0,
			Field:   "app_spec_request.request_id",
			Message: "el consejo debe aceptar la decision antes de programar la app",
		}}), nil
	}
	return executor.inner.Execute(ctx, input)
}
