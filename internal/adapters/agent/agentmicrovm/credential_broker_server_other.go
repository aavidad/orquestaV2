//go:build !linux

package agentmicrovm

import "context"

// CredentialBrokerServer is unavailable outside Linux because SO_PEERCRED and
// the private pathname publication contract are part of its trust boundary.
type CredentialBrokerServer struct{}

func NewCredentialBrokerServer(
	CredentialBrokerServerConfig,
	*CredentialBroker,
) (*CredentialBrokerServer, error) {
	return nil, fail(CodeCredentialBrokerServerUnavailable, nil)
}

func (*CredentialBrokerServer) Start(context.Context) error {
	return fail(CodeCredentialBrokerServerUnavailable, nil)
}

func (*CredentialBrokerServer) Shutdown(context.Context) error {
	return fail(CodeCredentialBrokerServerUnavailable, nil)
}

func (*CredentialBrokerServer) Wait() error {
	return fail(CodeCredentialBrokerServerUnavailable, nil)
}
