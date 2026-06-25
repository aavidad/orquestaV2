package orquestaappgateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestNewHTTPHandlerV0ExponeNuevaAppIntakeGuidedTurnV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(orquestaweb.WebNuevaAppIntakeGuidedRequestV0{
		SessionID: "session-app-gateway-guided-1",
		Locale:    "es",
		Need:      "Quiero una app para movil que muestre pisos en alquiler cercanos con mapas",
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestahttpgateway.RouteAppIntakeGuidedTurnV0, &body)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out orquestaweb.WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.SchemaVersion != orquestaweb.WebNuevaAppIntakeGuidedResponseSchemaV0 ||
		out.Session.Form.TipoApp != "mobile" ||
		len(out.Session.Form.Integraciones) == 0 {
		t.Fatalf("response=%+v", out)
	}
}
