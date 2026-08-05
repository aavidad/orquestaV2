package ports

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
)

const (
	microVMHostLaunchMaxRunRefBytes           = 512
	microVMHostLaunchMaxEffectAttemptRefBytes = 512
	microVMHostLaunchMaxServiceRefBytes       = 160
	microVMHostLaunchMaxPhysicalRefBytes      = 96
	microVMHostLaunchOneShotRequestDomainV1   = "orquesta.microvm-host-launch.one-shot-request.v1"
	microVMHostLaunchOneShotRequestPrefixV1   = "request:microvm-host-launch-one-shot:sha256:"
)

// MicroVMHostServiceRole identifies one signed host service without carrying
// its path, endpoint, process identity or credential material.
type MicroVMHostServiceRole string

const (
	MicroVMHostServiceControlBroker         MicroVMHostServiceRole = "control_broker"
	MicroVMHostServiceControlledEgressProxy MicroVMHostServiceRole = "controlled_egress_proxy"
)

// MicroVMHostLaunchAuthorityKey selects one historical physical launch
// authority. ActionFence is the fence persisted by its effect attempt, never a
// later recovery claim fence.
type MicroVMHostLaunchAuthorityKey struct {
	RunRef      goal.ExecutionRef
	ActionFence uint64
}

// MicroVMHostServiceAuthorityV1 is copied from one service in the exact signed
// launch plan. It contains identity metadata only.
type MicroVMHostServiceAuthorityV1 struct {
	Role           MicroVMHostServiceRole
	ServiceRef     string
	Port           uint32
	IdentityRef    string
	IdentitySHA256 string
}

// MicroVMHostLaunchAuthorityV1 binds an execution session to the exact signed
// physical launch. ExternalRef is empty while prepared and is set once by the
// durable registry when either the trusted host aperture or launch response
// first proves the sibling execution reference.
type MicroVMHostLaunchAuthorityV1 struct {
	Key              MicroVMHostLaunchAuthorityKey
	EffectAttemptRef string
	SessionRef       ExecutionSessionRef
	OneShotClaim     credentials.OneShotUseRequest
	PlanSHA256       string
	ConcessionSHA256 string
	Services         []MicroVMHostServiceAuthorityV1
	ExternalRef      string
}

// BuildMicroVMHostLaunchOneShotRequestRefV1 derives the exact causal request
// identity persisted before crossing client.Lanzar. The historical action
// fence is part of the identity; a later physical ExternalRef is deliberately
// absent because it is unknown at prepare time.
func BuildMicroVMHostLaunchOneShotRequestRefV1(
	key MicroVMHostLaunchAuthorityKey,
	effectAttemptRef string,
	sessionRef ExecutionSessionRef,
) (string, error) {
	if err := ValidateMicroVMHostLaunchAuthorityKey(key); err != nil {
		return "", err
	}
	if !validMicroVMHostLaunchOpaqueRef(effectAttemptRef, microVMHostLaunchMaxEffectAttemptRefBytes) {
		return "", microVMHostLaunchAuthorityError("effect_attempt_ref_invalid")
	}
	if len(sessionRef.String()) > microVMSessionMaxRefBytes {
		return "", microVMHostLaunchAuthorityError("session_ref_invalid")
	}
	if _, err := NewExecutionSessionRef(sessionRef.String()); err != nil {
		return "", microVMHostLaunchAuthorityError("session_ref_invalid")
	}

	digest := sha256.New()
	for _, field := range []string{
		microVMHostLaunchOneShotRequestDomainV1,
		key.RunRef.String(),
		strconv.FormatUint(key.ActionFence, 10),
		effectAttemptRef,
		sessionRef.String(),
	} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
	return microVMHostLaunchOneShotRequestPrefixV1 + hex.EncodeToString(digest.Sum(nil)), nil
}

// MicroVMHostLaunchAuthorityRegistry owns durable prepare-before-launch and
// set-once physical binding. Prepare must replay only an exact prepared
// authority. BindExternal must replay the same physical ref and reject a
// different ref. Returned values must not alias caller-owned Services slices.
type MicroVMHostLaunchAuthorityRegistry interface {
	Prepare(context.Context, MicroVMHostLaunchAuthorityV1) (MicroVMHostLaunchAuthorityV1, error)
	BindExternal(context.Context, MicroVMHostLaunchAuthorityKey, string) (MicroVMHostLaunchAuthorityV1, error)
	Resolve(context.Context, MicroVMHostLaunchAuthorityKey) (MicroVMHostLaunchAuthorityV1, error)
}

type MicroVMHostLaunchAuthorityContractError struct{ Code string }

func (err *MicroVMHostLaunchAuthorityContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func MicroVMHostLaunchAuthorityContractErrorCode(err error) string {
	var contractErr *MicroVMHostLaunchAuthorityContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

// ValidateMicroVMHostLaunchAuthorityKey validates the logical execution plus
// historical effect fence used for durable lookup.
func ValidateMicroVMHostLaunchAuthorityKey(key MicroVMHostLaunchAuthorityKey) error {
	runRef := key.RunRef.String()
	if !validMicroVMHostLaunchOpaqueRef(runRef, microVMHostLaunchMaxRunRefBytes) {
		return microVMHostLaunchAuthorityError("run_ref_invalid")
	}
	if _, err := goal.NewExecutionRef(runRef); err != nil {
		return microVMHostLaunchAuthorityError("run_ref_invalid")
	}
	if key.ActionFence == 0 {
		return microVMHostLaunchAuthorityError("action_fence_invalid")
	}
	return nil
}

// ValidateMicroVMHostLaunchAuthorityV1 accepts either a prepared authority or
// an authority carrying one valid set-once physical ExternalRef.
func ValidateMicroVMHostLaunchAuthorityV1(authority MicroVMHostLaunchAuthorityV1) error {
	expectedRequestRef, err := BuildMicroVMHostLaunchOneShotRequestRefV1(
		authority.Key,
		authority.EffectAttemptRef,
		authority.SessionRef,
	)
	if err != nil {
		return err
	}
	if err := credentials.ValidateOneShotUseRequest(authority.OneShotClaim); err != nil {
		return microVMHostLaunchAuthorityError("one_shot_claim_invalid")
	}
	if authority.OneShotClaim.RequestRef != expectedRequestRef {
		return microVMHostLaunchAuthorityError("one_shot_request_ref_mismatch")
	}
	if !validWorkspaceDigest(authority.PlanSHA256) {
		return microVMHostLaunchAuthorityError("plan_digest_invalid")
	}
	if !validWorkspaceDigest(authority.ConcessionSHA256) {
		return microVMHostLaunchAuthorityError("concession_digest_invalid")
	}
	if err := validateMicroVMHostServices(authority.Services); err != nil {
		return err
	}
	if authority.ExternalRef != "" && !validMicroVMHostPhysicalExecutionRef(authority.ExternalRef) {
		return microVMHostLaunchAuthorityError("external_ref_invalid")
	}
	return nil
}

// ValidateMicroVMHostLaunchAuthorityPreparedV1 requires the state persisted
// before crossing client.Lanzar.
func ValidateMicroVMHostLaunchAuthorityPreparedV1(authority MicroVMHostLaunchAuthorityV1) error {
	if err := ValidateMicroVMHostLaunchAuthorityV1(authority); err != nil {
		return err
	}
	if authority.ExternalRef != "" {
		return microVMHostLaunchAuthorityError("prepared_external_ref_present")
	}
	return nil
}

// ValidateMicroVMHostLaunchAuthorityBoundV1 requires a valid sibling physical
// execution reference after the set-once bind.
func ValidateMicroVMHostLaunchAuthorityBoundV1(authority MicroVMHostLaunchAuthorityV1) error {
	if err := ValidateMicroVMHostLaunchAuthorityV1(authority); err != nil {
		return err
	}
	if authority.ExternalRef == "" {
		return microVMHostLaunchAuthorityError("bound_external_ref_required")
	}
	return nil
}

// CloneMicroVMHostLaunchAuthorityV1 returns a detached material-free copy.
func CloneMicroVMHostLaunchAuthorityV1(authority MicroVMHostLaunchAuthorityV1) MicroVMHostLaunchAuthorityV1 {
	authority.Services = append([]MicroVMHostServiceAuthorityV1(nil), authority.Services...)
	return authority
}

func validateMicroVMHostServices(services []MicroVMHostServiceAuthorityV1) error {
	if len(services) < 1 || len(services) > 2 || services[0].Role != MicroVMHostServiceControlBroker ||
		(len(services) == 2 && services[1].Role != MicroVMHostServiceControlledEgressProxy) {
		return microVMHostLaunchAuthorityError("services_invalid")
	}
	seenPorts := make(map[uint32]struct{}, len(services))
	seenServiceRefs := make(map[string]struct{}, len(services))
	for _, service := range services {
		if service.Port == 0 ||
			!validMicroVMHostLaunchPrefixedRef(service.ServiceRef, "servicio:", microVMHostLaunchMaxServiceRefBytes) ||
			!validMicroVMHostLaunchPrefixedRef(service.IdentityRef, "identidad-servicio:", microVMHostLaunchMaxServiceRefBytes) ||
			!validWorkspaceDigest(service.IdentitySHA256) {
			return microVMHostLaunchAuthorityError("service_invalid")
		}
		if _, duplicate := seenPorts[service.Port]; duplicate {
			return microVMHostLaunchAuthorityError("service_duplicate")
		}
		if _, duplicate := seenServiceRefs[service.ServiceRef]; duplicate {
			return microVMHostLaunchAuthorityError("service_duplicate")
		}
		seenPorts[service.Port] = struct{}{}
		seenServiceRefs[service.ServiceRef] = struct{}{}
	}
	return nil
}

func validMicroVMHostLaunchOpaqueRef(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

func validMicroVMHostLaunchPrefixedRef(value, prefix string, maximum int) bool {
	if len(value) > maximum || !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return false
	}
	for _, character := range strings.TrimPrefix(value, prefix) {
		if !((character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') ||
			character == '-' || character == '_') {
			return false
		}
	}
	return true
}

func validMicroVMHostPhysicalExecutionRef(value string) bool {
	return validMicroVMHostLaunchPrefixedRef(value, "ejecucion:", microVMHostLaunchMaxPhysicalRefBytes)
}

func microVMHostLaunchAuthorityError(suffix string) error {
	return &MicroVMHostLaunchAuthorityContractError{Code: "microvm_host_launch_authority." + suffix}
}
