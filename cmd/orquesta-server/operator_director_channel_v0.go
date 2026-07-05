package main

import (
	"context"
	"sync"

	channel "orquesta/modulos/orquesta-operator-director-channel"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

type operatorDirectorChannelBridgeV0 struct {
	Query operator.OperatorMCPDirectedQueryPortV0
}

func newOperatorDirectorChannelServiceV0(
	query operator.OperatorMCPDirectedQueryPortV0,
	store channel.OperatorDirectorExchangeStorePortV0,
) channel.OperatorDirectorChannelServiceV0 {
	return channel.OperatorDirectorChannelServiceV0{
		Dispatcher: operatorDirectorChannelBridgeV0{Query: query},
		Store:      store,
	}
}

func (bridge operatorDirectorChannelBridgeV0) DispatchOperatorMessageV0(
	_ context.Context,
	message channel.OperatorMessageV0,
) (channel.OperatorDirectorResponseV0, error) {
	if bridge.Query == nil {
		return channel.OperatorDirectorResponseV0{}, channel.ErrOperatorDirectorDispatchPortUnavailableV0
	}
	queryRef := message.MessageRef
	if queryRef == "" {
		queryRef = "operator-query-ref-" + message.RequestRef
	}
	result, err := bridge.Query.RaiseOperatorDirectedQueryV0(operator.OperatorDirectedQueryV0{
		QueryRef:          queryRef,
		TargetRef:         message.TargetRef,
		QueryConnectorRef: firstNonEmptyOperatorDirectorChannelV0(message.AdapterRef, "query-connector-ref-operator-director-channel"),
		Question:          message.Body,
		EvidenceRefs:      message.EvidenceRefs,
	})
	if err != nil {
		return channel.OperatorDirectorResponseV0{}, err
	}
	status := "accepted"
	if !result.Accepted {
		status = "pending"
	}
	return channel.OperatorDirectorResponseV0{
		MessageRef:   message.MessageRef,
		Status:       status,
		Summary:      result.NextAction,
		ResponseRef:  result.AnswerRef,
		ResponseText: result.NextAction,
		EvidenceRefs: result.TraceRefs,
	}, nil
}

type operatorDirectorChannelMemoryStoreV0 struct {
	mu        sync.Mutex
	exchanges []channel.OperatorDirectorExchangeV0
}

func newOperatorDirectorChannelMemoryStoreV0() *operatorDirectorChannelMemoryStoreV0 {
	return &operatorDirectorChannelMemoryStoreV0{}
}

func (store *operatorDirectorChannelMemoryStoreV0) SaveOperatorDirectorExchangeV0(
	_ context.Context,
	exchange channel.OperatorDirectorExchangeV0,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.exchanges = append(store.exchanges, exchange)
	return nil
}

func (store *operatorDirectorChannelMemoryStoreV0) ExchangesV0() []channel.OperatorDirectorExchangeV0 {
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]channel.OperatorDirectorExchangeV0, len(store.exchanges))
	copy(out, store.exchanges)
	return out
}

func firstNonEmptyOperatorDirectorChannelV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
