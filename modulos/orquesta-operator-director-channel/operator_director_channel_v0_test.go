package orquestaoperatordirectorchannel

import (
	"context"
	"errors"
	"testing"
)

func TestOperatorMessageV0NormalizaValidaYPersisteAckDurable(t *testing.T) {
	dispatcher := &fakeOperatorDirectorDispatcherV0{}
	store := &fakeOperatorDirectorStoreV0{}
	result, issues, err := DispatchOperatorDirectorMessageV0(context.Background(), OperatorDirectorChannelServiceV0{
		Dispatcher: dispatcher,
		Store:      store,
	}, OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director-ref-1",
		Intent:     "estado",
		Body:       " Como va el run? ",
	})
	if err != nil || len(issues) != 0 {
		t.Fatalf("dispatch err=%v issues=%+v", err, issues)
	}
	if dispatcher.message.Intent != OperatorMessageIntentStatusV0 ||
		dispatcher.message.MessageRef != "operator-message-ref-request-ref-1" {
		t.Fatalf("message normalizado: %+v", dispatcher.message)
	}
	if result.AckRef == "" || result.MessageRef != dispatcher.message.MessageRef || result.ResponseText != "respuesta compacta" {
		t.Fatalf("ack inesperado: %+v", result)
	}
	if len(store.exchanges) != 1 || store.exchanges[0].Response.AckRef != result.AckRef {
		t.Fatalf("store durable no invocado: %+v", store.exchanges)
	}
}

func TestOperatorMessageV0RechazaRefsNoOpacasSinEjecutarPuertos(t *testing.T) {
	dispatcher := &fakeOperatorDirectorDispatcherV0{}
	store := &fakeOperatorDirectorStoreV0{}
	_, issues, err := DispatchOperatorDirectorMessageV0(context.Background(), OperatorDirectorChannelServiceV0{
		Dispatcher: dispatcher,
		Store:      store,
	}, OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director/ref",
		Body:       "hola",
	})
	if err != nil || len(issues) == 0 || dispatcher.called != 0 || len(store.exchanges) != 0 {
		t.Fatalf("validacion esperada err=%v issues=%+v called=%d store=%d", err, issues, dispatcher.called, len(store.exchanges))
	}
}

func TestOperatorMessageV0FallaSinStoreDurable(t *testing.T) {
	_, issues, err := DispatchOperatorDirectorMessageV0(context.Background(), OperatorDirectorChannelServiceV0{
		Dispatcher: &fakeOperatorDirectorDispatcherV0{},
	}, OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director-ref-1",
		Body:       "hola",
	})
	if len(issues) != 0 || !errors.Is(err, ErrOperatorDirectorStorePortUnavailableV0) {
		t.Fatalf("store requerido err=%v issues=%+v", err, issues)
	}
}

type fakeOperatorDirectorDispatcherV0 struct {
	called  int
	message OperatorMessageV0
}

func (f *fakeOperatorDirectorDispatcherV0) DispatchOperatorMessageV0(
	_ context.Context,
	message OperatorMessageV0,
) (OperatorDirectorResponseV0, error) {
	f.called++
	f.message = message
	return OperatorDirectorResponseV0{
		Status:       "accepted",
		ResponseRef:  "response-ref-1",
		ResponseText: "respuesta compacta",
		EvidenceRefs: []string{"evidence-ref-dispatch"},
	}, nil
}

type fakeOperatorDirectorStoreV0 struct {
	exchanges []OperatorDirectorExchangeV0
}

func (f *fakeOperatorDirectorStoreV0) SaveOperatorDirectorExchangeV0(
	_ context.Context,
	exchange OperatorDirectorExchangeV0,
) error {
	f.exchanges = append(f.exchanges, exchange)
	return nil
}
