package agentmicrovm

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

func TestPreserveWithAuthorityUsesPublicUnixAPIOnceButRejectsInsufficientReceipt(t *testing.T) {
	authority, request := historicalPreservationFixture(t)
	calls := 0
	handler := http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		response.Header().Set(microvm.CabeceraProtocolo, microvm.ProtocoloLocal)
		if incoming.Method != http.MethodPost ||
			incoming.URL.Path != "/v1/ejecuciones/"+request.Subject.ExternalRef+"/preservacion" {
			http.NotFound(response, incoming)
			return
		}
		calls++
		var got microvm.SolicitudPreservacion
		if err := json.NewDecoder(incoming.Body).Decode(&got); err != nil ||
			got != (microvm.SolicitudPreservacion{RevisionEsperada: 17, Cerca: 23}) ||
			incoming.Header.Get(microvm.CabeceraIdempotencia) != request.IdempotencyKey {
			t.Errorf("request=%+v key=%q err=%v", got, incoming.Header.Get(microvm.CabeceraIdempotencia), err)
		}
		_ = json.NewEncoder(response).Encode(microvm.RespuestaPreservacion{
			Ejecucion: microvm.RespuestaEjecucion{
				Referencia: request.Subject.ExternalRef, Estado: "preservada", Revision: 18, Cerca: 23,
			},
			ManifiestoSHA256: strings.Repeat("a", 64), ManifiestoBytes: 123, RevisionTrabajo: 9,
		})
	})
	client := newUnixHistoricalPreservationClient(t, handler)
	receipt, err := (&Adapter{client: client}).PreserveWithAuthority(context.Background(), authority, request)
	if ErrorCode(err) != CodePreserveResponseInsufficient || calls != 1 ||
		!reflect.DeepEqual(receipt, ports.AgentPreserveReceipt{}) || isDefinitelyNotApplied(err) {
		t.Fatalf("receipt=%+v calls=%d code=%q err=%v", receipt, calls, ErrorCode(err), err)
	}
}

func TestPreserveWithAuthorityRejectsCrossedSubjectAndDigestsBeforeUnixCall(t *testing.T) {
	authority, request := historicalPreservationFixture(t)
	client := &historicalPreservationCallProbe{launchClientStub: &launchClientStub{}}
	adapter := &Adapter{client: client}

	crossedSubject := authority
	crossedSubject.Subject.AgentRef = "agent:crossed"
	if _, err := adapter.PreserveWithAuthority(context.Background(), crossedSubject, request); ErrorCode(err) != CodeHistoricalSubjectMismatch {
		t.Fatalf("crossed subject err=%v", err)
	}
	invalidDigest := authority
	invalidDigest.Digests.KernelSHA256 = strings.Repeat("x", 64)
	if _, err := adapter.PreserveWithAuthority(context.Background(), invalidDigest, request); ErrorCode(err) != CodeHistoricalAuthorityInvalid {
		t.Fatalf("invalid digest err=%v", err)
	}
	if client.calls != 0 {
		t.Fatalf("physical calls=%d", client.calls)
	}
}

func TestGenericPreserveAndRecoveryRemainFailClosedWithoutHistoricalReceiptBinding(t *testing.T) {
	_, request := historicalPreservationFixture(t)
	adapter := &Adapter{}
	if receipt, err := adapter.Preserve(context.Background(), request); ErrorCode(err) != CodeHistoricalAuthorityInvalid || !reflect.DeepEqual(receipt, ports.AgentPreserveReceipt{}) {
		t.Fatalf("preserve receipt=%+v err=%v", receipt, err)
	}
	if receipt, err := adapter.ReconcilePreserve(context.Background(), request); ErrorCode(err) != CodePreserveRecoveryUnsupported || !reflect.DeepEqual(receipt, ports.AgentPreserveReceipt{}) {
		t.Fatalf("recover receipt=%+v err=%v", receipt, err)
	}
}

type historicalPreservationCallProbe struct {
	*launchClientStub
	calls int
}

func (probe *historicalPreservationCallProbe) Preservar(
	context.Context, string, string, microvm.SolicitudPreservacion,
) (microvm.RespuestaPreservacion, error) {
	probe.calls++
	return microvm.RespuestaPreservacion{}, nil
}

func historicalPreservationFixture(t *testing.T) (ports.AgentHistoricalRuntimeAuthority, ports.AgentPreserveRequest) {
	t.Helper()
	launch := validLaunchRequest(t)
	subject := ports.AgentEnvironmentLifecycleSubject{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash,
		ProviderRef: "provider:codex", ModelRef: "model:codex-microvm", AgentRef: "agent:codex-microvm",
		ExternalRef: "ejecucion:" + strings.Repeat("a", 64),
	}
	physical, _ := ports.NewAgentPhysicalToken(subject.ExternalRef)
	revision, _ := ports.NewAgentPhysicalRevision("17")
	fence, _ := ports.NewAgentPhysicalFence("23")
	digest := strings.Repeat("a", 64)
	authority := ports.AgentHistoricalRuntimeAuthority{
		Key:     ports.AgentHistoricalRuntimeAuthorityKey{ExecutionRef: subject.ExecutionRef, ActionFence: 23},
		Subject: subject,
		Digests: ports.AgentHistoricalRuntimeDigests{
			PlanSHA256: digest, GrantSHA256: digest, KernelSHA256: digest,
			InitramfsSHA256: digest, ProfileSHA256: digest,
		},
	}
	request := ports.AgentPreserveRequest{
		Subject: subject,
		ExpectedToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: physical, Revision: revision, Fence: fence, State: ports.AgentEnvironmentQuiesced,
		},
		IdempotencyKey: "preserve:b12:uds",
	}
	return authority, request
}

func newUnixHistoricalPreservationClient(t *testing.T, handler http.Handler) *microvm.Cliente {
	t.Helper()
	socket := t.TempDir() + "/agentmicrovm.sock"
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	client, err := microvm.Nuevo(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.LiberarConexiones)
	return client
}
