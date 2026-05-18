package orquestaappcodexstack

import (
	"context"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type compositeAgentDeliveryObservationSourceV0 struct {
	Sources []orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0
}

func (source compositeAgentDeliveryObservationSourceV0) BuildAgentDeliveryObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0)
	for _, child := range source.Sources {
		if child == nil {
			continue
		}
		next, err := child.BuildAgentDeliveryObservationsV0(ctx, request)
		if err != nil {
			return nil, err
		}
		observations = append(observations, next...)
	}
	return observations, nil
}

type domainWorkRecoveryDeliverySourceV0 struct {
	Stores         StoresV0
	DomainWork     orquestamcp.MCPDomainWorkExecutorPortV0
	DomainDelivery DomainWorkDeliveryBridgeConfigV0
}

func (source domainWorkRecoveryDeliverySourceV0) BuildAgentDeliveryObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	stack := source.stackV0()
	return stack.recoverableDomainWorkDeliveryObservationsV0(
		ctx,
		DrainRunRequestV0{
			RunRef:        request.Run.RunID,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			WaitAgentRefs: request.WaitAgentRefs,
		},
		request.Run,
	)
}

func (source domainWorkRecoveryDeliverySourceV0) readyV0() bool {
	return source.stackV0().domainWorkDeliveryBridgeReadyV0()
}

func (source domainWorkRecoveryDeliverySourceV0) stackV0() StackV0 {
	return StackV0{
		Stores:         source.Stores,
		DomainWork:     source.DomainWork,
		DomainDelivery: source.DomainDelivery,
	}
}
