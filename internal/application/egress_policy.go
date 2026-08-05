package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"orquesta/internal/goal"
)

const (
	maxEgressPolicyRefBytes              = 512
	maxEgressPolicyCanonicalPayloadBytes = 64 << 10
)

// EgressPolicyRef is an opaque reference. Application never interprets it as
// a URL, host, port, provider, repository type or physical runtime detail.
type EgressPolicyRef string

func NewEgressPolicyRef(value string) (EgressPolicyRef, error) {
	if value == "" || len(value) > maxEgressPolicyRefBytes || strings.TrimSpace(value) != value ||
		strings.ContainsRune(value, '\x00') || !utf8.ValidString(value) {
		return "", errors.New("application.egress_policy_ref_invalid")
	}
	return EgressPolicyRef(value), nil
}

func (ref EgressPolicyRef) String() string { return string(ref) }

// EgressPolicyAuthority is the immutable, non-secret resolution of one
// requested policy. CanonicalPayload is opaque to application; only its exact
// bytes, bound and SHA-256 digest are authoritative here.
type EgressPolicyAuthority struct {
	PolicyRef        EgressPolicyRef
	PayloadSHA256    string
	CanonicalPayload string
}

// EgressPolicyResolver is the consumer-side port for a future policy catalog.
// PFC-05a deliberately defines no catalog or bootstrap wiring.
type EgressPolicyResolver interface {
	ResolveEgressPolicy(context.Context, EgressPolicyRef) (EgressPolicyAuthority, error)
}

func ValidateEgressPolicyAuthority(authority EgressPolicyAuthority) error {
	if authority == (EgressPolicyAuthority{}) {
		return nil
	}
	ref, err := NewEgressPolicyRef(authority.PolicyRef.String())
	if err != nil || ref != authority.PolicyRef || len(authority.CanonicalPayload) == 0 ||
		len(authority.CanonicalPayload) > maxEgressPolicyCanonicalPayloadBytes ||
		authority.PayloadSHA256 != egressPolicyPayloadSHA256(authority.CanonicalPayload) {
		return errors.New("application.egress_policy_authority_invalid")
	}
	return nil
}

func (orchestrator *Orchestrator) resolvePlanEgressPolicies(
	ctx context.Context,
	spec *PlanSpec,
) ([]EgressPolicyAuthority, error) {
	if spec == nil {
		return nil, nil
	}
	resolved := make([]EgressPolicyAuthority, len(spec.WorkItems))
	byRef := make(map[EgressPolicyRef]EgressPolicyAuthority)
	for index, item := range spec.WorkItems {
		if item.EgressPolicyRef == "" {
			continue
		}
		requested, err := NewEgressPolicyRef(item.EgressPolicyRef)
		if err != nil {
			return nil, err
		}
		if authority, found := byRef[requested]; found {
			resolved[index] = authority
			continue
		}
		if orchestrator.egressPolicies == nil {
			return nil, errors.New("application.egress_policy_resolver_required")
		}
		authority, err := orchestrator.egressPolicies.ResolveEgressPolicy(ctx, requested)
		if err != nil {
			return nil, errors.New("application.egress_policy_resolution_failed")
		}
		if authority.PolicyRef != requested || ValidateEgressPolicyAuthority(authority) != nil {
			return nil, errors.New("application.egress_policy_resolution_invalid")
		}
		byRef[requested] = authority
		resolved[index] = authority
	}
	return resolved, nil
}

func validateHistoricalEgressAuthorities(spec *PlanSpec, record GoalRecord) error {
	if !planDeclaresEgressPolicy(spec) {
		return nil
	}
	items := record.Goal.WorkItems()
	if len(items) < len(spec.WorkItems) {
		return &StateError{Code: StateConflict}
	}
	authorities := make(map[goal.WorkItemRef]WorkItemAuthority, len(record.WorkItemAuthorities))
	for _, authority := range record.WorkItemAuthorities {
		if _, found := record.Goal.WorkItem(authority.WorkItemRef); !found ||
			validateWorkItemAuthority(record.Goal, authority) != nil {
			return &StateError{Code: StateConflict}
		}
		if _, duplicate := authorities[authority.WorkItemRef]; duplicate {
			return &StateError{Code: StateConflict}
		}
		authorities[authority.WorkItemRef] = authority
	}
	for index, itemSpec := range spec.WorkItems {
		authority, found := authorities[items[index].Ref()]
		if !found {
			return &StateError{Code: StateConflict}
		}
		if itemSpec.EgressPolicyRef == "" {
			if authority.EgressPolicy != (EgressPolicyAuthority{}) {
				return &StateError{Code: StateConflict}
			}
			continue
		}
		requested, err := NewEgressPolicyRef(itemSpec.EgressPolicyRef)
		if err != nil || authority.EgressPolicy.PolicyRef != requested ||
			ValidateEgressPolicyAuthority(authority.EgressPolicy) != nil {
			return &StateError{Code: StateConflict, Cause: err}
		}
	}
	return nil
}

func egressPolicyPayloadSHA256(payload string) string {
	digest := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(digest[:])
}
