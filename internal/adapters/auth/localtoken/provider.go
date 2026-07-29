package localtoken

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"

	"orquesta/internal/identity"
)

var _ identity.IdentityProvider = (*Authenticator)(nil)

func (*Authenticator) AuthenticationMethod() identity.AuthenticationMethod {
	return identity.AuthenticationMethod(AuthenticationMethod)
}

// Authenticate compares detached ephemeral material and returns only the
// principal explicitly bound by composition.
func (authenticator *Authenticator) Authenticate(
	ctx context.Context,
	credential identity.Credential,
) (identity.Principal, error) {
	if ctx == nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid}
	}
	if err := ctx.Err(); err != nil {
		return identity.Principal{}, &Error{Code: CodeContextInvalid, Cause: err}
	}
	if authenticator == nil || len(authenticator.bindings) == 0 {
		return identity.Principal{}, &Error{Code: CodePrincipalInvalid}
	}
	var candidateDigest [sha256.Size]byte
	if err := credential.Use(func(material []byte) error {
		candidateDigest = sha256.Sum256(material)
		return nil
	}); err != nil {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed, Cause: err}
	}
	selected, matches := 0, 0
	for index, binding := range authenticator.bindings {
		if identity.ValidatePrincipal(binding.principal) != nil ||
			binding.principal.Method != AuthenticationMethod {
			return identity.Principal{}, &Error{Code: CodePrincipalInvalid}
		}
		equal := subtle.ConstantTimeCompare(binding.digest[:], candidateDigest[:])
		selected = subtle.ConstantTimeSelect(equal, index, selected)
		matches += equal
	}
	if matches != 1 {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
	}
	return authenticator.bindings[selected].principal, nil
}

// Combine builds one request authenticator without exposing or copying token
// material. Duplicate credentials, principal refs, or actor refs fail closed.
func Combine(providers ...*Authenticator) (*Authenticator, error) {
	total := 0
	for _, provider := range providers {
		if provider == nil || len(provider.bindings) == 0 {
			return nil, &Error{Code: CodePrincipalInvalid}
		}
		total += len(provider.bindings)
	}
	bindings := make([]credentialBinding, 0, total)
	for _, provider := range providers {
		for _, candidate := range provider.bindings {
			if identity.ValidatePrincipal(candidate.principal) != nil ||
				candidate.principal.Method != AuthenticationMethod {
				return nil, &Error{Code: CodePrincipalInvalid}
			}
			for _, existing := range bindings {
				if existing.principal.Ref == candidate.principal.Ref ||
					existing.principal.ActorRef == candidate.principal.ActorRef ||
					subtle.ConstantTimeCompare(existing.digest[:], candidate.digest[:]) == 1 {
					return nil, &Error{Code: CodePrincipalInvalid}
				}
			}
			bindings = append(bindings, candidate)
		}
	}
	return &Authenticator{bindings: bindings}, nil
}
