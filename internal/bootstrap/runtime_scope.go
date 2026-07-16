package bootstrap

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
)

const runtimeScopeDomain = "orquesta.runtime-scope.local-state.v1"

type runtimeScopeBinder interface {
	BindRuntimeScope(string) error
}

type localStateIdentitySource interface {
	LocalStateIdentity() (string, bool, error)
}

func bindAgentRuntimeScope(agent AgentAdapter, state localStateIdentitySource) error {
	binder, ok := agent.(runtimeScopeBinder)
	if !ok {
		return nil
	}
	if state == nil {
		return errors.New("bootstrap.local_state_identity_unavailable")
	}
	identity, supported, err := state.LocalStateIdentity()
	if err != nil {
		return err
	}
	// A platform without a reliable local file identity keeps process controls
	// disabled. It must not receive a path-only scope that a backup could clone.
	if !supported {
		return nil
	}
	if identity == "" || strings.TrimSpace(identity) != identity || strings.ContainsRune(identity, '\x00') {
		return errors.New("bootstrap.local_state_identity_invalid")
	}
	return binder.BindRuntimeScope(runtimeScopeForLocalStateIdentity(identity))
}

func runtimeScopeForLocalStateIdentity(identity string) string {
	digest := sha256.New()
	writeRuntimeScopeField(digest, runtimeScopeDomain)
	writeRuntimeScopeField(digest, identity)
	return "runtime-scope:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func writeRuntimeScopeField(digest interface{ Write([]byte) (int, error) }, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}
