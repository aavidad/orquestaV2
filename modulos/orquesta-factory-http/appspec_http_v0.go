package orquestafactoryhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const AppSpecHTTPPathV0 = "/api/v0/apps/spec"

const appSpecHTTPJSONMaxBytesV0 int64 = 256 << 10

type AppSpecHTTPClockV0 func() time.Time

type AppSpecHTTPResponseV0 struct {
	AppSpec orquestafactory.AppSpecV0                 `json:"app_spec"`
	Backlog orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
}

type AppSpecHTTPErrorResponseV0 struct {
	Errores []orquestafactory.ValidationIssue `json:"errores"`
}

func NewAppSpecHTTPHandlerV0(clock AppSpecHTTPClockV0) http.Handler {
	return appSpecHTTPHandlerV0{clock: clock}
}

type appSpecHTTPHandlerV0 struct {
	clock AppSpecHTTPClockV0
}

func (h appSpecHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != AppSpecHTTPPathV0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusNotFound, []orquestafactory.ValidationIssue{
			issue(orquestafactory.ErrAppSpecInvalida, "path", "ruta no soportada"),
		}, correlationIDFromHeaderV0(r))
		return
	}
	if handleAppSpecHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setAppSpecHTTPAllowV0(w, http.MethodPost)
		writeAppSpecHTTPErrorsV0(w, http.StatusMethodNotAllowed, []orquestafactory.ValidationIssue{
			issue(orquestafactory.ErrAppSpecInvalida, "method", "metodo no permitido"),
		}, correlationIDFromHeaderV0(r))
		return
	}

	req, issues := decodeAppSpecHTTPJSONRequestV0(w, r)
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationIDV0(r, req))
		return
	}

	correlationID := correlationIDV0(r, req)
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, h.now())
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationID)
		return
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationID)
		return
	}

	writeAppSpecHTTPJSONV0(w, http.StatusOK, AppSpecHTTPResponseV0{
		AppSpec: spec,
		Backlog: backlog,
	}, correlationID)
}

func decodeAppSpecHTTPJSONRequestV0(w http.ResponseWriter, r *http.Request) (orquestafactory.AppSpecRequestV0, []orquestafactory.ValidationIssue) {
	var req orquestafactory.AppSpecRequestV0
	if !appSpecHTTPContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		return req, []orquestafactory.ValidationIssue{
			issue(orquestafactory.ErrAppSpecInvalida, "content_type", "content_type_no_soportado"),
		}
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, appSpecHTTPJSONMaxBytesV0))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return req, []orquestafactory.ValidationIssue{
				issue(orquestafactory.ErrAppSpecInvalida, "body", "request_body_too_large"),
			}
		}
		return req, []orquestafactory.ValidationIssue{
			issue(orquestafactory.ErrAppSpecInvalida, "body", "request_body_invalido"),
		}
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return req, []orquestafactory.ValidationIssue{
			issue(orquestafactory.ErrAppSpecInvalida, "body", "request_body_trailing_data"),
		}
	}
	return req, orquestafactory.ValidateAppSpecRequestV0(req)
}

func appSpecHTTPContentTypeAllowsJSONV0(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}
	mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func (h appSpecHTTPHandlerV0) now() time.Time {
	if h.clock == nil {
		return time.Now().UTC()
	}
	return h.clock().UTC()
}

func correlationIDV0(r *http.Request, req orquestafactory.AppSpecRequestV0) string {
	if id := correlationIDFromHeaderV0(r); id != "" {
		return id
	}
	return strings.TrimSpace(req.RequestID)
}

func correlationIDFromHeaderV0(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
}

func issue(code, field, message string) orquestafactory.ValidationIssue {
	return orquestafactory.ValidationIssue{Code: code, Field: field, Message: message}
}

func writeAppSpecHTTPErrorsV0(w http.ResponseWriter, status int, issues []orquestafactory.ValidationIssue, correlationID string) {
	writeAppSpecHTTPJSONV0(w, status, AppSpecHTTPErrorResponseV0{Errores: issues}, correlationID)
}

func writeAppSpecHTTPJSONV0(w http.ResponseWriter, status int, payload any, correlationID string) {
	w.Header().Set("Content-Type", "application/json")
	if correlationID != "" {
		w.Header().Set("X-Correlation-ID", correlationID)
	}
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}
