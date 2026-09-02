package agentmicrovm

import (
	"context"
	"strconv"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeLifecycleRequestInvalid       = "agentmicrovm.lifecycle_request_invalid"
	CodeLifecycleResponseInsufficient = "agentmicrovm.lifecycle_response_insufficient"
	CodeLifecycleReceiptUnavailable   = "agentmicrovm.lifecycle_receipt_unavailable"
)

var _ ports.AgentEnvironmentLifecycle = (*Adapter)(nil)
var _ ports.AgentEnvironmentLifecycleReconciler = (*Adapter)(nil)

func (adapter *Adapter) Inspect(ctx context.Context, request ports.AgentEnvironmentInspectRequest) (ports.AgentEnvironmentInspectReceipt, error) {
	if adapter == nil || nilInterface(ctx) || request.Subject.ExternalRef == "" {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleRequestInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentEnvironmentInspectReceipt{}, err
	}
	response, err := adapter.observer.Observar(ctx, request.Subject.ExternalRef)
	if err != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleReceiptUnavailable, err)
	}
	if response.Referencia != request.Subject.ExternalRef || response.Revision == 0 || response.Cerca == 0 {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	state, ok := inspectLifecycleState(response.Estado)
	if !ok {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	physical, err := ports.NewAgentPhysicalToken(response.Referencia)
	if err != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, err)
	}
	revision, err := ports.NewAgentPhysicalRevision(strconv.FormatUint(response.Revision, 10))
	if err != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, err)
	}
	fence, err := ports.NewAgentPhysicalFence(strconv.FormatUint(response.Cerca, 10))
	if err != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, err)
	}
	receipt := ports.AgentEnvironmentInspectReceipt{Subject: request.Subject, Token: ports.AgentEnvironmentLifecycleToken{PhysicalToken: physical, Revision: revision, Fence: fence, State: state}}
	if ports.ValidateAgentEnvironmentInspectReceipt(request, receipt) != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	return receipt, nil
}

func inspectLifecycleState(physical string) (ports.AgentEnvironmentLifecycleState, bool) {
	switch physical {
	case "disponible":
		return ports.AgentEnvironmentActive, true
	case "detenida":
		return ports.AgentEnvironmentQuiesced, true
	case "preservada":
		return ports.AgentEnvironmentPreserved, true
	case "cerrada":
		return ports.AgentEnvironmentClosed, true
	default:
		return "", false
	}
}

// Quiesce uses only the cooperative branch of the public Detener contract.
// Application completion remains distinct from AgentController.Stop because
// this adapter never asks for, nor accepts, a forced stop here.
func (adapter *Adapter) Quiesce(ctx context.Context, request ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	return adapter.quiesce(ctx, request)
}

type lifecycleCloseClient interface {
	Cerrar(context.Context, string, string, microvm.SolicitudCierre) (microvm.RespuestaCierre, error)
}

func (adapter *Adapter) Close(ctx context.Context, request ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	return adapter.close(ctx, request)
}
func (adapter *Adapter) ReconcileQuiesce(ctx context.Context, request ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	// Detener is the sibling's idempotent reconciliation boundary: repeating
	// the exact key and request retrieves the same pending or terminal fact.
	return adapter.quiesce(ctx, request)
}
func (adapter *Adapter) ReconcileClose(ctx context.Context, request ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	// Application calls this boundary only after Inspect has observed closed.
	// Repeating the exact idempotency key retrieves the durable C46 receipt.
	return adapter.close(ctx, request)
}

func (adapter *Adapter) quiesce(ctx context.Context, request ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(ctx) ||
		ports.ValidateAgentQuiesceRequest(request) != nil {
		return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleRequestInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentQuiesceReceipt{}, err
	}
	revision, err := strconv.ParseUint(request.ExpectedToken.Revision.String(), 10, 64)
	if err != nil || revision == 0 || revision > maxDurableCounter {
		return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleRequestInvalid, err)
	}
	fence, err := strconv.ParseUint(request.ExpectedToken.Fence.String(), 10, 64)
	if err != nil || fence == 0 || fence > maxDurableCounter {
		return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleRequestInvalid, err)
	}
	response, err := adapter.client.Detener(ctx, request.IdempotencyKey, request.Subject.ExternalRef,
		microvm.SolicitudDetencion{
			RevisionEsperada: revision,
			Cerca:            fence,
			Modo:             microvm.ModoDetencionCooperativa,
		})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentQuiesceReceipt{}, contextErr
		}
		return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleReceiptUnavailable, err)
	}
	receipt, ok := translateQuiesceReceipt(request, revision, fence, response)
	if !ok || ports.ValidateAgentQuiesceReceipt(request, receipt) != nil {
		return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	return receipt, nil
}

