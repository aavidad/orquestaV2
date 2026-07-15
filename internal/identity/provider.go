package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"

	"orquesta/internal/goal"
)

// AuthenticationMethod is a stable adapter identifier. It is identity
// metadata, never a provider-specific authorization policy.
type AuthenticationMethod string

// IdentityProvider is the single authentication boundary. Credentials remain
// ephemeral; project scope and RBAC are deliberately absent from this port.
type IdentityProvider interface {
	AuthenticationMethod() AuthenticationMethod
	Authenticate(context.Context, Credential) (Principal, error)
}

// Credential wraps transient authentication material without exporting it or
// allowing accidental JSON persistence.
type Credential struct {
	material []byte
}

func NewCredential(material []byte) (Credential, error) {
	if len(material) == 0 {
		return Credential{}, errors.New("identity.credential_required")
	}
	return Credential{material: append([]byte(nil), material...)}, nil
}

// Use exposes a detached copy only for the duration of callback execution.
func (credential Credential) Use(callback func([]byte) error) error {
	if len(credential.material) == 0 {
		return errors.New("identity.credential_required")
	}
	if callback == nil {
		return errors.New("identity.credential_callback_required")
	}
	material := append([]byte(nil), credential.material...)
	defer clear(material)
	return callback(material)
}

func (Credential) String() string   { return "[REDACTED]" }
func (Credential) GoString() string { return "identity.Credential{[REDACTED]}" }

func (Credential) MarshalJSON() ([]byte, error) {
	return nil, errors.New("identity.credential_not_serializable")
}

var _ json.Marshaler = Credential{}

// CanonicalIssuer validates an exact OIDC issuer identifier. HTTPS is required
// outside loopback; query, fragment, user info and non-canonical host casing are
// rejected instead of silently changing the identity namespace.
func CanonicalIssuer(raw string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsRune(raw, '\x00') {
		return "", errors.New("identity.issuer_invalid")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Opaque != "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.String() != raw {
		return "", errors.New("identity.issuer_invalid")
	}
	hostname := parsed.Hostname()
	if hostname == "" || hostname != strings.ToLower(hostname) || parsed.Scheme != strings.ToLower(parsed.Scheme) {
		return "", errors.New("identity.issuer_not_canonical")
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		address := net.ParseIP(hostname)
		if hostname != "localhost" && (address == nil || !address.IsLoopback()) {
			return "", errors.New("identity.issuer_insecure")
		}
	default:
		return "", errors.New("identity.issuer_invalid")
	}
	return raw, nil
}

// StablePrincipal derives both opaque refs only from authentication method,
// canonical issuer and exact OIDC subject. Email, display name and groups can
// therefore never change identity or grant RBAC authority.
func StablePrincipal(
	method AuthenticationMethod,
	canonicalIssuer string,
	subject string,
	kind PrincipalKind,
) (Principal, error) {
	if !validCode(string(method)) {
		return Principal{}, errors.New("identity.invalid_authentication_method")
	}
	issuer, err := CanonicalIssuer(canonicalIssuer)
	if err != nil {
		return Principal{}, err
	}
	if subject == "" || len(subject) > 255 || !utf8.ValidString(subject) ||
		strings.ContainsRune(subject, '\x00') || strings.ContainsAny(subject, "\r\n") {
		return Principal{}, errors.New("identity.subject_invalid")
	}
	digest := sha256.Sum256([]byte(string(method) + "\x00" + issuer + "\x00" + subject))
	suffix := hex.EncodeToString(digest[:])
	principalRef, err := NewPrincipalRef("principal:sha256:" + suffix)
	if err != nil {
		return Principal{}, err
	}
	actorRef, err := goal.NewActorRef("actor:sha256:" + suffix)
	if err != nil {
		return Principal{}, err
	}
	return NewPrincipal(principalRef, actorRef, kind, string(method))
}
