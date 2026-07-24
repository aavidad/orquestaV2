package executiontoken

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

const (
	AuthenticationMethod = "execution_token"
	credentialPurpose    = credentials.PurposeRef("orquesta.execution-session.v1")
	revocationReason     = "execution_terminal"
	secretBytes          = 32
)

type Broker struct {
	store     credentials.Store
	authority application.ExecutionSessionAuthoritySource
	random    io.Reader
}

var _ ports.ExecutionSessionBroker = (*Broker)(nil)

func New(store credentials.Store, authority application.ExecutionSessionAuthoritySource) (*Broker, error) {
	if store == nil || authority == nil {
		return nil, executionError(CodeDependenciesRequired)
	}
	return &Broker{store: store, authority: authority, random: rand.Reader}, nil
}

func (broker *Broker) validateContext(ctx context.Context) error {
	if broker == nil || broker.store == nil || ctx == nil {
		return executionError(CodeContextInvalid)
	}
	if err := ctx.Err(); err != nil {
		return executionError(CodeContextInvalid, err)
	}
	return nil
}

func executionAuthority(request ports.ExecutionSessionEnsureRequest) (ports.ExecutionSessionAuthority, error) {
	authority, err := application.DeriveExecutionSessionAuthority(request, AuthenticationMethod)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, executionError(CodeRequestInvalid)
	}
	return authority, nil
}

func (broker *Broker) Ensure(ctx context.Context, request ports.ExecutionSessionEnsureRequest) (ports.ExecutionSessionReceipt, error) {
	if err := broker.validateContext(ctx); err != nil {
		return ports.ExecutionSessionReceipt{}, err
	}
	if broker.random == nil {
		return ports.ExecutionSessionReceipt{}, executionError(CodeContextInvalid)
	}
	authority, err := executionAuthority(request)
	if err != nil {
		return ports.ExecutionSessionReceipt{}, err
	}
	material := make([]byte, secretBytes)
	defer clear(material)
	if _, err := io.ReadFull(broker.random, material); err != nil {
		return ports.ExecutionSessionReceipt{}, executionError(CodeRandomFailed, err)
	}
	secret, err := credentials.NewSecret(material)
	if err != nil {
		return ports.ExecutionSessionReceipt{}, executionError(CodeRandomFailed)
	}
	defer secret.Destroy()
	credentialRef, err := credentialRef(authority.SessionRef)
	if err != nil {
		return ports.ExecutionSessionReceipt{}, executionError(CodeRequestInvalid)
	}
	result, createErr := broker.store.Create(ctx, credentials.CreateRequest{
		ActorRef:      authority.ServicePrincipal.ActorRef.String(),
		RequestRef:    requestRef("ensure", authority.SessionRef),
		CredentialRef: credentialRef,
		OwnerRef:      credentials.OwnerRef(authority.ServicePrincipal.Ref.String()),
		ScopeRefs:     []credentials.ScopeRef{credentials.ScopeRef(request.ProjectRef.String())},
		PurposeRef:    credentialPurpose,
		Material:      secret,
	})
	if createErr != nil && !credentials.HasErrorCode(createErr, credentials.ErrorAlreadyExists) &&
		!credentials.HasErrorCode(createErr, credentials.ErrorIdempotencyConflict) {
		return ports.ExecutionSessionReceipt{}, executionError(CodeCredentialUnavailable, createErr)
	}
	if createErr == nil {
		return ports.ExecutionSessionReceipt{Authority: authority, EnsuredAt: result.Metadata.CreatedAt, Replayed: result.Replayed}, nil
	}
	receipt, err := broker.useSecret(ctx, authority, "probe", func(credentials.Secret) error { return nil })
	if err != nil {
		return ports.ExecutionSessionReceipt{}, err
	}
	return ports.ExecutionSessionReceipt{Authority: authority, EnsuredAt: receipt.OccurredAt, Replayed: true}, nil
}

func (broker *Broker) UseToken(ctx context.Context, request ports.ExecutionSessionEnsureRequest, callback func([]byte) error) error {
	if callback == nil {
		return executionError(CodeRequestInvalid)
	}
	if err := broker.validateContext(ctx); err != nil {
		return err
	}
	authority, err := executionAuthority(request)
	if err != nil {
		return err
	}
	_, err = broker.useSecret(ctx, authority, "materialize", func(secret credentials.Secret) error {
		material := secret.Bytes()
		defer clear(material)
		token := encodeToken(request.ExecutionRef, material)
		defer clear(token)
		if err := callback(token); err != nil {
			return credentials.NewError(credentials.ErrorConsumerFailed, "token_consumer")
		}
		return nil
	})
	return err
}

func (broker *Broker) Revoke(ctx context.Context, request ports.ExecutionSessionEnsureRequest) error {
	if err := broker.validateContext(ctx); err != nil {
		return err
	}
	authority, err := executionAuthority(request)
	if err != nil {
		return err
	}
	ref, err := credentialRef(authority.SessionRef)
	if err != nil {
		return executionError(CodeRequestInvalid)
	}
	result, err := broker.store.Revoke(ctx, credentials.RevokeRequest{
		ActorRef: authority.ServicePrincipal.ActorRef.String(), RequestRef: requestRef("revoke", authority.SessionRef),
		CredentialRef: ref, OwnerRef: credentials.OwnerRef(authority.ServicePrincipal.Ref.String()),
		ExpectedVersion: 1, Reason: revocationReason,
	})
	if credentials.HasErrorCode(err, credentials.ErrorNotFound) {
		return nil
	}
	if err != nil || !result.Metadata.Revoked || result.Metadata.RevokedAt.IsZero() ||
		result.Metadata.CredentialRef != ref ||
		result.Metadata.OwnerRef != credentials.OwnerRef(authority.ServicePrincipal.Ref.String()) {
		return executionError(CodeCredentialUnavailable, err)
	}
	return nil
}

func (broker *Broker) useSecret(ctx context.Context, authority ports.ExecutionSessionAuthority, operation string, callback func(credentials.Secret) error) (credentials.Receipt, error) {
	ref, err := credentialRef(authority.SessionRef)
	if err != nil {
		return credentials.Receipt{}, executionError(CodeRequestInvalid)
	}
	receipt, err := broker.store.Use(ctx, credentials.UseRequest{
		ActorRef:      authority.ServicePrincipal.ActorRef.String(),
		RequestRef:    requestRef(operation, authority.SessionRef),
		CredentialRef: ref,
		OwnerRef:      credentials.OwnerRef(authority.ServicePrincipal.Ref.String()),
		ScopeRef:      credentials.ScopeRef(authority.Request.ProjectRef.String()),
		PurposeRef:    credentialPurpose,
		Version:       0,
	}, callback)
	if err != nil {
		return credentials.Receipt{}, executionError(CodeCredentialUnavailable, err)
	}
	return receipt, nil
}

func credentialRef(session ports.ExecutionSessionRef) (credentials.CredentialRef, error) {
	const prefix = "execution-session:sha256:"
	value := session.String()
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return "", errors.New("executiontoken.session_ref_invalid")
	}
	ref := credentials.CredentialRef("credential:execution_" + strings.TrimPrefix(value, prefix))
	if err := credentials.ValidateCredentialRef(ref); err != nil {
		return "", err
	}
	return ref, nil
}

func requestRef(operation string, ref ports.ExecutionSessionRef) string {
	return "request:execution-session-" + operation + ":" + strings.TrimPrefix(ref.String(), "execution-session:")
}
