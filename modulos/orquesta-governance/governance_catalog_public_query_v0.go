package orquestagovernance

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const (
	GovernanceCatalogQueryHTTPPathV0             = "/api/v0/governance/catalog/query"
	GovernanceCatalogCurrentBlockEffectiveV0     = "effective"
	GovernanceCatalogInactiveBlockProposedV0     = "proposed"
	GovernanceCatalogInactiveBlockQuarantineV0   = "quarantine"
	GovernanceCatalogInvalidRequestErrorCodeV0   = "governance_catalog_invalid_request"
	GovernanceCatalogMethodNotAllowedErrorCodeV0 = "governance_catalog_method_not_allowed"
)

type GovernanceCatalogProviderV0 interface {
	LoadGovernanceCatalogV0() (GovernanceCatalogV0, error)
}

type GovernanceCatalogProviderFuncV0 func() (GovernanceCatalogV0, error)

func (fn GovernanceCatalogProviderFuncV0) LoadGovernanceCatalogV0() (GovernanceCatalogV0, error) {
	return fn()
}

type GovernanceCatalogPublicQueryRequestV0 struct {
	RequestID     string                                `json:"request_id,omitempty"`
	CorrelationID string                                `json:"correlation_id,omitempty"`
	Filters       GovernanceCatalogPublicQueryFiltersV0 `json:"filters"`
	OutputBudget  GovernanceCatalogPublicOutputBudgetV0 `json:"output_budget,omitempty"`
}

type GovernanceCatalogPublicQueryFiltersV0 struct {
	Module string   `json:"module,omitempty"`
	Role   string   `json:"role,omitempty"`
	Phase  string   `json:"phase,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type GovernanceCatalogPublicQueryResponseV0 struct {
	SchemaVersion   string                                   `json:"schema_version"`
	RequestID       string                                   `json:"request_id,omitempty"`
	CorrelationID   string                                   `json:"correlation_id,omitempty"`
	CatalogVersion  string                                   `json:"catalog_version"`
	CurrentBlock    string                                   `json:"current_block"`
	Freshness       GovernanceCatalogPublicFreshnessV0       `json:"freshness"`
	SourceRefs      []string                                 `json:"source_refs,omitempty"`
	Effective       []GovernanceCatalogEntryV0               `json:"effective"`
	Counters        GovernanceCatalogCountersV0              `json:"counters"`
	InactiveSummary GovernanceCatalogPublicInactiveSummaryV0 `json:"inactive_summary"`
	OutputBudget    GovernanceCatalogPublicOutputBudgetV0    `json:"output_budget"`
}

type GovernanceCatalogPublicErrorResponseV0 struct {
	RequestID     string                           `json:"request_id,omitempty"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
	Errors        []GovernanceCatalogPublicErrorV0 `json:"errors"`
}

type GovernanceCatalogPublicErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func QueryGovernanceCatalogPublicV0(provider GovernanceCatalogProviderV0, request GovernanceCatalogPublicQueryRequestV0) (GovernanceCatalogPublicQueryResponseV0, error) {
	response := GovernanceCatalogPublicQueryResponseV0{
		SchemaVersion:  GovernanceCatalogPublicSchemaV0,
		RequestID:      strings.TrimSpace(request.RequestID),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		CatalogVersion: GovernanceCatalogVersionV0,
		CurrentBlock:   GovernanceCatalogCurrentBlockEffectiveV0,
		OutputBudget:   NormalizeGovernanceCatalogPublicOutputBudgetV0(request.OutputBudget),
	}

	if provider == nil {
		return response, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")
	}

	catalog, err := provider.LoadGovernanceCatalogV0()
	if err != nil {
		return response, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")
	}

	if version := strings.TrimSpace(catalog.CatalogVersion); version != "" {
		response.CatalogVersion = version
	}
	query := GovernanceCatalogQueryV0{
		Module: strings.TrimSpace(request.Filters.Module),
		Role:   strings.TrimSpace(request.Filters.Role),
		Phase:  strings.TrimSpace(request.Filters.Phase),
		Tags:   governanceCleanTagsV0(request.Filters.Tags),
	}
	result, err := QueryEffectiveGovernanceCatalogV0(catalog, query)
	if err != nil {
		return response, err
	}

	response.Freshness = governanceCatalogPublicFreshnessV0(catalog)
	response.SourceRefs = response.Freshness.SourceRefs
	response.InactiveSummary = governanceCatalogPublicInactiveSummaryV0(catalog, query)
	response.Effective, response.OutputBudget = limitGovernanceCatalogPublicEffectiveV0(result.Effective, response.OutputBudget)
	response.Counters = result.Counters
	return response, nil
}

func GovernanceCatalogQueryHTTPHandlerV0(provider GovernanceCatalogProviderV0) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleGovernancePublicHTTPOptionsV0(w, r, http.MethodPost) {
			return
		}
		if r.Method != http.MethodPost {
			setGovernancePublicHTTPAllowV0(w, http.MethodPost)
			writeGovernanceCatalogPublicErrorV0(w, http.StatusMethodNotAllowed, GovernanceCatalogPublicErrorResponseV0{
				CorrelationID: governancePublicCorrelationIDFromHeaderV0(r),
				Errors:        []GovernanceCatalogPublicErrorV0{{Code: GovernanceCatalogMethodNotAllowedErrorCodeV0}},
			})
			return
		}

		var request GovernanceCatalogPublicQueryRequestV0
		if code := decodeGovernancePublicHTTPJSONV0(w, r, &request); code != "" {
			writeGovernanceCatalogPublicErrorV0(w, http.StatusBadRequest, GovernanceCatalogPublicErrorResponseV0{
				RequestID:     strings.TrimSpace(request.RequestID),
				CorrelationID: firstNonEmptyGovernancePublicV0(request.CorrelationID, governancePublicCorrelationIDFromHeaderV0(r)),
				Errors:        []GovernanceCatalogPublicErrorV0{{Code: GovernanceCatalogInvalidRequestErrorCodeV0}},
			})
			return
		}

		response, err := QueryGovernanceCatalogPublicV0(provider, request)
		if err != nil {
			statusCode := http.StatusBadRequest
			if errors.Is(err, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")) {
				statusCode = http.StatusServiceUnavailable
			}
			writeGovernanceCatalogPublicErrorV0(w, statusCode, GovernanceCatalogPublicErrorResponseV0{
				RequestID:     response.RequestID,
				CorrelationID: response.CorrelationID,
				Errors:        []GovernanceCatalogPublicErrorV0{{Code: err.Error()}},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if response.CorrelationID != "" {
			w.Header().Set("X-Correlation-ID", response.CorrelationID)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})
}

func writeGovernanceCatalogPublicErrorV0(w http.ResponseWriter, statusCode int, response GovernanceCatalogPublicErrorResponseV0) {
	w.Header().Set("Content-Type", "application/json")
	if response.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", response.CorrelationID)
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
