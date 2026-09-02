package agentmicrovm

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
)

type expiredContinuationClientStub struct {
	capabilities                                 microvm.RespuestaCapacidades
	prepared                                     microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1
	continued                                    microvm.RespuestaEjecucion
	capabilityErr, prepareErr, continueErr       error
	capabilityCalls, prepareCalls, continueCalls int
	keys                                         []string
	originals                                    []microvm.SolicitudLanzamiento
	authorities                                  []microvm.ExpiredLaunchContinuationAuthorityV1
}

func (client *expiredContinuationClientStub) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	client.capabilityCalls++
	return client.capabilities, client.capabilityErr
}

func (client *expiredContinuationClientStub) PrepararContinuacionLanzamientoCaducado(
	_ context.Context,
	key string,
	original microvm.SolicitudLanzamiento,
) (microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1, error) {
	client.prepareCalls++
	client.keys = append(client.keys, key)
	client.originals = append(client.originals, cloneOriginalContinuationRequest(original))
	return client.prepared, client.prepareErr
}

func (client *expiredContinuationClientStub) ContinuarLanzamientoCaducado(
	_ context.Context,
	key string,
	request microvm.SolicitudContinuacionLanzamientoCaducadoV1,
) (microvm.RespuestaEjecucion, error) {
	client.continueCalls++
	client.keys = append(client.keys, key)
	client.originals = append(client.originals, cloneOriginalContinuationRequest(request.SolicitudOriginal))
	client.authorities = append(client.authorities, request.Autoridad)
	return client.continued, client.continueErr
}

