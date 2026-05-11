package orquestafactory

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const AppSpecHTTPPathV0 = "/api/v0/apps/spec"

type AppSpecHTTPClockV0 func() time.Time

type AppSpecHTTPResponseV0 struct {
	AppSpec AppSpecV0                 `json:"app_spec"`
	Backlog BacklogInicialPropuestoV0 `json:"backlog"`
}

type AppSpecHTTPErrorResponseV0 struct {
	Errores []ValidationIssue `json:"errores"`
}

func NewAppSpecHTTPHandlerV0(clock AppSpecHTTPClockV0) http.Handler {
	return appSpecHTTPHandlerV0{clock: clock}
}

type appSpecHTTPHandlerV0 struct {
	clock AppSpecHTTPClockV0
}

func (h appSpecHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != AppSpecHTTPPathV0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusNotFound, []ValidationIssue{
			issue(ErrAppSpecInvalida, "path", "ruta no soportada"),
		}, correlationIDFromHeaderV0(r))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeAppSpecHTTPErrorsV0(w, http.StatusMethodNotAllowed, []ValidationIssue{
			issue(ErrAppSpecInvalida, "method", "metodo no permitido"),
		}, correlationIDFromHeaderV0(r))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, []ValidationIssue{
			issue(ErrAppSpecInvalida, "body", "request body invalido"),
		}, correlationIDFromHeaderV0(r))
		return
	}

	req, issues := DecodeAppSpecRequestV0(body)
	correlationID := correlationIDV0(r, req)
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationID)
		return
	}

	spec, issues := SolicitarNuevaAppV0(req, h.now())
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationID)
		return
	}
	backlog, issues := GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		writeAppSpecHTTPErrorsV0(w, http.StatusBadRequest, issues, correlationID)
		return
	}

	writeAppSpecHTTPJSONV0(w, http.StatusOK, AppSpecHTTPResponseV0{
		AppSpec: spec,
		Backlog: backlog,
	}, correlationID)
}

func (h appSpecHTTPHandlerV0) now() time.Time {
	if h.clock == nil {
		return time.Now().UTC()
	}
	return h.clock().UTC()
}

func correlationIDV0(r *http.Request, req AppSpecRequestV0) string {
	if id := correlationIDFromHeaderV0(r); id != "" {
		return id
	}
	return strings.TrimSpace(req.RequestID)
}

func correlationIDFromHeaderV0(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
}

func writeAppSpecHTTPErrorsV0(w http.ResponseWriter, status int, issues []ValidationIssue, correlationID string) {
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
