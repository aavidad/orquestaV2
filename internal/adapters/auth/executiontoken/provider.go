package executiontoken

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const tokenPrefix = "orqex1."

var errTokenInvalid = errors.New("executiontoken.token_invalid")
var _ identity.IdentityProvider = (*Broker)(nil)

func (*Broker) AuthenticationMethod() identity.AuthenticationMethod {
	return identity.AuthenticationMethod(AuthenticationMethod)
}

func (broker *Broker) Authenticate(ctx context.Context, credential identity.Credential) (identity.Principal, error) {
	if err := broker.validateContext(ctx); err != nil {
		return identity.Principal{}, err
	}
	if broker.authority == nil {
		return identity.Principal{}, executionError(CodeContextInvalid)
	}
	var executionRef goal.ExecutionRef
	var presented []byte
	err := credential.Use(func(material []byte) error {
		var parseErr error
		executionRef, presented, parseErr = parseToken(material)
		return parseErr
	})
	if err != nil {
		clear(presented)
		return identity.Principal{}, executionError(CodeAuthenticationFailed)
	}
	defer clear(presented)
	authority, err := broker.authority.ExecutionSessionAuthority(ctx, executionRef, AuthenticationMethod)
	expected, deriveErr := executionAuthority(authority.Request)
	if err != nil || deriveErr != nil || expected != authority ||
		identity.ValidatePrincipal(authority.ServicePrincipal) != nil ||
		authority.ServicePrincipal.Kind != identity.PrincipalKindService ||
		authority.ServicePrincipal.Method != AuthenticationMethod {
		return identity.Principal{}, executionError(CodeAuthenticationFailed)
	}
	matched := false
	_, err = broker.useSecret(ctx, expected, "authenticate", func(secret credentials.Secret) error {
		stored := secret.Bytes()
		defer clear(stored)
		matched = len(stored) == len(presented) &&
			subtle.ConstantTimeCompare(stored, presented) == 1
		return nil
	})
	if err != nil || !matched {
		return identity.Principal{}, executionError(CodeAuthenticationFailed)
	}
	return authority.ServicePrincipal, nil
}

func encodeToken(executionRef goal.ExecutionRef, secret []byte) []byte {
	token := make([]byte, 0, len(tokenPrefix)+base64.RawURLEncoding.EncodedLen(len(executionRef.String()))+
		1+base64.RawURLEncoding.EncodedLen(len(secret)))
	token = append(token, tokenPrefix...)
	token = base64.RawURLEncoding.AppendEncode(token, []byte(executionRef.String()))
	token = append(token, '.')
	return base64.RawURLEncoding.AppendEncode(token, secret)
}

func parseToken(token []byte) (goal.ExecutionRef, []byte, error) {
	if len(token) > 2048 || len(token) < len(tokenPrefix)+3 || string(token[:len(tokenPrefix)]) != tokenPrefix {
		return goal.ExecutionRef{}, nil, errTokenInvalid
	}
	body := token[len(tokenPrefix):]
	separator := len(body) - base64.RawURLEncoding.EncodedLen(secretBytes) - 1
	if separator <= 0 || body[separator] != '.' {
		return goal.ExecutionRef{}, nil, errTokenInvalid
	}
	executionBytes := make([]byte, base64.RawURLEncoding.DecodedLen(separator))
	executionLength, err := base64.RawURLEncoding.Decode(executionBytes, body[:separator])
	if err != nil {
		clear(executionBytes)
		return goal.ExecutionRef{}, nil, errTokenInvalid
	}
	defer clear(executionBytes)
	executionRef, err := goal.NewExecutionRef(string(executionBytes[:executionLength]))
	if err != nil {
		return goal.ExecutionRef{}, nil, errTokenInvalid
	}
	material := make([]byte, secretBytes)
	materialLength, err := base64.RawURLEncoding.Decode(material, body[separator+1:])
	if err != nil || materialLength != secretBytes {
		clear(material)
		return goal.ExecutionRef{}, nil, errTokenInvalid
	}
	return executionRef, material, nil
}
