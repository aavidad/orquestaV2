package agentmicrovm

import (
	"context"
	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
	"orquesta/internal/ports"
	"time"
)

const (
	CodeControlRequestInvalid     = "agentmicrovm.control_request_invalid"
	CodeControlObservationInvalid = "agentmicrovm.control_observation_invalid"
	CodeControlObservationFailed  = "agentmicrovm.control_observation_failed"
	CodeControlCanceledBeforeCall = "agentmicrovm.control_canceled_before_call"
	CodeControlUnavailable        = "agentmicrovm.control_unavailable"
	CodeControlResponseInvalid    = "agentmicrovm.control_response_invalid"
)

func (adapter *Adapter) ControlCapabilities(ctx context.Context) (ports.AgentControlCapabilities, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(ctx) {
		return ports.AgentControlCapabilities{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.negotiateControl(ctx); err != nil {
		return ports.AgentControlCapabilities{}, err
	}
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (adapter *Adapter) Stop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(adapter.observer) || nilInterface(ctx) {
		return ports.AgentStopReceipt{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentStopReceipt{}, fail(CodeControlCanceledBeforeCall, err)
	}
	if err := adapter.validateStopRequest(request); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if err := adapter.negotiateControl(ctx); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	physical, err := adapter.observer.Observar(ctx, request.ExternalRef)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentStopReceipt{}, fail(CodeControlCanceledBeforeCall, contextErr)
		}
		return ports.AgentStopReceipt{}, fail(CodeControlObservationFailed, err)
	}
	mode, modeOK := physicalStopMode(request.Mode)
	if physical.Referencia != request.ExternalRef || physical.Estado != "disponible" ||
		physical.Revision == 0 || physical.Revision > maxDurableCounter ||
		physical.Cerca == 0 || physical.Cerca > maxDurableCounter || !modeOK {
		return ports.AgentStopReceipt{}, fail(CodeControlObservationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentStopReceipt{}, fail(CodeControlCanceledBeforeCall, err)
	}
	response, err := adapter.client.Detener(ctx, request.IdempotencyKey, request.ExternalRef,
		microvm.SolicitudDetencion{RevisionEsperada: physical.Revision, Cerca: physical.Cerca, Modo: mode})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentStopReceipt{}, contextErr
		}
		return ports.AgentStopReceipt{}, fail(CodeControlUnavailable, err)
	}
	receipt, ok := translateStopReceipt(request, physical, response)
	if !ok || ports.ValidateAgentStopReceipt(request, receipt) != nil {
		return ports.AgentStopReceipt{}, fail(CodeControlResponseInvalid, nil)
	}
	return receipt, nil
}
func (adapter *Adapter) validateStopRequest(request ports.AgentStopRequest) error {
	if ports.ValidateAgentStopRequest(request) != nil || !validPhysicalExecutionRef(request.ExternalRef) ||
		request.ProviderRef != adapter.capabilities.ProviderRef || request.ModelRef != adapter.capabilities.ModelRef ||
		request.AgentRef != adapter.capabilities.AgentRef {
		return fail(CodeControlRequestInvalid, nil)
	}
	return nil
}
func (adapter *Adapter) negotiateControl(ctx context.Context) error {
	response, err := adapter.client.Capacidades(ctx)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return fail(CodeControlCanceledBeforeCall, contextErr)
		}
		return classifyCapabilitiesError(err)
	}
	if response.Protocolo != microvm.ProtocoloLocal || !validRemoteVersion(response.Version) {
		return fail(CodeProtocolIncompatible, nil)
	}
	if !supportsRemoteOperations(response.Operaciones, operationObserveExecution, operationStopExecution) {
		return fail(CodeOperationUnsupported, nil)
	}
	return nil
}
func physicalStopMode(mode ports.AgentStopMode) (microvm.ModoDetencionSolicitadoV1, bool) {
	if mode == ports.AgentStopCooperative {
		return microvm.ModoDetencionCooperativa, true
	}
	return microvm.ModoDetencionForzada, mode == ports.AgentStopForced
}
func translateStopReceipt(request ports.AgentStopRequest, physical microvm.RespuestaEjecucion,
	response microvm.RespuestaDetencion) (ports.AgentStopReceipt, bool) {
	wantedMode, validMode := physicalStopMode(request.Mode)
	if !validMode || response.Ejecucion.Referencia != request.ExternalRef ||
		response.Ejecucion.Revision < physical.Revision || response.Ejecucion.Revision > maxDurableCounter ||
		response.Ejecucion.Cerca != physical.Cerca || response.ClaveIdempotencia != request.IdempotencyKey ||
		response.ModoSolicitado != wantedMode {
		return ports.AgentStopReceipt{}, false
	}
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
	}
	if response.Estado == microvm.EstadoDetencionPendiente {
		if response.Ejecucion.Estado != "deteniendo" || response.ModoEfectivo != nil ||
			response.ReceiptRef != nil || response.ConfirmadaUnixMS != nil {
			return ports.AgentStopReceipt{}, false
		}
		receipt.Status = ports.AgentStopPending
		return receipt, true
	}
	if response.Estado != microvm.EstadoDetencionConfirmada || response.Ejecucion.Estado != "detenida" ||
		response.ModoEfectivo == nil || response.ReceiptRef == nil || response.ConfirmadaUnixMS == nil ||
		*response.ConfirmadaUnixMS == 0 || *response.ConfirmadaUnixMS > maxDurableCounter {
		return ports.AgentStopReceipt{}, false
	}
	switch *response.ModoEfectivo {
	case microvm.ModoDetencionEfectivoCooperativa:
		if request.Mode != ports.AgentStopCooperative {
			return ports.AgentStopReceipt{}, false
		}
	case microvm.ModoDetencionEfectivoForzada:
		if request.Mode != ports.AgentStopForced {
			return ports.AgentStopReceipt{}, false
		}
	case microvm.ModoDetencionEfectivoYaAusente:
		receipt.Status = ports.AgentStopAlreadyStopped
	default:
		return ports.AgentStopReceipt{}, false
	}
	if receipt.Status == "" {
		receipt.Status = ports.AgentStopped
	}
	receipt.ReceiptRef = *response.ReceiptRef
	receipt.ConfirmedAt = time.UnixMilli(int64(*response.ConfirmadaUnixMS)).UTC()
	return receipt, true
}
