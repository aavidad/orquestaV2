package agentmicrovm

import (
	"context"
	"crypto/ed25519"
	"errors"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

// LaunchGrantSigningCredentialPurpose is the exact credential authority used
// both when provisioning a signing key and when signing one physical launch.
// Exporting the typed contract keeps composition from duplicating its value.
const LaunchGrantSigningCredentialPurpose = credentials.PurposeRef("orquesta.microvm-launch-grant.v1")

// CredentialSignerConfig binds one public key identity to credential-store
// authority. Private material is intentionally absent from this durable config.
type CredentialSignerConfig struct {
	Store         credentials.Store
	CredentialRef credentials.CredentialRef
	KeyID         string
}

// CredentialSigner obtains private material for one callback and never retains
// a key, a prepared grant or a launch request between calls.
type CredentialSigner struct {
	store         credentials.Store
	credentialRef credentials.CredentialRef
	keyID         string
}

func NewCredentialSigner(config CredentialSignerConfig) (*CredentialSigner, error) {
	if nilInterface(config.Store) {
		return nil, fail(CodeConfigurationInvalid, nil)
	}
	if err := credentials.ValidateCredentialRef(config.CredentialRef); err != nil {
		return nil, fail(CodeConfigurationInvalid, err)
	}
	if err := validateSigningKeyID(config.KeyID); err != nil {
		return nil, fail(CodeConfigurationInvalid, err)
	}
	return &CredentialSigner{
		store: config.Store, credentialRef: config.CredentialRef, keyID: config.KeyID,
	}, nil
}

func (signer *CredentialSigner) Preparar(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	authorizedContext microvm.ContextoAutorizado,
	plan microvm.PlanLanzamiento,
	issuedAt time.Time,
	validity time.Duration,
) (microvm.SolicitudLanzamiento, error) {
	var signed microvm.SolicitudLanzamiento
	err := signer.useSigningKey(ctx, request, func(grantSigner *microvm.FirmanteConcesiones) error {
		var prepareErr error
		signed, prepareErr = grantSigner.Preparar(authorizedContext, plan, issuedAt, validity)
		return prepareErr
	})
	if err != nil {
		return microvm.SolicitudLanzamiento{}, err
	}
	return cloneSignedRequestUnchecked(signed), nil
}

func (signer *CredentialSigner) PrepararContenedor(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	binding DockerPhysicalBinding,
	compiled DockerCompilation,
) (microvm.SolicitudLanzarORecuperarContenedorV1, error) {
	if !validDockerCompilation(request, binding, compiled) {
		return microvm.SolicitudLanzarORecuperarContenedorV1{}, fail(CodeSigningFailed, nil)
	}
	var signed microvm.SolicitudLanzarORecuperarContenedorV1
	err := signer.useSigningKey(ctx, request, func(grantSigner *microvm.FirmanteConcesiones) error {
		var prepareErr error
		signed, prepareErr = grantSigner.PrepararContenedor(compiled.Plan, compiled.IssuedAt, compiled.Validity)
		return prepareErr
	})
	if err != nil {
		return microvm.SolicitudLanzarORecuperarContenedorV1{}, err
	}
	signed.Plan.BultosRef = append([]string(nil), signed.Plan.BultosRef...)
	return signed, nil
}

func (signer *CredentialSigner) useSigningKey(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	prepare func(*microvm.FirmanteConcesiones) error,
) error {
	if signer == nil || nilInterface(signer.store) || nilInterface(ctx) || prepare == nil {
		return fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	useRequest := credentials.UseRequest{
		ActorRef: request.ActorRef.String(), RequestRef: "request:microvm-launch:" + request.ExecutionRef.String(),
		CredentialRef: signer.credentialRef, OwnerRef: credentials.OwnerRef(request.ActorRef.String()),
		ScopeRef: credentials.ScopeRef(request.ProjectRef.String()), PurposeRef: LaunchGrantSigningCredentialPurpose,
		Version: 0,
	}
	if err := credentials.ValidateUseRequest(useRequest); err != nil {
		return fail(CodeSigningFailed, err)
	}
	_, err := signer.store.Use(ctx, useRequest, func(secret credentials.Secret) error {
		defer secret.Destroy()
		material := secret.Bytes()
		defer clear(material)
		if len(material) != ed25519.PrivateKeySize {
			return fail(CodeSigningFailed, nil)
		}
		grantSigner, signerErr := microvm.NuevoFirmanteConcesiones(signer.keyID, ed25519.PrivateKey(material))
		if signerErr != nil {
			return fail(CodeSigningFailed, signerErr)
		}
		defer grantSigner.Destruir()
		if signerErr = prepare(grantSigner); signerErr != nil {
			return fail(CodeSigningFailed, signerErr)
		}
		return nil
	})
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ErrorCode(err) == CodeSigningFailed {
		return err
	}
	return fail(CodeSigningFailed, err)
}

func validateSigningKeyID(keyID string) error {
	privateKey := make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	defer clear(privateKey)
	signer, err := microvm.NuevoFirmanteConcesiones(keyID, privateKey)
	if err != nil {
		return err
	}
	signer.Destruir()
	return nil
}
