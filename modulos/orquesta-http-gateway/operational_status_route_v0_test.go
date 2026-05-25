package orquestahttpgateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAppGatewayMuxV0RegistersOperationalStatusRoute(t *testing.T) {
	mux := NewAppGatewayMuxV0(RouteHandlersV0{
		OperationalStatus: markerHandler("operational-status"),
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, RouteOperationalStatusV0, nil)

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "operational-status" {
		t.Fatalf("body=%q", response.Body.String())
	}
}
