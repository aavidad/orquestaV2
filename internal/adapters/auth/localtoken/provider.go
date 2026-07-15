package localtoken

import (
	"context"

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
	if authenticator == nil || !authenticator.hasPrincipal ||
		identity.ValidatePrincipal(authenticator.principal) != nil ||
		authenticator.principal.Method != AuthenticationMethod {
		return identity.Principal{}, &Error{Code: CodePrincipalInvalid}
	}
	authenticated := false
	if err := credential.Use(func(material []byte) error {
		authenticated = authenticator.matches(string(material))
		return nil
	}); err != nil {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed, Cause: err}
	}
	if !authenticated {
		return identity.Principal{}, &Error{Code: CodeAuthenticationFailed}
	}
	return authenticator.principal, nil
}
