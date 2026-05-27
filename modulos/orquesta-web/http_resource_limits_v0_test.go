package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebControlJSONV0RechazaTrailingDataV0(t *testing.T) {
	var input WebRunControlCommandV0
	req := httptest.NewRequest(http.MethodPost, "/run-control", strings.NewReader(`{} {}`))

	err := decodeWebControlJSONV0(httptest.NewRecorder(), req, &input)

	if err == nil || !strings.Contains(err.Error(), "request_body_trailing_data") {
		t.Fatalf("err=%v", err)
	}
}

func TestWebControlJSONV0RechazaBodyTooLargeV0(t *testing.T) {
	var input WebRunControlCommandV0
	body := strings.NewReader(`{"reason":"` + strings.Repeat("a", int(webControlJSONMaxBytesV0)+1) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/run-control", body)

	err := decodeWebControlJSONV0(httptest.NewRecorder(), req, &input)

	if err == nil || !strings.Contains(err.Error(), "http: request body too large") {
		t.Fatalf("err=%v", err)
	}
}

func TestWebControlContentTypeV0DeclaraJSONYFormLegacyV0(t *testing.T) {
	if !webControlContentTypeAllowsJSONV0("") {
		t.Fatalf("content-type vacio debe conservar compatibilidad JSON legacy")
	}
	if !webControlContentTypeAllowsJSONV0("application/vnd.orquesta+json; charset=utf-8") {
		t.Fatalf("+json debe aceptarse")
	}
	if webControlContentTypeAllowsJSONV0("text/plain") {
		t.Fatalf("text/plain no debe aceptarse como JSON")
	}
	if !webControlContentTypeAllowsFormV0("application/x-www-form-urlencoded; charset=utf-8") {
		t.Fatalf("form urlencoded debe aceptarse")
	}
	if !webControlContentTypeAllowsFormV0("multipart/form-data; boundary=x") {
		t.Fatalf("multipart form debe aceptarse")
	}
}

func TestWebPublicQueryV0RechazaRawQueryGrandeV0(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/run-queue?reason="+strings.Repeat("a", webPublicQueryMaxBytesV0+1), nil)

	err := validateWebPublicQueryV0(req)

	if err == nil || !strings.Contains(err.Error(), "public_query_too_large") {
		t.Fatalf("err=%v", err)
	}
}

func TestWebControlFormV0RechazaClaveRepetidaV0(t *testing.T) {
	body := strings.Repeat("evidence_refs=x&", webPublicParamMaxValuesPerKeyV0+1)
	req := httptest.NewRequest(http.MethodPost, "/run-control", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := parseWebControlFormV0(httptest.NewRecorder(), req)

	if err == nil || !strings.Contains(err.Error(), "public_parameter_repeated") {
		t.Fatalf("err=%v", err)
	}
}

func TestWebRunQueueGetV0RechazaQuerySinPersistirRawQueryV0(t *testing.T) {
	endpoint := NewRunQueueWebEndpointV0(nil)
	req := httptest.NewRequest(http.MethodGet, "/run-queue?token="+strings.Repeat("x", webPublicQueryMaxBytesV0+1), nil)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest ||
		strings.Contains(rec.Body.String(), "token=") ||
		strings.Contains(rec.Body.String(), strings.Repeat("x", 64)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
