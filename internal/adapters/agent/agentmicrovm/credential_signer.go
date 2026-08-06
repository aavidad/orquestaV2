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
	if signer == nil || nilInterface(signer.store) {
		return microvm.SolicitudLanzamiento{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return microvm.SolicitudLanzamiento{}, err
	}

	useRequest := credentials.UseRequest{
		ActorRef:      request.ActorRef.String(),
		RequestRef:    "request:microvm-launch:" + request.ExecutionRef.String(),
		CredentialRef: signer.credentialRef,
		OwnerRef:      credentials.OwnerRef(request.ActorRef.String()),
		ScopeRef:      credentials.ScopeRef(request.ProjectRef.String()),
		PurposeRef:    LaunchGrantSigningCredentialPurpose,
		Version:       0,
	}
	if err := credentials.ValidateUseRequest(useRequest); err != nil {
		return microvm.SolicitudLanzamiento{}, fail(CodeSigningFailed, err)
	}

	var signed microvm.SolicitudLanzamiento
	var signingErr error
	_, useErr := signer.store.Use(ctx, useRequest, func(secret credentials.Secret) error {
		defer secret.Destroy()
		material := secret.Bytes()
		defer clear(material)
		if len(material) != ed25519.PrivateKeySize {
			signingErr = fail(CodeSigningFailed, nil)
			return signingErr
		}

		grantSigner, err := microvm.NuevoFirmanteConcesiones(
			signer.keyID,
			ed25519.PrivateKey(material),
		)
		if err != nil {
			signingErr = fail(CodeSigningFailed, err)
			return signingErr
		}
		defer grantSigner.Destruir()
		signed, err = grantSigner.Preparar(authorizedContext, plan, issuedAt, validity)
		if err != nil {
			signingErr = fail(CodeSigningFailed, err)
			return signingErr
		}
		return nil
	})
	if signingErr != nil {
		return microvm.SolicitudLanzamiento{}, signingErr
	}
	if useErr != nil {
		if errors.Is(useErr, context.Canceled) || errors.Is(useErr, context.DeadlineExceeded) {
			return microvm.SolicitudLanzamiento{}, useErr
		}
		return microvm.SolicitudLanzamiento{}, fail(CodeSigningFailed, useErr)
	}
	return cloneSignedRequestUnchecked(signed), nil
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
