package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	channel "orquesta/modulos/orquesta-operator-director-channel"
)

func TestOperatorDirectorMessageMCPV0RegistraToolYDelegaConStoreDurable(t *testing.T) {
	dispatcher := &fakeOperatorDirectorMessageDispatchMCPV0{}
	store := &fakeOperatorDirectorMessageStoreMCPV0{}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		OperatorDirectorMessage: channel.OperatorDirectorChannelServiceV0{
			Dispatcher: dispatcher,
			Store:      store,
		},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	tool, ok := transport.tools[channel.OperatorDirectorMessageToolNameV0]
	if !ok || tool.ResourceURI != channel.OperatorDirectorMessageResourceURIV0 {
		t.Fatalf("tool no registrado: %+v ok=%v", tool, ok)
	}

	output, err := transport.CallToolV0(context.Background(), channel.OperatorDirectorMessageToolNameV0, channel.OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director-ref-1",
		Intent:     "observe_run",
		Body:       "observa run-ref-1",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPOperatorDirectorMessageToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPOperatorDirectorMessageEstadoOKV0 ||
		result.Response == nil ||
		result.Response.AckRef == "" ||
		dispatcher.called != 1 ||
		len(store.exchanges) != 1 {
		t.Fatalf("resultado inesperado: result=%+v called=%d store=%d", result, dispatcher.called, len(store.exchanges))
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1200)
}

func TestOperatorMessageMCPV0SinStoreDevuelveErrorCompacto(t *testing.T) {
	executor := MCPTransportOperatorDirectorMessageExecutorV0{
		Service: channel.OperatorDirectorChannelServiceV0{Dispatcher: &fakeOperatorDirectorMessageDispatchMCPV0{}},
	}
	result := executor.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director-ref-1",
		Body:       "status",
	})
	if result.Estado != MCPOperatorDirectorMessageEstadoErrorV0 ||
		result.ErrorCode != channel.ErrOperatorMessageStoreFailedV0 {
		t.Fatalf("error esperado: %+v", result)
	}
}

type fakeOperatorDirectorMessageDispatchMCPV0 struct {
	called int
}

func (f *fakeOperatorDirectorMessageDispatchMCPV0) DispatchOperatorMessageV0(
	_ context.Context,
	message channel.OperatorMessageV0,
) (channel.OperatorDirectorResponseV0, error) {
	f.called++
	return channel.OperatorDirectorResponseV0{
		MessageRef:   message.MessageRef,
		Status:       "accepted",
		ResponseRef:  "response-ref-mcp",
		ResponseText: "ack compacto",
		EvidenceRefs: []string{"evidence-ref-mcp-dispatch"},
	}, nil
}

type fakeOperatorDirectorMessageStoreMCPV0 struct {
	exchanges []channel.OperatorDirectorExchangeV0
}

func (f *fakeOperatorDirectorMessageStoreMCPV0) SaveOperatorDirectorExchangeV0(
	_ context.Context,
	exchange channel.OperatorDirectorExchangeV0,
) error {
	f.exchanges = append(f.exchanges, exchange)
	return nil
}
