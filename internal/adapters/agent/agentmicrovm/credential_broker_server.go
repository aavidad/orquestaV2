package agentmicrovm

import "time"

const (
	CodeCredentialBrokerServerInvalid     = "agentmicrovm.credential_broker_server_invalid"
	CodeCredentialBrokerSocketUnsafe      = "agentmicrovm.credential_broker_socket_unsafe"
	CodeCredentialBrokerPeerDenied        = "agentmicrovm.credential_broker_peer_denied"
	CodeCredentialBrokerServerUnavailable = "agentmicrovm.credential_broker_server_unavailable"
	CodeCredentialBrokerCleanupFailed     = "agentmicrovm.credential_broker_cleanup_failed"

	maximumCredentialBrokerHandlers = uint32(256)
)

// CredentialBrokerServerConfig binds the resident broker to one private UDS
// and one physical agente_microvm UID. Every operational limit is explicit;
// this security boundary has no permissive fallback.
type CredentialBrokerServerConfig struct {
	SocketPath        string
	OwnerUID          uint32
	PeerUID           uint32
	ConnectionTimeout time.Duration
	MaxConcurrent     uint32
}
