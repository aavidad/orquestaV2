package executiontoken

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const tokenPrefix = "orqex1."

var _ identity.IdentityProvider = (*Broker)(nil)

func (*Broker) AuthenticationMethod() identity.AuthenticationMethod {
	return identity.AuthenticationMethod(AuthenticationMethod)
}

func (broker *Broker) Authenticate(
	ctx context.Context,
	credential identity.Credential,
) (identity.Principal, error) {
	if broker == nil || broker.store == nil || broker.authority == nil || ctx == nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid}
	}
	if err := ctx.Err(); err != nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid, Cause: err}
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
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
	}
	defer clear(presented)
	authority, err := broker.authority.ExecutionSessionAuthority(ctx, executionRef, AuthenticationMethod)
	if err != nil || identity.ValidatePrincipal(authority.ServicePrincipal) != nil ||
		authority.ServicePrincipal.Kind != identity.PrincipalKindService ||
		authority.ServicePrincipal.Method != AuthenticationMethod {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
	}
	expected, err := applicationAuthority(authority)
	if err != nil {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
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
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
	}
	return authority.ServicePrincipal, nil
}

func applicationAuthority(authority ports.ExecutionSessionAuthority) (ports.ExecutionSessionAuthority, error) {
	expected, err := application.DeriveExecutionSessionAuthority(authority.Request, AuthenticationMethod)
	if err != nil || !application.SameExecutionSessionAuthority(expected, authority) {
		return ports.ExecutionSessionAuthority{}, errors.New("executiontoken.authority_invalid")
	}
	return expected, nil
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
	if len(token) < len(tokenPrefix)+3 || len(token) > 2048 ||
		!bytes.HasPrefix(token, []byte(tokenPrefix)) {
		return goal.ExecutionRef{}, nil, errors.New("executiontoken.token_invalid")
	}
	body := token[len(tokenPrefix):]
	separator := bytes.IndexByte(body, '.')
	if separator <= 0 || separator == len(body)-1 || bytes.IndexByte(body[separator+1:], '.') >= 0 {
		return goal.ExecutionRef{}, nil, errors.New("executiontoken.token_invalid")
	}
	executionBytes := make([]byte, base64.RawURLEncoding.DecodedLen(separator))
	executionLength, err := base64.RawURLEncoding.Decode(executionBytes, body[:separator])
	if err != nil {
		clear(executionBytes)
		return goal.ExecutionRef{}, nil, errors.New("executiontoken.token_invalid")
	}
	defer clear(executionBytes)
	executionRef, err := goal.NewExecutionRef(string(executionBytes[:executionLength]))
	if err != nil {
		return goal.ExecutionRef{}, nil, errors.New("executiontoken.token_invalid")
	}
	encodedMaterial := body[separator+1:]
	material := make([]byte, base64.RawURLEncoding.DecodedLen(len(encodedMaterial)))
	materialLength, err := base64.RawURLEncoding.Decode(material, encodedMaterial)
	if err != nil || materialLength != secretBytes {
		clear(material)
		return goal.ExecutionRef{}, nil, errors.New("executiontoken.token_invalid")
	}
	return executionRef, material[:materialLength], nil
}
