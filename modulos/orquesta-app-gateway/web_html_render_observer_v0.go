package orquestaappgateway

import (
	"net/http"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func ObserveWebHTMLRenderErrorsV0(
	next http.Handler,
	observer orquestaobservability.WebHTMLRenderObserverV0,
) http.Handler {
	if next == nil || observer == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &webHTMLRenderObservationWriterV0{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(writer, r)
		observeWebHTMLRenderResultV0(observer, writer, r)
	})
}

type webHTMLRenderObservationWriterV0 struct {
	http.ResponseWriter
	statusCode int
	writeErr   bool
}

func (writer *webHTMLRenderObservationWriterV0) WriteHeader(statusCode int) {
	writer.statusCode = statusCode
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *webHTMLRenderObservationWriterV0) Write(data []byte) (int, error) {
	written, err := writer.ResponseWriter.Write(data)
	if err != nil {
		writer.writeErr = true
	}
	return written, err
}

func observeWebHTMLRenderResultV0(
	observer orquestaobservability.WebHTMLRenderObserverV0,
	writer *webHTMLRenderObservationWriterV0,
	r *http.Request,
) {
	reason := strings.TrimSpace(writer.Header().Get(orquestaweb.WebHTMLErrorCodeHeaderV0))
	stage := orquestaobservability.WebHTMLRenderStageRenderV0
	if writer.writeErr {
		reason = orquestaobservability.WebHTMLResponseWriteFailedReasonV0
		stage = orquestaobservability.WebHTMLRenderStageWriteV0
	}
	if reason == "" {
		return
	}
	observer.ObserveWebHTMLRenderV0(orquestaobservability.NewWebHTMLRenderObservationV0(
		webHTMLRenderRouteRefV0(r),
		reason,
		stage,
		writer.statusCode,
		strings.TrimSpace(writer.Header().Get(orquestaweb.WebHTMLLocaleHeaderV0)),
	))
}

func webHTMLRenderRouteRefV0(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "route-ref-web-unknown"
	}
	switch r.URL.Path {
	case "/nueva-app":
		return "route-ref-web-nueva-app"
	case "/app-change":
		return "route-ref-web-app-change"
	case "/ops":
		return "route-ref-web-ops"
	default:
		return "route-ref-web-other"
	}
}
