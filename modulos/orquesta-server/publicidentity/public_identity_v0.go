package publicidentity

import "strings"

const (
	PublicCorrelationHeaderV0          = "X-Correlation-ID"
	PublicIdempotencyKeyHeaderV0       = "Idempotency-Key"
	PublicLegacyIdempotencyKeyHeaderV0 = "X-Idempotency-Key"

	PublicIdentityIssueIdempotencyKeyRequiredV0 = "idempotency_key_requerida"
)

const (
	PublicIdentityModeReadOnlyV0              = "read_only"
	PublicIdentityModeMutationIdempotentV0    = "mutation_idempotent"
	PublicIdentityModeMutationNonIdempotentV0 = "mutation_non_idempotent"
	PublicIdentityModeOperatorBreakglassV0    = "operator_breakglass"
)

const (
	PublicIdentitySourceEmptyV0          = ""
	PublicIdentitySourceHeaderV0         = "header"
	PublicIdentitySourceBodyV0           = "body"
	PublicIdentitySourceDerivedAllowedV0 = "derived_allowed"
	PublicIdentitySourceMissingBlockedV0 = "missing_blocked"
)

type PublicIdentityInputV0 struct {
	Mode                 string
	RequestID            string
	CorrelationID        string
	IdempotencyKey       string
	HeaderCorrelationID  string
	HeaderIdempotencyKey string
	Mutating             bool
}

type PublicIdentityV0 struct {
	Mode                     string
	RequestID                string
	CorrelationID            string
	IdempotencyKey           string
	CorrelationIDDerived     bool
	IdempotencyKeyDerived    bool
	CorrelationSource        string
	IdempotencyKeySource     string
	CorrelationHeaderName    string
	IdempotencyKeyHeaderName string
	Issues                   []PublicIdentityIssueV0
}

type PublicIdentityIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func NormalizePublicIdentityV0(input PublicIdentityInputV0) PublicIdentityV0 {
	mode := normalizePublicIdentityModeV0(input.Mode, input.Mutating)
	requestID := strings.TrimSpace(input.RequestID)
	headerCorrelationID := strings.TrimSpace(input.HeaderCorrelationID)
	declaredCorrelationID := strings.TrimSpace(input.CorrelationID)
	correlationID := firstNonEmptyPublicIdentityV0(headerCorrelationID, declaredCorrelationID, requestID)
	correlationSource := publicIdentitySourceV0(headerCorrelationID, declaredCorrelationID, correlationID)

	headerIdempotencyKey := strings.TrimSpace(input.HeaderIdempotencyKey)
	declaredIdempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	idempotencyKey := firstNonEmptyPublicIdentityV0(headerIdempotencyKey, declaredIdempotencyKey)
	idempotencyDerived := false
	idempotencySource := publicIdentitySourceV0(headerIdempotencyKey, declaredIdempotencyKey, idempotencyKey)
	if publicIdentityCanDeriveIdempotencyV0(mode) && idempotencyKey == "" && requestID != "" {
		idempotencyKey = "idem-" + requestID
		idempotencyDerived = true
		idempotencySource = PublicIdentitySourceDerivedAllowedV0
	}

	var issues []PublicIdentityIssueV0
	if publicIdentityRequiresIdempotencyV0(mode) && idempotencyKey == "" {
		idempotencySource = PublicIdentitySourceMissingBlockedV0
		issues = append(issues, PublicIdentityIssueV0{
			Code:    PublicIdentityIssueIdempotencyKeyRequiredV0,
			Field:   "idempotency_key",
			Message: PublicIdentityIssueIdempotencyKeyRequiredV0,
		})
	}

	return PublicIdentityV0{
		Mode:                     mode,
		RequestID:                requestID,
		CorrelationID:            correlationID,
		IdempotencyKey:           idempotencyKey,
		CorrelationIDDerived:     headerCorrelationID == "" && declaredCorrelationID == "" && correlationID != "",
		IdempotencyKeyDerived:    idempotencyDerived,
		CorrelationSource:        correlationSource,
		IdempotencyKeySource:     idempotencySource,
		CorrelationHeaderName:    PublicCorrelationHeaderV0,
		IdempotencyKeyHeaderName: PublicIdempotencyKeyHeaderV0,
		Issues:                   issues,
	}
}

func HeaderIdempotencyKeyV0(headerGetter func(string) string) string {
	if headerGetter == nil {
		return ""
	}
	return firstNonEmptyPublicIdentityV0(
		headerGetter(PublicIdempotencyKeyHeaderV0),
		headerGetter(PublicLegacyIdempotencyKeyHeaderV0),
	)
}

func firstNonEmptyPublicIdentityV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizePublicIdentityModeV0(mode string, mutating bool) string {
	switch strings.TrimSpace(mode) {
	case PublicIdentityModeReadOnlyV0:
		return PublicIdentityModeReadOnlyV0
	case PublicIdentityModeMutationIdempotentV0:
		return PublicIdentityModeMutationIdempotentV0
	case PublicIdentityModeMutationNonIdempotentV0:
		return PublicIdentityModeMutationNonIdempotentV0
	case PublicIdentityModeOperatorBreakglassV0:
		return PublicIdentityModeOperatorBreakglassV0
	default:
		if mutating {
			return PublicIdentityModeMutationIdempotentV0
		}
		return PublicIdentityModeReadOnlyV0
	}
}

func publicIdentityRequiresIdempotencyV0(mode string) bool {
	return mode != PublicIdentityModeReadOnlyV0
}

func publicIdentityCanDeriveIdempotencyV0(mode string) bool {
	return mode == PublicIdentityModeMutationIdempotentV0 || mode == PublicIdentityModeOperatorBreakglassV0
}

func publicIdentitySourceV0(headerValue string, bodyValue string, effectiveValue string) string {
	switch {
	case strings.TrimSpace(headerValue) != "":
		return PublicIdentitySourceHeaderV0
	case strings.TrimSpace(bodyValue) != "":
		return PublicIdentitySourceBodyV0
	case strings.TrimSpace(effectiveValue) != "":
		return PublicIdentitySourceDerivedAllowedV0
	default:
		return PublicIdentitySourceEmptyV0
	}
}
