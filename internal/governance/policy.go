package governance

// SecurityCriticality is declared risk metadata. It is never inferred from
// provider names, prompts, paths, or keyword rails.
type SecurityCriticality string

const (
	SecurityCriticalityNormal    SecurityCriticality = "normal"
	SecurityCriticalitySensitive SecurityCriticality = "sensitive"
	SecurityCriticalityCritical  SecurityCriticality = "critical"
)

// ReasoningEffort describes requested model effort, independently of risk.
type ReasoningEffort string

const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortXHigh  ReasoningEffort = "xhigh"
)

func ValidateSecurityCriticality(criticality SecurityCriticality) error {
	switch criticality {
	case SecurityCriticalityNormal, SecurityCriticalitySensitive, SecurityCriticalityCritical:
		return nil
	default:
		return domainError(ErrorInvalidArgument, "security_criticality")
	}
}

func ValidateReasoningEffort(effort ReasoningEffort) error {
	switch effort {
	case ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh, ReasoningEffortXHigh:
		return nil
	default:
		return domainError(ErrorInvalidArgument, "reasoning_effort")
	}
}
