package orquestaruntime

import (
	"context"
	"sync"
)

type ExternalAgentProcessCommandResolverWithReceiptV0 struct {
	Resolver ExternalAgentProcessCommandResolverV0

	mu      sync.Mutex
	receipt ProcessRuntimeLaunchReceiptV0
}

func NewExternalAgentProcessCommandResolverWithReceiptV0(
	resolver ExternalAgentProcessCommandResolverV0,
) *ExternalAgentProcessCommandResolverWithReceiptV0 {
	return &ExternalAgentProcessCommandResolverWithReceiptV0{Resolver: resolver}
}

func (r *ExternalAgentProcessCommandResolverWithReceiptV0) ResolveExternalAgentProcessCommandV0(
	ctx context.Context,
	spec ExternalAgentLaunchSpecV0,
) (ProcessRuntimeLaunchRequestV0, []ExternalAgentConnectorErrorV0) {
	if r == nil || r.Resolver == nil {
		return ProcessRuntimeLaunchRequestV0{}, []ExternalAgentConnectorErrorV0{
			externalAgentProcessErrorV0(ExternalAgentResolverUnavailableV0, "resolver", spec.CorrelationID, false),
		}
	}
	req, issues := r.Resolver.ResolveExternalAgentProcessCommandV0(ctx, spec)
	if len(issues) == 0 {
		r.setReceiptV0(NewProcessRuntimeLaunchReceiptFromSpecV0(spec, "command_resolver_resolution"))
	}
	return req, issues
}

func (r *ExternalAgentProcessCommandResolverWithReceiptV0) ExternalAgentProcessCommandResolutionReceiptV0() ProcessRuntimeLaunchReceiptV0 {
	if r == nil {
		return ProcessRuntimeLaunchReceiptV0{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.receipt
}

func (r *ExternalAgentProcessCommandResolverWithReceiptV0) setReceiptV0(
	receipt ProcessRuntimeLaunchReceiptV0,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.receipt = receipt
}
