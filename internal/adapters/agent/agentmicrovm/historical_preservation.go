package agentmicrovm

import (
	"context"
	"strconv"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeHistoricalAuthorityInvalid   = "agentmicrovm.historical_authority_invalid"
	CodeHistoricalSubjectMismatch    = "agentmicrovm.historical_subject_mismatch"
	CodePreserveUnavailable          = "agentmicrovm.preserve_unavailable"
	CodePreserveResponseInsufficient = "agentmicrovm.preserve_response_insufficient"
	CodePreserveRecoveryUnsupported  = "agentmicrovm.preserve_recovery_unsupported"
)

type historicalPreservationClient interface {
	Preservar(context.Context, string, string, microvm.SolicitudPreservacion) (microvm.RespuestaPreservacion, error)
}

// PreserveWithAuthority translates only immutable authority resolved by
// application. Agente MicroVM's stable response currently omits manifest ref,
// seal time and an idempotency receipt. The physical call may therefore have
// applied, but it cannot be promoted to an AgentPreserveReceipt.
func (adapter *Adapter) PreserveWithAuthority(
	ctx context.Context,
	authority ports.AgentHistoricalRuntimeAuthority,
	request ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	if adapter == nil || nilInterface(ctx) || ports.ValidateAgentHistoricalRuntimeAuthority(authority) != nil ||
		ports.ValidateAgentPreserveRequest(request) != nil {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, nil)
	}
	if authority.Subject != request.Subject {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalSubjectMismatch, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentPreserveReceipt{}, err
	}
	client, ok := adapter.client.(historicalPreservationClient)
	if !ok || nilInterface(client) {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveUnavailable, nil)
	}
	revision, err := strconv.ParseUint(request.ExpectedToken.Revision.String(), 10, 64)
	if err != nil || revision == 0 || revision > maxDurableCounter {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, err)
	}
	fence, err := strconv.ParseUint(request.ExpectedToken.Fence.String(), 10, 64)
	if err != nil || fence == 0 || fence > maxDurableCounter {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, err)
	}
	response, err := client.Preservar(ctx, request.IdempotencyKey, request.Subject.ExternalRef,
		microvm.SolicitudPreservacion{RevisionEsperada: revision, Cerca: fence})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentPreserveReceipt{}, contextErr
		}
		return ports.AgentPreserveReceipt{}, fail(CodePreserveUnavailable, err)
	}
	// Validate every subject value the public response does expose. Even a
	// coherent response remains insufficient as a terminal receipt.
	if response.Ejecucion.Referencia != request.Subject.ExternalRef ||
		response.Ejecucion.Cerca != fence || response.Ejecucion.Revision == 0 ||
		response.RevisionTrabajo == 0 || response.ManifiestoSHA256 == "" || response.ManifiestoBytes == 0 {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, nil)
	}
	return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, nil)
}

// Preserve cannot recover the missing historical launch fence and digests
// from AgentPreserveRequest, so the generic lifecycle boundary is closed.
func (adapter *Adapter) Preserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, nil)
}

// ReconcilePreserve is read-only, but the neutral request contains neither the
// manifest ref nor its exact byte count required by the stable sibling API.
func (adapter *Adapter) ReconcilePreserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, fail(CodePreserveRecoveryUnsupported, nil)
}
