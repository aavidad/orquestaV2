package orquestamcp

import publicidentity "orquesta/modulos/orquesta-server/publicidentity"

const (
	MCPPublicCorrelationHeaderV0          = publicidentity.PublicCorrelationHeaderV0
	MCPPublicIdempotencyKeyHeaderV0       = publicidentity.PublicIdempotencyKeyHeaderV0
	MCPPublicLegacyIdempotencyKeyHeaderV0 = publicidentity.PublicLegacyIdempotencyKeyHeaderV0

	MCPPublicMutationIssueIdempotencyKeyRequiredV0 = publicidentity.PublicIdentityIssueIdempotencyKeyRequiredV0
)

type MCPPublicMutationIdentityInputV0 struct {
	Mode                 string
	RequestID            string
	CorrelationID        string
	IdempotencyKey       string
	HeaderCorrelationID  string
	HeaderIdempotencyKey string
	Mutating             bool
}

type MCPPublicMutationIdentityV0 struct {
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
	Issues                   []MCPValidationIssueV0
}

func NormalizeMCPPublicMutationIdentityV0(
	input MCPPublicMutationIdentityInputV0,
) MCPPublicMutationIdentityV0 {
	identity := publicidentity.NormalizePublicIdentityV0(publicidentity.PublicIdentityInputV0(input))
	return MCPPublicMutationIdentityV0{
		Mode:                     identity.Mode,
		RequestID:                identity.RequestID,
		CorrelationID:            identity.CorrelationID,
		IdempotencyKey:           identity.IdempotencyKey,
		CorrelationIDDerived:     identity.CorrelationIDDerived,
		IdempotencyKeyDerived:    identity.IdempotencyKeyDerived,
		CorrelationSource:        identity.CorrelationSource,
		IdempotencyKeySource:     identity.IdempotencyKeySource,
		CorrelationHeaderName:    identity.CorrelationHeaderName,
		IdempotencyKeyHeaderName: identity.IdempotencyKeyHeaderName,
		Issues:                   mcpPublicIdentityIssuesV0(identity.Issues),
	}
}

func MCPPublicMutationHeaderIdempotencyKeyV0(headerGetter func(string) string) string {
	return publicidentity.HeaderIdempotencyKeyV0(headerGetter)
}

func mcpPublicIdentityIssuesV0(values []publicidentity.PublicIdentityIssueV0) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(values))
	for _, value := range values {
		out = append(out, MCPValidationIssueV0{
			Code:    value.Code,
			Field:   value.Field,
			Message: value.Message,
		})
	}
	return out
}
