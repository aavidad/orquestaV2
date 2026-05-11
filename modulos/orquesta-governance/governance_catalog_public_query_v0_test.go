package orquestagovernance

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryGovernanceCatalogPublicV0ReturnsEffectiveOnly(t *testing.T) {
	provider := GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
		return GovernanceCatalogV0{
			Catalogs: GovernanceCatalogBlocksV0{
				Effective: []GovernanceCatalogEntryV0{
					testEffectiveEntryV0("regla governance review", GovernanceScopeV0{
						Level:   "module",
						Modules: []string{"orquesta-governance"},
						Roles:   []string{"director"},
						Phases:  []string{"review"},
						Tags:    []string{"contracts", "catalog"},
					}),
				},
				Proposed: []GovernanceCatalogEntryV0{
					testInactiveEntryV0("candidato governance", GovernanceCatalogStateProposedV0, func(entry *GovernanceCatalogEntryV0) {
						entry.Scope.Modules = []string{"orquesta-governance"}
						entry.Scope.Roles = []string{"director"}
						entry.Scope.Phases = []string{"review"}
						entry.Scope.Tags = []string{"contracts", "catalog"}
					}),
				},
				Quarantine: []GovernanceCatalogEntryV0{
					testInactiveEntryV0("historico governance", GovernanceCatalogStateQuarantineV0, func(entry *GovernanceCatalogEntryV0) {
						entry.Scope.Modules = []string{"orquesta-governance"}
						entry.Scope.Roles = []string{"director"}
						entry.Scope.Phases = []string{"review"}
						entry.Scope.Tags = []string{"contracts", "catalog"}
					}),
				},
			},
		}, nil
	})

	response, err := QueryGovernanceCatalogPublicV0(provider, GovernanceCatalogPublicQueryRequestV0{
		RequestID:     "req-1",
		CorrelationID: "corr-1",
		Filters: GovernanceCatalogPublicQueryFiltersV0{
			Module: "orquesta-governance",
			Role:   "director",
			Phase:  "review",
			Tags:   []string{"contracts", "catalog", "contracts"},
		},
	})
	if err != nil {
		t.Fatalf("QueryGovernanceCatalogPublicV0() error = %v", err)
	}

	if response.Result.CurrentBlock != GovernanceCatalogCurrentBlockEffectiveV0 {
		t.Fatalf("CurrentBlock = %q, want %q", response.Result.CurrentBlock, GovernanceCatalogCurrentBlockEffectiveV0)
	}
	if len(response.Result.InactiveBlocks) != 2 {
		t.Fatalf("InactiveBlocks len = %d, want 2", len(response.Result.InactiveBlocks))
	}
	if got := response.Result.Counters; got.Effective != 1 || got.Proposed != 1 || got.Quarantine != 1 {
		t.Fatalf("Counters = %+v, want effective=1 proposed=1 quarantine=1", got)
	}
	if len(response.Result.Effective) != 1 || response.Result.Effective[0].Name != "regla governance review" {
		t.Fatalf("Effective = %+v, want one governance rule", response.Result.Effective)
	}
}

func TestQueryGovernanceCatalogPublicV0MapsProviderFailure(t *testing.T) {
	provider := GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
		return GovernanceCatalogV0{}, errors.New("fixture unavailable")
	})

	_, err := QueryGovernanceCatalogPublicV0(provider, GovernanceCatalogPublicQueryRequestV0{})
	if !errors.Is(err, GovernanceCatalogErrorV0("governance_catalog_source_unavailable")) {
		t.Fatalf("error = %v, want governance_catalog_source_unavailable", err)
	}
}

func TestGovernanceCatalogQueryHTTPHandlerV0ReturnsJSONResponse(t *testing.T) {
	provider := GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
		return GovernanceCatalogV0{
			Catalogs: GovernanceCatalogBlocksV0{
				Effective: []GovernanceCatalogEntryV0{
					testEffectiveEntryV0("regla cli governance", GovernanceScopeV0{
						Level:   "module",
						Modules: []string{"orquesta-cli"},
						Roles:   []string{"director"},
						Phases:  []string{"review"},
						Tags:    []string{"public-read"},
					}),
				},
			},
		}, nil
	})

	requestBody := `{"request_id":"req-http","correlation_id":"corr-http","filters":{"module":"orquesta-cli","role":"director","phase":"review","tags":["public-read"]}}`
	request := httptest.NewRequest(http.MethodPost, GovernanceCatalogQueryHTTPPathV0, strings.NewReader(requestBody))
	recorder := httptest.NewRecorder()

	GovernanceCatalogQueryHTTPHandlerV0(provider).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response GovernanceCatalogPublicQueryResponseV0
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.RequestID != "req-http" || response.CorrelationID != "corr-http" {
		t.Fatalf("response ids = %q/%q, want req-http/corr-http", response.RequestID, response.CorrelationID)
	}
	if response.Result.Counters.Effective != 1 {
		t.Fatalf("effective counter = %d, want 1", response.Result.Counters.Effective)
	}
}

func TestGovernanceCatalogQueryHTTPHandlerV0RejectsInvalidJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, GovernanceCatalogQueryHTTPPathV0, strings.NewReader(`{"filters":{"module":"orquesta-cli"},"extra":true}`))
	recorder := httptest.NewRecorder()

	GovernanceCatalogQueryHTTPHandlerV0(GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
		return GovernanceCatalogV0{}, nil
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var response GovernanceCatalogPublicErrorResponseV0
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Errors) != 1 || response.Errors[0].Code != GovernanceCatalogInvalidRequestErrorCodeV0 {
		t.Fatalf("errors = %+v, want governance_catalog_invalid_request", response.Errors)
	}
}

func TestGovernanceCatalogQueryHTTPHandlerV0RejectsMethod(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, GovernanceCatalogQueryHTTPPathV0, nil)
	recorder := httptest.NewRecorder()

	GovernanceCatalogQueryHTTPHandlerV0(nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