func TestExpiredLaunchContinuationTransportKeepsRun7AttemptAndExactReplay(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	subject := expiredContinuationSubjectFromFixture(t, fixture)
	fixture.Manifest = subject.ExpectedManifest
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(fixture.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	running, motor := true, "Running"
	client := &expiredContinuationClientStub{
		capabilities: validExpiredContinuationCapabilities(),
		prepared: microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1{
			Manifiesto: fixture.Manifest, ManifiestoSHA256: manifestDigest,
		},
		continued: microvm.RespuestaEjecucion{
			Referencia: fixture.Manifest.ExecutionRef, Estado: "disponible", Revision: 2,
			Cerca: fixture.Manifest.Fence, VCPU: 2, MemoriaMiB: 2048,
			Identidad:   &microvm.IdentidadProceso{PID: 123, InicioTicks: 456},
			ProcesoVivo: &running, EstadoMotor: &motor,
		},
	}
	transport, err := NewExpiredLaunchContinuationTransportV41(client)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := transport.Prepare(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	reprepared, err := transport.Prepare(context.Background(), subject)
	if err != nil || !reflect.DeepEqual(reprepared, prepared) {
		t.Fatalf("replay tras Prepare no fue exacto: err=%v first=%+v replay=%+v", err, prepared, reprepared)
	}
	if prepared.Subject.Binding.EffectAttemptRef != "effect-attempt:run7" ||
		prepared.Subject.Binding.ReconciliationAttemptRef != "agent-launch-reconciliation-attempt:run7" ||
		!reflect.DeepEqual(prepared.Manifest, subject.ExpectedManifest) {
		t.Fatalf("preparación perdió causalidad: %+v", prepared)
	}
	seed := sha256.Sum256([]byte("orquesta-expired-launch-continuation-test-key-v1"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	defer clear(privateKey)
	issuance := ExpiredLaunchContinuationIssuanceV41{
		AuthorityRef: fixture.Content.AuthorityRef, IssuedUnixMS: fixture.Content.IssuedUnixMS,
		ExpiresUnixMS: fixture.Content.ExpiresUnixMS, KeyID: fixture.Content.KeyID,
		KeyEpoch: fixture.Content.KeyEpoch, TrustRevision: fixture.Content.TrustRevision,
	}
	issued, err := transport.Issue(prepared, issuance, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := transport.Issue(prepared, issuance, privateKey)
	if err != nil || !reflect.DeepEqual(issued, replayed) {
		t.Fatalf("emisión no fue replay exacto: err=%v\nfirst=%+v\nsecond=%+v", err, issued, replayed)
	}
	record, err := BuildExpiredAgentLaunchContinuationRecordV41(
		"expired-launch-continuation-subject:run7", issued,
		time.UnixMilli(int64(issuance.IssuedUnixMS)).UTC(), issuance.IssuedUnixMS,
	)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := issuedExpiredAgentLaunchContinuationFromRecordV41(record, subject.IdempotencyKey)
	if err != nil || !reflect.DeepEqual(rebuilt, issued) {
		t.Fatalf("registro no reconstruyó bytes exactos: err=%v\nwant=%+v\ngot=%+v", err, issued, rebuilt)
	}
	divergentRecord := record
	divergentRecord.KeyEpoch++
	if _, err := issuedExpiredAgentLaunchContinuationFromRecordV41(
		divergentRecord, subject.IdempotencyKey,
	); !errors.Is(err, ErrExpiredLaunchContinuationInvalid) {
		t.Fatalf("sombra divergente aceptada: %v", err)
	}
	response, err := transport.Continue(context.Background(), issued)
	if err != nil || response.Referencia != fixture.Manifest.ExecutionRef {
		t.Fatalf("continuación=%+v err=%v", response, err)
	}
	if client.capabilityCalls != 3 || client.prepareCalls != 2 || client.continueCalls != 1 ||
		!reflect.DeepEqual(client.keys, []string{"launch:run7", "launch:run7", "launch:run7"}) ||
		!reflect.DeepEqual(client.originals[0], client.originals[1]) ||
		!reflect.DeepEqual(client.originals[1], client.originals[2]) ||
		!reflect.DeepEqual(client.authorities[0], issued.Authority) {
		t.Fatalf("wire divergente: %+v", client)
	}
}

func TestExpiredLaunchContinuationTransportFailsClosedWithoutCapabilityOrOnDivergence(t *testing.T) {
	fixture := readExpiredLaunchContinuationFixture(t)
	subject := expiredContinuationSubjectFromFixture(t, fixture)
	fixture.Manifest = subject.ExpectedManifest
	client := &expiredContinuationClientStub{capabilities: validExpiredContinuationCapabilities()}
	client.capabilities.Operaciones = []string{operationPrepareExpiredLaunchContinuation}
	transport, err := NewExpiredLaunchContinuationTransportV41(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.Prepare(context.Background(), subject); !errors.Is(err, ErrExpiredLaunchContinuationUnsupported) ||
		client.prepareCalls != 0 || client.continueCalls != 0 {
		t.Fatalf("capacidad ausente cruzó mutación: err=%v client=%+v", err, client)
	}

	client.capabilities = validExpiredContinuationCapabilities()
	divergent := fixture.Manifest
	divergent.CID++
	digest, err := ExpiredLaunchContinuationManifestSHA256V1(divergent)
	if err != nil {
		t.Fatal(err)
	}
	client.prepared = microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1{
		Manifiesto: divergent, ManifiestoSHA256: digest,
	}
	if _, err := transport.Prepare(context.Background(), subject); !errors.Is(err, ErrExpiredLaunchContinuationDivergent) ||
		client.prepareCalls != 1 || client.continueCalls != 0 {
		t.Fatalf("manifiesto divergente aceptado: err=%v client=%+v", err, client)
	}
}

func expiredContinuationSubjectFromFixture(
	t *testing.T,
	fixture expiredLaunchContinuationFixture,
) ExpiredLaunchContinuationSubjectV41 {
	t.Helper()
	original := microvm.SolicitudLanzamiento{
		Perfil: []byte(`{"perfil":"run7"}`), Plan: []byte(`{"plan":"run7"}`),
		Concesion: []byte(`{"concesion":"run7"}`),
	}
	manifest := fixture.Manifest
	manifest.RequestKeySHA256 = expiredContinuationRequestKeySHA256("launch:run7")
	manifest.OriginalRequestSHA256 = expiredContinuationOriginalRequestSHA256("launch:run7", original)
	return ExpiredLaunchContinuationSubjectV41{
		Binding: ExpiredLaunchContinuationCausalBindingV41{
			ReconciliationAuthorityRef: "agent-launch-reconciliation-authority:run7",
			ReconciliationAttemptRef:   "agent-launch-reconciliation-attempt:run7",
			ProjectRef:                 "project:run7", GoalRef: "goal:run7", WorkItemRef: "work-item:run7",
			ExecutionRef: "execution:run7", ActionRef: "action:launch:run7",
			EffectIntentRef: "effect-intent:run7", EffectIntentDigest: strings.Repeat("e", 64),
			EffectAttemptRef: "effect-attempt:run7", PlanGeneration: fixture.Manifest.Generation,
			WorkItemGeneration: 1, ActionFence: fixture.Manifest.Fence,
		},
		IdempotencyKey: "launch:run7", Original: original,
		ProfileSHA256: bytesSHA256(original.Perfil), PlanSHA256: bytesSHA256(original.Plan),
		ConcessionSHA256: bytesSHA256(original.Concesion), ExpectedManifest: manifest,
	}
}

func validExpiredContinuationCapabilities() microvm.RespuestaCapacidades {
	return microvm.RespuestaCapacidades{
		Protocolo: microvm.ProtocoloLocal, Version: "0.1.0",
		Operaciones: []string{
			operationPrepareExpiredLaunchContinuation,
			operationContinueExpiredLaunchContinuation,
		},
		KVMDisponible: true, FirecrackerConfigurado: true, FirecrackerEjecutable: true,
		MaximoEjecuciones: 1,
	}
}
