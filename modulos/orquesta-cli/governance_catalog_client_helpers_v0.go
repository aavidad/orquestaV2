package orquestacli

import (
	"errors"
	"net/http"
	"strings"
	"time"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func clientErrorGovernanceEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliSingleGovernanceErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func transportGovernanceEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return cliSingleGovernanceErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func cliSingleGovernanceErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		CliContractGovernanceCatalogV0,
		CliContractVersionGovernanceV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func endpointGovernanceV0(client *GovernanceCatalogCliReaderV0) string {
	if client != nil && strings.TrimSpace(client.Endpoint) != "" {
		return client.Endpoint
	}
	return GovernanceCatalogCliEndpointV0
}

func httpClientGovernanceV0(client *GovernanceCatalogCliReaderV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutGovernanceV0(client *GovernanceCatalogCliReaderV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func isKnownGovernanceIssueCliV0(code string) bool {
	switch strings.TrimSpace(code) {
	case "governance_catalog_source_unavailable",
		"governance_catalog_invalid_source",
		"governance_catalog_entry_without_origin",
		"governance_catalog_entry_without_promotion_criteria",
		string(orquestagovernance.GovernanceCatalogForbiddenActivationErrorV0),
		"governance_catalog_secret_detected",
		"governance_catalog_text_only_secret_control",
		string(orquestagovernance.GovernanceEntryEffectiveWithoutDecisionV0),
		"governance_entry_text_only_secret_control":
		return true
	default:
		return false
	}
}

func governanceCatalogCleanTagsV0(tags []string) []string {
	clean := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		clean = append(clean, tag)
	}
	if clean == nil {
		return []string{}
	}
	return clean
}
