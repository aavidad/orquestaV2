package orquestagovernance

import (
	"encoding/json"
	"errors"
	"io"
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
}

type GovernanceCatalogPublicQueryFiltersV0 struct {
	Module string   `json:"module,omitempty"`
	Role   string   `json:"role,omitempty"`
	Phase  string   `json:"phase,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type GovernanceCatalogPublicQueryResponseV0 struct {
	RequestID     string                      `json:"request_id,omitempty"`
	CorrelationID string                      `json:"correlation_id,omitempty"`
	Effective     []GovernanceCatalogEntryV0  `json:"effective"`
	Counters      GovernanceCatalogCountersV0 `json:"counters"`
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
		RequestID:     strings.TrimSpace(request.RequestID),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
	}

	if provider == nil {
		return response, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")
	}

	catalog, err := provider.LoadGovernanceCatalogV0()
	if err != nil {
		return response, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")
	}

	result, err := QueryEffectiveGovernanceCatalogV0(catalog, GovernanceCatalogQueryV0{
		Module: strings.TrimSpace(request.Filters.Module),
		Role:   strings.TrimSpace(request.Filters.Role),
		Phase:  strings.TrimSpace(request.Filters.Phase),
		Tags:   governanceCleanTagsV0(request.Filters.Tags),
	})
	if err != nil {
		return response, err
	}

	response.Effective = result.Effective
	response.Counters = result.Counters
	return response, nil
}

func GovernanceCatalogQueryHTTPHandlerV0(provider GovernanceCatalogProviderV0) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeGovernanceCatalogPublicErrorV0(w, http.StatusMethodNotAllowed, GovernanceCatalogPublicErrorResponseV0{
				Errors: []GovernanceCatalogPublicErrorV0{{Code: GovernanceCatalogMethodNotAllowedErrorCodeV0}},
			})
			return
		}

		var request GovernanceCatalogPublicQueryRequestV0
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeGovernanceCatalogPublicErrorV0(w, http.StatusBadRequest, GovernanceCatalogPublicErrorResponseV0{
				Errors: []GovernanceCatalogPublicErrorV0{{Code: GovernanceCatalogInvalidRequestErrorCodeV0}},
			})
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeGovernanceCatalogPublicErrorV0(w, http.StatusBadRequest, GovernanceCatalogPublicErrorResponseV0{
				RequestID:     strings.TrimSpace(request.RequestID),
				CorrelationID: strings.TrimSpace(request.CorrelationID),
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
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})
}

func writeGovernanceCatalogPublicErrorV0(w http.ResponseWriter, statusCode int, response GovernanceCatalogPublicErrorResponseV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
