package orquestaweb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAppChangeWebEndpointV0POSTLlamaClienteYRenderizaDirector(t *testing.T) {
	client := &fakeAppChangeClientV0{
		result: WebAppChangeViewModelV0{
			Estado:              WebAppChangeEstadoAceptadoV0,
			RunRef:              "run-ref-web-change-001",
			ChangeRef:           "change-ref-web-001",
			DirectorQuestionRef: "question-ref-web-change-001",
		},
	}
	endpoint := NewAppChangeWebEndpointV0(client)
	values := url.Values{}
	values.Set("locale", "es")
	values.Set("run_ref", "run-ref-web-change-001")
	values.Set("change_ref", "change-ref-web-001")
	values.Set("user_intent", "Cambiar la web para vista semanal.")
	req := httptest.NewRequest(http.MethodPost, "/app-change", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.form.ChangeRef != "change-ref-web-001" ||
		client.form.UserIntent != "Cambiar la web para vista semanal." ||
		!strings.Contains(rec.Body.String(), "question-ref-web-change-001") {
		t.Fatalf("form=%+v body=%s", client.form, rec.Body.String())
	}
}

func TestAppChangeWebEndpointV0GETMuestraMensajeParaDirector(t *testing.T) {
	endpoint := NewAppChangeWebEndpointV0(&fakeAppChangeClientV0{})
	req := httptest.NewRequest(http.MethodGet, "/app-change", nil)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK ||
		!strings.Contains(body, `name="user_intent"`) ||
		!strings.Contains(body, "Mensaje para el director") ||
		!strings.Contains(body, "cambios, instrucciones o peticiones") {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}

type fakeAppChangeClientV0 struct {
	form   WebAppChangeFormV0
	result WebAppChangeViewModelV0
}

func (client *fakeAppChangeClientV0) RequestAppChange(
	_ context.Context,
	form WebAppChangeFormV0,
) (WebAppChangeViewModelV0, error) {
	client.form = form
	return client.result, nil
}
