package agentmicrovm

import (
	"context"
	"strconv"

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
	receipt := ports.AgentEnvironmentInspectReceipt{Subject: request.Subject, Token: ports.AgentEnvironmentLifecycleToken{PhysicalToken: physical, Revision: revision, Fence: fence, State: ports.AgentEnvironmentActive}}
	if ports.ValidateAgentEnvironmentInspectReceipt(request, receipt) != nil {
		return ports.AgentEnvironmentInspectReceipt{}, fail(CodeLifecycleResponseInsufficient, nil)
	}
	return receipt, nil
}

func (*Adapter) Quiesce(context.Context, ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
}
func (*Adapter) Close(context.Context, ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	return ports.AgentCloseReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
}
func (*Adapter) ReconcileQuiesce(context.Context, ports.AgentQuiesceRequest) (ports.AgentQuiesceReceipt, error) {
	return ports.AgentQuiesceReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
}
func (*Adapter) ReconcileClose(context.Context, ports.AgentCloseRequest) (ports.AgentCloseReceipt, error) {
	return ports.AgentCloseReceipt{}, fail(CodeLifecycleReceiptUnavailable, nil)
}
