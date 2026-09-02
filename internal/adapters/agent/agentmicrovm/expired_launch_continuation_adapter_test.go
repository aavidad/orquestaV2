package agentmicrovm

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"reflect"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
)

type adapterExpiredContinuationClientStub struct {
	*launchClientStub
	prepared          microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1
	continued         microvm.RespuestaEjecucion
	prepareCalls      int
	continuationCalls int
}

func (client *adapterExpiredContinuationClientStub) PrepararContinuacionLanzamientoCaducado(
	_ context.Context,
	_ string,
	_ microvm.SolicitudLanzamiento,
) (microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1, error) {
	client.prepareCalls++
	return client.prepared, nil
}

func (client *adapterExpiredContinuationClientStub) ContinuarLanzamientoCaducado(
	_ context.Context,
	_ string,
	_ microvm.SolicitudContinuacionLanzamientoCaducadoV1,
) (microvm.RespuestaEjecucion, error) {
	client.continuationCalls++
	return client.continued, nil
}

func TestAdapterExpiredLaunchContinuationV41NeverFallsBackOrResigns(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	remote := validRemoteCapabilities()
	remote.Operaciones = append(
		remote.Operaciones,
		operationPrepareExpiredLaunchContinuation,
		operationContinueExpiredLaunchContinuation,
	)
	running, motor := true, "Running"
	response := validPhysicalResponse(t, request, descriptor)
	response.ProcesoVivo, response.EstadoMotor = &running, &motor
	base := &launchClientStub{capabilities: remote, response: response}
	client := &adapterExpiredContinuationClientStub{launchClientStub: base, continued: response}
	signer := validSigner()
	adapter, err := New(validAdapterConfig(client, signer, request, descriptor))
	if err != nil {
		t.Fatal(err)
	}

	initial, err := adapter.Launch(context.Background(), request)
	if err != nil || len(base.launchRequests) != 1 || len(signer.calls) != 1 {
		t.Fatalf("historical launch=%+v err=%v launches=%d signs=%d", initial, err, len(base.launchRequests), len(signer.calls))
	}
	original := cloneOriginalContinuationRequest(base.launchRequests[0])
	fixture := readExpiredLaunchContinuationFixture(t)
	manifest := fixture.Manifest
	manifest.RequestKeySHA256 = expiredContinuationRequestKeySHA256(request.IdempotencyKey)
	manifest.OriginalRequestSHA256 = expiredContinuationOriginalRequestSHA256(request.IdempotencyKey, original)
	manifest.ExecutionRef = response.Referencia
	manifest.Fence = request.EffectAuthority.ActionFence
	manifest.Generation = uint64(request.PlanGeneration)
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	client.prepared = microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1{
		Manifiesto: manifest, ManifiestoSHA256: manifestDigest,
	}
	subject := ExpiredLaunchContinuationSubjectV41{
		Binding: ExpiredLaunchContinuationCausalBindingV41{
			ReconciliationAuthorityRef: "agent-launch-reconciliation-authority:run7",
			ReconciliationAttemptRef:   "agent-launch-reconciliation-attempt:run7",
			ProjectRef:                 request.ProjectRef.String(), GoalRef: request.GoalRef.String(),
			WorkItemRef: request.WorkItemRef.String(), ExecutionRef: request.ExecutionRef.String(),
			ActionRef:       "action:launch:" + request.ExecutionRef.String(),
			EffectIntentRef: "effect-intent:run7", EffectIntentDigest: fixture.Manifest.SourceDigest,
			EffectAttemptRef: request.EffectAuthority.EffectAttemptRef,
			PlanGeneration:   uint64(request.PlanGeneration), WorkItemGeneration: 1,
			ActionFence: request.EffectAuthority.ActionFence,
		},
		IdempotencyKey: request.IdempotencyKey, Original: original,
		ProfileSHA256: bytesSHA256(original.Perfil), PlanSHA256: bytesSHA256(original.Plan),
		ConcessionSHA256: bytesSHA256(original.Concesion), ExpectedManifest: manifest,
	}
	prepared, err := adapter.expiredContinuation.Prepare(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	seed := sha256.Sum256([]byte("orquesta-expired-launch-continuation-adapter-key-v1"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	defer clear(privateKey)
	issued, err := adapter.expiredContinuation.Issue(prepared, ExpiredLaunchContinuationIssuanceV41{
		AuthorityRef: "continuation:run7-adapter", IssuedUnixMS: 1_788_000_000_000,
		ExpiresUnixMS: 1_788_000_300_000, KeyID: "continuation-run7-adapter",
		KeyEpoch: 1, TrustRevision: 1,
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	record, err := BuildExpiredAgentLaunchContinuationRecordV41(
		"expired-launch-continuation-subject:run7-adapter", issued,
		time.UnixMilli(1_788_000_000_000).UTC(), 1_788_000_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}

	removeOperation(&base.capabilities, operationContinueExpiredLaunchContinuation)
	if _, err := adapter.ContinueExpiredAgentLaunchV41(
		context.Background(), request, record,
	); ErrorCode(err) != CodeLaunchRejected || isTemporary(err) ||
		len(base.launchRequests) != 1 || len(base.reconcileRequests) != 0 ||
		client.continuationCalls != 0 || len(signer.calls) != 1 {
		t.Fatalf("missing capability err=%v launch=%d reconcile=%d continue=%d signs=%d",
			err, len(base.launchRequests), len(base.reconcileRequests),
			client.continuationCalls, len(signer.calls))
	}
	base.capabilities.Operaciones = append(
		base.capabilities.Operaciones, operationContinueExpiredLaunchContinuation,
	)
	continued, err := adapter.ContinueExpiredAgentLaunchV41(context.Background(), request, record)
	if err != nil || !reflect.DeepEqual(continued, initial) {
		t.Fatalf("continued=%+v initial=%+v err=%v", continued, initial, err)
	}
	if len(base.launchRequests) != 1 || len(base.reconcileRequests) != 0 ||
		client.prepareCalls != 1 || client.continuationCalls != 1 || len(signer.calls) != 1 {
		t.Fatalf("launch=%d reconcile=%d prepare=%d continue=%d signs=%d",
			len(base.launchRequests), len(base.reconcileRequests), client.prepareCalls,
			client.continuationCalls, len(signer.calls))
	}
}
