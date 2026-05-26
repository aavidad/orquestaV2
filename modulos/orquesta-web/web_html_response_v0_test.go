package orquestaweb

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebHTMLTemplateResponseV0RenderErrorDevuelveReasonPublico(t *testing.T) {
	endpoint := NewNuevaAppWebEndpointV0(&fakeNuevaAppClientV0{})
	page := endpoint.page("es", initialNuevaAppViewModelV0("es"))
	brokenTemplate := template.Must(template.New("broken").Parse(`{{.CampoInexistente.RefPrivada}}`))
	rec := httptest.NewRecorder()

	result := writeNuevaAppHTMLPageWithTemplateV0(rec, http.StatusOK, page, brokenTemplate)

	body := rec.Body.String()
	if result.OK ||
		result.ReasonCode != WebHTMLRenderFailedV0 ||
		result.StatusCode != http.StatusInternalServerError ||
		rec.Code != http.StatusInternalServerError {
		t.Fatalf("result=%+v status=%d body=%s", result, rec.Code, body)
	}
	if rec.Header().Get("X-Orquesta-Web-Error-Code") != WebHTMLRenderFailedV0 ||
		!strings.Contains(body, WebHTMLRenderFailedV0) ||
		!strings.Contains(body, `<html lang="es">`) {
		t.Fatalf("fallback publico inesperado headers=%v body=%s", rec.Header(), body)
	}
	for _, forbidden := range []string{"CampoInexistente", "RefPrivada", "HOME", "token", "stack"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("fallback no debe exponer %q: %s", forbidden, body)
		}
	}
}

func TestWebHTMLTemplateResponseV0WriteErrorDevuelveReasonCompacto(t *testing.T) {
	writer := &webHTMLFailingWriterV0{header: http.Header{}}

	result := writeWebHTMLStringResponseV0(writer, http.StatusOK, "<html></html>", "es")

	if result.OK ||
		result.ReasonCode != WebResponseWriteFailedV0 ||
		result.StatusCode != http.StatusOK ||
		writer.status != http.StatusOK {
		t.Fatalf("result=%+v writer=%+v", result, writer)
	}
	if writer.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type=%q", writer.header.Get("Content-Type"))
	}
}

func TestAppChangeHTMLV0UsaFallbackComunAnteRenderError(t *testing.T) {
	brokenTemplate := template.Must(template.New("broken_app_change").Parse(`{{.NoExiste.DetallePrivado}}`))
	rec := httptest.NewRecorder()

	result := writeAppChangeHTMLWithTemplateV0(rec, http.StatusOK,
		appChangePageV0("es", InitialWebAppChangeViewModelV0()), brokenTemplate)

	if result.OK ||
		result.ReasonCode != WebHTMLRenderFailedV0 ||
		rec.Code != http.StatusInternalServerError ||
		strings.Contains(rec.Body.String(), "DetallePrivado") {
		t.Fatalf("result=%+v status=%d body=%s", result, rec.Code, rec.Body.String())
	}
}

type webHTMLFailingWriterV0 struct {
	header http.Header
	status int
}

func (writer *webHTMLFailingWriterV0) Header() http.Header {
	return writer.header
}

func (writer *webHTMLFailingWriterV0) WriteHeader(status int) {
	writer.status = status
}

func (writer *webHTMLFailingWriterV0) Write(_ []byte) (int, error) {
	return 0, errors.New("socket write failed")
}
