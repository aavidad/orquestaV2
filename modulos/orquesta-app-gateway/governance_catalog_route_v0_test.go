package orquestaappgateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func TestGovernanceCatalogAPIDelegaEnProviderPublicoV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{
		GovernanceCatalog: orquestagovernance.GovernanceCatalogProviderFuncV0(func() (orquestagovernance.GovernanceCatalogV0, error) {
			return governanceCatalogForAppGatewayTestV0(), nil
		}),
		Timeout: time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/governance/catalog/query",
		strings.NewReader(`{"request_id":"req-gov-app-gateway","correlation_id":"corr-gov-app-gateway","filters":{"module":"orquesta-app-gateway","tags":["public-read"]}}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response orquestagovernance.GovernanceCatalogPublicQueryResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequestID != "req-gov-app-gateway" ||
		response.CorrelationID != "corr-gov-app-gateway" ||
		len(response.Effective) != 1 ||
		response.Counters.Effective != 1 ||
		response.Counters.Proposed != 1 ||
		response.Counters.Quarantine != 1 {
		t.Fatalf("response=%+v", response)
	}
}

func TestGovernanceCatalogAPISinProviderDevuelveErrorPublicoV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/governance/catalog/query",
		strings.NewReader(`{"request_id":"req-gov-missing","correlation_id":"corr-gov-missing","filters":{"module":"orquesta-app-gateway"}}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response orquestagovernance.GovernanceCatalogPublicErrorResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequestID != "req-gov-missing" ||
		response.CorrelationID != "corr-gov-missing" ||
		len(response.Errors) != 1 ||
		response.Errors[0].Code != "governance_catalog_source_unavailable" {
		t.Fatalf("response=%+v", response)
	}
}

func TestGovernanceCatalogAPIFalloProviderDevuelveErrorPublicoV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{
		GovernanceCatalog: orquestagovernance.GovernanceCatalogProviderFuncV0(func() (orquestagovernance.GovernanceCatalogV0, error) {
			return orquestagovernance.GovernanceCatalogV0{}, errors.New("private source detail")
		}),
		Timeout: time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/governance/catalog/query",
		strings.NewReader(`{"request_id":"req-gov-fail","correlation_id":"corr-gov-fail","filters":{"module":"orquesta-app-gateway"}}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), "private source detail") {
		t.Fatalf("response filtra detalle privado: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func governanceCatalogForAppGatewayTestV0() orquestagovernance.GovernanceCatalogV0 {
	return orquestagovernance.GovernanceCatalogV0{
		Catalogs: orquestagovernance.GovernanceCatalogBlocksV0{
			Effective: []orquestagovernance.GovernanceCatalogEntryV0{
				governanceCatalogEntryForAppGatewayTestV0(
					"governance app gateway effective",
					orquestagovernance.GovernanceCatalogStateEffectiveV0,
					orquestagovernance.GovernanceReviewStateApprovedEffectiveV0,
				),
			},
			Proposed: []orquestagovernance.GovernanceCatalogEntryV0{
				governanceCatalogEntryForAppGatewayTestV0("governance app gateway proposed", orquestagovernance.GovernanceCatalogStateProposedV0, "pending_review"),
			},
			Quarantine: []orquestagovernance.GovernanceCatalogEntryV0{
				governanceCatalogEntryForAppGatewayTestV0("governance app gateway quarantine", orquestagovernance.GovernanceCatalogStateQuarantineV0, "quarantined"),
			},
		},
	}
}

func governanceCatalogEntryForAppGatewayTestV0(name string, state string, review string) orquestagovernance.GovernanceCatalogEntryV0 {
	decisionRef := "decision-ref-governance-app-gateway"
	promotedAt := "2026-05-25T00:00:00Z"
	return orquestagovernance.GovernanceCatalogEntryV0{
		Kind:    "rule",
		Name:    name,
		Summary: "public route shape sync",
		Scope: orquestagovernance.GovernanceScopeV0{
			Modules: []string{"orquesta-app-gateway"},
			Tags:    []string{"public-read"},
		},
		Status: orquestagovernance.GovernanceStatusV0{CatalogState: state, ReviewState: review},
		Promotion: orquestagovernance.GovernancePromotionV0{
			DecisionRef: &decisionRef,
			PromotedAt:  &promotedAt,
		},
	}
}
