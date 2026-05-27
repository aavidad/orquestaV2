package orquestaweb

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
)

const (
	WebHTMLRenderFailedV0      = "web_html_render_failed"
	WebResponseWriteFailedV0   = "web_response_write_failed"
	WebHTMLErrorCodeHeaderV0   = "X-Orquesta-Web-Error-Code"
	WebHTMLLocaleHeaderV0      = "X-Orquesta-Web-Locale"
	WebHTMLContentTypeHeaderV0 = "text/html; charset=utf-8"
	WebHTMLContentLanguageV0   = "Content-Language"
)

type WebHTMLWriteResultV0 struct {
	OK         bool
	ReasonCode string
	StatusCode int
}

func writeWebHTMLTemplateResponseV0(
	w http.ResponseWriter,
	status int,
	tmpl *template.Template,
	data any,
	locale string,
) WebHTMLWriteResultV0 {
	if tmpl == nil {
		return writeWebHTMLFallbackV0(w, http.StatusInternalServerError, WebHTMLRenderFailedV0, locale)
	}
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return writeWebHTMLFallbackV0(w, http.StatusInternalServerError, WebHTMLRenderFailedV0, locale)
	}
	setWebHTMLResponseHeadersV0(w, locale)
	w.WriteHeader(status)
	if _, err := w.Write(body.Bytes()); err != nil {
		return WebHTMLWriteResultV0{OK: false, ReasonCode: WebResponseWriteFailedV0, StatusCode: status}
	}
	return WebHTMLWriteResultV0{OK: true, StatusCode: status}
}

func writeWebHTMLStringResponseV0(
	w http.ResponseWriter,
	status int,
	body string,
	locale string,
) WebHTMLWriteResultV0 {
	setWebHTMLResponseHeadersV0(w, locale)
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		return WebHTMLWriteResultV0{OK: false, ReasonCode: WebResponseWriteFailedV0, StatusCode: status}
	}
	return WebHTMLWriteResultV0{OK: true, StatusCode: status}
}

func writeWebHTMLFallbackV0(
	w http.ResponseWriter,
	status int,
	reason string,
	locale string,
) WebHTMLWriteResultV0 {
	if status < http.StatusBadRequest {
		status = http.StatusInternalServerError
	}
	if reason == "" {
		reason = WebHTMLRenderFailedV0
	}
	body := `<!doctype html><html lang="` + template.HTMLEscapeString(safeWebHTMLLocaleV0(locale)) +
		`"><head><meta charset="utf-8"><title>` + template.HTMLEscapeString(reason) +
		`</title></head><body><main><h1>` + template.HTMLEscapeString(reason) +
		`</h1></main></body></html>`
	setWebHTMLResponseHeadersV0(w, locale)
	w.Header().Set(WebHTMLErrorCodeHeaderV0, reason)
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		return WebHTMLWriteResultV0{OK: false, ReasonCode: WebResponseWriteFailedV0, StatusCode: status}
	}
	return WebHTMLWriteResultV0{OK: false, ReasonCode: reason, StatusCode: status}
}

func safeWebHTMLLocaleV0(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" || len(locale) > 16 {
		return "es"
	}
	if index := strings.Index(locale, "-"); index > 0 {
		return locale[:index]
	}
	return locale
}

func setWebHTMLResponseHeadersV0(w http.ResponseWriter, locale string) {
	safeLocale := safeWebHTMLLocaleV0(locale)
	w.Header().Set("Content-Type", WebHTMLContentTypeHeaderV0)
	w.Header().Set(WebHTMLContentLanguageV0, safeLocale)
	w.Header().Set(WebHTMLLocaleHeaderV0, safeLocale)
}
