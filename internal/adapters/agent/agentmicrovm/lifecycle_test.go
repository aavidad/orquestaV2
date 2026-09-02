package agentmicrovm

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

func TestAgentMicroVMLifecycleContractsRemainCompleteAndFailClosed(t *testing.T) {
	var lifecycle ports.AgentEnvironmentLifecycle = (*Adapter)(nil)
	var reconciler ports.AgentEnvironmentLifecycleReconciler = (*Adapter)(nil)
	adapter := &Adapter{}

	if _, err := lifecycle.(*Adapter).Quiesce(context.Background(), ports.AgentQuiesceRequest{}); ErrorCode(err) != CodeLifecycleRequestInvalid {
		t.Fatalf("Quiesce code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.Close(context.Background(), ports.AgentCloseRequest{}); ErrorCode(err) != CodeLifecycleRequestInvalid {
		t.Fatalf("Close code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := reconciler.(*Adapter).ReconcileQuiesce(context.Background(), ports.AgentQuiesceRequest{}); ErrorCode(err) != CodeLifecycleRequestInvalid {
		t.Fatalf("ReconcileQuiesce code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.ReconcilePreserve(context.Background(), ports.AgentPreserveRequest{}); ErrorCode(err) != CodePreserveRecoveryUnsupported {
		t.Fatalf("ReconcilePreserve code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.ReconcileClose(context.Background(), ports.AgentCloseRequest{}); ErrorCode(err) != CodeLifecycleRequestInvalid {
		t.Fatalf("ReconcileClose code=%q err=%v", ErrorCode(err), err)
	}
}

type lifecycleCloseClientStub struct {
	*launchClientStub
	reply    microvm.RespuestaCierre
	err      error
	keys     []string
	refs     []string
	requests []microvm.SolicitudCierre
}

func (client *lifecycleCloseClientStub) Cerrar(
	_ context.Context,
	key string,
	ref string,
	request microvm.SolicitudCierre,
) (microvm.RespuestaCierre, error) {
	client.keys = append(client.keys, key)
	client.refs = append(client.refs, ref)
	client.requests = append(client.requests, request)
	return client.reply, client.err
}

func TestAgentMicroVMCloseReturnsOnlyExactDurableC46Receipt(t *testing.T) {
	request := lifecycleCloseRequest(t)
	receiptRef := "cierre:" + strings.Repeat("c", 64)
	confirmedAt := uint64(43_000)
	client := &lifecycleCloseClientStub{launchClientStub: &launchClientStub{}}
	client.reply = microvm.RespuestaCierre{
		Ejecucion: microvm.RespuestaEjecucion{
			Referencia: request.Subject.ExternalRef, Estado: "cerrada", Revision: 20, Cerca: 23,
		},
		ClaveIdempotencia: request.IdempotencyKey, Estado: microvm.EstadoCierreConfirmada,
		ReceiptRef: &receiptRef, ConfirmadaUnixMS: &confirmedAt,
	}
	receipt, err := (&Adapter{client: client}).Close(context.Background(), request)
	if err != nil || ports.ValidateAgentCloseReceipt(request, receipt) != nil ||
		receipt.ReceiptRef != receiptRef || !receipt.ConfirmedAt.Equal(time.UnixMilli(43_000).UTC()) {
		t.Fatalf("receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
	}
	want := microvm.SolicitudCierre{
		RevisionEsperada: 18, Cerca: 23,
		ManifiestoSHA256Esperado: request.Preservation.PhysicalManifest.ManifestSHA256,
	}
	if !reflect.DeepEqual(client.keys, []string{request.IdempotencyKey}) ||
		!reflect.DeepEqual(client.refs, []string{request.Subject.ExternalRef}) ||
		!reflect.DeepEqual(client.requests, []microvm.SolicitudCierre{want}) {
		t.Fatalf("keys=%v refs=%v requests=%+v", client.keys, client.refs, client.requests)
	}
}

func TestAgentMicroVMCloseLeavesPendingEffectForReadOnlyReconciliation(t *testing.T) {
	request := lifecycleCloseRequest(t)
	client := &lifecycleCloseClientStub{launchClientStub: &launchClientStub{}}
	client.reply = microvm.RespuestaCierre{
		Ejecucion: microvm.RespuestaEjecucion{
			Referencia: request.Subject.ExternalRef, Estado: "preservada", Revision: 18, Cerca: 23,
		},
		ClaveIdempotencia: request.IdempotencyKey, Estado: microvm.EstadoCierrePendiente,
	}
	if receipt, err := (&Adapter{client: client}).Close(context.Background(), request); ErrorCode(err) != CodeLifecycleReceiptUnavailable || receipt != (ports.AgentCloseReceipt{}) {
		t.Fatalf("pending receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
	}
}

func lifecycleCloseRequest(t *testing.T) ports.AgentCloseRequest {
	t.Helper()
	_, preserve := historicalPreservationFixture(t)
	preserve.ExpectedToken.State = ports.AgentEnvironmentPreserved
	preserve.ExpectedToken.Revision, _ = ports.NewAgentPhysicalRevision("18")
	digest := strings.Repeat("b", 64)
	return ports.AgentCloseRequest{
		Subject: preserve.Subject, ExpectedToken: preserve.ExpectedToken,
		Preservation: ports.AgentPreservationBinding{
			ApplicationReceiptRef: "environment-receipt:b12:uds",
			PhysicalManifest: ports.AgentPhysicalPreservationBinding{
				ManifestRef: digest, ManifestSHA256: digest,
			},
		},
		IdempotencyKey: "close:b12:uds",
	}
}

func TestAgentMicroVMInspectTranslatesOnlyExactPhysicalLifecycleStates(t *testing.T) {
	_, preserve := historicalPreservationFixture(t)
	request := ports.AgentEnvironmentInspectRequest{Subject: preserve.Subject}
	cases := map[string]ports.AgentEnvironmentLifecycleState{
		"disponible": ports.AgentEnvironmentActive,
		"detenida":   ports.AgentEnvironmentQuiesced,
		"preservada": ports.AgentEnvironmentPreserved,
		"cerrada":    ports.AgentEnvironmentClosed,
	}
	for physicalState, expected := range cases {
		t.Run(physicalState, func(t *testing.T) {
			observer := &observationClientStub{physical: microvm.RespuestaEjecucion{
				Referencia: request.Subject.ExternalRef,
				Estado:     physicalState,
				Revision:   18,
				Cerca:      23,
			}}
			receipt, err := (&Adapter{observer: observer}).Inspect(context.Background(), request)
			if err != nil || ports.ValidateAgentEnvironmentInspectReceipt(request, receipt) != nil ||
				receipt.Token.State != expected {
				t.Fatalf("receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
			}
		})
	}

	for _, physicalState := range []string{
		"", "iniciando", "deteniendo", "preservando", "cerrando",
		"fallida", "cuarentena", "recuperando", "PRESERVADA",
	} {
		t.Run("rejected_"+physicalState, func(t *testing.T) {
			observer := &observationClientStub{physical: microvm.RespuestaEjecucion{
				Referencia: request.Subject.ExternalRef,
				Estado:     physicalState,
				Revision:   18,
				Cerca:      23,
			}}
			receipt, err := (&Adapter{observer: observer}).Inspect(context.Background(), request)
			if ErrorCode(err) != CodeLifecycleResponseInsufficient ||
				receipt != (ports.AgentEnvironmentInspectReceipt{}) {
				t.Fatalf("receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
			}
		})
	}
}

type lifecycleQuiesceClient struct {
	*launchClientStub
	reply    microvm.RespuestaDetencion
	err      error
	keys     []string
	refs     []string
	requests []microvm.SolicitudDetencion
}

func (client *lifecycleQuiesceClient) Detener(
	_ context.Context,
	key string,
	ref string,
	request microvm.SolicitudDetencion,
) (microvm.RespuestaDetencion, error) {
	client.keys = append(client.keys, key)
	client.refs = append(client.refs, ref)
	client.requests = append(client.requests, request)
	return client.reply, client.err
}

func TestAgentMicroVMQuiesceAndReconcileRepeatExactCooperativeRequest(t *testing.T) {
	request := lifecycleQuiesceRequest(t)
	client := &lifecycleQuiesceClient{launchClientStub: &launchClientStub{}}
	client.reply = lifecycleQuiescePendingResponse(request)
	adapter := &Adapter{client: client}

	first, err := adapter.Quiesce(context.Background(), request)
	if err != nil || ports.ValidateAgentQuiesceReceipt(request, first) != nil ||
		first.NextToken.State != ports.AgentEnvironmentQuiescing || first.ReceiptRef != "" || !first.ConfirmedAt.IsZero() {
		t.Fatalf("pending receipt=%+v code=%q err=%v", first, ErrorCode(err), err)
	}
	replayed, err := adapter.ReconcileQuiesce(context.Background(), request)
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("replayed receipt=%+v code=%q err=%v", replayed, ErrorCode(err), err)
	}
	wantPhysical := microvm.SolicitudDetencion{
		RevisionEsperada: 17,
		Cerca:            23,
		Modo:             microvm.ModoDetencionCooperativa,
	}
	if !reflect.DeepEqual(client.keys, []string{request.IdempotencyKey, request.IdempotencyKey}) ||
		!reflect.DeepEqual(client.refs, []string{request.Subject.ExternalRef, request.Subject.ExternalRef}) ||
		!reflect.DeepEqual(client.requests, []microvm.SolicitudDetencion{wantPhysical, wantPhysical}) {
		t.Fatalf("keys=%v refs=%v requests=%+v", client.keys, client.refs, client.requests)
	}
}

func TestAgentMicroVMQuiesceAcceptsOnlyCooperativeOrAbsentConfirmation(t *testing.T) {
	request := lifecycleQuiesceRequest(t)
	for _, effective := range []microvm.ModoDetencionEfectivoV1{
		microvm.ModoDetencionEfectivoCooperativa,
		microvm.ModoDetencionEfectivoYaAusente,
	} {
		t.Run(string(effective), func(t *testing.T) {
			client := &lifecycleQuiesceClient{launchClientStub: &launchClientStub{}}
			client.reply = lifecycleQuiesceConfirmedResponse(request, effective)
			receipt, err := (&Adapter{client: client}).Quiesce(context.Background(), request)
			if err != nil || ports.ValidateAgentQuiesceReceipt(request, receipt) != nil ||
				receipt.NextToken.State != ports.AgentEnvironmentQuiesced ||
				receipt.ReceiptRef != *client.reply.ReceiptRef ||
				!receipt.ConfirmedAt.Equal(time.UnixMilli(int64(*client.reply.ConfirmadaUnixMS)).UTC()) {
				t.Fatalf("receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
			}
		})
	}

	client := &lifecycleQuiesceClient{launchClientStub: &launchClientStub{}}
	client.reply = lifecycleQuiesceConfirmedResponse(request, microvm.ModoDetencionEfectivoForzada)
	if receipt, err := (&Adapter{client: client}).Quiesce(context.Background(), request); ErrorCode(err) != CodeLifecycleResponseInsufficient || receipt != (ports.AgentQuiesceReceipt{}) {
		t.Fatalf("forced receipt=%+v code=%q err=%v", receipt, ErrorCode(err), err)
	}
}

func TestAgentMicroVMQuiesceRejectsNonPhysicalCounterBeforeEffect(t *testing.T) {
	request := lifecycleQuiesceRequest(t)
	request.ExpectedToken.Revision, _ = ports.NewAgentPhysicalRevision("revision:opaque")
	client := &lifecycleQuiesceClient{launchClientStub: &launchClientStub{}}
	if _, err := (&Adapter{client: client}).Quiesce(context.Background(), request); ErrorCode(err) != CodeLifecycleRequestInvalid || len(client.requests) != 0 {
		t.Fatalf("calls=%d code=%q err=%v", len(client.requests), ErrorCode(err), err)
	}
}

func lifecycleQuiesceRequest(t *testing.T) ports.AgentQuiesceRequest {
	t.Helper()
	_, preserve := historicalPreservationFixture(t)
	preserve.ExpectedToken.State = ports.AgentEnvironmentActive
	return ports.AgentQuiesceRequest{
		Subject:        preserve.Subject,
		ExpectedToken:  preserve.ExpectedToken,
		IdempotencyKey: "quiesce:b12:uds",
	}
}

func lifecycleQuiescePendingResponse(request ports.AgentQuiesceRequest) microvm.RespuestaDetencion {
	return microvm.RespuestaDetencion{
		Ejecucion: microvm.RespuestaEjecucion{
			Referencia: request.Subject.ExternalRef,
			Estado:     "deteniendo",
			Revision:   18,
			Cerca:      23,
		},
		ClaveIdempotencia: request.IdempotencyKey,
		Estado:            microvm.EstadoDetencionPendiente,
		ModoSolicitado:    microvm.ModoDetencionCooperativa,
	}
}

func lifecycleQuiesceConfirmedResponse(
	request ports.AgentQuiesceRequest,
	effective microvm.ModoDetencionEfectivoV1,
) microvm.RespuestaDetencion {
	receiptRef := "detencion:quiesce:b12:uds"
	confirmedAt := uint64(42_000)
	return microvm.RespuestaDetencion{
		Ejecucion: microvm.RespuestaEjecucion{
			Referencia: request.Subject.ExternalRef,
			Estado:     "detenida",
			Revision:   18,
			Cerca:      23,
		},
		ClaveIdempotencia: request.IdempotencyKey,
		Estado:            microvm.EstadoDetencionConfirmada,
		ModoSolicitado:    microvm.ModoDetencionCooperativa,
		ModoEfectivo:      &effective,
		ReceiptRef:        &receiptRef,
		ConfirmadaUnixMS:  &confirmedAt,
	}
}
