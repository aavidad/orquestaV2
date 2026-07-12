package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
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

// CouncilRefForAppRequestV0 deriva el consejo de la PETICION DE CREACION.
//
// Incluye el HASH DE LA APPSPEC, no solo el request_id: si solo dependiera del
// identificador, se podria aprobar una app, mutar su especificacion reutilizando
// el mismo request_id, y colarse por una puerta abierta para OTRA cosa. El
// consejo aprueba una especificacion concreta, no un nombre.
func CouncilRefForAppRequestV0(requestID string, spec orquestafactory.AppSpecRequestV0) string {
	ref := strings.TrimSpace(requestID)
	if ref == "" {
		return ""
	}
	return "council-app-" + ref + "-" + appSpecFingerprintV0(spec)
}

func appSpecFingerprintV0(spec orquestafactory.AppSpecRequestV0) string {
	bytes, err := json.Marshal(spec)
	if err != nil {
		// Un spec que no serializa no puede acreditarse: se le da una huella
		// imposible de igualar, de modo que el gate cierre.
		return "unhashable"
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])[:16]
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
	councilRef := CouncilRefForAppRequestV0(requestID, input.AppSpecRequest)

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

// El gate no puede cubrir UNA ruta y dejar otra abierta: una puerta con una
// ventana al lado no es una puerta. `ejecutar_orquestacion` tambien programa a
// partir de una AppSpec, asi que tambien pasa por el consejo.
func CouncilRefForOrchestrationV0(requestID string, spec orquestafactory.AppSpecV0) string {
	ref := strings.TrimSpace(requestID)
	if ref == "" {
		return ""
	}
	return "council-orq-" + ref + "-" + appSpecFingerprintForOrchestrationV0(spec)
}

func appSpecFingerprintForOrchestrationV0(spec orquestafactory.AppSpecV0) string {
	bytes, err := json.Marshal(spec)
	if err != nil {
		return "unhashable"
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])[:16]
}

type councilGatedEjecutarOrquestacionExecutorV0 struct {
	inner  orquestamcp.MCPTransportEjecutarOrquestacionAppExecutorV0
	config CouncilGateConfigV0
}

var _ orquestamcp.MCPTransportEjecutarOrquestacionAppExecutorV0 = councilGatedEjecutarOrquestacionExecutorV0{}

func newCouncilGatedEjecutarOrquestacionExecutorV0(
	inner orquestamcp.MCPTransportEjecutarOrquestacionAppExecutorV0,
	config CouncilGateConfigV0,
) orquestamcp.MCPTransportEjecutarOrquestacionAppExecutorV0 {
	if !config.Required {
		return inner
	}
	return councilGatedEjecutarOrquestacionExecutorV0{inner: inner, config: config}
}

func (executor councilGatedEjecutarOrquestacionExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPEjecutarOrquestacionAppToolInputV0,
) (orquestamcp.MCPEjecutarOrquestacionAppToolResultV0, error) {
	// Mismo fail-closed que la otra ruta: la mala configuracion cierra, no abre.
	if executor.config.Decision == nil || executor.inner == nil {
		return orquestamcp.MCPEjecutarOrquestacionAppToolResultV0{}, fmt.Errorf(
			"%s: el gate del consejo esta exigido pero no hay puerto de decision cableado",
			CouncilGateMisconfiguredCodeV0,
		)
	}
	councilRef := CouncilRefForOrchestrationV0(input.RequestID, input.AppSpec)
	accepted, err := executor.config.Decision.CouncilDecisionAcceptedV0(ctx, councilRef)
	if err != nil || !accepted {
		return orquestamcp.MCPEjecutarOrquestacionAppToolResultV0{}, fmt.Errorf(
			"%s: el consejo debe aceptar la decision antes de orquestar la app",
			CouncilGateDecisionRequiredCodeV0,
		)
	}
	return executor.inner.Execute(ctx, input)
}
