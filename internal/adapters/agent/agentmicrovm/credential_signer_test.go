package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
)

const testCredentialRef = credentials.CredentialRef("credential:microvm-launch-signing")

type credentialSignerStoreStub struct {
	material []byte
	err      error
	calls    []credentials.UseRequest
	contexts []context.Context
	retained []credentials.Secret
}

func (*credentialSignerStoreStub) Create(context.Context, credentials.CreateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("unused")
}

func (store *credentialSignerStoreStub) Use(
	ctx context.Context,
	request credentials.UseRequest,
	callback func(credentials.Secret) error,
) (credentials.Receipt, error) {
	store.calls = append(store.calls, request)
	store.contexts = append(store.contexts, ctx)
	if store.err != nil {
		return credentials.Receipt{}, store.err
	}
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.Receipt{}, err
	}
	store.retained = append(store.retained, secret)
	if err := callback(secret); err != nil {
		return credentials.Receipt{}, err
	}
	return credentials.Receipt{
		CredentialRef: request.CredentialRef,
		OwnerRef:      request.OwnerRef,
		ScopeRef:      request.ScopeRef,
		PurposeRef:    request.PurposeRef,
		Version:       1,
		RequestRef:    request.RequestRef,
		ActorRef:      request.ActorRef,
		Operation:     "use",
	}, nil
}

func (*credentialSignerStoreStub) Rotate(context.Context, credentials.RotateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("unused")
}

func (*credentialSignerStoreStub) Revoke(context.Context, credentials.RevokeRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, errors.New("unused")
}

func TestCredentialSignerUsesExactAuthorityForEveryReplayAndProducesVerifiableSignature(t *testing.T) {
	request := validLaunchRequest(t)
	compiled := mustCompile(t, request, validDescriptor(t, false))
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	store := &credentialSignerStoreStub{material: append([]byte(nil), privateKey...)}
	signer := mustCredentialSigner(t, store)
	ctx := context.Background()

	first, err := signer.Preparar(ctx, request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity)
	if err != nil {
		t.Fatalf("Preparar() first error = %v", err)
	}
	second, err := signer.Preparar(ctx, request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity)
	if err != nil {
		t.Fatalf("Preparar() replay error = %v", err)
	}
	if !bytes.Equal(first.Plan, second.Plan) || !bytes.Equal(first.Concesion, second.Concesion) {
		t.Fatal("credential replay produced different signed bytes")
	}

	wantUse := credentials.UseRequest{
		ActorRef:      request.ActorRef.String(),
		RequestRef:    "request:microvm-launch:" + request.ExecutionRef.String(),
		CredentialRef: testCredentialRef,
		OwnerRef:      credentials.OwnerRef(request.ActorRef.String()),
		ScopeRef:      credentials.ScopeRef(request.ProjectRef.String()),
		PurposeRef:    credentialSigningPurpose,
		Version:       0,
	}
	if len(store.calls) != 2 || store.calls[0] != wantUse || store.calls[1] != wantUse ||
		len(store.contexts) != 2 || store.contexts[0] != ctx || store.contexts[1] != ctx {
		t.Fatalf("Use calls = %+v contexts=%d, want two exact calls", store.calls, len(store.contexts))
	}
	for index, retained := range store.retained {
		if material := retained.Bytes(); len(material) != ed25519.PrivateKeySize ||
			!bytes.Equal(material, make([]byte, ed25519.PrivateKeySize)) {
			t.Fatalf("callback secret %d was not destroyed", index)
		}
	}
	verifyCredentialGrant(t, first, privateKey.Public().(ed25519.PublicKey))
}

func TestCredentialSignerAcquiresCredentialAgainForAdapterReplay(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	store := &credentialSignerStoreStub{material: ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))}
	signer := mustCredentialSigner(t, store)
	client := &launchClientStub{
		capabilities: validRemoteCapabilities(),
		response:     validPhysicalResponse(t, request, descriptor),
	}
	adapter := mustNewAdapter(t, client, signer, request, descriptor)

	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() first = %v", err)
	}
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() replay = %v", err)
	}
	if len(store.calls) != 2 || len(client.launchRequests) != 2 {
		t.Fatalf("calls: credential=%d launch=%d", len(store.calls), len(client.launchRequests))
	}
}

func TestCredentialSignerRejectsBadMaterialAndStoreFailureBeforeSocket(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	secretText := "private-material-must-not-leak"
	tests := []struct {
		name     string
		store    *credentialSignerStoreStub
		wantIs   error
		wantCode string
		wantCall int
	}{
		{
			name:     "wrong private key length",
			store:    &credentialSignerStoreStub{material: []byte("short")},
			wantCode: CodeSigningFailed,
			wantCall: 1,
		},
		{
			name:     "store denied",
			store:    &credentialSignerStoreStub{err: errors.New("denied: " + secretText)},
			wantCode: CodeSigningFailed,
			wantCall: 1,
		},
		{
			name:     "store canceled",
			store:    &credentialSignerStoreStub{err: context.Canceled},
			wantIs:   context.Canceled,
			wantCall: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signer := mustCredentialSigner(t, test.store)
			client := &launchClientStub{
				capabilities: validRemoteCapabilities(),
				response:     validPhysicalResponse(t, request, descriptor),
			}
			adapter := mustNewAdapter(t, client, signer, request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.wantCode || len(client.launchRequests) != 0 ||
				len(test.store.calls) != test.wantCall || strings.Contains(err.Error(), secretText) {
				t.Fatalf("Launch() error=%v code=%q uses=%d launches=%d", err, ErrorCode(err), len(test.store.calls), len(client.launchRequests))
			}
			if test.wantIs != nil && !errors.Is(err, test.wantIs) {
				t.Fatalf("Launch() error=%v, want errors.Is(%v)", err, test.wantIs)
			}
		})
	}
}

