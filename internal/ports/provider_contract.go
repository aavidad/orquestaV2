package ports

import (
	"errors"
	"strings"
	"time"

	"orquesta/internal/governance"
)

// ProviderAvailabilityStatus is an observed fact. Unknown never means
// available and must therefore fail closed during routing.
type ProviderAvailabilityStatus string

const (
	ProviderAvailabilityAvailable   ProviderAvailabilityStatus = "available"
	ProviderAvailabilityUnavailable ProviderAvailabilityStatus = "unavailable"
	ProviderAvailabilityUnknown     ProviderAvailabilityStatus = "unknown"
)

// ProviderQuotaStatus deliberately exposes no inferred balance. Physical
// pools, licences and reservations belong to the capacity contracts.
type ProviderQuotaStatus string

const (
	ProviderQuotaAvailable ProviderQuotaStatus = "available"
	ProviderQuotaExhausted ProviderQuotaStatus = "exhausted"
	ProviderQuotaUnknown   ProviderQuotaStatus = "unknown"
)

// ProviderModel describes only capabilities explicitly advertised by a
// provider. Similar names never imply capability or quality parity.
type ProviderModel struct {
	ProviderRef      string
	ModelRef         string
	CapabilityRefs   []string
	ReasoningEfforts []governance.ReasoningEffort
}

// ProviderCatalogObservation is a read-only, expiring provider snapshot.
// Usage is telemetry, not a budget settlement or lifecycle mutation.
type ProviderCatalogObservation struct {
	ProviderRef  string
	Models       []ProviderModel
	Availability ProviderAvailabilityStatus
	Quota        ProviderQuotaStatus
	Usage        governance.ResourceUsage
	ObservedAt   time.Time
	ExpiresAt    time.Time
}

type ProviderContractError struct {
	Code string
}

func (err *ProviderContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ProviderContractErrorCode(err error) string {
	var contractErr *ProviderContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateProviderModel(model ProviderModel) error {
	switch {
	case !validProviderContractRef(model.ProviderRef):
		return &ProviderContractError{Code: "provider.provider_ref_invalid"}
	case !validProviderContractRef(model.ModelRef):
		return &ProviderContractError{Code: "provider.model_ref_invalid"}
	case !validCapabilityRefs(model.CapabilityRefs):
		return &ProviderContractError{Code: "provider.capability_refs_invalid"}
	case !validProviderReasoningEfforts(model.ReasoningEfforts):
		return &ProviderContractError{Code: "provider.reasoning_efforts_invalid"}
	default:
		return nil
	}
}

func ValidateProviderCatalogObservation(observation ProviderCatalogObservation) error {
	if !validProviderContractRef(observation.ProviderRef) {
		return &ProviderContractError{Code: "provider.provider_ref_invalid"}
	}
	switch observation.Availability {
	case ProviderAvailabilityAvailable, ProviderAvailabilityUnavailable, ProviderAvailabilityUnknown:
	default:
		return &ProviderContractError{Code: "provider.availability_invalid"}
	}
	switch observation.Quota {
	case ProviderQuotaAvailable, ProviderQuotaExhausted, ProviderQuotaUnknown:
	default:
		return &ProviderContractError{Code: "provider.quota_invalid"}
	}
	if observation.ObservedAt.IsZero() || !observation.ExpiresAt.After(observation.ObservedAt) {
		return &ProviderContractError{Code: "provider.observation_window_invalid"}
	}
	if err := governance.ValidateResourceUsage(observation.Usage); err != nil {
		return &ProviderContractError{Code: "provider.usage_invalid"}
	}
	seenModels := make(map[string]struct{}, len(observation.Models))
	for _, model := range observation.Models {
		if err := ValidateProviderModel(model); err != nil {
			return err
		}
		if model.ProviderRef != observation.ProviderRef {
			return &ProviderContractError{Code: "provider.model_provider_mismatch"}
		}
		if _, duplicate := seenModels[model.ModelRef]; duplicate {
			return &ProviderContractError{Code: "provider.model_ref_duplicate"}
		}
		seenModels[model.ModelRef] = struct{}{}
	}
	return nil
}

func ProviderModelSupports(
	model ProviderModel,
	capabilityRefs []string,
	effort governance.ReasoningEffort,
) bool {
	if ValidateProviderModel(model) != nil || !validCapabilityRefs(capabilityRefs) ||
		governance.ValidateReasoningEffort(effort) != nil {
		return false
	}
	for _, required := range capabilityRefs {
		if !containsAgentRef(model.CapabilityRefs, required) {
			return false
		}
	}
	for _, supported := range model.ReasoningEfforts {
		if supported == effort {
			return true
		}
	}
	return false
}

func validProviderContractRef(value string) bool {
	return validAgentIdentityRef(value) && !strings.ContainsRune(value, '\x00')
}

func validProviderReasoningEfforts(values []governance.ReasoningEffort) bool {
	seen := make(map[governance.ReasoningEffort]struct{}, len(values))
	for _, value := range values {
		if governance.ValidateReasoningEffort(value) != nil {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}