func translateQuiesceReceipt(
	request ports.AgentQuiesceRequest,
	revision uint64,
	fence uint64,
	response microvm.RespuestaDetencion,
) (ports.AgentQuiesceReceipt, bool) {
	if response.Ejecucion.Referencia != request.Subject.ExternalRef ||
		response.Ejecucion.Revision <= revision || response.Ejecucion.Revision > maxDurableCounter ||
		response.Ejecucion.Cerca != fence || response.ClaveIdempotencia != request.IdempotencyKey ||
		response.ModoSolicitado != microvm.ModoDetencionCooperativa {
		return ports.AgentQuiesceReceipt{}, false
	}
	nextRevision, err := ports.NewAgentPhysicalRevision(strconv.FormatUint(response.Ejecucion.Revision, 10))
	if err != nil {
		return ports.AgentQuiesceReceipt{}, false
	}
	receipt := ports.AgentQuiesceReceipt{
		Subject:       request.Subject,
		PreviousToken: request.ExpectedToken,
		NextToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: request.ExpectedToken.PhysicalToken,
			Revision:      nextRevision,
			Fence:         request.ExpectedToken.Fence,
		},
		IdempotencyKey: request.IdempotencyKey,
	}
	if response.Estado == microvm.EstadoDetencionPendiente {
		if response.Ejecucion.Estado != "deteniendo" || response.ModoEfectivo != nil ||
			response.ReceiptRef != nil || response.ConfirmadaUnixMS != nil {
			return ports.AgentQuiesceReceipt{}, false
		}
		receipt.NextToken.State = ports.AgentEnvironmentQuiescing
		return receipt, true
	}
	if response.Estado != microvm.EstadoDetencionConfirmada || response.Ejecucion.Estado != "detenida" ||
		response.ModoEfectivo == nil || response.ReceiptRef == nil || response.ConfirmadaUnixMS == nil ||
		*response.ConfirmadaUnixMS == 0 || *response.ConfirmadaUnixMS > maxDurableCounter {
		return ports.AgentQuiesceReceipt{}, false
	}
	if *response.ModoEfectivo != microvm.ModoDetencionEfectivoCooperativa &&
		*response.ModoEfectivo != microvm.ModoDetencionEfectivoYaAusente {
		return ports.AgentQuiesceReceipt{}, false
	}
	receipt.NextToken.State = ports.AgentEnvironmentQuiesced
	receipt.ReceiptRef = *response.ReceiptRef
	receipt.ConfirmedAt = time.UnixMilli(int64(*response.ConfirmadaUnixMS)).UTC()
	return receipt, true
}

func (adapter *Adapter) close(ctx context.Context, request ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	if adapter == nil || nilInterface(ctx) || ports.ValidateAgentCloseRequest(request) != nil {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleRequestInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentCloseReceipt{}, err
	}
	revision, err := strconv.ParseUint(request.ExpectedToken.Revision.String(), 10, 64)
	if err != nil || revision == 0 || revision > maxDurableCounter {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleRequestInvalid, err)
	}
	fence, err := strconv.ParseUint(request.ExpectedToken.Fence.String(), 10, 64)
	if err != nil || fence == 0 || fence > maxDurableCounter {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleRequestInvalid, err)
	}
	client, ok := adapter.client.(lifecycleCloseClient)
	if !ok || nilInterface(client) {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
	}
	response, err := client.Cerrar(ctx, request.IdempotencyKey, request.Subject.ExternalRef,
		microvm.SolicitudCierre{
			RevisionEsperada: revision, Cerca: fence,
			ManifiestoSHA256Esperado: request.Preservation.PhysicalManifest.ManifestSHA256,
		})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentCloseReceipt{}, contextErr
		}
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleReceiptUnavailable, err)
	}
	// Pending proves admission, not closure. We deliberately leave the durable
	// attempt in reconciliation and do not repeat Cerrar until Inspect observes
	// the terminal physical state.
	if response.Estado != microvm.EstadoCierreConfirmada || response.ReceiptRef == nil ||
		response.ConfirmadaUnixMS == nil || *response.ConfirmadaUnixMS == 0 ||
		*response.ConfirmadaUnixMS > maxDurableCounter {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
	}
	nextRevision, err := ports.NewAgentPhysicalRevision(strconv.FormatUint(response.Ejecucion.Revision, 10))
	if err != nil {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleResponseInsufficient, err)
	}
	receipt := ports.AgentCloseReceipt{
		Subject: request.Subject, PreviousToken: request.ExpectedToken,
		NextToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: request.ExpectedToken.PhysicalToken,
			Revision:      nextRevision, Fence: request.ExpectedToken.Fence, State: ports.AgentEnvironmentClosed,
		},
		Preservation: request.Preservation, IdempotencyKey: request.IdempotencyKey,
		ReceiptRef:  *response.ReceiptRef,
		ConfirmedAt: time.UnixMilli(int64(*response.ConfirmadaUnixMS)).UTC(),
	}
	if ports.ValidateAgentCloseReceipt(request, receipt) != nil {
		return ports.AgentCloseReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	return receipt, nil
}