func TestCredentialSignerPreservesPreCanceledContextWithoutCredentialUse(t *testing.T) {
	request := validLaunchRequest(t)
	compiled := mustCompile(t, request, validDescriptor(t, false))
	store := &credentialSignerStoreStub{material: ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))}
	signer := mustCredentialSigner(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := signer.Preparar(ctx, request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity)
	if !errors.Is(err, context.Canceled) || len(store.calls) != 0 {
		t.Fatalf("Preparar() error=%v uses=%d", err, len(store.calls))
	}
}

func TestCredentialSignerPreservesStoreCancellation(t *testing.T) {
	request := validLaunchRequest(t)
	compiled := mustCompile(t, request, validDescriptor(t, false))
	store := &credentialSignerStoreStub{err: context.DeadlineExceeded}
	signer := mustCredentialSigner(t, store)

	_, err := signer.Preparar(
		context.Background(), request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity,
	)
	if !errors.Is(err, context.DeadlineExceeded) || ErrorCode(err) != "" || len(store.calls) != 1 {
		t.Fatalf("Preparar() error=%v code=%q uses=%d", err, ErrorCode(err), len(store.calls))
	}
}

func TestNewCredentialSignerRejectsInvalidConfiguration(t *testing.T) {
	validStore := &credentialSignerStoreStub{}
	var typedNil *credentialSignerStoreStub
	tests := []CredentialSignerConfig{
		{},
		{Store: typedNil, CredentialRef: testCredentialRef, KeyID: "clave-publica:test"},
		{Store: validStore, CredentialRef: "credential:Bad", KeyID: "clave-publica:test"},
		{Store: validStore, CredentialRef: testCredentialRef, KeyID: "public-key:test"},
	}
	for index, config := range tests {
		if signer, err := NewCredentialSigner(config); signer != nil || ErrorCode(err) != CodeConfigurationInvalid {
			t.Fatalf("case %d: signer=%v error=%v", index, signer, err)
		}
	}
}

func mustCredentialSigner(t *testing.T, store credentials.Store) *CredentialSigner {
	t.Helper()
	signer, err := NewCredentialSigner(CredentialSignerConfig{
		Store: store, CredentialRef: testCredentialRef, KeyID: "clave-publica:test",
	})
	if err != nil {
		t.Fatalf("NewCredentialSigner() = %v", err)
	}
	return signer
}

type credentialGrantContent struct {
	Esquema       string `json:"esquema"`
	Audiencia     string `json:"audiencia"`
	ConcesionRef  string `json:"concesion_ref"`
	RunRef        string `json:"run_ref"`
	Cerca         uint64 `json:"cerca"`
	PlanSHA256    string `json:"plan_sha256"`
	EmitidaUnixMS int64  `json:"emitida_unix_ms"`
	NoAntesUnixMS int64  `json:"no_antes_unix_ms"`
	ExpiraUnixMS  int64  `json:"expira_unix_ms"`
	ClaveID       string `json:"clave_id"`
	Algoritmo     string `json:"algoritmo"`
}

func verifyCredentialGrant(t *testing.T, request microvm.SolicitudLanzamiento, publicKey ed25519.PublicKey) {
	t.Helper()
	var grant struct {
		Content   credentialGrantContent `json:"contenido"`
		Signature string                 `json:"firma_base64"`
	}
	if err := json.Unmarshal(request.Concesion, &grant); err != nil {
		t.Fatalf("decode grant = %v", err)
	}
	signature, err := base64.StdEncoding.DecodeString(grant.Signature)
	if err != nil || !ed25519.Verify(publicKey, credentialGrantMessage(grant.Content), signature) {
		t.Fatal("credential-backed signature is not verifiable")
	}
}

func credentialGrantMessage(content credentialGrantContent) []byte {
	message := append([]byte("agentmicrovm.concesion-lanzamiento.v1\x00"), credentialGrantField(content.Esquema)...)
	message = append(message, credentialGrantField(content.Audiencia)...)
	message = append(message, credentialGrantField(content.ConcesionRef)...)
	message = append(message, credentialGrantField(content.RunRef)...)
	message = binary.BigEndian.AppendUint64(message, content.Cerca)
	message = append(message, credentialGrantField(content.PlanSHA256)...)
	message = binary.BigEndian.AppendUint64(message, uint64(content.EmitidaUnixMS))
	message = binary.BigEndian.AppendUint64(message, uint64(content.NoAntesUnixMS))
	message = binary.BigEndian.AppendUint64(message, uint64(content.ExpiraUnixMS))
	message = append(message, credentialGrantField(content.ClaveID)...)
	return append(message, credentialGrantField(content.Algoritmo)...)
}

func credentialGrantField(value string) []byte {
	field := binary.BigEndian.AppendUint32(nil, uint32(len(value)))
	return append(field, value...)
}
