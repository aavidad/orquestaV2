package agentmicrovm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

func TestPreserveWithAuthorityUsesPublicUnixAPIAndReturnsExactHistoricalReceipt(t *testing.T) {
	authority, request := historicalPreservationFixture(t)
	content, artifacts := physicalPreservationManifestFixture(t, authority, request, false)
	digest := sha256.Sum256(content)
	manifestSHA := hex.EncodeToString(digest[:])
	mutations, reads := 0, 0
	handler := http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		response.Header().Set(microvm.CabeceraProtocolo, microvm.ProtocoloLocal)
		switch {
		case incoming.Method == http.MethodPost &&
			incoming.URL.Path == "/v1/ejecuciones/"+request.Subject.ExternalRef+"/preservacion":
			mutations++
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
				ManifiestoSHA256: manifestSHA, ManifiestoBytes: uint64(len(content)),
				Artefactos: artifacts, BytesUtiles: 7, RevisionTrabajo: 9,
			})
		case incoming.Method == http.MethodGet &&
			incoming.URL.Path == "/v1/ejecuciones/"+request.Subject.ExternalRef+"/preservacion/manifiesto":
			reads++
			query := incoming.URL.Query()
			if query.Get("cerca") != "23" || query.Get("revision_trabajo") != "9" ||
				query.Get("manifiesto_ref") != manifestSHA || query.Get("manifiesto_sha256") != manifestSHA ||
				query.Get("manifiesto_bytes") != strconv.Itoa(len(content)) {
				t.Errorf("manifest query=%v", query)
			}
			_ = json.NewEncoder(response).Encode(microvm.RespuestaManifiestoPreservacion{
				Referencia: request.Subject.ExternalRef, Cerca: 23, RevisionTrabajo: 9,
				ManifiestoRef: manifestSHA, ManifiestoSHA256: manifestSHA,
				ManifiestoBytes: uint64(len(content)), SelladaUnixMS: 42_000,
				ContenidoBase64: base64.StdEncoding.EncodeToString(content),
			})
		default:
			http.NotFound(response, incoming)
		}
	})
	client := newUnixHistoricalPreservationClient(t, handler)
	receipt, err := (&Adapter{client: client}).PreserveWithAuthority(context.Background(), authority, request)
	if err != nil || mutations != 1 || reads != 1 || ports.ValidateAgentPreserveReceipt(request, receipt) != nil {
		t.Fatalf("receipt=%+v mutations=%d reads=%d code=%q err=%v", receipt, mutations, reads, ErrorCode(err), err)
	}
	if receipt.ReceiptRef != manifestSHA || !reflect.DeepEqual(receipt.Manifest.Content, content) ||
		receipt.Manifest.Causality.PlanSHA256 != authority.Digests.PlanSHA256 ||
		receipt.Manifest.Causality.GrantSHA256 != authority.Digests.GrantSHA256 ||
		!receipt.ConfirmedAt.Equal(time.UnixMilli(42_000).UTC()) {
		t.Fatalf("receipt lost physical or historical binding: %+v", receipt)
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
	crossedFence := authority
	crossedFence.Key.ActionFence++
	if _, err := adapter.PreserveWithAuthority(context.Background(), crossedFence, request); ErrorCode(err) != CodeHistoricalAuthorityInvalid {
		t.Fatalf("crossed historical fence err=%v", err)
	}
	if client.calls != 0 {
		t.Fatalf("physical calls=%d", client.calls)
	}
}

func TestPhysicalPreservationManifestRejectsUnknownShapeAndCrossedHistoricalDigest(t *testing.T) {
	authority, request := historicalPreservationFixture(t)
	response := microvm.RespuestaPreservacion{
		Ejecucion: microvm.RespuestaEjecucion{
			Referencia: request.Subject.ExternalRef, Estado: "preservada", Revision: 18, Cerca: 23,
		},
		BytesUtiles: 7, RevisionTrabajo: 9,
	}
	unknown, artifacts := physicalPreservationManifestFixture(t, authority, request, true)
	response.Artefactos = artifacts
	if validPhysicalPreservationManifest(unknown, request, authority, response) {
		t.Fatal("manifest with unknown top-level field accepted")
	}
	content, _ := physicalPreservationManifestFixture(t, authority, request, false)
	crossed := authority
	crossed.Digests.PlanSHA256 = strings.Repeat("c", 64)
	if validPhysicalPreservationManifest(content, request, crossed, response) {
		t.Fatal("manifest bound to a different historical plan accepted")
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

func physicalPreservationManifestFixture(
	t *testing.T,
	authority ports.AgentHistoricalRuntimeAuthority,
	request ports.AgentPreserveRequest,
	unknown bool,
) ([]byte, []microvm.ArtefactoPreservado) {
	t.Helper()
	contentDigest := strings.Repeat("b", 64)
	artifacts := []microvm.ArtefactoPreservado{{
		Origen: "orden:principal", Clase: "orden", ContenidoSHA256: contentDigest,
		ContenidoBytes: 11, BytesUtiles: 7,
	}}
	manifest := map[string]any{
		"protocolo": physicalPreservationProtocolV1,
		"contexto": map[string]any{
			"referencia": request.Subject.ExternalRef, "revision_lifecycle": uint64(18),
			"cerca": uint64(23), "revision_trabajo": uint64(9),
			"plan_sha256": authority.Digests.PlanSHA256, "concesion_sha256": authority.Digests.GrantSHA256,
			"kernel_sha256":    authority.Digests.KernelSHA256,
			"initramfs_sha256": authority.Digests.InitramfsSHA256,
			"perfil_sha256":    authority.Digests.ProfileSHA256,
		},
		"artefactos": []any{map[string]any{
			"origen": "orden:principal", "clase": "orden", "contenido": "contenido:principal",
			"contenido_sha256": contentDigest, "contenido_bytes": uint64(11),
			"raiz_sha256": nil, "archivos": nil, "bytes_utiles": uint64(7),
			"detalle": map[string]any{
				"tipo": "orden", "codigo_salida": int32(0), "senal": nil,
				"agotada": false, "salida_truncada": false,
			},
		}},
	}
	if unknown {
		manifest["desconocido"] = true
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return content, artifacts
}
