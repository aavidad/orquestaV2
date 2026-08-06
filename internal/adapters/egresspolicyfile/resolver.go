// Package egresspolicyfile resolves one immutable egress policy from a
// composition-owned file. It is deliberately not a mutable policy catalog.
package egresspolicyfile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"reflect"
	"strings"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/application"
)

const (
	CodeOptionsInvalid      = "egress_policy_file.options_invalid"
	CodePlatformUnsupported = "egress_policy_file.platform_unsupported"
	CodeFileUnsafe          = "egress_policy_file.file_unsafe"
	CodeDigestMismatch      = "egress_policy_file.digest_mismatch"
	CodePayloadInvalid      = "egress_policy_file.payload_invalid"
	CodePolicyRefMismatch   = "egress_policy_file.policy_ref_mismatch"
	CodePolicyNotFound      = "egress_policy_file.policy_not_found"
	CodeResolverUnavailable = "egress_policy_file.resolver_unavailable"
)

const maximumCanonicalPayloadBytes = int64(64 << 10)

// Error exposes a stable machine code without including the local pathname or
// policy contents in its human representation.
type Error struct {
	Code string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

// IsError reports whether err was produced by this adapter with code.
func IsError(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target != nil && target.Code == code
}

// Resolver is a read-only, single-policy implementation of
// application.EgressPolicyResolver. Construction loads and seals the source;
// Resolve never reaches the filesystem again.
type Resolver struct {
	authority application.EgressPolicyAuthority
}

var _ application.EgressPolicyResolver = (*Resolver)(nil)

// New loads one exact policy from path. expectedSHA256 is the lowercase
// hexadecimal SHA-256 of the exact source bytes. maximumBytes is both the read
// bound and an explicit deployment limit; values above the application
// authority limit are rejected before opening the file.
func New(
	policyRef application.EgressPolicyRef,
	path string,
	expectedSHA256 string,
	ownerUID uint32,
	maximumBytes int64,
) (*Resolver, error) {
	return newWithHooks(policyRef, path, expectedSHA256, ownerUID, maximumBytes, loaderHooks{})
}

func newWithHooks(
	policyRef application.EgressPolicyRef,
	path string,
	expectedSHA256 string,
	ownerUID uint32,
	maximumBytes int64,
	hooks loaderHooks,
) (*Resolver, error) {
	canonicalRef, err := application.NewEgressPolicyRef(policyRef.String())
	if err != nil || canonicalRef != policyRef || !canonicalAbsolutePath(path) ||
		!validSHA256(expectedSHA256) || maximumBytes <= 0 ||
		maximumBytes > maximumCanonicalPayloadBytes {
		return nil, &Error{Code: CodeOptionsInvalid}
	}
	payload, err := loadPinnedPolicy(path, expectedSHA256, ownerUID, maximumBytes, hooks)
	if err != nil {
		return nil, err
	}
	grant, err := microvm.DecodificarConcesionEgresoV1(payload)
	if err != nil || grant.Esquema != microvm.EsquemaConcesionEgreso {
		clear(payload)
		return nil, &Error{Code: CodePayloadInvalid}
	}
	if grant.Referencia != policyRef.String() {
		clear(payload)
		return nil, &Error{Code: CodePolicyRefMismatch}
	}
	authority := application.EgressPolicyAuthority{
		PolicyRef: policyRef, PayloadSHA256: expectedSHA256,
		CanonicalPayload: string(payload),
	}
	clear(payload)
	if application.ValidateEgressPolicyAuthority(authority) != nil {
		return nil, &Error{Code: CodePayloadInvalid}
	}
	return &Resolver{authority: authority}, nil
}

// ResolveEgressPolicy returns the sealed bytes only for the exact reference
// supplied to New. A different or malformed reference never falls back to the
// sole configured policy.
func (resolver *Resolver) ResolveEgressPolicy(
	ctx context.Context,
	policyRef application.EgressPolicyRef,
) (application.EgressPolicyAuthority, error) {
	if resolver == nil || resolver.authority == (application.EgressPolicyAuthority{}) {
		return application.EgressPolicyAuthority{}, &Error{Code: CodeResolverUnavailable}
	}
	if nilContext(ctx) {
		return application.EgressPolicyAuthority{}, &Error{Code: CodeOptionsInvalid}
	}
	if err := ctx.Err(); err != nil {
		return application.EgressPolicyAuthority{}, err
	}
	if policyRef != resolver.authority.PolicyRef {
		return application.EgressPolicyAuthority{}, &Error{Code: CodePolicyNotFound}
	}
	return resolver.authority, nil
}

func nilContext(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	value := reflect.ValueOf(ctx)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func canonicalAbsolutePath(path string) bool {
	return path != "" && strings.IndexByte(path, 0) < 0 && filepath.IsAbs(path) &&
		filepath.Clean(path) == path && path != string(filepath.Separator)
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
